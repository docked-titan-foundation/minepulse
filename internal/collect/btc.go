package collect

import (
	"context"
	"fmt"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/btc"
	"github.com/docked-titan-foundation/minepulse/internal/config"
	"github.com/docked-titan-foundation/minepulse/internal/model"
)

const (
	// btcSourceTimeout bounds each pool read so a wedged pool degrades its own
	// panel instead of eating the refresh interval (FR-017).
	btcSourceTimeout = 5 * time.Second
	// ckpoolLogTail is how far back to look for a status cycle. ckpool publishes
	// one about every 20s, and a busy pool interleaves share lines between them.
	ckpoolLogTail = 400
)

// ckpoolNoStatsRemedy is what an operator can actually do about the stock ckpool
// deployment, which logs its status records to a file inside the container
// rather than to the pod's log stream (research R1a).
const ckpoolNoStatsRemedy = "ckpool writes stats to its logfile, not stdout — tail /var/lib/ckpool/logs/ckpool.log into the pod log (chart sidecar), or run public-pool for a stats API"

// hintAddressForWorkers is guidance rather than a fault: public-pool cannot list
// the addresses mining to it, so per-worker rows need one naming the payout
// address. It renders as a remedy, not a warning.
const hintAddressForWorkers = "pass --btc-address <your payout address> for per-worker rows"

// detectTTL is how long a discovery result is reused. Pools are Deployments that
// move rarely, and a cluster-wide pod list every tick would be the one way this
// read-only tool could weigh on the API server (Constitution II: don't perturb
// what you observe). Any failed pool read invalidates it immediately, so a
// rescheduled pod is picked up on the next tick rather than after the TTL.
const detectTTL = 60 * time.Second

// btcCollector gathers the Bitcoin side of a snapshot. It keeps the last good
// stats per pool so a failed read renders stale-but-honest rather than empty.
type btcCollector struct {
	k    *kubeClient
	cfg  config.Config
	last map[string]*model.BitcoinStats // pod key -> last good stats
	seen map[string][]model.BitcoinMiner
	when map[string]time.Time

	// cached discovery
	pods      []poolPod
	scope     string
	scopeWarn string
	detected  time.Time

	// net is the cluster's own addresses, set by the caller each tick so both
	// tabs classify localities against the same reading.
	net clusterNet
}

func newBTCCollector(k *kubeClient, cfg config.Config) *btcCollector {
	return &btcCollector{
		k:    k,
		cfg:  cfg,
		last: map[string]*model.BitcoinStats{},
		seen: map[string][]model.BitcoinMiner{},
		when: map[string]time.Time{},
	}
}

// Gather returns the Bitcoin view, or nil when collection is disabled or no pool
// exists. It never returns an error: a Bitcoin problem is a warning on the
// snapshot, never a failed snapshot (Constitution III).
func (c *btcCollector) Gather(ctx context.Context, now time.Time) (*model.BitcoinView, []string) {
	if c.cfg.NoBTC {
		return nil, nil
	}

	var warnings []string
	pods, scope, warn := c.discover(ctx, now)
	if warn != "" {
		warnings = append(warnings, warn)
	}
	view := &model.BitcoinView{Scope: scope}
	if len(pods) == 0 {
		view.Note = "no Bitcoin pool found in " + scope
		return view, warnings
	}

	for i := range pods {
		p, w := c.gatherPool(ctx, &pods[i], now)
		view.Pools = append(view.Pools, p)
		warnings = append(warnings, w...)
		if p.Source == model.SourceNone && len(w) > 0 {
			// The pod answered nothing and complained: it may have been
			// rescheduled or replaced, so rediscover on the next tick.
			c.detected = time.Time{}
		}
	}
	return view, warnings
}

// discover returns the detected pools, reusing the previous result until the TTL
// expires or a read failure invalidates it.
func (c *btcCollector) discover(ctx context.Context, now time.Time) ([]poolPod, string, string) {
	if !c.detected.IsZero() && now.Sub(c.detected) < detectTTL {
		// The narrowed-search warning still applies while the result is reused —
		// it describes the search, not the moment it ran.
		return c.pods, c.scope, c.scopeWarn
	}
	pods, scope, warn := c.k.detectPools(ctx, c.cfg)
	c.pods, c.scope, c.scopeWarn, c.detected = pods, scope, warn, now
	return pods, scope, warn
}

func (c *btcCollector) gatherPool(ctx context.Context, pp *poolPod, now time.Time) (model.BitcoinPool, []string) {
	out := model.BitcoinPool{
		Impl: pp.impl, Namespace: pp.namespace, Pod: pp.pod, Node: pp.node,
		Phase: pp.phase, Running: pp.running,
		Endpoint: c.k.bitcoinEndpoint(ctx, *pp, c.net),
		Source:   model.SourceNone, Detail: model.DetailTotals,
	}
	if !pp.start.IsZero() {
		out.Uptime = time.Since(pp.start)
	}

	key := pp.namespace + "/" + pp.pod
	if !pp.running {
		out.Note = "pool pod is not Running"
		c.attachLastKnown(&out, key)
		return out, nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.sourceTimeout())
	defer cancel()

	var (
		stats  *model.BitcoinStats
		miners []model.BitcoinMiner
		detail model.DetailLevel
		src    model.StatsSource
		note   string
		warns  []string
	)

	switch pp.impl {
	case model.ImplPublicPool:
		stats, miners, detail, note, warns = c.gatherPublicPool(ctx, pp)
		if stats != nil {
			src = model.SourceAPI
		}
	case model.ImplCkpool:
		stats, miners, detail, note, warns = c.gatherCkpool(ctx, pp)
		if stats != nil {
			src = model.SourceLogs
		}
	default:
		out.Note = "unrecognized pool implementation — image " + pp.image
		out.Remedy = "minepulse reads public-pool and ckpool; tell us what this one is"
		c.attachLastKnown(&out, key)
		return out, nil
	}

	out.Note = note
	if note == hintAddressForWorkers {
		// Guidance, not a fault: carry it as a remedy so it does not read as a
		// warning about a pool that is working fine (standard P8).
		out.Note, out.Remedy = "", note
	}
	if pp.impl == model.ImplCkpool && stats == nil {
		out.Remedy = ckpoolNoStatsRemedy
	}

	if stats == nil {
		// Nothing readable this tick: show the last good values, marked stale,
		// exactly as the Monero pool panel does.
		c.attachLastKnown(&out, key)
		return out, warns
	}

	out.Source = src
	out.Detail = detail
	out.Stats = stats
	out.Miners = miners
	out.AsOf = now
	c.last[key] = stats
	c.seen[key] = miners
	c.when[key] = now
	return out, warns
}

// gatherPublicPool reads the live per-address endpoint when an address is
// configured, and the pool-wide totals otherwise. The totals are cached five
// minutes upstream, so they are only ever the headline when nothing better exists.
func (c *btcCollector) gatherPublicPool(ctx context.Context, pp *poolPod) (
	*model.BitcoinStats, []model.BitcoinMiner, model.DetailLevel, string, []string) {

	var warns []string

	totals, err := c.k.proxyGetIn(ctx, pp.namespace, pp.pod, pp.apiPort, "api/pool")
	var poolStats *model.BitcoinStats
	if err == nil {
		if s, e := btc.ParsePool(totals, time.Now()); e == nil {
			poolStats = s
		} else {
			warns = append(warns, "public-pool /api/pool: "+e.Error())
		}
	} else {
		warns = append(warns, "public-pool /api/pool unreachable: "+err.Error())
	}

	if c.cfg.BTCAddress == "" {
		if poolStats == nil {
			return nil, nil, model.DetailTotals, "public-pool API unreachable", warns
		}
		// Not a problem, an instruction: the panel is complete for what it can
		// know without a payout address. gatherPool turns this into a remedy.
		return poolStats, nil, model.DetailTotals, hintAddressForWorkers, warns
	}

	body, err := c.k.proxyGetIn(ctx, pp.namespace, pp.pod, pp.apiPort,
		"api/client/"+c.cfg.BTCAddress)
	if err != nil {
		warns = append(warns, "public-pool /api/client unreachable: "+err.Error())
		if poolStats == nil {
			return nil, nil, model.DetailTotals, "public-pool API unreachable", warns
		}
		return poolStats, nil, model.DetailTotals, "pool-wide totals only (per-address read failed)", warns
	}
	stats, miners, err := btc.ParseClient(body)
	if err != nil {
		warns = append(warns, "public-pool /api/client: "+err.Error())
		if poolStats == nil {
			return nil, nil, model.DetailTotals, "public-pool returned an unreadable response", warns
		}
		return poolStats, nil, model.DetailTotals, "pool-wide totals only (per-address response unreadable)", warns
	}

	// Carry the chain-level figures from /api/pool onto the live per-address view.
	if poolStats != nil {
		stats.BlockHeight = poolStats.BlockHeight
		stats.BlocksFound = poolStats.BlocksFound
		stats.TotalsAsOf = poolStats.TotalsAsOf
	}
	if len(miners) == 0 {
		return stats, nil, model.DetailTotals, "no workers connected for " + shortAddr(c.cfg.BTCAddress), warns
	}
	return stats, miners, model.DetailDevice, "", warns
}

// gatherCkpool parses ckpool's status records out of the pod log. They are only
// there if the operator routed ckpool's logfile into the log stream, so their
// absence is an expected, explained outcome rather than a failure (FR-009a).
func (c *btcCollector) gatherCkpool(ctx context.Context, pp *poolPod) (
	*model.BitcoinStats, []model.BitcoinMiner, model.DetailLevel, string, []string) {

	logs, err := c.k.podLogsIn(ctx, pp.namespace, pp.pod, pp.container, ckpoolLogTail)
	if err != nil {
		return nil, nil, model.DetailTotals, "ckpool log unreadable: " + err.Error(),
			[]string{"ckpool logs unreadable: " + err.Error()}
	}
	stats, miners, err := btc.ParseStatusLog(logs)
	if err != nil {
		return nil, nil, model.DetailTotals, "ckpool log unparseable", []string{"ckpool log: " + err.Error()}
	}
	if stats == nil {
		if len(logs) == 0 {
			return nil, nil, model.DetailTotals, "ckpool has not logged anything yet", nil
		}
		return nil, nil, model.DetailTotals, "ckpool publishes no stats to its log stream", nil
	}
	if len(miners) == 0 {
		return stats, nil, model.DetailTotals, "no miners connected", nil
	}
	return stats, miners, model.DetailAddress,
		"per-device detail unavailable from ckpool logs (addresses only)", nil
}

// attachLastKnown fills a pool panel with the previous tick's values, marked
// stale, so a transient failure does not blank the view (VR-4).
func (c *btcCollector) attachLastKnown(out *model.BitcoinPool, key string) {
	stats, ok := c.last[key]
	if !ok {
		return
	}
	out.Stats = stats
	out.Miners = c.seen[key]
	out.AsOf = c.when[key]
	out.Stale = true
	if len(out.Miners) > 0 {
		out.Detail = model.DetailAddress
		if out.Impl == model.ImplPublicPool {
			out.Detail = model.DetailDevice
		}
	}
}

func (c *btcCollector) sourceTimeout() time.Duration {
	if c.cfg.Interval > 0 && c.cfg.Interval < btcSourceTimeout {
		return c.cfg.Interval
	}
	return btcSourceTimeout
}

// shortAddr truncates a payout address for display. Addresses are never logged
// (Constitution VII); this is only for panel text.
func shortAddr(a string) string {
	if len(a) <= 14 {
		return a
	}
	return fmt.Sprintf("%s…%s", a[:8], a[len(a)-4:])
}
