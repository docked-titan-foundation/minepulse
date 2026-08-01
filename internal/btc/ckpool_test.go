package btc

import (
	"os"
	"testing"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestParseCkpoolStatusLog(t *testing.T) {
	stats, miners, err := ParseStatusLog(readFixture(t, "ckpool_status.log"))
	if err != nil {
		t.Fatalf("ParseStatusLog: %v", err)
	}
	if stats == nil {
		t.Fatal("stats is nil for a log containing two full status cycles")
	}

	// The newest cycle wins: the fixture's second cycle reports 480T and 12483
	// accepted, the first 471T and 12470.
	if stats.Hashrate1m != 480e12 {
		t.Errorf("Hashrate1m = %v, want 480e12 (newest cycle)", stats.Hashrate1m)
	}
	if stats.Accepted != 12483 {
		t.Errorf("Accepted = %v, want 12483 (newest cycle)", stats.Accepted)
	}
	// Records are told apart by their keys, not their order, so all three land.
	if stats.Users != 1 || stats.Workers != 2 || stats.WorkersIdle != 0 {
		t.Errorf("counters = users %d workers %d idle %d, want 1/2/0",
			stats.Users, stats.Workers, stats.WorkersIdle)
	}
	if stats.BestShare != 1.2e9 {
		t.Errorf("BestShare = %v, want 1.2e9", stats.BestShare)
	}
	if stats.NetworkDiffPct != 0.02 {
		t.Errorf("NetworkDiffPct = %v, want 0.02", stats.NetworkDiffPct)
	}
	if stats.HashrateWindow != "1m" {
		t.Errorf("HashrateWindow = %q, want \"1m\"", stats.HashrateWindow)
	}
	if got := stats.LastUpdate.UTC(); got != time.Unix(1785500418, 0).UTC() {
		t.Errorf("LastUpdate = %v, want the newest lastupdate", got)
	}
	if stats.Runtime != 512000*time.Second {
		t.Errorf("Runtime = %v, want 512000s", stats.Runtime)
	}

	// ckpool reports per payout address, never per device (research R1).
	if len(miners) != 1 {
		t.Fatalf("miners = %d, want 1 address row", len(miners))
	}
	m := miners[0]
	if m.Name != "bc1qexampleaddressxxxxxxxxxxxxxxxxxxxxxxx" {
		t.Errorf("miner name = %q, want the payout address", m.Name)
	}
	if m.Hashrate != 480e12 || m.Workers != 2 || m.Shares != 12483 {
		t.Errorf("miner = %v H/s, %d workers, %v shares; want 480e12/2/12483",
			m.Hashrate, m.Workers, m.Shares)
	}
	if m.BestEver != 4.1e9 {
		t.Errorf("BestEver = %v, want 4.1e9", m.BestEver)
	}
	if m.LastSeen.UTC() != time.Unix(1785500411, 0).UTC() {
		t.Errorf("LastSeen = %v, want the newest lastshare", m.LastSeen)
	}
	if m.StartTime.UTC() != time.Unix(1785000720, 0).UTC() {
		t.Errorf("StartTime = %v, want the authorization time", m.StartTime)
	}
}

// One unreadable value must cost that field only — the rest of the cycle still
// lands, and the bad field reads as unavailable rather than idle.
func TestParseCkpoolStatusLogDegradesPerField(t *testing.T) {
	stats, _, err := ParseStatusLog(readFixture(t, "ckpool_status_degraded.log"))
	if err != nil {
		t.Fatalf("ParseStatusLog: %v", err)
	}
	if stats == nil {
		t.Fatal("stats is nil despite two readable records")
	}
	if stats.Hashrate1m != model.Unknown {
		t.Errorf("Hashrate1m = %v, want Unknown for an unparseable value", stats.Hashrate1m)
	}
	if stats.Hashrate5m != 472e12 {
		t.Errorf("Hashrate5m = %v, want 472e12 — a sibling field must survive", stats.Hashrate5m)
	}
	if stats.Hashrate1h != 1e15 {
		t.Errorf("Hashrate1h = %v, want 1e15 from \"1e+03T\"", stats.Hashrate1h)
	}
	if stats.Hashrate1d != 999 {
		t.Errorf("Hashrate1d = %v, want 999 (unsuffixed)", stats.Hashrate1d)
	}
	if stats.Accepted != 12483 {
		t.Errorf("Accepted = %v, want 12483 — the shares record must still land", stats.Accepted)
	}
	if stats.WorkersIdle != 1 {
		t.Errorf("WorkersIdle = %d, want 1", stats.WorkersIdle)
	}
}

// A log with no status record at all yields no stats — not a pool reporting zero.
func TestParseCkpoolStatusLogWithoutRecords(t *testing.T) {
	stats, miners, err := ParseStatusLog(readFixture(t, "ckpool_no_status.log"))
	if err != nil {
		t.Fatalf("ParseStatusLog: %v", err)
	}
	if stats != nil {
		t.Errorf("stats = %+v, want nil when the log carries no Pool: record", stats)
	}
	if len(miners) != 0 {
		t.Errorf("miners = %d, want none", len(miners))
	}
}

func TestParseCkpoolStatusLogEmpty(t *testing.T) {
	stats, miners, err := ParseStatusLog(nil)
	if err != nil {
		t.Fatalf("ParseStatusLog(nil): %v", err)
	}
	if stats != nil || len(miners) != 0 {
		t.Errorf("empty input must yield nothing, got %+v / %d miners", stats, len(miners))
	}
}

// The log prefix is not a contract: ckpool's own timestamp, a container runtime's
// prefix, or none at all must all parse.
func TestParseCkpoolStatusLogPrefixAgnostic(t *testing.T) {
	lines := []byte(
		`Pool:{"runtime": 60, "lastupdate": 1785500418, "Users": 1, "Workers": 1, "Idle": 0, "Disconnected": 0}` + "\n" +
			`2026-07-31T12:40:18.421Z stdout F [2026-07-31 12:40:18.421] ckpool: Pool:{"hashrate1m": "12.3T", "hashrate5m": "12T", "hashrate15m": "12T", "hashrate1hr": "12T", "hashrate6hr": "12T", "hashrate1d": "12T", "hashrate7d": "12T"}` + "\n")
	stats, _, err := ParseStatusLog(lines)
	if err != nil {
		t.Fatalf("ParseStatusLog: %v", err)
	}
	if stats == nil || stats.Hashrate1m != 12.3e12 || stats.Users != 1 {
		t.Errorf("stats = %+v, want both records parsed regardless of prefix", stats)
	}
}
