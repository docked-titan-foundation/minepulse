package model

import "time"

// PoolImpl is which solo-mining pool implementation a workload runs.
type PoolImpl string

const (
	ImplPublicPool PoolImpl = "public-pool"
	ImplCkpool     PoolImpl = "ckpool"
	ImplUnknown    PoolImpl = "unknown" // pool-shaped workload, no fingerprint matched
)

// DetailLevel is the finest per-miner granularity a pool's stats source reached.
// public-pool can name each device; ckpool's records stop at the payout address;
// with neither, only pool totals exist.
type DetailLevel string

const (
	DetailDevice  DetailLevel = "device"
	DetailAddress DetailLevel = "address"
	DetailTotals  DetailLevel = "totals"
)

// BitcoinView is the Bitcoin tab's whole payload: every pool detected, plus what
// detection itself has to say. A nil *BitcoinView means no pool was detected (or
// collection was disabled); a view holding a pool whose Source is SourceNone
// means the pool exists but nothing readable reports its stats.
type BitcoinView struct {
	Pools []BitcoinPool `json:"pools"`
	Scope string        `json:"scope"`          // namespaces detection actually searched
	Note  string        `json:"note,omitempty"` // why the view is thin
}

// BitcoinPool is one solo-mining pool workload as minepulse sees it.
type BitcoinPool struct {
	Impl      PoolImpl      `json:"impl"`
	Namespace string        `json:"namespace"`
	Pod       string        `json:"pod"`
	Node      string        `json:"node"`
	Phase     string        `json:"phase"`
	Running   bool          `json:"running"`
	Uptime    time.Duration `json:"uptime"`

	Source StatsSource `json:"stats_source"`
	Detail DetailLevel `json:"detail_level"`

	Stats  *BitcoinStats  `json:"stats,omitempty"`
	Miners []BitcoinMiner `json:"miners,omitempty"`

	Stale bool      `json:"stale"`
	AsOf  time.Time `json:"as_of"`

	Note   string `json:"note,omitempty"`   // the degradation, in one line
	Remedy string `json:"remedy,omitempty"` // what the operator can do about it
}

// BitcoinStats is the pool-level aggregate. Every numeric field is Unknown when
// the source does not provide it — never 0, which would read as "mining stopped".
type BitcoinStats struct {
	Hashrate1m     float64 `json:"hashrate_1m"`
	Hashrate5m     float64 `json:"hashrate_5m"`
	Hashrate1h     float64 `json:"hashrate_1h"`
	Hashrate1d     float64 `json:"hashrate_1d"`
	HashrateWindow string  `json:"hashrate_window"` // which window the headline figure is

	Users        int `json:"users"`
	Workers      int `json:"workers"`
	WorkersIdle  int `json:"workers_idle"`
	Disconnected int `json:"disconnected"`

	Accepted       float64 `json:"accepted"` // difficulty shares, not share counts
	Rejected       float64 `json:"rejected"`
	BestShare      float64 `json:"best_share"`
	NetworkDiffPct float64 `json:"network_diff_pct"`
	SPS1m          float64 `json:"sps_1m"`

	BlockHeight int64 `json:"block_height"`
	BlocksFound int   `json:"blocks_found"`

	TotalsAsOf time.Time     `json:"totals_as_of,omitempty"` // marks cached pool-wide figures
	LastUpdate time.Time     `json:"last_update"`
	Runtime    time.Duration `json:"runtime"`
}

// NewBitcoinStats returns stats with every numeric field marked unavailable, so
// a parser only has to set what its source actually reports (VR-2).
func NewBitcoinStats() *BitcoinStats {
	return &BitcoinStats{
		Hashrate1m: Unknown, Hashrate5m: Unknown, Hashrate1h: Unknown, Hashrate1d: Unknown,
		Users: -1, Workers: -1, WorkersIdle: -1, Disconnected: -1,
		Accepted: Unknown, Rejected: Unknown, BestShare: Unknown,
		NetworkDiffPct: Unknown, SPS1m: Unknown,
		BlockHeight: -1, BlocksFound: -1,
	}
}

// BitcoinMiner is one per-miner row, at whichever granularity DetailLevel names:
// a device (public-pool) or a payout address (ckpool).
type BitcoinMiner struct {
	Name           string    `json:"name"` // worker name, or payout address
	Hashrate       float64   `json:"hashrate"`
	BestDifficulty float64   `json:"best_difficulty"`
	BestEver       float64   `json:"best_ever"`
	Workers        int       `json:"workers"` // devices under this address
	Shares         float64   `json:"shares"`
	StartTime      time.Time `json:"start_time"`
	LastSeen       time.Time `json:"last_seen"`
	SessionID      string    `json:"session_id,omitempty"`
}
