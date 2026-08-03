// Package render turns a model.Snapshot into terminal or machine output.
package render

import (
	"fmt"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// unavailable is how every panel, column and output mode spells "this source
// does not report it" (standard P7). One mark, no exceptions — a second
// spelling of "unknown" is what the standard exists to prevent, and the Monero
// tab spent one feature cycle as that exception because a freeze outranked it.
const unavailable = "—"

// Hashrate formats H/s with a sensible unit. Unknown (<0) renders as the
// unavailable mark. The scale runs to PH/s so a Bitcoin ASIC and a CPU miner
// read the same way.
func Hashrate(hs float64) string {
	if hs < 0 {
		return unavailable
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
// from thousands to trillions). Unknown (<0) renders as the unavailable mark.
func Difficulty(d float64) string {
	if d < 0 {
		return unavailable
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
		return unavailable
	case bad < 0:
		return Count(good) + "✓"
	case good < 0:
		return unavailable + "/" + Count(bad) + "✗"
	default:
		return Count(good) + "✓/" + Count(bad) + "✗"
	}
}

// Count formats a share count, which ckpool reports as difficulty-weighted
// floats rather than integers. Unknown (<0) renders as the unavailable mark.
func Count(v float64) string {
	if v < 0 {
		return unavailable
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

// Pct formats a percentage; negative means unavailable, which is the one mark
// the standard allows (P7).
func Pct(p float64) string {
	if p < 0 {
		return unavailable
	}
	return fmt.Sprintf("%.0f%%", p)
}

// XMR formats a Monero amount.
func XMR(v float64) string { return fmt.Sprintf("%.6f XMR", v) }

// Dur formats a duration compactly (e.g. 3h12m, 45s).
func Dur(d time.Duration) string {
	if d <= 0 {
		return unavailable
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
		return unavailable
	}
	return Dur(time.Since(t)) + " ago"
}

// PoolLine is the identity line each tab opens with: where the work goes, whose
// pool it is, how that pool shares work, and at what address (specs/007).
//
//	[internal] mining-pool.bitcoin.svc:3333 - public-pool - solo - 10.43.7.12
//
// Every field degrades to the unavailable mark on its own, so a pool minepulse
// half-recognizes still produces a line rather than nothing. Locality is the one
// field that never guesses: it reports what it could prove, and its basis is
// available for the caller to dim in beside it.
func PoolLine(e *model.PoolEndpoint) string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("[%s] %s - %s - %s - %s",
		locality(e.Locality), orMark(e.URL), orMark(e.Brand),
		mode(e.Mode), orMark(e.IP))
}

func locality(l model.Locality) string {
	if l == model.LocalityInternal || l == model.LocalityExternal {
		return string(l)
	}
	return unavailable
}

func mode(m model.MiningMode) string {
	if m == model.ModeSolo || m == model.ModeShared {
		return string(m)
	}
	return unavailable
}

func orMark(s string) string {
	if s == "" {
		return unavailable
	}
	return s
}
