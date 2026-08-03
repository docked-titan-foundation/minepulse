package render

import (
	"strings"
	"testing"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

func devicePool() model.BitcoinPool {
	s := model.NewBitcoinStats()
	s.HashrateWindow = "now"
	s.Hashrate1m = 1.68e12
	s.Workers = 2
	s.BestShare = 4.1e9
	s.BlockHeight = 907213
	return model.BitcoinPool{
		Impl: model.ImplPublicPool, Namespace: "bitcoin", Pod: "mining-pool-abc",
		Node: "andromeda", Phase: "Running", Running: true, Uptime: 73 * time.Hour,
		Source: model.SourceAPI, Detail: model.DetailDevice, Stats: s, AsOf: time.Now(),
		Miners: []model.BitcoinMiner{
			{Name: "bitaxe-01", Hashrate: 480e9, BestDifficulty: 1.2e9, LastSeen: time.Now().Add(-11 * time.Second)},
			{Name: "nerdqaxe-01", Hashrate: 1.2e12, BestDifficulty: 8.9e8, LastSeen: time.Now().Add(-13 * time.Second)},
		},
	}
}

func addressPool() model.BitcoinPool {
	s := model.NewBitcoinStats()
	s.HashrateWindow = "1m"
	s.Hashrate1m = 480e12
	s.Users, s.Workers, s.WorkersIdle = 1, 2, 1
	s.Accepted, s.Rejected, s.BestShare = 12483, 3, 1.2e9
	s.NetworkDiffPct = 0.02
	return model.BitcoinPool{
		Impl: model.ImplCkpool, Namespace: "bitcoin-solo", Pod: "mining-pool-xyz",
		Node: "orion", Phase: "Running", Running: true, Uptime: 142 * time.Hour,
		Source: model.SourceLogs, Detail: model.DetailAddress, Stats: s, AsOf: time.Now(),
		Note: "per-device detail unavailable from ckpool logs (addresses only)",
		Miners: []model.BitcoinMiner{
			{
				Name: "bc1qexampleaddressxxxxxxxxxxxxxxxxxxxxxxx", Hashrate: 480e12,
				BestDifficulty: 1.2e9, BestEver: 4.1e9, Workers: 2, Shares: 12483,
				LastSeen: time.Now().Add(-31 * time.Second),
			},
		},
	}
}

// public-pool reaches per-device rows and says so by listing each worker.
func TestBitcoinBodyDeviceDetail(t *testing.T) {
	out := bitcoinBody(&model.BitcoinView{Scope: "all namespaces", Pools: []model.BitcoinPool{devicePool()}})
	for _, want := range []string{"public-pool", "mining-pool-abc", "1.68 TH/s", "WORKER", "bitaxe-01", "nerdqaxe-01", "via API"} {
		if !strings.Contains(out, want) {
			t.Errorf("device view missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "ADDRESS") {
		t.Errorf("device view must not use the address table\n---\n%s", out)
	}
}

// ckpool stops at payout addresses, and the panel names what is missing rather
// than implying the devices vanished.
func TestBitcoinBodyAddressDetail(t *testing.T) {
	out := bitcoinBody(&model.BitcoinView{Scope: "all namespaces", Pools: []model.BitcoinPool{addressPool()}})
	for _, want := range []string{"ckpool", "480.00 TH/s", "1 user", "2 workers (1 idle)", "ADDRESS", "via logs", "per-device detail unavailable"} {
		if !strings.Contains(out, want) {
			t.Errorf("address view missing %q\n---\n%s", want, out)
		}
	}
	// The payout address is truncated on screen (Constitution VII).
	if strings.Contains(out, "bc1qexampleaddressxxxxxxxxxxxxxxxxxxxxxxx") {
		t.Errorf("address must be truncated in the TUI\n---\n%s", out)
	}
	if !strings.Contains(out, "…") {
		t.Errorf("expected a truncated address\n---\n%s", out)
	}
}

// A detected pool with no readable stats keeps its identity and carries the
// remedy — the stock-ckpool case (FR-009a).
func TestBitcoinBodyNoStatsSource(t *testing.T) {
	p := model.BitcoinPool{
		Impl: model.ImplCkpool, Namespace: "bitcoin", Pod: "mining-pool-xyz", Node: "orion",
		Phase: "Running", Running: true, Uptime: time.Hour, Source: model.SourceNone,
		Detail: model.DetailTotals,
		Note:   "ckpool publishes no stats to its log stream",
		Remedy: "tail /var/lib/ckpool/logs/ckpool.log into the pod log",
	}
	out := bitcoinBody(&model.BitcoinView{Scope: "all namespaces", Pools: []model.BitcoinPool{p}})
	for _, want := range []string{"ckpool", "mining-pool-xyz", "no stats source", "publishes no stats", "tail /var/lib/ckpool"} {
		if !strings.Contains(out, want) {
			t.Errorf("no-stats view missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "0 H/s") {
		t.Errorf("a pool with no stats must never render a zero hashrate\n---\n%s", out)
	}
}

// The three "nothing to show" states must read differently from each other.
func TestBitcoinBodyEmptyStates(t *testing.T) {
	none := bitcoinBody(nil)
	if !strings.Contains(none, "no Bitcoin pool detected") {
		t.Errorf("nil view = %q", none)
	}
	empty := bitcoinBody(&model.BitcoinView{Scope: "namespace bitcoin", Note: "no Bitcoin pool found in namespace bitcoin"})
	if !strings.Contains(empty, "no Bitcoin pool found in namespace bitcoin") {
		t.Errorf("empty view = %q", empty)
	}
	if none == empty {
		t.Error("undetected and searched-but-absent must not render identically")
	}
}

// A failed read shows the last known values, marked stale with their age.
func TestBitcoinBodyStale(t *testing.T) {
	p := addressPool()
	p.Stale = true
	p.AsOf = time.Now().Add(-4 * time.Minute)
	out := bitcoinBody(&model.BitcoinView{Scope: "all namespaces", Pools: []model.BitcoinPool{p}})
	if !strings.Contains(out, "stale") {
		t.Errorf("stale panel must say so\n---\n%s", out)
	}
	if !strings.Contains(out, "480.00 TH/s") {
		t.Errorf("stale panel must keep the last known values\n---\n%s", out)
	}
}

// Both pools appear when both run; neither hides the other (FR-007).
func TestBitcoinBodyBothPools(t *testing.T) {
	out := bitcoinBody(&model.BitcoinView{
		Scope: "all namespaces",
		Pools: []model.BitcoinPool{devicePool(), addressPool()},
	})
	if !strings.Contains(out, "public-pool") || !strings.Contains(out, "ckpool") {
		t.Errorf("both implementations must be listed\n---\n%s", out)
	}
	if !strings.Contains(out, "searched all namespaces") {
		t.Errorf("the searched scope must be stated\n---\n%s", out)
	}
}

// P6: one share format everywhere, and no invented zeros.
func TestShares(t *testing.T) {
	tests := []struct {
		good, bad float64
		want      string
	}{
		{12483, 3, "12483✓/3✗"},
		{42, 0, "42✓/0✗"},
		{480, model.Unknown, "480✓"},        // public-pool reports no rejects
		{model.Unknown, model.Unknown, "—"}, // nothing known
		{model.Unknown, 2, "—/2✗"},          // rejects only
		{2_500_000, 1, "2.50M✓/1✗"},         // large counts stay readable
	}
	for _, tt := range tests {
		if got := Shares(tt.good, tt.bad); got != tt.want {
			t.Errorf("Shares(%v, %v) = %q, want %q", tt.good, tt.bad, got, tt.want)
		}
	}
}

func TestFormatHelpers(t *testing.T) {
	tests := []struct{ got, want string }{
		{Hashrate(480e12), "480.00 TH/s"},
		{Hashrate(1.2e15), "1.20 PH/s"},
		{Hashrate(480e9), "480.00 GH/s"},
		{Hashrate(377), "377 H/s"},
		{Hashrate(model.Unknown), "—"},
		{Difficulty(1.2e9), "1.20 G"},
		{Difficulty(4.1e12), "4.10 T"},
		{Difficulty(model.Unknown), "—"},
		{Count(12483), "12483"},
		{Count(model.Unknown), "—"},
		{ShortAddress("bc1qexampleaddressxxxxxxxxxxxxxxxxxxxxxxx"), "bc1qexam…xxxx"},
		{ShortAddress("short"), "short"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}
