package render

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// moneroBodyGolden is the Monero tab's body, exactly as it rendered before the
// presentation standard (specs/004) was applied to the Bitcoin tab.
//
// FR-001 says the reference tab does not move. This pins it: any change to the
// Monero renderer — including one made "while tidying" the shared helpers it
// happens to use — fails here rather than reaching a terminal.
const moneroBodyGolden = "cluster  2/2 mining · 617 H/s · 42 shares✓ · miner 5000m\n" +
	"\n" +
	"node CPU free ███░░░░░░░░░░░░░░░░░░░░░ 12%\n" +
	"\n" +
	"NODE         STATE             HASH/60s    THR     SHARES    MINER   FREE  CPU-FREE ~2m   POOL\n" +
	// The POOL column lines up across rows with and without a sparkline: the
	// width helpers measure display columns, so the 6-rune (18-byte) sparkline
	// is padded to its 14-column cell like any other value. Before that fix this
	// row's POOL started flush against the sparkline while the next row's was
	// aligned — the one change to this tab since it became the reference.
	"andromeda    Running            377 H/s    6/8     42✓/2✗    5000m    12%  ▃▃▂▁▂▃         pool.supportxmr.com:443\n" +
	"orion        DONATE⚠            240 H/s    4/6      0✓/0✗      n/a    n/a                 donate.v2.xmrig.com:3333\n" +
	"\n" +
	"pool  662 H/s reported · due 0.003120 XMR · paid 0.184000 XMR · last share —"

func TestMoneroTabIsUnchanged(t *testing.T) {
	snap := sampleSnapshot()
	// A Bitcoin view must not leak into the Monero body either.
	snap.Bitcoin = &model.BitcoinView{Scope: "all namespaces", Pools: []model.BitcoinPool{addressPool()}}
	m := tuiModel{snap: snap, updated: time.Now(), tab: tabMonero}

	got := stripANSI(m.body())
	if got != moneroBodyGolden {
		t.Errorf("the Monero tab moved (FR-001)\n--- want\n%s\n--- got\n%s", moneroBodyGolden, got)
	}
}

// P4: a table column starts at the same display column on every row, whether or
// not the cells before it contain multi-byte glyphs. This is what the width
// helpers exist for — a sparkline is 18 bytes and 6 columns wide, and measuring
// the wrong one ragged the POOL column for rows that had CPU history.
func TestNodeTableColumnsAlign(t *testing.T) {
	snap := sampleSnapshot()
	var starts []int
	for i := range snap.Nodes {
		row := stripANSI(renderNodeRow(snap.Nodes[i]))
		idx := strings.Index(row, "pool.supportxmr.com")
		if idx < 0 {
			idx = strings.Index(row, "donate.v2.xmrig.com")
		}
		if idx < 0 {
			t.Fatalf("no POOL cell in row %q", row)
		}
		starts = append(starts, lipgloss.Width(row[:idx]))
	}
	for i := 1; i < len(starts); i++ {
		if starts[i] != starts[0] {
			t.Errorf("POOL column starts at %d on row %d but %d on row 0 — the sparkline cell is not padded to its width",
				starts[i], i, starts[0])
		}
	}
}

// Width helpers must count display columns, not bytes, and must never slice a
// rune in half.
func TestWidthHelpers(t *testing.T) {
	if got := pad("▃▃▂▁▂▃", 14); lipgloss.Width(got) != 14 {
		t.Errorf("pad(sparkline, 14) is %d columns wide, want 14", lipgloss.Width(got))
	}
	if got := pad("abc", 6); got != "abc   " {
		t.Errorf("pad(\"abc\", 6) = %q", got)
	}
	if got := truncate("bitaxe-01", 16); got != "bitaxe-01" {
		t.Errorf("truncate must leave a short string alone, got %q", got)
	}
	got := truncate("ɱonero-node-with-a-long-name", 12)
	if lipgloss.Width(got) > 12 {
		t.Errorf("truncate(…, 12) = %q, %d columns wide", got, lipgloss.Width(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated text must end in an ellipsis, got %q", got)
	}
	if !utf8.ValidString(got) {
		t.Errorf("truncate sliced a rune in half: %q", got)
	}
}

// The rules a reviewer would otherwise have to check by eye, as tests.
func TestBitcoinTabFollowsTheStandard(t *testing.T) {
	view := &model.BitcoinView{
		Scope: "all namespaces",
		Pools: []model.BitcoinPool{devicePool(), addressPool()},
	}
	body := stripANSI(bitcoinBody(view))
	lines := strings.Split(body, "\n")

	// P1: panel content is flush left — the box's padding is the only indent.
	for i, l := range lines {
		if l != "" && (strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t")) {
			t.Errorf("P1: line %d is indented: %q", i+1, l)
		}
	}

	// P7: one unavailable mark on this tab. "n/a" is the Monero tab's exception.
	if strings.Contains(body, "n/a") {
		t.Errorf("P7: the Bitcoin tab must use — for unavailable, found n/a\n---\n%s", body)
	}

	// P5: an averaged hashrate column names its window; an instantaneous one
	// must not borrow one it does not have.
	if !strings.Contains(body, "HASH/1m") {
		t.Errorf("P5: ckpool's averaged column must name its window\n---\n%s", body)
	}
	if strings.Contains(body, "HASHRATE") {
		t.Errorf("P5: a bare HASHRATE column head hides its averaging window\n---\n%s", body)
	}

	// P6: shares read the same way everywhere they appear.
	if !strings.Contains(body, "12483✓/3✗") {
		t.Errorf("P6: shares must render as N✓/M✗ in the table too\n---\n%s", body)
	}

	// P2: each panel opens with its bold role label followed by labeled
	// metrics, and omits figures its source does not report rather than
	// printing a meaningless bare dash into the list.
	for _, want := range []string{
		"ckpool  480.00 TH/s (1m)", "12483✓/3✗ shares",
		"public-pool  1.68 TH/s", "best 4.10 G",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("P2: panel header must carry the metrics, missing %q\n---\n%s", want, body)
		}
	}
	if strings.Contains(body, "· — ·") {
		t.Errorf("P2: an unreported figure must be omitted from the header, not dashed\n---\n%s", body)
	}

	// P3: provenance is stated once, on the context line.
	if strings.Count(body, "via logs") != 1 {
		t.Errorf("P3: the stats source must be stated exactly once per panel\n---\n%s", body)
	}

	// P9: one blank line between panels, as between Monero's sections.
	if strings.Contains(body, "\n\n\n") {
		t.Errorf("P9: panels are separated by exactly one blank line\n---\n%s", body)
	}
}

// stripANSI removes the escape sequences lipgloss adds, so the tests compare the
// text a reader sees rather than the styling around it.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++ // skip the 'm'
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
