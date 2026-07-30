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
	for _, warn := range s.Warnings {
		fmt.Fprintf(w, "! %s\n", warn)
	}
	fmt.Fprintln(w, strings.Repeat("─", 40))
}

func minerCPU(n model.NodeStatus) string {
	if n.CPU == nil {
		return "n/a"
	}
	return fmt.Sprintf("%dm", n.CPU.MinerMilli)
}

func nodeFree(n model.NodeStatus) string {
	if n.CPU == nil {
		return "n/a"
	}
	return Pct(n.CPU.FreePct)
}

func note(n model.NodeStatus) string {
	if n.Note != "" {
		return n.Note
	}
	return "—"
}
