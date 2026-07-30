// Package model holds the plain-data types minepulse gathers and renders.
// They carry no cluster clients so aggregation and rendering stay pure and
// unit-testable (Constitution IV). JSON tags define the `--output json` contract.
package model

import "time"

// StatsSource indicates where a node's mining stats came from.
type StatsSource string

const (
	SourceAPI  StatsSource = "api"  // XMRig HTTP API
	SourceLogs StatsSource = "logs" // parsed from pod logs (reduced fidelity)
	SourceNone StatsSource = "none" // no mining data available
)

// Sentinel for a numeric field whose value is unknown/unavailable.
const Unknown = -1.0

// Snapshot is one moment's view: the unit of refresh, of one-shot output, and
// of a single `json` line.
type Snapshot struct {
	Timestamp time.Time      `json:"timestamp"`
	Cluster   ClusterSummary `json:"cluster"`
	Nodes     []NodeStatus   `json:"nodes"`
	Pool      *PoolStats     `json:"pool,omitempty"`
	Warnings  []string       `json:"warnings,omitempty"`
}

// ClusterSummary is the aggregate across all mining nodes.
type ClusterSummary struct {
	NodesMining    int     `json:"nodes_mining"`
	NodesTotal     int     `json:"nodes_total"`
	TotalHashrate  float64 `json:"total_hashrate"`
	AcceptedShares int64   `json:"accepted_shares"`
	RejectedShares int64   `json:"rejected_shares"`
	MinerCPUMilli  int64   `json:"miner_cpu_milli"`
	NodeCPUFreePct float64 `json:"node_cpu_free_pct"` // -1 when metrics unavailable
}

// NodeStatus is one miner instance on one node.
type NodeStatus struct {
	Node        string        `json:"node"`
	Pod         string        `json:"pod"`
	Phase       string        `json:"phase"`
	Restarts    int32         `json:"restarts"`
	Image       string        `json:"image,omitempty"`
	Uptime      time.Duration `json:"uptime"`
	Mining      *MiningStats  `json:"mining,omitempty"`
	CPU         *CPUSample    `json:"cpu,omitempty"`
	History     *Ring         `json:"-"` // session UI state, not serialized
	StatsSource StatsSource   `json:"stats_source"`
	Note        string        `json:"note,omitempty"`
}

// MiningStats is what a running XMRig reports.
type MiningStats struct {
	Hashrate10s    float64 `json:"hashrate_10s"`
	Hashrate60s    float64 `json:"hashrate_60s"`
	Hashrate15m    float64 `json:"hashrate_15m"`
	ThreadsActive  int     `json:"threads_active"`
	ThreadsTotal   int     `json:"threads_total"`
	SharesGood     int64   `json:"shares_good"`
	SharesTotal    int64   `json:"shares_total"`
	Pool           string  `json:"pool"`
	Connected      bool    `json:"connected"`
	PingMs         int     `json:"ping_ms"`
	DonateFallback bool    `json:"donate_fallback"`
	WorkerID       string  `json:"worker_id,omitempty"`
	Version        string  `json:"version,omitempty"`
}

// CPUSample is one point of miner-vs-node CPU.
type CPUSample struct {
	T                 time.Time `json:"t"`
	MinerMilli        int64     `json:"miner_milli"`
	NodeUsedMilli     int64     `json:"node_used_milli"`
	NodeCapacityMilli int64     `json:"node_capacity_milli"`
	FreePct           float64   `json:"free_pct"`
}

// MinerPctOfNode is the miner's CPU as a percentage of the node's capacity.
func (c CPUSample) MinerPctOfNode() float64 {
	if c.NodeCapacityMilli == 0 {
		return 0
	}
	return float64(c.MinerMilli) / float64(c.NodeCapacityMilli) * 100
}

// PoolStats is the pool-side view for a wallet.
type PoolStats struct {
	Wallet           string    `json:"wallet"`
	ReportedHashrate float64   `json:"reported_hashrate"`
	AmountDueXMR     float64   `json:"amount_due_xmr"`
	AmountPaidXMR    float64   `json:"amount_paid_xmr"`
	TotalHashes      int64     `json:"total_hashes"`
	LastShare        time.Time `json:"last_share"`
	Stale            bool      `json:"stale"`
	AsOf             time.Time `json:"as_of"`
}

// Summarize (re)computes Cluster from Nodes. Nodes without CPU metrics are
// excluded from the free-CPU average; if none have metrics, NodeCPUFreePct is
// -1 (unavailable) rather than a misleading 0.
func (s *Snapshot) Summarize() {
	var sum ClusterSummary
	sum.NodesTotal = len(s.Nodes)
	var capMilli, freeMilli int64
	for i := range s.Nodes {
		n := &s.Nodes[i]
		if n.Mining != nil && n.Mining.Connected {
			sum.NodesMining++
			if n.Mining.Hashrate60s > 0 {
				sum.TotalHashrate += n.Mining.Hashrate60s
			}
			sum.AcceptedShares += n.Mining.SharesGood
			if n.Mining.SharesTotal >= n.Mining.SharesGood {
				sum.RejectedShares += n.Mining.SharesTotal - n.Mining.SharesGood
			}
		}
		if n.CPU != nil {
			sum.MinerCPUMilli += n.CPU.MinerMilli
			capMilli += n.CPU.NodeCapacityMilli
			freeMilli += n.CPU.NodeCapacityMilli - n.CPU.NodeUsedMilli
		}
	}
	if capMilli > 0 {
		sum.NodeCPUFreePct = float64(freeMilli) / float64(capMilli) * 100
	} else {
		sum.NodeCPUFreePct = Unknown
	}
	s.Cluster = sum
}
