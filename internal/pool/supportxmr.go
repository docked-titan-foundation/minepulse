// Package pool fetches and parses pool-side miner statistics. v1 targets the
// SupportXMR public API shape. Parsing is pure and fixture-tested (Constitution
// IV); the HTTP fetch is isolated in Client.
package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// piconeroPerXMR is the number of atomic units in one XMR (10^12).
const piconeroPerXMR = 1e12

// Stats is the subset of GET /api/miner/{address}/stats that minepulse uses.
// Amounts are in atomic units (piconero).
type Stats struct {
	Hash        float64 `json:"hash"`        // reported hashrate, H/s
	LastHash    int64   `json:"lastHash"`    // unix seconds
	TotalHashes int64   `json:"totalHashes"` // lifetime
	AmtPaid     int64   `json:"amtPaid"`     // atomic
	AmtDue      int64   `json:"amtDue"`      // atomic
}

// ParseStats decodes a SupportXMR miner stats payload.
func ParseStats(b []byte) (*Stats, error) {
	var s Stats
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse pool stats: %w", err)
	}
	return &s, nil
}

// ToPoolStats maps raw stats into the model for a wallet, as of the given time.
func (s *Stats) ToPoolStats(wallet string, asOf time.Time) model.PoolStats {
	var last time.Time
	if s.LastHash > 0 {
		last = time.Unix(s.LastHash, 0)
	}
	return model.PoolStats{
		Wallet:           wallet,
		ReportedHashrate: s.Hash,
		AmountDueXMR:     float64(s.AmtDue) / piconeroPerXMR,
		AmountPaidXMR:    float64(s.AmtPaid) / piconeroPerXMR,
		TotalHashes:      s.TotalHashes,
		LastShare:        last,
		AsOf:             asOf,
	}
}

// Client fetches pool-side stats over HTTP (read-only).
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient returns a Client for a SupportXMR-compatible API base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 8 * time.Second},
	}
}

// MinerStats fetches and parses stats for a wallet address.
func (c *Client) MinerStats(ctx context.Context, wallet string) (*Stats, error) {
	endpoint := fmt.Sprintf("%s/miner/%s/stats", c.BaseURL, url.PathEscape(wallet))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pool API status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return ParseStats(body)
}
