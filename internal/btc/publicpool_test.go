package btc

import (
	"testing"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

func TestParseClient(t *testing.T) {
	stats, miners, err := ParseClient(readFixture(t, "publicpool_client.json"))
	if err != nil {
		t.Fatalf("ParseClient: %v", err)
	}
	if len(miners) != 2 {
		t.Fatalf("miners = %d, want 2 devices", len(miners))
	}

	// The headline hashrate is the sum of the devices — the live figure, unlike
	// /api/pool's five-minute cache (research R3).
	if want := 480e9 + 1200e9; stats.Hashrate1m != want {
		t.Errorf("Hashrate1m = %v, want %v (sum of workers)", stats.Hashrate1m, want)
	}
	if stats.Workers != 2 {
		t.Errorf("Workers = %d, want 2", stats.Workers)
	}
	if stats.BestShare != 4.1e9 {
		t.Errorf("BestShare = %v, want the address's bestDifficulty", stats.BestShare)
	}

	m := miners[0]
	if m.Name != "bitaxe-01" || m.SessionID != "a1b2c3d4" {
		t.Errorf("first miner = %q/%q, want bitaxe-01/a1b2c3d4", m.Name, m.SessionID)
	}
	if m.Hashrate != 480e9 {
		t.Errorf("Hashrate = %v, want 480e9", m.Hashrate)
	}
	// bestDifficulty arrives as a 2-decimal string, not a number.
	if m.BestDifficulty != 1.2e9 {
		t.Errorf("BestDifficulty = %v, want 1.2e9 parsed from a string", m.BestDifficulty)
	}
	if got := m.StartTime.UTC(); got != time.Date(2026, 7, 25, 9, 12, 0, 0, time.UTC) {
		t.Errorf("StartTime = %v, want the ISO-8601 startTime", got)
	}
	if got := m.LastSeen.UTC(); got != time.Date(2026, 7, 31, 12, 40, 31, 0, time.UTC) {
		t.Errorf("LastSeen = %v, want the ISO-8601 lastSeen", got)
	}
}

// An address the pool has never seen: no workers, null bestDifficulty. That is
// "nothing connected", not "hashrate zero" for fields the pool did not report.
func TestParseClientEmpty(t *testing.T) {
	stats, miners, err := ParseClient(readFixture(t, "publicpool_client_empty.json"))
	if err != nil {
		t.Fatalf("ParseClient: %v", err)
	}
	if len(miners) != 0 {
		t.Fatalf("miners = %d, want none", len(miners))
	}
	if stats.Workers != 0 {
		t.Errorf("Workers = %d, want 0 — the pool did report a count", stats.Workers)
	}
	if stats.BestShare != model.Unknown {
		t.Errorf("BestShare = %v, want Unknown for a null bestDifficulty", stats.BestShare)
	}
	if stats.Hashrate1m != model.Unknown {
		t.Errorf("Hashrate1m = %v, want Unknown when no worker reported one", stats.Hashrate1m)
	}
}

func TestParsePool(t *testing.T) {
	asOf := time.Date(2026, 7, 31, 12, 40, 42, 0, time.UTC)
	stats, err := ParsePool(readFixture(t, "publicpool_pool.json"), asOf)
	if err != nil {
		t.Fatalf("ParsePool: %v", err)
	}
	if stats.Hashrate1m != 1680e9 {
		t.Errorf("Hashrate1m = %v, want totalHashRate", stats.Hashrate1m)
	}
	if stats.Workers != 2 {
		t.Errorf("Workers = %d, want totalMiners", stats.Workers)
	}
	if stats.BlockHeight != 907213 {
		t.Errorf("BlockHeight = %d, want 907213", stats.BlockHeight)
	}
	if stats.BlocksFound != 0 {
		t.Errorf("BlocksFound = %d, want 0", stats.BlocksFound)
	}
	// These figures are up to five minutes old by construction, and the view has
	// to be able to say so.
	if !stats.TotalsAsOf.Equal(asOf) {
		t.Errorf("TotalsAsOf = %v, want %v", stats.TotalsAsOf, asOf)
	}
}

func TestParsePublicPoolGarbage(t *testing.T) {
	if _, _, err := ParseClient([]byte("<html>404</html>")); err == nil {
		t.Error("ParseClient accepted non-JSON")
	}
	if _, err := ParsePool([]byte("nope"), time.Now()); err == nil {
		t.Error("ParsePool accepted non-JSON")
	}
}
