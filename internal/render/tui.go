package render

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/docked-titan-foundation/minepulse/internal/config"
	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// GatherFunc produces one snapshot; the TUI calls it on each tick.
type GatherFunc func(ctx context.Context) (*model.Snapshot, error)

// RunTUI runs the interactive dashboard until the user quits or ctx is canceled.
// tab is the coin it opens on; m/b still switch freely once it is running.
func RunTUI(ctx context.Context, interval time.Duration, tab config.Tab, gather GatherFunc) error {
	m := tuiModel{gather: gather, interval: interval, ctx: ctx, tab: tabFor(tab)}
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

// tabFor maps the configured coin onto the view's tab. An unknown value falls
// back to Monero; the command layer has already rejected it by this point.
func tabFor(t config.Tab) coinTab {
	if t == config.TabBitcoin {
		return tabBitcoin
	}
	return tabMonero
}

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
	tabOnSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	btcOrange = lipgloss.Color("#f7931a")
	xmrIconSt = lipgloss.NewStyle().Foreground(xmrOrange)
	btcIconSt = lipgloss.NewStyle().Foreground(btcOrange)
	boxSt     = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
)

func (m tuiModel) View() string {
	var b strings.Builder
	b.WriteString(m.title() + "\n\n")
	b.WriteString(m.tabs() + "\n")
	b.WriteString(m.box(m.body()))
	b.WriteString(footer())
	return b.String()
}

// Coin marks for the tab strip: Monero's ɱ (U+0271) and Bitcoin's ₿ (U+20BF).
// Both are single-width glyphs in a terminal font, so they cost one column and
// leave the strip's alignment alone.
const (
	xmrIcon = "ɱ"
	btcIcon = "₿"
)

// tabs is the strip above the box naming the two coins, active one highlighted.
// Each coin's mark keeps its own color whether or not its tab is selected, so
// the strip reads at a glance; only the label dims.
func (m tuiModel) tabs() string {
	name := func(icon, label string, iconSt lipgloss.Style, t coinTab) string {
		if m.tab == t {
			return tabOnSt.Render("[") + iconSt.Bold(true).Render(icon) +
				tabOnSt.Render(" "+label+"]")
		}
		return " " + iconSt.Render(icon) + dimStyle.Render(" "+label+" ")
	}
	return " " + name(xmrIcon, "monero", xmrIconSt, tabMonero) +
		"  " + name(btcIcon, "bitcoin", btcIconSt, tabBitcoin)
}

// title is the banner above the box: name, last update, pause state.
func (m tuiModel) title() string {
	status := ""
	if m.paused {
		status = warnSt.Render(" [paused]")
	}
	upd := unavailable
	if !m.updated.IsZero() {
		upd = m.updated.Format("15:04:05")
	}
	return fmt.Sprintf("%s  %s%s",
		titleSt.Render("⛏  minepulse"),
		dimStyle.Render("updated "+upd),
		status)
}

// box frames the body. It hugs the content rather than stretching across the
// terminal: on a wide screen a short panel framed at 250 columns is mostly
// border, and the eye has to travel the whole width to find the next value.
func (m tuiModel) box(body string) string {
	st := boxSt
	// Style.Width covers the content plus this style's horizontal padding (one
	// column each side), and the border sits outside it — so a box that hugs
	// its content asks for content+2, and needs content+4 columns on screen.
	want := lipgloss.Width(body) + 2
	if avail := m.w - 2; avail > 0 && avail < want {
		return st.Render(body) // too narrow to frame neatly; overflow as before
	}
	return st.Width(want).Render(body)
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

// moneroBody is the Monero tab: cluster totals, nodes, pool, warnings.
func (m tuiModel) moneroBody() string {
	var b strings.Builder
	s := m.snap
	c := s.Cluster

	// Header: cluster totals. Free CPU is a figure here rather than a gauge —
	// the bar that answers "which node is tight" belongs on the rows, where the
	// comparison is (specs/006).
	fmt.Fprintf(&b, "%s  %s mining · %s · %s shares✓ · miner %s · node free %s\n\n",
		headSt.Render("cluster"),
		fmt.Sprintf("%d/%d", c.NodesMining, c.NodesTotal),
		Hashrate(c.TotalHashrate),
		fmt.Sprintf("%d", c.AcceptedShares),
		fmt.Sprintf("%dm", c.MinerCPUMilli),
		Pct(c.NodeCPUFreePct))

	// Node table.
	b.WriteString(nodeTable(s.Nodes))

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

// Per-column truncation bounds. They exist so one pathological value — a
// fully-qualified node name, an unusually verbose phase — cannot widen the whole
// table; every other column is free to size itself to its content.
const (
	nodeColMax  = 20
	stateColMax = 20
)

// nodeTable is the Monero tab's node grid.
func nodeTable(nodes []model.NodeStatus) string {
	t := newTable(
		column{"NODE", alignLeft},
		column{"STATE", alignLeft},
		column{"HASH/60s", alignRight},
		column{"THR", alignRight},
		column{"SHARES", alignRight},
		column{"MINER", alignRight},
		column{"NODE CPU FREE", alignLeft},
		column{"POOL", alignLeft},
	)
	for i := range nodes {
		t.add(nodeRow(nodes[i])...)
	}
	return t.render()
}

func nodeRow(n model.NodeStatus) []string {
	state := truncate(n.Phase, stateColMax)
	var stateStyled string
	switch {
	case n.Mining != nil && n.Mining.DonateFallback:
		stateStyled = badSt.Render("DONATE⚠")
	case n.Mining != nil && n.Mining.Connected:
		stateStyled = goodSt.Render(state)
	case !isRunning(n):
		stateStyled = badSt.Render(state)
	default:
		stateStyled = warnSt.Render(state)
	}

	hash, thr, shares := unavailable, unavailable, unavailable
	pool := unavailable
	if n.Mining != nil {
		hash = Hashrate(n.Mining.Hashrate60s)
		thr = fmt.Sprintf("%d/%d", n.Mining.ThreadsActive, n.Mining.ThreadsTotal)
		shares = fmt.Sprintf("%d✓/%d✗", n.Mining.SharesGood, n.Mining.SharesTotal-n.Mining.SharesGood)
		pool = n.Mining.Pool
		if n.StatsSource == model.SourceLogs {
			pool += dimStyle.Render(" (logs)")
		}
	}
	miner, free := unavailable, unavailable
	if n.CPU != nil {
		miner = fmt.Sprintf("%dm", n.CPU.MinerMilli)
		free = cpuBar(n.CPU.FreePct, cpuBarWidth)
	}

	return []string{
		truncate(n.Node, nodeColMax), stateStyled,
		hash, thr, shares, miner, free, pool,
	}
}

func isRunning(n model.NodeStatus) bool { return n.Phase == "Running" }

// cpuBarWidth is how many cells the trough gets. Eight is enough to rank a
// handful of nodes by eye, and keeps the column (bar, space, percentage) no
// wider than its own head.
const cpuBarWidth = 8

// cpuBar draws a 0..100 percentage as a filled trough with the figure beside
// it, in the thresholds the cluster gauge used before this bar replaced it:
// green with headroom, amber when tight, red when the node is starved.
//
// The percentage stays next to the bar because color may not be the only thing
// carrying a reading (Constitution: terminal-safe, no reliance on color alone),
// and because two bars of similar length are quicker to separate by number than
// by eye. No metrics at all is the unavailable mark — an empty trough would read
// as "0% free", which is a very different and much more alarming claim.
func cpuBar(pct float64, width int) string {
	if pct < 0 {
		return unavailable
	}
	// Round rather than truncate, and never let a node with headroom left draw an
	// empty trough: at 10% free an 8-cell bar truncates to nothing, which is the
	// same picture a saturated node paints. A sliver is the honest rendering.
	filled := int(math.Round(pct / 100 * float64(width)))
	if filled == 0 && pct > 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	st := goodSt
	if pct < 15 {
		st = badSt
	} else if pct < 30 {
		st = warnSt
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return st.Render(bar) + " " + st.Render(Pct(pct))
}

func footer() string {
	return "\n" + dimStyle.Render("q quit · p pause · r refresh · m/b coin")
}
