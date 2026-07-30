package render

import (
	"strings"
	"testing"
	"time"

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
