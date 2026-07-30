package xmrig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}

func TestParseSummaryAndMap(t *testing.T) {
	s, err := ParseSummary(read(t, "summary.json"))
	if err != nil {
		t.Fatalf("ParseSummary: %v", err)
	}
	backends, err := ParseBackends(read(t, "backends.json"))
	if err != nil {
		t.Fatalf("ParseBackends: %v", err)
	}

	active := ActiveCPUThreads(backends)
	if active != 6 {
		t.Fatalf("ActiveCPUThreads = %d, want 6", active)
	}

	ms := s.ToMiningStats(active, "pool.supportxmr.com:443")

	if ms.Hashrate10s != 376.61 || ms.Hashrate60s != 371.02 {
		t.Errorf("hashrate 10s/60s = %v/%v, want 376.61/371.02", ms.Hashrate10s, ms.Hashrate60s)
	}
	if ms.Hashrate15m != model.Unknown {
		t.Errorf("hashrate 15m = %v, want Unknown (null in fixture)", ms.Hashrate15m)
	}
	if ms.ThreadsActive != 6 || ms.ThreadsTotal != 8 {
		t.Errorf("threads = %d/%d, want 6/8", ms.ThreadsActive, ms.ThreadsTotal)
	}
	if ms.SharesGood != 42 || ms.SharesTotal != 44 {
		t.Errorf("shares = %d/%d, want 42/44", ms.SharesGood, ms.SharesTotal)
	}
	if !ms.Connected {
		t.Error("expected Connected (pool set, uptime>0)")
	}
	if ms.PingMs != 27 {
		t.Errorf("ping = %d, want 27", ms.PingMs)
	}
	if ms.DonateFallback {
		t.Error("configured pool matches; DonateFallback should be false")
	}
	if ms.WorkerID != "homelab-andromeda" || ms.Version != "6.20.0" {
		t.Errorf("worker/version = %q/%q", ms.WorkerID, ms.Version)
	}
}

func TestDonateDetection(t *testing.T) {
	cases := []struct {
		pool, configured string
		want             bool
	}{
		{"donate.v2.xmrig.com:3333", "pool.supportxmr.com:443", true}, // built-in donate host
		{"donate.ssl.xmrig.com:443", "", true},                        // donate host, no configured
		{"pool.supportxmr.com:443", "pool.supportxmr.com:443", false}, // matches configured
		{"pool.supportxmr.com:443", "", false},                        // no configured, not donate
		{"someother.pool.com:3333", "pool.supportxmr.com:443", true},  // not the configured pool
	}
	for _, c := range cases {
		if got := IsDonatePool(c.pool, c.configured); got != c.want {
			t.Errorf("IsDonatePool(%q,%q) = %v, want %v", c.pool, c.configured, got, c.want)
		}
	}
}

func TestActiveCPUThreadsAbsent(t *testing.T) {
	if got := ActiveCPUThreads([]Backend{{Type: "cpu", Enabled: false}}); got != -1 {
		t.Errorf("disabled cpu backend threads = %d, want -1", got)
	}
}
