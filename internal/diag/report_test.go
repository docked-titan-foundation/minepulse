package diag

import (
	"strings"
	"testing"
)

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
