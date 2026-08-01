package render

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// GatherFunc produces one snapshot; the TUI calls it on each tick.
type GatherFunc func(ctx context.Context) (*model.Snapshot, error)

// RunTUI runs the interactive dashboard until the user quits or ctx is canceled.
func RunTUI(ctx context.Context, interval time.Duration, gather GatherFunc) error {
	m := tuiModel{gather: gather, interval: interval, ctx: ctx}
	p := tea.NewProgram(m, tea.WithAltScreen())
	go func() {
		<-ctx.Done()
		p.Quit()
	}()
	_, err := p.Run()
	return err
}

type tickMsg time.Time
type snapMsg struct {
	snap *model.Snapshot
	err  error
}

// coinTab is which view the box shows. It is pure UI state: both coins are
// gathered every tick, so switching costs nothing and shows nothing staler.
type coinTab int

const (
	tabMonero coinTab = iota
	tabBitcoin
)

type tuiModel struct {
	gather    GatherFunc
	interval  time.Duration
	ctx       context.Context
	snap      *model.Snapshot
	err       error
	w, h      int
	paused    bool
	gathering bool
	tab       coinTab
	updated   time.Time
}

func (m tuiModel) Init() tea.Cmd { return tea.Batch(m.gatherCmd(), tickCmd(m.interval)) }

func (m tuiModel) gatherCmd() tea.Cmd {
	return func() tea.Msg {
		s, err := m.gather(m.ctx)
		return snapMsg{snap: s, err: err}
	}
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "p":
			m.paused = !m.paused
			return m, nil
		case "r":
			if !m.gathering {
				m.gathering = true
				return m, m.gatherCmd()
			}
			return m, nil
		case "m":
			m.tab = tabMonero
			return m, nil
		case "b":
			m.tab = tabBitcoin
			return m, nil
		case "tab":
			if m.tab == tabMonero {
				m.tab = tabBitcoin
			} else {
				m.tab = tabMonero
			}
			return m, nil
		}
	case tickMsg:
		cmds := []tea.Cmd{tickCmd(m.interval)}
		if !m.paused && !m.gathering {
			m.gathering = true
			cmds = append(cmds, m.gatherCmd())
		}
		return m, tea.Batch(cmds...)
	case snapMsg:
		m.gathering = false
		if msg.err != nil {
			m.err = msg.err
		} else if msg.snap != nil {
			m.snap = msg.snap
			m.err = nil
			m.updated = time.Now()
		}
		return m, nil
	}
	return m, nil
}

// ── styles ──────────────────────────────────────────────────────────────────
var (
	xmrOrange = lipgloss.Color("#ff6600")
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	titleSt   = lipgloss.NewStyle().Foreground(xmrOrange).Bold(true)
	headSt    = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Bold(true)
	goodSt    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnSt    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	badSt     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	sparkSt   = lipgloss.NewStyle().Foreground(xmrOrange)
	tabOnSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	boxSt     = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
)

func (m tuiModel) View() string {
	var b strings.Builder
	b.WriteString(m.title() + "\n")
	b.WriteString(m.tabs() + "\n")
	b.WriteString(m.box(m.body()))
	b.WriteString(footer())
	return b.String()
}

// tabs is the strip above the box naming the two coins, active one highlighted.
func (m tuiModel) tabs() string {
	name := func(label string, t coinTab) string {
		if m.tab == t {
			return tabOnSt.Render("[" + label + "]")
		}
		return dimStyle.Render(" " + label + " ")
	}
	return " " + name("monero", tabMonero) + "  " + name("bitcoin", tabBitcoin)
}

// title is the banner above the box: name, last update, pause state.
func (m tuiModel) title() string {
	status := ""
	if m.paused {
		status = warnSt.Render(" [paused]")
	}
	upd := "—"
	if !m.updated.IsZero() {
		upd = m.updated.Format("15:04:05")
	}
	return fmt.Sprintf("%s  %s%s",
		titleSt.Render("⛏  minepulse"),
		dimStyle.Render("updated "+upd),
		status)
}

// box frames the body, widening to the terminal when the content allows it.
func (m tuiModel) box(body string) string {
	st := boxSt
	// Border + padding cost 4 columns; only stretch when nothing would wrap.
	if avail := m.w - 4; avail > 0 && avail >= lipgloss.Width(body) {
		st = st.Width(avail)
	}
	return st.Render(body)
}

// body is the active tab's content.
func (m tuiModel) body() string {
	var b strings.Builder

	if m.err != nil {
		b.WriteString(badSt.Render("error: "+m.err.Error()) + "\n")
	}
	if m.snap == nil {
		b.WriteString(dimStyle.Render("gathering…"))
		return b.String()
	}
	if m.tab == tabBitcoin {
		b.WriteString(bitcoinBody(m.snap.Bitcoin))
		return strings.TrimRight(b.String(), "\n")
	}
	return b.String() + m.moneroBody()
}

// moneroBody is the Monero tab: cluster totals, gauge, nodes, pool, warnings.
func (m tuiModel) moneroBody() string {
	var b strings.Builder
	s := m.snap
	c := s.Cluster

	// Header: cluster totals + free-CPU gauge.
	fmt.Fprintf(&b, "%s  %s mining · %s · %s shares✓ · miner %s\n",
		headSt.Render("cluster"),
		fmt.Sprintf("%d/%d", c.NodesMining, c.NodesTotal),
		Hashrate(c.TotalHashrate),
		fmt.Sprintf("%d", c.AcceptedShares),
		fmt.Sprintf("%dm", c.MinerCPUMilli))
	b.WriteString("\n" + gauge("node CPU free", c.NodeCPUFreePct, 24) + "\n\n")

	// Node table.
	fmt.Fprintf(&b, "%-12s %-14s %11s %6s %10s %8s %6s  %-14s %s\n",
		"NODE", "STATE", "HASH/60s", "THR", "SHARES", "MINER", "FREE", "CPU-FREE ~2m", "POOL")
	for i := range s.Nodes {
		b.WriteString(renderNodeRow(s.Nodes[i]))
	}

	// Pool panel.
	if s.Pool != nil {
		p := s.Pool
		stale := ""
		if p.Stale {
			stale = warnSt.Render(" (stale " + Ago(p.AsOf) + ")")
		}
		fmt.Fprintf(&b, "\n%s  %s reported · due %s · paid %s · last share %s%s\n",
			headSt.Render("pool"), Hashrate(p.ReportedHashrate),
			XMR(p.AmountDueXMR), XMR(p.AmountPaidXMR), Ago(p.LastShare), stale)
	}
	for _, w := range s.Warnings {
		b.WriteString(warnSt.Render("! "+w) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderNodeRow(n model.NodeStatus) string {
	state := n.Phase
	var stateStyled string
	switch {
	case n.Mining != nil && n.Mining.DonateFallback:
		stateStyled = badSt.Render("DONATE⚠")
	case n.Mining != nil && n.Mining.Connected:
		stateStyled = goodSt.Render(truncate(state, 14))
	case !isRunning(n):
		stateStyled = badSt.Render(truncate(state, 14))
	default:
		stateStyled = warnSt.Render(truncate(state, 14))
	}

	hash, thr, shares := "—", "—", "—"
	pool := "—"
	if n.Mining != nil {
		hash = Hashrate(n.Mining.Hashrate60s)
		thr = fmt.Sprintf("%d/%d", n.Mining.ThreadsActive, n.Mining.ThreadsTotal)
		shares = fmt.Sprintf("%d✓/%d✗", n.Mining.SharesGood, n.Mining.SharesTotal-n.Mining.SharesGood)
		pool = n.Mining.Pool
		if n.StatsSource == model.SourceLogs {
			pool += dimStyle.Render(" (logs)")
		}
	}
	miner, free, spark := "n/a", "n/a", ""
	if n.CPU != nil {
		miner = fmt.Sprintf("%dm", n.CPU.MinerMilli)
		free = Pct(n.CPU.FreePct)
	}
	if n.History != nil {
		spark = sparkSt.Render(pad(Sparkline(n.History.FreePctSeries(), 0, 100, 14), 14))
	} else {
		spark = pad("", 14)
	}

	// NODE STATE HASH THR SHARES MINER FREE SPARK POOL
	return fmt.Sprintf("%-12s %-14s %11s %6s %10s %8s %6s  %s %s\n",
		truncate(n.Node, 12), padStyled(stateStyled, 14),
		hash, thr, shares, miner, free, spark, pool)
}

func isRunning(n model.NodeStatus) bool { return n.Phase == "Running" }

// gauge renders a labeled horizontal bar for a 0..100 percentage (or n/a).
func gauge(label string, pct float64, width int) string {
	if pct < 0 {
		return dimStyle.Render(fmt.Sprintf("%s: n/a", label))
	}
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	st := goodSt
	if pct < 15 {
		st = badSt
	} else if pct < 30 {
		st = warnSt
	}
	return fmt.Sprintf("%s %s %s", dimStyle.Render(label), st.Render(bar), st.Render(Pct(pct)))
}

func footer() string {
	return "\n" + dimStyle.Render("q quit · p pause · r refresh · m/b coin")
}

// ── small width helpers (styled strings break %-Ns padding) ──────────────────
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// padStyled pads based on the plain-text length of an already-styled cell.
func padStyled(styled string, n int) string {
	w := lipgloss.Width(styled)
	if w >= n {
		return styled
	}
	return styled + strings.Repeat(" ", n-w)
}
