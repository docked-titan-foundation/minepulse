package model

import (
	"math"
	"testing"
)

func TestSummarizeAggregates(t *testing.T) {
	s := &Snapshot{Nodes: []NodeStatus{
		{
			Node:   "a",
			Mining: &MiningStats{Hashrate60s: 300, SharesGood: 10, SharesTotal: 12, Connected: true},
			CPU:    &CPUSample{MinerMilli: 3000, NodeUsedMilli: 6000, NodeCapacityMilli: 8000},
		},
		{
			Node:   "b",
			Mining: &MiningStats{Hashrate60s: 200, SharesGood: 5, SharesTotal: 5, Connected: true, DonateFallback: true},
			CPU:    &CPUSample{MinerMilli: 1000, NodeUsedMilli: 3000, NodeCapacityMilli: 4000},
		},
		{
			Node:  "c", // crashloop: no mining, no cpu
			Phase: "CrashLoopBackOff",
		},
	}}

	s.Summarize()
	c := s.Cluster

	if c.NodesTotal != 3 || c.NodesMining != 2 {
		t.Errorf("nodes: total=%d mining=%d, want 3/2", c.NodesTotal, c.NodesMining)
	}
	if c.TotalHashrate != 500 {
		t.Errorf("total hashrate = %v, want 500", c.TotalHashrate)
	}
	if c.AcceptedShares != 15 {
		t.Errorf("accepted = %d, want 15", c.AcceptedShares)
	}
	if c.RejectedShares != 2 { // a: 12-10=2, b: 5-5=0
		t.Errorf("rejected = %d, want 2", c.RejectedShares)
	}
	if c.MinerCPUMilli != 4000 {
		t.Errorf("miner cpu = %d, want 4000", c.MinerCPUMilli)
	}
	// free = (8000-6000)+(4000-3000)=3000 of 12000 capacity = 25%
	if math.Abs(c.NodeCPUFreePct-25) > 1e-9 {
		t.Errorf("free pct = %v, want 25", c.NodeCPUFreePct)
	}
}

func TestSummarizeFreePctUnavailableWhenNoMetrics(t *testing.T) {
	s := &Snapshot{Nodes: []NodeStatus{
		{Node: "a", Mining: &MiningStats{Hashrate60s: 100, Connected: true}}, // no CPU
	}}
	s.Summarize()
	if s.Cluster.NodeCPUFreePct != Unknown {
		t.Errorf("free pct = %v, want %v (unavailable)", s.Cluster.NodeCPUFreePct, Unknown)
	}
}

func TestSummarizeSkipsDisconnectedMiner(t *testing.T) {
	s := &Snapshot{Nodes: []NodeStatus{
		{Node: "a", Mining: &MiningStats{Hashrate60s: 100, Connected: false}},
	}}
	s.Summarize()
	if s.Cluster.NodesMining != 0 || s.Cluster.TotalHashrate != 0 {
		t.Errorf("disconnected miner counted: mining=%d hr=%v", s.Cluster.NodesMining, s.Cluster.TotalHashrate)
	}
}

func TestMinerPctOfNode(t *testing.T) {
	c := CPUSample{MinerMilli: 2000, NodeCapacityMilli: 8000}
	if got := c.MinerPctOfNode(); math.Abs(got-25) > 1e-9 {
		t.Errorf("MinerPctOfNode = %v, want 25", got)
	}
	if got := (CPUSample{}).MinerPctOfNode(); got != 0 {
		t.Errorf("zero-capacity pct = %v, want 0", got)
	}
}
