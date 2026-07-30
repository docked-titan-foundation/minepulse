// Package xmrig parses the XMRig HTTP API responses (/1/summary, /2/backends)
// and maps them to minepulse's model. All functions here are pure so they can
// be unit-tested against recorded fixtures (Constitution IV).
package xmrig

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// Summary is the subset of GET /1/summary that minepulse uses.
type Summary struct {
	Version  string `json:"version"`
	WorkerID string `json:"worker_id"`
	Hashrate struct {
		// Total is [10s, 60s, 15m]; entries may be null before a window fills.
		Total []*float64 `json:"total"`
	} `json:"hashrate"`
	Results struct {
		SharesGood  int64 `json:"shares_good"`
		SharesTotal int64 `json:"shares_total"`
	} `json:"results"`
	Connection struct {
		Pool   string `json:"pool"`
		Uptime int64  `json:"uptime"`
		Ping   int    `json:"ping"`
	} `json:"connection"`
	CPU struct {
		Threads int `json:"threads"`
	} `json:"cpu"`
}

// ParseSummary decodes a /1/summary payload.
func ParseSummary(b []byte) (*Summary, error) {
	var s Summary
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse xmrig summary: %w", err)
	}
	return &s, nil
}

// hr returns the hashrate window at index i, or model.Unknown if missing/null.
func (s *Summary) hr(i int) float64 {
	if i < 0 || i >= len(s.Hashrate.Total) || s.Hashrate.Total[i] == nil {
		return model.Unknown
	}
	return *s.Hashrate.Total[i]
}

// IsDonatePool reports whether pool is XMRig's built-in donation pool, or — when
// a configured pool is known — any pool other than it. This is how a silently
// mis-wired miner (mining to xmrig.com instead of the operator) is caught.
func IsDonatePool(pool, configured string) bool {
	host := hostOnly(pool)
	if strings.HasSuffix(host, "xmrig.com") {
		return true
	}
	if configured != "" && host != hostOnly(configured) {
		return true
	}
	return false
}

func hostOnly(hostPort string) string {
	if i := strings.LastIndex(hostPort, ":"); i > 0 {
		return hostPort[:i]
	}
	return hostPort
}

// ToMiningStats maps a summary (+ active thread count from /2/backends and the
// operator's configured pool) into the model. activeThreads < 0 means unknown.
func (s *Summary) ToMiningStats(activeThreads int, configuredPool string) model.MiningStats {
	ping := s.Connection.Ping
	if ping == 0 {
		ping = -1
	}
	return model.MiningStats{
		Hashrate10s:    s.hr(0),
		Hashrate60s:    s.hr(1),
		Hashrate15m:    s.hr(2),
		ThreadsActive:  activeThreads,
		ThreadsTotal:   s.CPU.Threads,
		SharesGood:     s.Results.SharesGood,
		SharesTotal:    s.Results.SharesTotal,
		Pool:           s.Connection.Pool,
		Connected:      s.Connection.Pool != "" && s.Connection.Uptime > 0,
		PingMs:         ping,
		DonateFallback: s.Connection.Pool != "" && IsDonatePool(s.Connection.Pool, configuredPool),
		WorkerID:       s.WorkerID,
		Version:        s.Version,
	}
}
