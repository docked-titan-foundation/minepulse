package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// A fresh stats block must report every figure as unavailable, so a parser that
// sets nothing can never be mistaken for a pool reporting zeros (VR-2).
func TestNewBitcoinStatsIsAllUnknown(t *testing.T) {
	s := NewBitcoinStats()
	floats := map[string]float64{
		"Hashrate1m": s.Hashrate1m, "Hashrate5m": s.Hashrate5m, "Hashrate1h": s.Hashrate1h,
		"Hashrate1d": s.Hashrate1d, "Accepted": s.Accepted, "Rejected": s.Rejected,
		"BestShare": s.BestShare, "NetworkDiffPct": s.NetworkDiffPct, "SPS1m": s.SPS1m,
	}
	for name, v := range floats {
		if v != Unknown {
			t.Errorf("%s = %v, want Unknown (%v)", name, v, Unknown)
		}
	}
	ints := map[string]int{
		"Users": s.Users, "Workers": s.Workers,
		"WorkersIdle": s.WorkersIdle, "Disconnected": s.Disconnected, "BlocksFound": s.BlocksFound,
	}
	for name, v := range ints {
		if v != -1 {
			t.Errorf("%s = %d, want -1", name, v)
		}
	}
	if s.BlockHeight != -1 {
		t.Errorf("BlockHeight = %d, want -1", s.BlockHeight)
	}
}

// Detail and Miners must agree: totals means no rows, the other rungs mean rows
// (VR-3). This is the invariant the collectors are held to.
func TestDetailLevelMatchesMiners(t *testing.T) {
	tests := []struct {
		name   string
		detail DetailLevel
		miners []BitcoinMiner
		ok     bool
	}{
		{"totals with no rows", DetailTotals, nil, true},
		{"totals with rows", DetailTotals, []BitcoinMiner{{Name: "x"}}, false},
		{"device with rows", DetailDevice, []BitcoinMiner{{Name: "bitaxe-01"}}, true},
		{"device with no rows", DetailDevice, nil, false},
		{"address with rows", DetailAddress, []BitcoinMiner{{Name: "bc1q"}}, true},
		{"address with no rows", DetailAddress, nil, false},
	}
	for _, tt := range tests {
		p := BitcoinPool{Detail: tt.detail, Miners: tt.miners}
		if got := detailConsistent(p); got != tt.ok {
			t.Errorf("%s: consistent = %v, want %v", tt.name, got, tt.ok)
		}
	}
}

// detailConsistent states VR-3 as code so both the test and future collectors
// have one definition to point at.
func detailConsistent(p BitcoinPool) bool {
	if p.Detail == DetailTotals {
		return len(p.Miners) == 0
	}
	return len(p.Miners) > 0
}

// The JSON contract is additive: without Bitcoin data the key is absent entirely,
// so existing consumers see the record they always saw (FR-015).
func TestSnapshotOmitsBitcoinWhenNil(t *testing.T) {
	b, err := json.Marshal(&Snapshot{Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "bitcoin") {
		t.Errorf("nil Bitcoin must not emit a key, got %s", b)
	}

	b, err = json.Marshal(&Snapshot{Timestamp: time.Now(), Bitcoin: &BitcoinView{Scope: "all"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"bitcoin"`) {
		t.Errorf("non-nil Bitcoin must emit a key, got %s", b)
	}
}

// Summarize is the Monero cluster summary and must ignore Bitcoin entirely (VR-6).
func TestSummarizeIgnoresBitcoin(t *testing.T) {
	s := &Snapshot{
		Nodes: []NodeStatus{{
			Node:   "andromeda",
			Mining: &MiningStats{Hashrate60s: 100, Connected: true, SharesGood: 5, SharesTotal: 5},
		}},
		Bitcoin: &BitcoinView{Pools: []BitcoinPool{{
			Impl:  ImplCkpool,
			Stats: &BitcoinStats{Hashrate1m: 480e12, Workers: 2},
		}}},
	}
	s.Summarize()
	if s.Cluster.TotalHashrate != 100 {
		t.Errorf("TotalHashrate = %v, want 100 (Bitcoin must not be folded in)", s.Cluster.TotalHashrate)
	}
	if s.Cluster.NodesTotal != 1 {
		t.Errorf("NodesTotal = %d, want 1", s.Cluster.NodesTotal)
	}
}
