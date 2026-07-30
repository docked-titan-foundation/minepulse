// Package diag holds minepulse's doctor (preflight) result types and the pure
// classification logic behind them, kept free of cluster clients so it is
// unit-testable (Constitution IV).
package diag

import "fmt"

// CheckStatus is the outcome of a single check.
type CheckStatus string

const (
	StatusOK   CheckStatus = "ok"
	StatusWarn CheckStatus = "warn"
	StatusFail CheckStatus = "fail"
	StatusInfo CheckStatus = "info"
)

// severity orders statuses for computing the worst result (higher = worse).
func (s CheckStatus) severity() int {
	switch s {
	case StatusFail:
		return 3
	case StatusWarn:
		return 2
	case StatusOK:
		return 1
	default: // info
		return 0
	}
}

// Check is one preflight result.
type Check struct {
	Name   string      `json:"name"`
	Status CheckStatus `json:"status"`
	Detail string      `json:"detail,omitempty"`
	Remedy string      `json:"remedy,omitempty"` // shown when not OK
}

// Report is the ordered set of checks a doctor run produced.
type Report struct {
	Checks []Check `json:"checks"`
}

// Add appends a check.
func (r *Report) Add(c Check) { r.Checks = append(r.Checks, c) }

// Worst returns the most severe status across all checks (StatusInfo if empty).
func (r *Report) Worst() CheckStatus {
	worst := StatusInfo
	for _, c := range r.Checks {
		if c.Status.severity() > worst.severity() {
			worst = c.Status
		}
	}
	return worst
}

// Failed reports whether any check hard-failed (used for the process exit code).
func (r *Report) Failed() bool { return r.Worst() == StatusFail }

// apiRemedy is the actionable fix for a disabled/unreachable XMRig HTTP API.
const apiRemedy = "Enable it: set `httpApi.enabled: true` on the monero-idle-miner chart " +
	"(binds pod-internally, restricted). Until then minepulse reads stats from logs (reduced fidelity)."

// APIReachabilityCheck classifies XMRig HTTP API reachability across the running
// miners. It is pure so the P1 behavior (warn + recommend enabling) is tested
// directly.
func APIReachabilityCheck(running, reachable int) Check {
	const name = "XMRig HTTP API"
	switch {
	case running == 0:
		return Check{name, StatusInfo, "no miners are Running to probe", ""}
	case reachable == running:
		return Check{name, StatusOK, fmt.Sprintf("reachable on all %d miner(s)", running), ""}
	case reachable == 0:
		return Check{name, StatusWarn,
			fmt.Sprintf("not reachable on any of %d miner(s) — minepulse is falling back to log parsing", running),
			apiRemedy}
	default:
		return Check{name, StatusWarn,
			fmt.Sprintf("reachable on only %d of %d miner(s)", reachable, running),
			apiRemedy}
	}
}
