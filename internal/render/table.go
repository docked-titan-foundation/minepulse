package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The dashboard's one table renderer (standard P4/P13).
//
// Panels hand it columns and already-formatted cells; it decides the widths and
// draws the rules. Two things follow from that which the per-panel format
// strings it replaced could not do: a column is exactly as wide as its content
// needs — so a phase is never truncated to pay for padding a hashrate — and the
// grid lines land in the same place on every row, because every width here is
// measured in display columns rather than bytes.

type colAlign int

const (
	alignLeft colAlign = iota
	alignRight
)

// column is a head and how its cells sit under it. Identity left, numbers right.
type column struct {
	head  string
	align colAlign
}

type table struct {
	cols []column
	rows [][]string
}

func newTable(cols ...column) *table { return &table{cols: cols} }

// add appends one row. Cells are already styled and formatted — the table only
// places them.
func (t *table) add(cells ...string) { t.rows = append(t.rows, cells) }

// Grid pieces. All single-width, like every other glyph the dashboard draws.
const (
	colSep    = "│"
	ruleLine  = "─"
	ruleCross = "┼"
	// gutter is the space between a rule and the cell on either side of it.
	gutter = 1
)

// render draws the head row, the rule beneath it, and the rows. It ends in a
// newline like the format-string tables it replaced, so callers append to it
// the same way.
func (t *table) render() string {
	if len(t.cols) == 0 {
		return ""
	}
	w := t.widths()
	last := len(t.cols) - 1

	var b strings.Builder
	line := func(cells []string) {
		for i := range t.cols {
			if i > 0 {
				b.WriteString(dimStyle.Render(colSep))
			}
			v := ""
			if i < len(cells) {
				v = cells[i]
			}
			b.WriteString(t.cell(v, w[i], i == 0, i == last, t.cols[i].align))
		}
		b.WriteString("\n")
	}

	heads := make([]string, len(t.cols))
	for i, c := range t.cols {
		heads[i] = headSt.Render(c.head)
	}
	line(heads)

	segs := make([]string, len(t.cols))
	for i := range t.cols {
		// Each segment spans its cell plus the gutters beside it — which the
		// outermost columns do not have, so the rule begins and ends with the
		// table rather than hanging a column past it at either edge.
		n := w[i] + 2*gutter
		if i == 0 {
			n -= gutter
		}
		if i == last {
			n -= gutter
		}
		segs[i] = strings.Repeat(ruleLine, n)
	}
	b.WriteString(dimStyle.Render(strings.Join(segs, ruleCross)) + "\n")

	for _, r := range t.rows {
		line(r)
	}
	return b.String()
}

// cell lays one value into its column: the value padded on the side its
// alignment asks for, with a gutter on each side that borders a rule. The first
// column drops its leading gutter so the table starts flush left like every
// other line in the panel (P1); the last drops its trailing one, since nothing
// follows it but the end of the line and trailing blanks would widen the box
// that hugs this content.
func (t *table) cell(v string, width int, first, last bool, a colAlign) string {
	switch {
	case a == alignRight:
		v = padLeft(v, width)
	case !last:
		v = pad(v, width)
	}
	if !first {
		v = strings.Repeat(" ", gutter) + v
	}
	if !last {
		v += strings.Repeat(" ", gutter)
	}
	return v
}

// widths sizes every column to the wider of its head and its widest cell
// (FR-002). A row shorter than the column list contributes nothing to the
// columns it does not reach.
func (t *table) widths() []int {
	w := make([]int, len(t.cols))
	for i, c := range t.cols {
		w[i] = lipgloss.Width(c.head)
	}
	for _, r := range t.rows {
		for i, v := range r {
			if i >= len(w) {
				break
			}
			if n := lipgloss.Width(v); n > w[i] {
				w[i] = n
			}
		}
	}
	return w
}

// dots is the placeholder for a cell with nothing to draw yet, sized to match
// the real values beside it: inside a ruled grid a blank cell reads as a
// rendering fault, a dotted one as "nothing recorded" (FR-006).
func dots(width int) string {
	if width < 1 {
		width = 1
	}
	return dimStyle.Render(strings.Repeat("·", width))
}

// ── width helpers ───────────────────────────────────────────────────────────
//
// All of these measure *display columns*, not bytes: a sparkline of six block
// characters is 18 bytes but occupies six columns, and byte arithmetic on it
// both misaligns the following column and can slice a rune in half. lipgloss's
// measurement also ignores the escape sequences styling adds, so an already
// colored cell pads to the same width as a plain one.
func truncate(s string, n int) string {
	if lipgloss.Width(s) <= n {
		return s
	}
	if n <= 1 {
		return string([]rune(s)[:n])
	}
	// Take runes until one more would not leave room for the ellipsis.
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > n-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "…"
}

func pad(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}

func padLeft(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return strings.Repeat(" ", n-w) + s
	}
	return s
}
