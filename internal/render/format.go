// Package render turns a model.Snapshot into terminal or machine output.
package render

import (
	"fmt"
	"time"
)

// Hashrate formats H/s with a sensible unit. Unknown (<0) renders as "—". The
// scale runs to PH/s so a Bitcoin ASIC and a CPU miner read the same way.
func Hashrate(hs float64) string {
	if hs < 0 {
		return "—"
	}
	switch {
	case hs >= 1e15:
		return fmt.Sprintf("%.2f PH/s", hs/1e15)
	case hs >= 1e12:
		return fmt.Sprintf("%.2f TH/s", hs/1e12)
	case hs >= 1e9:
		return fmt.Sprintf("%.2f GH/s", hs/1e9)
	case hs >= 1e6:
		return fmt.Sprintf("%.2f MH/s", hs/1e6)
	case hs >= 1e3:
		return fmt.Sprintf("%.2f kH/s", hs/1e3)
	default:
		return fmt.Sprintf("%.0f H/s", hs)
	}
}

// Difficulty formats a share difficulty with an SI suffix (Bitcoin shares run
// from thousands to trillions). Unknown (<0) renders as "—".
func Difficulty(d float64) string {
	if d < 0 {
		return "—"
	}
	switch {
	case d >= 1e12:
		return fmt.Sprintf("%.2f T", d/1e12)
	case d >= 1e9:
		return fmt.Sprintf("%.2f G", d/1e9)
	case d >= 1e6:
		return fmt.Sprintf("%.2f M", d/1e6)
	case d >= 1e3:
		return fmt.Sprintf("%.2f k", d/1e3)
	default:
		return fmt.Sprintf("%.0f", d)
	}
}

// Shares formats accepted/rejected shares the one way they are written across
// the dashboard (standard P6): "42✓/2✗". A source that does not report rejects
// gets the accepted half alone — never "42✓/0✗", which would claim a clean run
// nobody measured. Both unknown renders as the unavailable mark.
func Shares(good, bad float64) string {
	switch {
	case good < 0 && bad < 0:
		return "—"
	case bad < 0:
		return Count(good) + "✓"
	case good < 0:
		return "—/" + Count(bad) + "✗"
	default:
		return Count(good) + "✓/" + Count(bad) + "✗"
	}
}

// Count formats a share count, which ckpool reports as difficulty-weighted
// floats rather than integers. Unknown (<0) renders as "—".
func Count(v float64) string {
	if v < 0 {
		return "—"
	}
	if v >= 1e6 {
		return fmt.Sprintf("%.2fM", v/1e6)
	}
	return fmt.Sprintf("%.0f", v)
}

// ShortAddress truncates a payout address for display. Addresses are shown
// truncated in the terminal and never logged (Constitution VII); the JSON
// output carries them whole because that is a data contract.
func ShortAddress(a string) string {
	const head, tail = 8, 4
	if len(a) <= head+tail+1 {
		return a
	}
	return a[:head] + "…" + a[len(a)-tail:]
}

// durSince is how long ago t was, or 0 when unknown, for Dur.
func durSince(t time.Time) time.Duration {
	if t.IsZero() {
		return 0
	}
	return time.Since(t)
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
