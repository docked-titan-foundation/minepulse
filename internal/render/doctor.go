package render

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"

	"github.com/docked-titan-foundation/minepulse/internal/diag"
)

var statusStyle = map[diag.CheckStatus]lipgloss.Style{
	diag.StatusOK:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	diag.StatusWarn: lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
	diag.StatusFail: lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
	diag.StatusInfo: lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
}

var statusGlyph = map[diag.CheckStatus]string{
	diag.StatusOK:   "✓",
	diag.StatusWarn: "⚠",
	diag.StatusFail: "✗",
	diag.StatusInfo: "•",
}

// Doctor writes a doctor report as a human-readable checklist. Remedies are
// shown indented under any non-OK check.
func Doctor(w io.Writer, r *diag.Report) {
	fmt.Fprintln(w, titleSt.Render("⛏  minepulse doctor"))
	for _, c := range r.Checks {
		st := statusStyle[c.Status]
		fmt.Fprintf(w, "  %s %-16s %s\n",
			st.Render(statusGlyph[c.Status]),
			c.Name,
			dimStyle.Render(c.Detail))
		if c.Remedy != "" && c.Status != diag.StatusOK {
			fmt.Fprintf(w, "      %s %s\n", dimStyle.Render("→"), st.Render(c.Remedy))
		}
	}
	// Summary line.
	switch r.Worst() {
	case diag.StatusFail:
		fmt.Fprintln(w, badSt.Render("\nsome checks failed."))
	case diag.StatusWarn:
		fmt.Fprintln(w, warnSt.Render("\ncompleted with warnings."))
	default:
		fmt.Fprintln(w, goodSt.Render("\nall good."))
	}
}
