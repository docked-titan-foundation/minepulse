package pool

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseStatsAndMap(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "stats.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	s, err := ParseStats(b)
	if err != nil {
		t.Fatalf("ParseStats: %v", err)
	}

	asOf := time.Now()
	ps := s.ToPoolStats("47walletaddr", asOf)

	if ps.ReportedHashrate != 662 {
		t.Errorf("reported hashrate = %v, want 662", ps.ReportedHashrate)
	}
	// 184000000000 / 1e12 = 0.184
	if math.Abs(ps.AmountPaidXMR-0.184) > 1e-9 {
		t.Errorf("paid = %v XMR, want 0.184", ps.AmountPaidXMR)
	}
	// 3120000000 / 1e12 = 0.00312
	if math.Abs(ps.AmountDueXMR-0.00312) > 1e-12 {
		t.Errorf("due = %v XMR, want 0.00312", ps.AmountDueXMR)
	}
	if ps.TotalHashes != 4203000000 {
		t.Errorf("total hashes = %d, want 4203000000", ps.TotalHashes)
	}
	if ps.LastShare.Unix() != 1785540000 {
		t.Errorf("last share = %v, want unix 1785540000", ps.LastShare.Unix())
	}
	if ps.Wallet != "47walletaddr" || !ps.AsOf.Equal(asOf) {
		t.Errorf("wallet/asOf not carried through")
	}
}

func TestParseStatsZeroLastHash(t *testing.T) {
	s, err := ParseStats([]byte(`{"hash":0,"lastHash":0,"amtDue":0,"amtPaid":0}`))
	if err != nil {
		t.Fatalf("ParseStats: %v", err)
	}
	if got := s.ToPoolStats("w", time.Now()); !got.LastShare.IsZero() {
		t.Errorf("lastHash 0 should map to zero time, got %v", got.LastShare)
	}
}
