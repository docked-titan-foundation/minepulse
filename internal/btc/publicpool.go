package btc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// clientResponse is GET /api/client/{address} — uncached upstream, so this is
// the live view of what is actually mining.
type clientResponse struct {
	BestDifficulty *float64 `json:"bestDifficulty"` // null until the address has a settings row
	WorkersCount   *int     `json:"workersCount"`
	Workers        []struct {
		SessionID      string  `json:"sessionId"`
		Name           string  `json:"name"`
		BestDifficulty string  `json:"bestDifficulty"` // a string with 2 decimals
		HashRate       float64 `json:"hashRate"`       // H/s
		StartTime      string  `json:"startTime"`
		LastSeen       string  `json:"lastSeen"`
	} `json:"workers"`
}

// poolResponse is GET /api/pool — cached five minutes upstream (research R3).
type poolResponse struct {
	TotalHashRate *float64 `json:"totalHashRate"`
	BlockHeight   *int64   `json:"blockHeight"`
	TotalMiners   *int     `json:"totalMiners"`
	BlocksFound   []any    `json:"blocksFound"`
}

// ParseClient decodes public-pool's per-address response into pool stats and one
// row per device. The address's own best difficulty becomes the pool's best
// share; the headline hashrate is the sum of the devices, which is live where
// /api/pool is not.
func ParseClient(b []byte) (*model.BitcoinStats, []model.BitcoinMiner, error) {
	var r clientResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, nil, fmt.Errorf("parse public-pool client: %w", err)
	}

	stats := model.NewBitcoinStats()
	stats.HashrateWindow = "now"
	if r.BestDifficulty != nil {
		stats.BestShare = *r.BestDifficulty
	}
	if r.WorkersCount != nil {
		stats.Workers = *r.WorkersCount
	} else {
		stats.Workers = len(r.Workers)
	}

	miners := make([]model.BitcoinMiner, 0, len(r.Workers))
	var total float64
	for _, w := range r.Workers {
		total += w.HashRate
		m := model.BitcoinMiner{
			Name:           w.Name,
			SessionID:      w.SessionID,
			Hashrate:       w.HashRate,
			BestDifficulty: parseFloatOrUnknown(w.BestDifficulty),
			BestEver:       model.Unknown,
			Shares:         model.Unknown,
			Workers:        -1,
			StartTime:      parseISO(w.StartTime),
			LastSeen:       parseISO(w.LastSeen),
		}
		miners = append(miners, m)
	}
	if len(r.Workers) > 0 {
		stats.Hashrate1m = total
	}
	return stats, miners, nil
}

// ParsePool decodes public-pool's pool-wide totals. asOf marks when they were
// fetched — upstream caches them for five minutes, so the view can say how old
// they may be rather than presenting them as current.
func ParsePool(b []byte, asOf time.Time) (*model.BitcoinStats, error) {
	var r poolResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("parse public-pool pool: %w", err)
	}
	stats := model.NewBitcoinStats()
	stats.HashrateWindow = "pool"
	if r.TotalHashRate != nil {
		stats.Hashrate1m = *r.TotalHashRate
	}
	if r.TotalMiners != nil {
		stats.Workers = *r.TotalMiners
	}
	if r.BlockHeight != nil {
		stats.BlockHeight = *r.BlockHeight
	}
	stats.BlocksFound = len(r.BlocksFound)
	stats.TotalsAsOf = asOf
	return stats, nil
}

func parseFloatOrUnknown(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return model.Unknown
	}
	return v
}

func parseISO(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
