package collect

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/config"
	"github.com/docked-titan-foundation/minepulse/internal/model"
	"github.com/docked-titan-foundation/minepulse/internal/pool"
	"github.com/docked-titan-foundation/minepulse/internal/xmrig"
)

// historyCap is how many CPU samples each node's sparkline retains.
const historyCap = 120

// clusterSource is the live, read-only Source. It composes the kube, metrics,
// XMRig, and pool collectors; a failure of any one becomes a warning or an
// "unavailable" field rather than aborting the snapshot (Constitution III).
type clusterSource struct {
	cfg            config.Config
	k              *kubeClient
	poolClient     *pool.Client
	btc            *btcCollector
	rings          map[string]*model.Ring
	wallet         string           // resolved once
	configuredPool string           // resolved from miner args (for donate detection)
	lastPool       *model.PoolStats // last successful pool fetch (for stale display)
}

func newClusterSource(cfg config.Config) (Source, error) {
	k, err := newKubeClient(cfg)
	if err != nil {
		return nil, err
	}
	c := &clusterSource{
		cfg:    cfg,
		k:      k,
		rings:  map[string]*model.Ring{},
		wallet: cfg.Wallet,
	}
	if !cfg.NoPool {
		c.poolClient = pool.NewClient(cfg.PoolAPIURL)
	}
	if !cfg.NoBTC {
		c.btc = newBTCCollector(k, cfg)
	}
	return c, nil
}

func (c *clusterSource) Close() error { return nil }

func (c *clusterSource) ring(node string) *model.Ring {
	r, ok := c.rings[node]
	if !ok {
		r = model.NewRing(historyCap)
		c.rings[node] = r
	}
	return r
}

func (c *clusterSource) Gather(ctx context.Context) (*model.Snapshot, error) {
	miners, err := c.k.listMiners(ctx)
	if err != nil {
		// No pods at all is a fatal-for-this-tick error (can't reach the API);
		// an empty list is not (handled below).
		return nil, fmt.Errorf("list miner pods: %w", err)
	}

	snap := &model.Snapshot{Timestamp: time.Now()}
	if len(miners) == 0 {
		snap.Warnings = append(snap.Warnings,
			fmt.Sprintf("no pods match %q in namespace %q — is the miner installed?", c.cfg.Selector, c.cfg.Namespace))
	}

	// Resolve wallet + configured pool from the miners' args, once.
	nodes := make([]string, 0, len(miners))
	for i := range miners {
		m := &miners[i]
		if m.node != "" {
			nodes = append(nodes, m.node)
		}
		if c.configuredPool == "" && m.pool != "" {
			c.configuredPool = m.pool
		}
		if c.wallet == "" && m.wallet != "" {
			c.wallet = m.wallet
		}
	}

	capMilli := c.k.nodeCPUCapacity(ctx, nodes)
	nodeUsed, nErr := c.k.nodeCPUUsed(ctx)
	podCPU, pErr := c.k.podCPUUsed(ctx)
	metricsOK := nErr == nil && pErr == nil
	if !metricsOK {
		snap.Warnings = append(snap.Warnings, "CPU metrics unavailable (metrics-server?)")
	}

	for i := range miners {
		m := &miners[i]
		ns := model.NodeStatus{
			Node: m.node, Pod: m.pod, Phase: m.phase, Restarts: m.restarts,
			Image: m.image, StatsSource: model.SourceNone,
		}
		if !m.start.IsZero() {
			ns.Uptime = time.Since(m.start)
		}

		if m.running {
			if ms, src := c.gatherMining(ctx, m); ms != nil {
				ns.Mining = ms
				ns.StatsSource = src
				if src == model.SourceLogs {
					ns.Note = "XMRig API unavailable — stats from logs"
				}
			}
		} else {
			ns.Note = "pod not Running"
		}

		if metricsOK {
			if capM := capMilli[m.node]; capM > 0 {
				used := nodeUsed[m.node]
				cpu := &model.CPUSample{
					T: snap.Timestamp, MinerMilli: podCPU[m.pod],
					NodeUsedMilli: used, NodeCapacityMilli: capM,
					FreePct: float64(capM-used) / float64(capM) * 100,
				}
				r := c.ring(m.node)
				r.Append(*cpu)
				ns.CPU = cpu
				ns.History = r
			}
		}

		snap.Nodes = append(snap.Nodes, ns)
	}

	sort.Slice(snap.Nodes, func(i, j int) bool { return snap.Nodes[i].Node < snap.Nodes[j].Node })

	c.gatherPool(ctx, snap)

	// The Bitcoin side is collected every tick regardless of which tab is
	// showing, so switching tabs never displays staler data than the timestamp
	// claims (FR-005). It can only add warnings, never fail the snapshot.
	if c.btc != nil {
		view, warns := c.btc.Gather(ctx, snap.Timestamp)
		snap.Bitcoin = view
		snap.Warnings = append(snap.Warnings, warns...)
	}

	snap.Summarize()
	return snap, nil
}

// gatherMining tries the XMRig HTTP API (via pod proxy) then falls back to logs,
// honoring --xmrig-api.
func (c *clusterSource) gatherMining(ctx context.Context, m *minerPod) (*model.MiningStats, model.StatsSource) {
	if c.cfg.XMRigAPI != config.XMRigOff {
		if sumB, err := c.k.proxyGet(ctx, m.pod, "1/summary"); err == nil {
			if sum, e := xmrig.ParseSummary(sumB); e == nil {
				active := -1
				if beB, e2 := c.k.proxyGet(ctx, m.pod, "2/backends"); e2 == nil {
					if bs, e3 := xmrig.ParseBackends(beB); e3 == nil {
						active = xmrig.ActiveCPUThreads(bs)
					}
				}
				ms := sum.ToMiningStats(active, c.configuredPool)
				return &ms, model.SourceAPI
			}
		}
		if c.cfg.XMRigAPI == config.XMRigOn {
			return nil, model.SourceNone // API required but unreachable
		}
	}
	if logB, err := c.k.podLogs(ctx, m.pod, m.container, 200); err == nil {
		if ms := ParseLogs(string(logB), c.configuredPool); ms != nil {
			return ms, model.SourceLogs
		}
	}
	return nil, model.SourceNone
}

// gatherPool fetches pool-side earnings, showing the last-known value marked
// stale if the pool API is unreachable this tick.
func (c *clusterSource) gatherPool(ctx context.Context, snap *model.Snapshot) {
	if c.poolClient == nil || c.wallet == "" {
		return
	}
	st, err := c.poolClient.MinerStats(ctx, c.wallet)
	if err != nil {
		if c.lastPool != nil {
			stale := *c.lastPool
			stale.Stale = true
			snap.Pool = &stale
		}
		snap.Warnings = append(snap.Warnings, "pool API unreachable: "+err.Error())
		return
	}
	ps := st.ToPoolStats(c.wallet, snap.Timestamp)
	c.lastPool = &ps
	snap.Pool = &ps
}
