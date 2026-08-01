package diag

import (
	"strings"
	"testing"
)

func TestBitcoinCheck(t *testing.T) {
	const remedy = "tail /var/lib/ckpool/logs/ckpool.log into the pod log"

	tests := []struct {
		name       string
		scope      string
		pools      []BitcoinPoolResult
		skipped    bool
		searchWarn string
		want       CheckStatus
		wantDetail string
		wantRemedy bool
	}{
		{
			name: "disabled", skipped: true, want: StatusInfo, wantDetail: "skipped",
		},
		{
			// No pool must never fail the exit code: mining only Monero is normal.
			name: "no pool", scope: "all namespaces", want: StatusInfo, wantDetail: "none found",
			wantRemedy: true,
		},
		{
			name: "search denied", searchWarn: "search limited to namespace monero-idle-miner",
			want: StatusWarn, wantDetail: "search limited", wantRemedy: true,
		},
		{
			name:  "pool with stats",
			scope: "all namespaces",
			pools: []BitcoinPoolResult{{Where: "public-pool in bitcoin/mining-pool-abc", Source: "api", HasStats: true}},
			want:  StatusOK, wantDetail: "stats via api",
		},
		{
			// The stock ckpool case: detected, silent, and the operator is told why.
			name:  "pool without a readable stats source",
			scope: "all namespaces",
			pools: []BitcoinPoolResult{{Where: "ckpool in bitcoin/mining-pool-xyz", Source: "none", Remedy: remedy}},
			want:  StatusWarn, wantDetail: "no readable stats", wantRemedy: true,
		},
		{
			name:  "one healthy, one silent",
			scope: "all namespaces",
			pools: []BitcoinPoolResult{
				{Where: "public-pool in bitcoin/mining-pool-abc", Source: "api", HasStats: true},
				{Where: "ckpool in solo/mining-pool-xyz", Source: "none", Remedy: remedy},
			},
			want: StatusWarn, wantDetail: "stats via api; ckpool in solo/mining-pool-xyz: no readable stats",
			wantRemedy: true,
		},
	}

	for _, tt := range tests {
		got := BitcoinCheck(tt.scope, tt.pools, tt.skipped, tt.searchWarn)
		if got.Status != tt.want {
			t.Errorf("%s: status = %q, want %q", tt.name, got.Status, tt.want)
		}
		if !strings.Contains(got.Detail, tt.wantDetail) {
			t.Errorf("%s: detail = %q, want it to contain %q", tt.name, got.Detail, tt.wantDetail)
		}
		if tt.wantRemedy && got.Remedy == "" {
			t.Errorf("%s: expected an actionable remedy", tt.name)
		}
	}
}

// A Bitcoin warning must never turn into a non-zero exit code on its own.
func TestBitcoinWarningDoesNotFail(t *testing.T) {
	r := &Report{}
	r.Add(Check{Name: "cluster API", Status: StatusOK})
	r.Add(BitcoinCheck("all namespaces",
		[]BitcoinPoolResult{{Where: "ckpool in bitcoin/x", Source: "none", Remedy: "…"}}, false, ""))
	if r.Failed() {
		t.Error("a Bitcoin warning must not fail the doctor exit code")
	}
}

func TestAPIReachabilityCheck(t *testing.T) {
	cases := []struct {
		running, reachable int
		want               CheckStatus
		remedy             bool // expect the enable-httpApi remedy
	}{
		{0, 0, StatusInfo, false}, // nothing running → n/a
		{2, 2, StatusOK, false},   // all reachable
		{2, 0, StatusWarn, true},  // none reachable → warn + remedy (the P1 case)
		{3, 1, StatusWarn, true},  // partial → warn + remedy
	}
	for _, c := range cases {
		got := APIReachabilityCheck(c.running, c.reachable)
		if got.Status != c.want {
			t.Errorf("APIReachabilityCheck(%d,%d) status = %s, want %s", c.running, c.reachable, got.Status, c.want)
		}
		hasRemedy := got.Remedy != ""
		if hasRemedy != c.remedy {
			t.Errorf("APIReachabilityCheck(%d,%d) remedy present = %v, want %v", c.running, c.reachable, hasRemedy, c.remedy)
		}
		if c.remedy && !strings.Contains(got.Remedy, "httpApi.enabled: true") {
			t.Errorf("remedy should recommend httpApi.enabled: true, got: %q", got.Remedy)
		}
	}
}

func TestReportWorstAndFailed(t *testing.T) {
	r := &Report{}
	if r.Worst() != StatusInfo || r.Failed() {
		t.Fatalf("empty report: worst=%s failed=%v", r.Worst(), r.Failed())
	}
	r.Add(Check{Status: StatusOK})
	r.Add(Check{Status: StatusWarn})
	if r.Worst() != StatusWarn || r.Failed() {
		t.Errorf("ok+warn: worst=%s failed=%v, want warn/false", r.Worst(), r.Failed())
	}
	r.Add(Check{Status: StatusFail})
	if r.Worst() != StatusFail || !r.Failed() {
		t.Errorf("with fail: worst=%s failed=%v, want fail/true", r.Worst(), r.Failed())
	}
}
