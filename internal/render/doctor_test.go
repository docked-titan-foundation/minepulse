package render

import (
	"strings"
	"testing"

	"github.com/docked-titan-foundation/minepulse/internal/diag"
)

func TestRenderReportDisabledAPI(t *testing.T) {
	r := &diag.Report{}
	r.Add(diag.Check{Name: "cluster API", Status: diag.StatusOK, Detail: "reachable"})
	r.Add(diag.APIReachabilityCheck(2, 0)) // disabled on all → warn + remedy

	var b strings.Builder
	Doctor(&b, r)
	out := b.String()

	for _, want := range []string{"doctor", "XMRig HTTP API", "httpApi.enabled: true", "warning"} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor render missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderReportAllGood(t *testing.T) {
	r := &diag.Report{}
	r.Add(diag.Check{Name: "cluster API", Status: diag.StatusOK, Detail: "reachable"})
	r.Add(diag.APIReachabilityCheck(2, 2))

	var b strings.Builder
	Doctor(&b, r)
	if out := b.String(); !strings.Contains(out, "all good") {
		t.Errorf("expected 'all good' summary, got:\n%s", out)
	}
}
