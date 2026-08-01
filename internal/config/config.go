// Package config holds the resolved runtime options for minepulse.
package config

import "time"

// Output selects how snapshots are rendered.
type Output string

const (
	OutputTUI    Output = "tui"    // full-screen interactive (TTY)
	OutputStream Output = "stream" // compact text block per tick
	OutputJSON   Output = "json"   // one JSON snapshot per line (jsonl)
)

// XMRigAPIMode controls the miner-stats data source.
type XMRigAPIMode string

const (
	XMRigAuto XMRigAPIMode = "auto" // use the HTTP API if reachable, else logs
	XMRigOn   XMRigAPIMode = "on"   // require the HTTP API
	XMRigOff  XMRigAPIMode = "off"  // logs only
)

// Config is the fully-resolved set of options a run uses.
type Config struct {
	Namespace  string
	Selector   string
	Interval   time.Duration
	Output     Output
	Kubeconfig string
	Context    string
	Wallet     string // empty = auto-detect from the miner
	NoPool     bool
	PoolAPIURL string
	XMRigAPI   XMRigAPIMode
	Mock       bool

	// Bitcoin side. Every field's zero value means "auto-detect", so the
	// Bitcoin tab needs no configuration at all (FR-006).
	BTCNamespace string // empty = search every namespace the credentials allow
	BTCSelector  string // extra label selector, ANDed with the fingerprint check
	BTCAddress   string // payout address, for public-pool per-device rows
	BTCAPIPort   int    // 0 = the container port named "api", else 3334
	NoBTC        bool   // skip Bitcoin discovery and collection entirely
}

// Defaults returns the baseline configuration.
func Defaults() Config {
	return Config{
		Namespace:  "monero-idle-miner",
		Selector:   "app.kubernetes.io/name=monero-idle-miner",
		Interval:   3 * time.Second,
		Output:     OutputTUI,
		PoolAPIURL: "https://supportxmr.com/api",
		XMRigAPI:   XMRigAuto,
	}
}
