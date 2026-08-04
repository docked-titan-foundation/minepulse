package render

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/docked-titan-foundation/minepulse/internal/config"
	"github.com/docked-titan-foundation/minepulse/internal/model"
)

func sampleSnapshot() *model.Snapshot {
	hist := model.NewRing(20)
	for _, v := range []float64{40, 35, 20, 12, 18, 30} {
		hist.Append(model.CPUSample{FreePct: v})
	}
	s := &model.Snapshot{
		Timestamp: time.Now(),
		Nodes: []model.NodeStatus{
			{
				Node: "andromeda", Phase: "Running", StatsSource: model.SourceAPI,
				Mining: &model.MiningStats{
					Hashrate60s: 377, ThreadsActive: 6, ThreadsTotal: 8,
					SharesGood: 42, SharesTotal: 44, Pool: "pool.supportxmr.com:443", Connected: true,
				},
				CPU:     &model.CPUSample{MinerMilli: 5000, NodeUsedMilli: 7000, NodeCapacityMilli: 8000, FreePct: 12.5},
				History: hist,
			},
			{
				Node: "orion", Phase: "Running", StatsSource: model.SourceAPI,
				Mining: &model.MiningStats{
					Hashrate60s: 240, ThreadsActive: 4, ThreadsTotal: 6,
					Pool: "donate.v2.xmrig.com:3333", DonateFallback: true, Connected: true,
				},
			},
		},
		Pool: &model.PoolStats{ReportedHashrate: 662, AmountDueXMR: 0.00312, AmountPaidXMR: 0.184},
		Endpoint: &model.PoolEndpoint{
			URL: "pool.supportxmr.com:443", IP: "116.202.180.221",
			Brand: "SupportXMR", Mode: model.ModeShared,
			Locality: model.LocalityExternal, Basis: "public address, no cluster object matches",
			Diverged: true,
		},
	}
	s.Summarize()
	return s
}

func TestTUIViewRendersKeyContent(t *testing.T) {
	m := tuiModel{snap: sampleSnapshot(), updated: time.Now()}
	out := m.View()
	for _, want := range []string{"minepulse", "cluster", "andromeda", "orion", "DONATE", "pool", "quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("TUI View missing %q\n---\n%s", want, out)
		}
	}
}

func TestTUIViewNilSnapshot(t *testing.T) {
	m := tuiModel{}
	if out := m.View(); !strings.Contains(out, "gathering") {
		t.Errorf("nil-snapshot view should say gathering, got:\n%s", out)
	}
}

// The tab strip is always visible, and m/b/tab only change what is drawn — no
// gather, no pause change, no new timestamp (FR-004).
func TestTUITabsSwitchBodyOnly(t *testing.T) {
	snap := sampleSnapshot()
	snap.Bitcoin = &model.BitcoinView{Scope: "all namespaces", Pools: []model.BitcoinPool{addressPool()}}
	updated := time.Now()
	m := tuiModel{snap: snap, updated: updated}

	monero := m.View()
	if !strings.Contains(monero, "monero") || !strings.Contains(monero, "bitcoin") {
		t.Errorf("tab strip missing\n---\n%s", monero)
	}
	// Each coin's mark is on its tab whichever tab is active.
	if !strings.Contains(monero, xmrIcon) || !strings.Contains(monero, btcIcon) {
		t.Errorf("tab strip must carry both coin icons\n---\n%s", monero)
	}
	if !strings.Contains(monero, "andromeda") {
		t.Errorf("Monero tab must show the node table\n---\n%s", monero)
	}
	if !strings.Contains(monero, "m/b coin") {
		t.Errorf("footer must advertise the coin keys\n---\n%s", monero)
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	bm, ok := next.(tuiModel)
	if !ok {
		t.Fatalf("Update returned %T, want tuiModel", next)
	}
	if bm.tab != tabBitcoin {
		t.Fatalf("pressing b selected tab %v, want bitcoin", bm.tab)
	}
	if !bm.updated.Equal(updated) || bm.gathering {
		t.Error("switching tabs must not trigger a gather or change the update time")
	}
	btc := bm.View()
	if !strings.Contains(btc, "ckpool") || strings.Contains(btc, "andromeda") {
		t.Errorf("Bitcoin tab must replace the Monero body\n---\n%s", btc)
	}

	back, _ := bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if got := back.(tuiModel).View(); got != monero {
		t.Errorf("returning to Monero must render the same view\n--- want\n%s\n--- got\n%s", monero, got)
	}

	toggled, _ := bm.Update(tea.KeyMsg{Type: tea.KeyTab})
	if toggled.(tuiModel).tab != tabMonero {
		t.Error("tab key must toggle back to Monero")
	}
}

// --tab picks the coin the dashboard opens on; the keys still work from there.
func TestTabForOpensOnConfiguredCoin(t *testing.T) {
	if got := tabFor(config.TabBitcoin); got != tabBitcoin {
		t.Errorf("tabFor(bitcoin) = %v, want bitcoin", got)
	}
	if got := tabFor(config.TabMonero); got != tabMonero {
		t.Errorf("tabFor(monero) = %v, want monero", got)
	}
	// The command layer rejects unknown values; the view still refuses to panic.
	if got := tabFor(config.Tab("dogecoin")); got != tabMonero {
		t.Errorf("tabFor(unknown) = %v, want monero", got)
	}

	snap := sampleSnapshot()
	snap.Bitcoin = &model.BitcoinView{Scope: "all namespaces", Pools: []model.BitcoinPool{addressPool()}}
	m := tuiModel{snap: snap, updated: time.Now(), tab: tabFor(config.TabBitcoin)}
	if out := m.View(); !strings.Contains(out, "ckpool") {
		t.Errorf("opening on bitcoin must render the Bitcoin body\n---\n%s", out)
	}
}

// With no Bitcoin data the tab still renders, saying which of the empty states
// it is in rather than showing an empty box.
func TestTUIBitcoinTabWithoutData(t *testing.T) {
	m := tuiModel{snap: sampleSnapshot(), updated: time.Now(), tab: tabBitcoin}
	out := m.View()
	if !strings.Contains(out, "no Bitcoin pool detected") {
		t.Errorf("Bitcoin tab without data = %q", out)
	}
}

func TestSparkline(t *testing.T) {
	if got := Sparkline([]float64{0, 50, 100}, 0, 100, 10); got != "▁▄█" {
		t.Errorf("Sparkline = %q, want ▁▄█", got)
	}
	if got := Sparkline(nil, 0, 100, 10); got != "" {
		t.Errorf("empty sparkline = %q, want empty", got)
	}
	// width caps to most recent points
	if got := Sparkline([]float64{0, 0, 0, 100}, 0, 100, 2); got != "▁█" {
		t.Errorf("capped sparkline = %q, want ▁█", got)
	}
}

// The bar's thresholds are what the cluster gauge's were, and its no-metrics
// case is the unavailable mark — an empty trough would read as "0% free", which
// is a much more alarming claim than "we could not measure it" (006-FR-006).
func TestCPUBar(t *testing.T) {
	for _, tc := range []struct {
		pct   float64
		want  string
		style string
	}{
		{-1, unavailable, ""},
		{0, "░░░░░░░░ 0%", "bad"},
		// A node with headroom left never draws an empty trough, however little:
		// that is the picture a saturated node paints, and they must differ.
		{10, "█░░░░░░░ 10%", "bad"},
		{12.5, "█░░░░░░░ 12%", "bad"},
		{50, "████░░░░ 50%", "good"},
		{100, "████████ 100%", "good"},
		{150, "████████ 150%", "good"}, // never overflows its cell
	} {
		if got := stripANSI(cpuBar(tc.pct, 8)); got != tc.want {
			t.Errorf("cpuBar(%v) = %q, want %q", tc.pct, got, tc.want)
		}
	}
	// Color tracks the same thresholds the gauge used; compare the styled
	// output rather than the plain text, since that is where it lives.
	if cpuBar(10, 8) == cpuBar(50, 8) {
		t.Error("a starved node and an idle one must not render identically")
	}
}

// Every field of the identity line degrades on its own, so a pool minepulse only
// half-recognizes still produces a line rather than nothing (007-FR-001).
func TestPoolLine(t *testing.T) {
	full := &model.PoolEndpoint{
		URL: "mining-pool.bitcoin.svc:3333", IP: "10.43.7.12",
		Brand: "public-pool", Mode: model.ModeSolo, Locality: model.LocalityInternal,
	}
	if got, want := PoolLine(full), "[internal] mining-pool.bitcoin.svc:3333 - public-pool - solo - 10.43.7.12"; got != want {
		t.Errorf("PoolLine =\n%s\nwant\n%s", got, want)
	}

	// Nothing known but the address: four marks, still a line.
	bare := &model.PoolEndpoint{URL: "10.43.7.12:3333", Locality: model.LocalityUnknown, Mode: model.ModeUnknown}
	if got, want := PoolLine(bare), "[—] 10.43.7.12:3333 - — - — - —"; got != want {
		t.Errorf("PoolLine(bare) = %q, want %q", got, want)
	}

	if got := PoolLine(nil); got != "" {
		t.Errorf("PoolLine(nil) = %q, want empty", got)
	}
}

// P14: color is a second channel, never the only one. Whatever the styled line
// paints, stripping the escapes must leave exactly the text `stream` prints —
// so a monochrome terminal, a pipe and a screen reader all lose nothing.
func TestStyledPoolLineSaysNothingExtra(t *testing.T) {
	for _, loc := range []model.Locality{
		model.LocalityInternal, model.LocalityExternal, model.LocalityUnknown,
	} {
		e := &model.PoolEndpoint{
			URL: "mining-pool.bitcoin.svc:3333", IP: "10.43.7.12",
			Brand: "public-pool", Mode: model.ModeSolo, Locality: loc,
		}
		if got, want := stripANSI(poolLineStyled(e)), PoolLine(e); got != want {
			t.Errorf("styled line for %q reads %q, want %q", loc, got, want)
		}
	}
	if poolLineStyled(nil) != "" {
		t.Error("no endpoint, no line")
	}
}

// Each locality gets its own color, and "outside the cluster" must not borrow
// the warning color: for Monero it is the ordinary case, not a fault.
func TestLocalityColorsAreDistinct(t *testing.T) {
	in := localityStyle(model.LocalityInternal)
	out := localityStyle(model.LocalityExternal)
	unknown := localityStyle(model.LocalityUnknown)

	seen := map[lipgloss.TerminalColor]string{}
	for name, st := range map[string]lipgloss.Style{
		"internal": in, "external": out, "unknown": unknown,
	} {
		fg := st.GetForeground()
		if prev, dup := seen[fg]; dup {
			t.Errorf("%s and %s share a color; the marker stops distinguishing them", name, prev)
		}
		seen[fg] = name
	}
	if out.GetForeground() == warnSt.GetForeground() {
		t.Error("external must not use the warning color — it is the normal case, not a fault")
	}
}

// A clean run should look calm rather than green, and a node that is not mining
// when it should be is the thing worth coloring.
func TestMiningFractionColors(t *testing.T) {
	if stripANSI(miningFraction(3, 4)) != "3/4" || stripANSI(miningFraction(4, 4)) != "4/4" {
		t.Error("color must not alter the text")
	}
	if miningFraction(4, 4) == miningFraction(3, 4) {
		t.Error("all-mining and partly-mining must not render identically")
	}
	// Nothing installed is not a fault to flag.
	if miningFraction(0, 0) == miningFraction(0, 1) {
		t.Error("no nodes at all must not read as a failure")
	}
}
