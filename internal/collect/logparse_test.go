package collect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

func TestParseLogs(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "xmrig.log"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	ms := ParseLogs(string(b), "pool.supportxmr.com:443")
	if ms == nil {
		t.Fatal("ParseLogs returned nil for a real log")
	}
	if ms.Hashrate10s != 376.6 || ms.Hashrate60s != 371.0 {
		t.Errorf("speed 10s/60s = %v/%v, want 376.6/371.0", ms.Hashrate10s, ms.Hashrate60s)
	}
	if ms.Hashrate15m != model.Unknown {
		t.Errorf("speed 15m = %v, want Unknown (n/a)", ms.Hashrate15m)
	}
	if ms.ThreadsActive != 6 || ms.ThreadsTotal != 8 {
		t.Errorf("threads = %d/%d, want 6/8", ms.ThreadsActive, ms.ThreadsTotal)
	}
	if ms.SharesGood != 42 || ms.SharesTotal != 44 {
		t.Errorf("shares = %d/%d, want 42/44", ms.SharesGood, ms.SharesTotal)
	}
	if ms.Pool != "pool.supportxmr.com:443" || !ms.Connected {
		t.Errorf("pool = %q connected=%v", ms.Pool, ms.Connected)
	}
	if ms.DonateFallback {
		t.Error("configured pool matches; DonateFallback should be false")
	}
	if ms.Version != "6.20.0" {
		t.Errorf("version = %q, want 6.20.0", ms.Version)
	}
}

func TestParseLogsDonateFallback(t *testing.T) {
	logs := `[t]  net      use pool donate.v2.xmrig.com:3333 1.2.3.4
[t]  miner    speed 10s/60s/15m 5.0 5.0 5.0 H/s max 6 H/s`
	ms := ParseLogs(logs, "pool.supportxmr.com:443")
	if ms == nil || !ms.DonateFallback {
		t.Fatalf("expected donate fallback detected, got %+v", ms)
	}
}

func TestParseLogsNoSignal(t *testing.T) {
	if ms := ParseLogs("nothing useful here\njust noise", ""); ms != nil {
		t.Errorf("expected nil for logs with no mining signal, got %+v", ms)
	}
}
