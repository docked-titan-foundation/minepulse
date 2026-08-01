package render

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

func TestGaugeUnavailable(t *testing.T) {
	if got := gauge("x", -1, 10); !strings.Contains(got, "n/a") {
		t.Errorf("gauge(-1) = %q, want n/a", got)
	}
}
