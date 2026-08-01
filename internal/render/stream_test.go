package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

func streamOf(t *testing.T, s *model.Snapshot) string {
	t.Helper()
	var b bytes.Buffer
	Stream(&b, s)
	return b.String()
}

func TestStreamBitcoinBlock(t *testing.T) {
	s := sampleSnapshot()
	s.Bitcoin = &model.BitcoinView{
		Scope: "all namespaces",
		Pools: []model.BitcoinPool{devicePool(), addressPool()},
	}
	out := streamOf(t, s)

	for _, want := range []string{
		"bitcoin: public-pool (api)", "bitcoin: ckpool (logs)",
		"1.68 TH/s", "480.00 TH/s", "bitaxe-01", "WORKER", "ADDRESS", "per-device detail unavailable",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stream missing %q\n---\n%s", want, out)
		}
	}
	// The Monero block still comes first and is untouched.
	if i, j := strings.Index(out, "cluster:"), strings.Index(out, "bitcoin:"); i < 0 || j < i {
		t.Errorf("Monero block must precede the Bitcoin block\n---\n%s", out)
	}
	// Stream output is for pipes: no ANSI, ever.
	if strings.Contains(out, "\x1b[") {
		t.Errorf("stream output must carry no ANSI escapes\n---\n%q", out)
	}
	// Addresses are truncated in text output.
	if strings.Contains(out, "bc1qexampleaddressxxxxxxxxxxxxxxxxxxxxxxx") {
		t.Errorf("stream must truncate payout addresses\n---\n%s", out)
	}
}

func TestStreamBitcoinAbsent(t *testing.T) {
	out := streamOf(t, sampleSnapshot())
	if !strings.Contains(out, "bitcoin: no pool detected") {
		t.Errorf("a nil Bitcoin view must say so in one line\n---\n%s", out)
	}
}

func TestStreamBitcoinNoStatsSource(t *testing.T) {
	s := sampleSnapshot()
	s.Bitcoin = &model.BitcoinView{
		Scope: "all namespaces",
		Pools: []model.BitcoinPool{{
			Impl: model.ImplCkpool, Namespace: "bitcoin", Pod: "mining-pool-xyz",
			Phase: "Running", Running: true, Uptime: time.Hour,
			Source: model.SourceNone, Detail: model.DetailTotals,
			Note:   "ckpool publishes no stats to its log stream",
			Remedy: "tail /var/lib/ckpool/logs/ckpool.log into the pod log",
		}},
	}
	out := streamOf(t, s)
	if !strings.Contains(out, "ckpool publishes no stats") || !strings.Contains(out, "→ tail ") {
		t.Errorf("no-stats pool must print its note and remedy\n---\n%s", out)
	}
	// Look only at the Bitcoin block: "H/s" appears throughout the Monero table.
	btcBlock := out[strings.Index(out, "bitcoin:"):]
	if strings.Contains(btcBlock, "H/s") {
		t.Errorf("no-stats pool must not print a hashrate at all\n---\n%s", btcBlock)
	}
}

// The JSON contract: additive, -1 for unknown, whole addresses, H/s throughout.
func TestJSONBitcoinContract(t *testing.T) {
	s := sampleSnapshot()
	s.Bitcoin = &model.BitcoinView{Scope: "all namespaces", Pools: []model.BitcoinPool{addressPool()}}

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	btc, ok := doc["bitcoin"].(map[string]any)
	if !ok {
		t.Fatalf("no bitcoin object in %s", b)
	}
	pools := btc["pools"].([]any)
	pool := pools[0].(map[string]any)
	if pool["impl"] != "ckpool" || pool["stats_source"] != "logs" || pool["detail_level"] != "address" {
		t.Errorf("pool identity fields wrong: %v", pool)
	}
	stats := pool["stats"].(map[string]any)
	if stats["hashrate_1m"].(float64) != 480e12 {
		t.Errorf("hashrate must be plain H/s, got %v", stats["hashrate_1m"])
	}
	// Fields ckpool does not report stay -1 rather than becoming a claim of zero.
	if stats["block_height"].(float64) != -1 {
		t.Errorf("unreported block_height = %v, want -1", stats["block_height"])
	}
	// The address is a data contract: whole in JSON, truncated only on screen.
	miners := pool["miners"].([]any)
	if miners[0].(map[string]any)["name"] != "bc1qexampleaddressxxxxxxxxxxxxxxxxxxxxxxx" {
		t.Errorf("JSON must carry the whole address, got %v", miners[0])
	}

	// And the Monero half is untouched by any of it.
	if _, ok := doc["cluster"].(map[string]any); !ok {
		t.Errorf("cluster key missing from %s", b)
	}
	delete(doc, "bitcoin")
	plain, err := json.Marshal(sampleSnapshot())
	if err != nil {
		t.Fatalf("marshal plain: %v", err)
	}
	var want map[string]any
	if err := json.Unmarshal(plain, &want); err != nil {
		t.Fatalf("unmarshal plain: %v", err)
	}
	if len(doc) != len(want) {
		t.Errorf("removing bitcoin must reproduce the pre-feature record: got keys %v, want %v",
			keysOf(doc), keysOf(want))
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
