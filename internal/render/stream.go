package render

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// Stream writes one compact, self-contained text block for a snapshot — meant
// for piping, background runs, and agent consumption (no ANSI, no cursor moves).
func Stream(w io.Writer, s *model.Snapshot) {
	c := s.Cluster
	fmt.Fprintf(w, "── minepulse %s ──\n", s.Timestamp.Format("15:04:05"))
	if e := s.Endpoint; e != nil {
		div := ""
		if e.Diverged {
			div = "  ! miners disagree on the pool"
		}
		fmt.Fprintf(w, "%s%s\n", PoolLine(e), div)
	}
	fmt.Fprintf(w, "cluster: %d/%d mining · %s · shares %d✓/%d✗ · miner CPU %dm · node free %s\n",
		c.NodesMining, c.NodesTotal, Hashrate(c.TotalHashrate),
		c.AcceptedShares, c.RejectedShares, c.MinerCPUMilli, Pct(c.NodeCPUFreePct))

	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "  NODE\tSTATE\tHASH/60s\tTHREADS\tSHARES\tMINER CPU\tNODE FREE\tPOOL")
	for _, n := range s.Nodes {
		state := n.Phase
		if n.Mining == nil {
			fmt.Fprintf(tw, "  %s\t%s\t—\t—\t—\t%s\t%s\t%s\n",
				n.Node, state, minerCPU(n), nodeFree(n), note(n))
			continue
		}
		m := n.Mining
		pool := m.Pool
		if m.DonateFallback {
			pool += " ⚠DONATE"
		} else if !m.Connected {
			pool += " (disconnected)"
		}
		src := ""
		if n.StatsSource == model.SourceLogs {
			src = " (logs)"
		}
		fmt.Fprintf(tw, "  %s\t%s%s\t%s\t%d/%d\t%d✓/%d✗\t%s\t%s\t%s\n",
			n.Node, state, src, Hashrate(m.Hashrate60s),
			m.ThreadsActive, m.ThreadsTotal,
			m.SharesGood, m.SharesTotal-m.SharesGood,
			minerCPU(n), nodeFree(n), pool)
	}
	_ = tw.Flush()

	if s.Pool != nil {
		p := s.Pool
		stale := ""
		if p.Stale {
			stale = " (stale, as of " + Ago(p.AsOf) + ")"
		}
		fmt.Fprintf(w, "pool: %s reported · due %s · paid %s · last share %s%s\n",
			Hashrate(p.ReportedHashrate), XMR(p.AmountDueXMR), XMR(p.AmountPaidXMR),
			Ago(p.LastShare), stale)
	}
	streamBitcoin(w, s.Bitcoin)

	for _, warn := range s.Warnings {
		fmt.Fprintf(w, "! %s\n", warn)
	}
	fmt.Fprintln(w, strings.Repeat("─", 40))
}

// streamBitcoin writes the Bitcoin block. A single line when there is nothing to
// report, so a script can tell "no pool" from "broken pool" without parsing JSON.
func streamBitcoin(w io.Writer, v *model.BitcoinView) {
	if v == nil {
		fmt.Fprintln(w, "bitcoin: no pool detected")
		return
	}
	if len(v.Pools) == 0 {
		note := v.Note
		if note == "" {
			note = "no pool found in " + v.Scope
		}
		fmt.Fprintf(w, "bitcoin: %s\n", note)
		return
	}

	for i := range v.Pools {
		p := &v.Pools[i]
		state := p.Phase
		if !p.Running {
			state += " (not running)"
		}
		src := string(p.Source)
		if p.Stale {
			src += ", stale " + Ago(p.AsOf)
		}
		if e := p.Endpoint; e != nil {
			fmt.Fprintf(w, "bitcoin: %s\n", PoolLine(e))
		}
		fmt.Fprintf(w, "bitcoin: %s (%s) · %s/%s · %s %s\n",
			p.Impl, src, p.Namespace, p.Pod, state, Dur(p.Uptime))

		if p.Stats == nil {
			if p.Note != "" {
				fmt.Fprintf(w, "  ! %s\n", p.Note)
			}
			if p.Remedy != "" {
				fmt.Fprintf(w, "  → %s\n", p.Remedy)
			}
			continue
		}
		// The same headline the TUI builds, so both outputs report a figure —
		// or omit it — identically (P2/P12).
		fmt.Fprintf(w, "  %s\n", headlineMetrics(p.Stats))

		if len(p.Miners) > 0 {
			// Columns follow the granularity: a device has no worker count or
			// share total to report, an address has no session to have started.
			// Same column vocabulary as the TUI (P4-P7, P12): the hashrate head
			// names its window, shares read N✓/M✗, identities truncate.
			tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
			hash := hashHead(p.Stats)
			if p.Detail == model.DetailDevice {
				fmt.Fprintf(tw, "  WORKER\t%s\tBEST\tLAST SHARE\n", hash)
				for _, m := range p.Miners {
					fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
						truncate(m.Name, 16), Hashrate(m.Hashrate),
						Difficulty(m.BestDifficulty), Ago(m.LastSeen))
				}
			} else {
				fmt.Fprintf(tw, "  ADDRESS\t%s\tWORKERS\tSHARES\tBEST\tLAST SHARE\n", hash)
				for _, m := range p.Miners {
					workers := unavailable
					if m.Workers >= 0 {
						workers = fmt.Sprintf("%d", m.Workers)
					}
					fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\t%s\n",
						ShortAddress(m.Name), Hashrate(m.Hashrate), workers,
						Shares(m.Shares, model.Unknown),
						Difficulty(m.BestDifficulty), Ago(m.LastSeen))
				}
			}
			_ = tw.Flush()
		}
		if p.Note != "" {
			fmt.Fprintf(w, "  ! %s\n", p.Note)
		}
		if p.Remedy != "" {
			fmt.Fprintf(w, "  → %s\n", p.Remedy)
		}
	}
}

func minerCPU(n model.NodeStatus) string {
	if n.CPU == nil {
		return unavailable
	}
	return fmt.Sprintf("%dm", n.CPU.MinerMilli)
}

func nodeFree(n model.NodeStatus) string {
	if n.CPU == nil {
		return unavailable
	}
	return Pct(n.CPU.FreePct)
}

func note(n model.NodeStatus) string {
	if n.Note != "" {
		return n.Note
	}
	return unavailable
}
