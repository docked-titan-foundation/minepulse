package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// A column is as wide as the wider of its head and its widest cell, and no
// wider — the whole point of replacing the hardcoded format strings (FR-002).
func TestTableSizesColumnsToContent(t *testing.T) {
	tb := newTable(
		column{"NODE", alignLeft},
		column{"HASH", alignRight},
	)
	tb.add("andromeda", "377 H/s")
	tb.add("a", "1 H/s")

	got := stripANSI(tb.render())
	want := "NODE      │    HASH\n" +
		"──────────┼────────\n" +
		"andromeda │ 377 H/s\n" +
		"a         │   1 H/s\n"
	if got != want {
		t.Errorf("table render\n--- want\n%s\n--- got\n%s", want, got)
	}
}

// The rules must land in the same display column on every line even when cells
// carry color and multi-byte glyphs — the failure the old byte-counted padding
// produced, and the reason every width here goes through lipgloss (FR-003).
func TestTableAlignsStyledAndMultiByteCells(t *testing.T) {
	tb := newTable(
		column{"STATE", alignLeft},
		column{"TREND", alignLeft},
		column{"SHARES", alignRight},
	)
	tb.add(badSt.Render("DONATE⚠"), goodSt.Render("▃▃▂▁▂▃"), "42✓/2✗")
	tb.add(goodSt.Render("Running"), dots(6), "0✓/0✗")

	assertRulesAlign(t, tb.render())

	// The cells arrive intact — the renderer places them, it does not rewrite
	// them. (Whether they carry escape sequences depends on the terminal
	// lipgloss detects, which under `go test` is none; that is not this test's
	// business.)
	for _, want := range []string{"DONATE⚠", "▃▃▂▁▂▃", "42✓/2✗"} {
		if !strings.Contains(stripANSI(tb.render()), want) {
			t.Errorf("the table dropped cell %q\n%s", want, tb.render())
		}
	}
}

// A row may be short: a panel that has nothing for a trailing column should get
// an empty cell, not a panic or a ragged grid.
func TestTableToleratesShortRows(t *testing.T) {
	tb := newTable(
		column{"A", alignLeft},
		column{"B", alignLeft},
		column{"C", alignRight},
	)
	tb.add("one", "two", "three")
	tb.add("only")

	assertRulesAlign(t, tb.render())
}

func TestTableEmpty(t *testing.T) {
	if got := newTable().render(); got != "" {
		t.Errorf("a table with no columns renders %q, want empty", got)
	}
}

// dots stands in for a cell with nothing to draw, at the width of the values
// beside it (FR-006).
func TestDotsPlaceholder(t *testing.T) {
	if got := lipgloss.Width(dots(6)); got != 6 {
		t.Errorf("dots(6) is %d columns wide, want 6", got)
	}
	if got := lipgloss.Width(dots(0)); got != 1 {
		t.Errorf("dots(0) is %d columns wide, want 1 — a cell is never nothing", got)
	}
}

func TestPadLeft(t *testing.T) {
	if got := padLeft("42%", 6); got != "   42%" {
		t.Errorf("padLeft(\"42%%\", 6) = %q", got)
	}
	if got := padLeft("▃▃▂▁▂▃", 8); lipgloss.Width(got) != 8 {
		t.Errorf("padLeft(sparkline, 8) is %d columns wide, want 8", lipgloss.Width(got))
	}
	if got := padLeft("toolong", 3); got != "toolong" {
		t.Errorf("padLeft must not truncate, got %q", got)
	}
}
