package render

import (
	"fmt"
	"strings"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// bitcoinBody renders the Bitcoin tab: one panel per detected pool, each naming
// its stats source and what that source cannot tell us. The four lifecycle
// states — no pool, nothing published yet, live, stale — each read differently,
// so "not found" is never confused with "idle" (SC-004).
func bitcoinBody(v *model.BitcoinView) string {
	if v == nil {
		return dimStyle.Render("no Bitcoin pool detected") + "\n" +
			dimStyle.Render("minepulse looks for public-pool and ckpool workloads; --no-btc disables the search")
	}
	if len(v.Pools) == 0 {
		note := v.Note
		if note == "" {
			note = "no Bitcoin pool found in " + v.Scope
		}
		return dimStyle.Render(note)
	}

	var b strings.Builder
	for i := range v.Pools {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(bitcoinPool(&v.Pools[i]))
	}
	b.WriteString("\n" + dimStyle.Render("searched "+v.Scope))
	return b.String()
}

// bitcoinPool renders one pool as a panel in the dashboard's standard grammar
// (specs/004): a bold role label followed by the metrics, identity dimmed
// beneath it, then the table, then notes. Everything flush left.
func bitcoinPool(p *model.BitcoinPool) string {
	var b strings.Builder

	// P2: header line is the label plus the metrics, most important first.
	if s := p.Stats; s != nil {
		fmt.Fprintf(&b, "%s  %s\n", headSt.Render(string(p.Impl)), headlineMetrics(s))
	} else {
		// No metrics to carry: the label stands alone and the context line below
		// reports why, once.
		b.WriteString(headSt.Render(string(p.Impl)) + "\n")
	}

	// P3: identity and provenance, dimmed, on their own line.
	b.WriteString(contextLine(p) + "\n")

	if p.Stats != nil {
		if extra := chainSummary(p.Stats); extra != "" {
			b.WriteString(dimStyle.Render(extra) + "\n")
		}
		if len(p.Miners) > 0 {
			b.WriteString(minerTable(p))
		}
	}

	// P8: a problem is a warning; a remedy is a dimmed instruction.
	if p.Note != "" {
		b.WriteString(warnSt.Render("! "+p.Note) + "\n")
	}
	if p.Remedy != "" {
		b.WriteString(dimStyle.Render("→ "+p.Remedy) + "\n")
	}
	return b.String()
}

// headlineMetrics is the ` · `-separated figure list on a panel header, most
// important first and each labeled the way the Monero header labels its own
// ("42 shares✓", "miner 5000m").
//
// A figure the source does not report is left out rather than rendered as a
// dash: in a prose list a bare "—" says nothing, while in a table cell it holds
// the column open. The headline hashrate is the exception — it is the panel's
// primary metric, so it appears even when unavailable.
func headlineMetrics(s *model.BitcoinStats) string {
	parts := []string{Hashrate(s.Hashrate1m)}
	if s.HashrateWindow != "" && s.HashrateWindow != "now" && s.HashrateWindow != "pool" {
		parts[0] += " (" + s.HashrateWindow + ")"
	}
	if w := workerSummary(s); w != "workers —" {
		parts = append(parts, w)
	}
	if s.Accepted >= 0 || s.Rejected >= 0 {
		parts = append(parts, Shares(s.Accepted, s.Rejected)+" shares")
	}
	if s.BestShare >= 0 {
		parts = append(parts, "best "+Difficulty(s.BestShare))
	}
	return strings.Join(parts, " · ")
}

// contextLine is where the pool runs, in what state, and where its numbers came
// from — the provenance the metrics line must not carry (P3).
func contextLine(p *model.BitcoinPool) string {
	state := goodSt.Render(p.Phase)
	if !p.Running {
		state = badSt.Render(p.Phase)
	}
	// Provenance is dim when the numbers are current, and a warning when there
	// are none or they are stale — the one place either fact is stated.
	source := dimStyle.Render("· " + sourceLabel(p))
	if p.Stale || p.Source == model.SourceNone {
		source = dimStyle.Render("· ") + warnSt.Render(sourceLabel(p))
	}
	return fmt.Sprintf("%s %s %s %s %s",
		dimStyle.Render(p.Namespace+"/"+p.Pod),
		dimStyle.Render("on "+orDash(p.Node)),
		dimStyle.Render("·"),
		state+dimStyle.Render(" "+Dur(p.Uptime)),
		source)
}

// sourceLabel says where the numbers came from, and how old they are when the
// source failed this tick. Plain text: the caller styles the context line.
func sourceLabel(p *model.BitcoinPool) string {
	switch {
	case p.Stale:
		return "stale " + Ago(p.AsOf)
	case p.Source == model.SourceAPI:
		return "via API"
	case p.Source == model.SourceLogs:
		return "via logs"
	default:
		return "no stats source"
	}
}

func workerSummary(s *model.BitcoinStats) string {
	parts := make([]string, 0, 3)
	if s.Users >= 0 {
		parts = append(parts, plural(s.Users, "user"))
	}
	if s.Workers >= 0 {
		w := plural(s.Workers, "worker")
		if s.WorkersIdle > 0 {
			w += fmt.Sprintf(" (%d idle)", s.WorkersIdle)
		}
		parts = append(parts, w)
	}
	if len(parts) == 0 {
		return "workers —"
	}
	return strings.Join(parts, " · ")
}

// plural counts a thing in readable English: "1 worker", "2 workers".
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func chainSummary(s *model.BitcoinStats) string {
	parts := make([]string, 0, 4)
	if s.NetworkDiffPct >= 0 {
		parts = append(parts, fmt.Sprintf("%.2f%% of network diff", s.NetworkDiffPct))
	}
	if s.SPS1m >= 0 {
		parts = append(parts, fmt.Sprintf("%.1f shares/s", s.SPS1m))
	}
	if s.BlockHeight >= 0 {
		parts = append(parts, fmt.Sprintf("block %d", s.BlockHeight))
	}
	if s.BlocksFound > 0 {
		parts = append(parts, fmt.Sprintf("%d blocks found", s.BlocksFound))
	}
	if !s.TotalsAsOf.IsZero() {
		parts = append(parts, "pool totals as of "+Ago(s.TotalsAsOf))
	}
	return strings.Join(parts, " · ")
}

// minerTable lists per-miner rows at whichever granularity the source reached:
// devices for public-pool, payout addresses for ckpool. Identity columns left,
// numbers right, hashrate heads naming their window (P4/P5).
func minerTable(p *model.BitcoinPool) string {
	var b strings.Builder
	hash := hashHead(p.Stats)

	if p.Detail == model.DetailDevice {
		fmt.Fprintf(&b, "%-16s %12s %10s %10s %11s\n",
			"WORKER", hash, "BEST", "UP", "LAST SHARE")
		for _, m := range p.Miners {
			fmt.Fprintf(&b, "%-16s %12s %10s %10s %11s\n",
				truncate(m.Name, 16), Hashrate(m.Hashrate), Difficulty(m.BestDifficulty),
				Dur(durSince(m.StartTime)), Ago(m.LastSeen))
		}
		return b.String()
	}

	fmt.Fprintf(&b, "%-16s %12s %8s %12s %10s %11s\n",
		"ADDRESS", hash, "WORKERS", "SHARES", "BEST", "LAST SHARE")
	for _, m := range p.Miners {
		workers := "—"
		if m.Workers >= 0 {
			workers = fmt.Sprintf("%d", m.Workers)
		}
		fmt.Fprintf(&b, "%-16s %12s %8s %12s %10s %11s\n",
			ShortAddress(m.Name), Hashrate(m.Hashrate), workers,
			Shares(m.Shares, model.Unknown), Difficulty(m.BestDifficulty), Ago(m.LastSeen))
	}
	return b.String()
}

// hashHead names the averaging window in the column head, or leaves it off when
// the figure is instantaneous — a column must not imply an average it is not
// (P5). ckpool averages over a minute; public-pool reports what is live now.
func hashHead(s *model.BitcoinStats) string {
	if s == nil || s.HashrateWindow == "" || s.HashrateWindow == "now" || s.HashrateWindow == "pool" {
		return "HASH"
	}
	return "HASH/" + s.HashrateWindow
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
