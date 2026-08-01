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

func bitcoinPool(p *model.BitcoinPool) string {
	var b strings.Builder

	// Identity line: which pool, where, in what state, reading from what.
	state := goodSt.Render(p.Phase)
	if !p.Running {
		state = badSt.Render(p.Phase)
	}
	fmt.Fprintf(&b, "%s  %s · %s/%s · %s %s · %s\n",
		headSt.Render(string(p.Impl)),
		dimStyle.Render("on "+orDash(p.Node)),
		p.Namespace, p.Pod,
		state, dimStyle.Render(Dur(p.Uptime)),
		sourceLabel(p))

	if p.Stats == nil {
		if p.Note != "" {
			b.WriteString("  " + warnSt.Render(p.Note) + "\n")
		}
		if p.Remedy != "" {
			b.WriteString("  " + dimStyle.Render("→ "+p.Remedy) + "\n")
		}
		return b.String()
	}

	s := p.Stats
	fmt.Fprintf(&b, "  %s (%s) · %s · %s · best %s\n",
		Hashrate(s.Hashrate1m), orDash(s.HashrateWindow),
		workerSummary(s), shareSummary(s), Difficulty(s.BestShare))

	if extra := chainSummary(s); extra != "" {
		b.WriteString("  " + dimStyle.Render(extra) + "\n")
	}
	if len(p.Miners) > 0 {
		b.WriteString(minerTable(p))
	}
	if p.Note != "" {
		b.WriteString("  " + dimStyle.Render("! "+p.Note) + "\n")
	}
	if p.Remedy != "" {
		b.WriteString("  " + dimStyle.Render("→ "+p.Remedy) + "\n")
	}
	return b.String()
}

// sourceLabel says where the numbers came from, and how old they are when the
// source failed this tick.
func sourceLabel(p *model.BitcoinPool) string {
	switch {
	case p.Stale:
		return warnSt.Render("stale " + Ago(p.AsOf))
	case p.Source == model.SourceAPI:
		return dimStyle.Render("via API")
	case p.Source == model.SourceLogs:
		return dimStyle.Render("via logs")
	default:
		return warnSt.Render("no stats source")
	}
}

func workerSummary(s *model.BitcoinStats) string {
	parts := make([]string, 0, 3)
	if s.Users >= 0 {
		parts = append(parts, fmt.Sprintf("%d users", s.Users))
	}
	if s.Workers >= 0 {
		w := fmt.Sprintf("%d workers", s.Workers)
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

func shareSummary(s *model.BitcoinStats) string {
	if s.Accepted < 0 && s.Rejected < 0 {
		return "shares —"
	}
	return fmt.Sprintf("%s✓/%s✗", Count(s.Accepted), Count(s.Rejected))
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
// devices for public-pool, payout addresses for ckpool.
func minerTable(p *model.BitcoinPool) string {
	var b strings.Builder
	if p.Detail == model.DetailDevice {
		fmt.Fprintf(&b, "  %-16s %12s %10s %10s %10s\n",
			"WORKER", "HASHRATE", "BEST", "UP", "LAST SHARE")
		for _, m := range p.Miners {
			fmt.Fprintf(&b, "  %-16s %12s %10s %10s %10s\n",
				truncate(m.Name, 16), Hashrate(m.Hashrate), Difficulty(m.BestDifficulty),
				Dur(durSince(m.StartTime)), Ago(m.LastSeen))
		}
		return b.String()
	}
	fmt.Fprintf(&b, "  %-20s %12s %8s %10s %10s %10s\n",
		"ADDRESS", "HASHRATE", "WORKERS", "SHARES", "BEST", "LAST SHARE")
	for _, m := range p.Miners {
		workers := "—"
		if m.Workers >= 0 {
			workers = fmt.Sprintf("%d", m.Workers)
		}
		fmt.Fprintf(&b, "  %-20s %12s %8s %10s %10s %10s\n",
			ShortAddress(m.Name), Hashrate(m.Hashrate), workers,
			Count(m.Shares), Difficulty(m.BestDifficulty), Ago(m.LastSeen))
	}
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
