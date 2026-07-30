// Package render turns a model.Snapshot into terminal or machine output.
package render

import (
	"fmt"
	"time"
)

// Hashrate formats H/s with a sensible unit. Unknown (<0) renders as "—".
func Hashrate(hs float64) string {
	if hs < 0 {
		return "—"
	}
	switch {
	case hs >= 1e6:
		return fmt.Sprintf("%.2f MH/s", hs/1e6)
	case hs >= 1e3:
		return fmt.Sprintf("%.2f kH/s", hs/1e3)
	default:
		return fmt.Sprintf("%.0f H/s", hs)
	}
}

// Pct formats a percentage; negative means unavailable.
func Pct(p float64) string {
	if p < 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.0f%%", p)
}

// XMR formats a Monero amount.
func XMR(v float64) string { return fmt.Sprintf("%.6f XMR", v) }

// Dur formats a duration compactly (e.g. 3h12m, 45s).
func Dur(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	d = d.Round(time.Minute)
	if d < time.Minute {
		return "<1m"
	}
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// Ago formats how long ago t was.
func Ago(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return Dur(time.Since(t)) + " ago"
}
