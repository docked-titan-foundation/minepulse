package render

import (
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// moneroBodyGolden is the Monero tab's body as it renders today: the ruled grid
// from specs/005, with the per-node CPU bar from specs/006.
//
// 004-FR-001 froze this tab byte-for-byte; 005-FR-008 superseded that freeze.
// The guard survives it: the tab may not move by accident, including "while
// tidying" a shared helper it happens to use.
//
// Note what the identity line does with a divergent cluster: it names the pool
// most miners are on and says out loud that they disagree, rather than reporting
// the majority as the whole truth — orion is on the donate pool (007-FR-007).
//
// Three properties to notice. Columns are sized to their content, so STATE fits
// a real phase and HASH/60s does not reserve four columns nothing uses. The
// rules line up across a row with a bar and a row without, because widths are
// display columns. And a node with no CPU metrics gets the unavailable mark in
// NODE CPU FREE rather than an empty trough, which would claim 0% free.
const moneroBodyGolden = "[external] pool.supportxmr.com:443 - SupportXMR - shared - 116.202.180.221  ! miners disagree on the pool\n" +
	"public address, no cluster object matches\n" +
	"\n" +
	"cluster  2/2 mining · 617 H/s · 42 shares✓ · miner 5000m · node free 12%\n" +
	"\n" +
	"NODE      │ STATE   │ HASH/60s │ THR │ SHARES │ MINER │ NODE CPU FREE │ POOL\n" +
	"──────────┼─────────┼──────────┼─────┼────────┼───────┼───────────────┼─────────────────────────\n" +
	"andromeda │ Running │  377 H/s │ 6/8 │ 42✓/2✗ │ 5000m │ █░░░░░░░ 12%  │ pool.supportxmr.com:443\n" +
	"orion     │ DONATE⚠ │  240 H/s │ 4/6 │  0✓/0✗ │     — │ —             │ donate.v2.xmrig.com:3333\n" +
	"\n" +
	"pool  662 H/s reported · due 0.003120 XMR · paid 0.184000 XMR · last share —"

func TestMoneroTabIsPinned(t *testing.T) {
	snap := sampleSnapshot()
	// A Bitcoin view must not leak into the Monero body either.
	snap.Bitcoin = &model.BitcoinView{Scope: "all namespaces", Pools: []model.BitcoinPool{addressPool()}}
	m := tuiModel{snap: snap, updated: time.Now(), tab: tabMonero}

	got := stripANSI(m.body())
	if got != moneroBodyGolden {
		t.Errorf("the Monero tab moved (005-FR-008, 006-SC-006)\n--- want\n%s\n--- got\n%s", moneroBodyGolden, got)
	}
}

// P4/P13: every rule in a table sits at the same display column on every line
// of it, whether or not the cells before it contain multi-byte glyphs. This is
// what the width helpers exist for — a sparkline is 18 bytes and 6 columns
// wide, and measuring the wrong one ragged the columns after it on any row that
// had CPU history.
func TestNodeTableColumnsAlign(t *testing.T) {
	assertRulesAlign(t, nodeTable(sampleSnapshot().Nodes))
}

// assertRulesAlign checks a rendered table's grid: every line carries its rules
// at identical display columns, and the head rule crosses each of them.
func assertRulesAlign(t *testing.T, rendered string) {
	t.Helper()
	lines := strings.Split(strings.TrimRight(stripANSI(rendered), "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("a table is at least a head, a rule and a row, got %d line(s):\n%s", len(lines), rendered)
	}

	want := rulePositions(lines[0], colSep)
	if len(want) == 0 {
		t.Fatalf("no column rules in the head row: %q", lines[0])
	}
	if got := rulePositions(lines[1], ruleCross); !slices.Equal(got, want) {
		t.Errorf("the head rule crosses at %v, but the columns divide at %v\n%s", got, want, rendered)
	}
	for i, l := range lines[2:] {
		if got := rulePositions(l, colSep); !slices.Equal(got, want) {
			t.Errorf("row %d rules at %v, want %v\n%s", i, got, want, rendered)
		}
	}
}

// rulePositions is where mark falls in a line, measured in display columns so a
// multi-byte cell before it does not shift the answer.
func rulePositions(line, mark string) []int {
	var at []int
	for i, r := range []rune(line) {
		if string(r) == mark {
			at = append(at, lipgloss.Width(string([]rune(line)[:i])))
		}
	}
	return at
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

	// P7: one unavailable mark — see TestNoSecondSpellingOfUnknown, which now
	// holds every mode to it rather than this tab alone.

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

	// P13: both tabs' tables are the same grid, drawn by the same renderer.
	for _, p := range view.Pools {
		assertRulesAlign(t, minerTable(&p))
	}
}

// P7: one mark for unavailable, in every mode. This is the rule 004 had to
// exempt the Monero tab from, because FR-001 froze the tab and the tab said
// "n/a"; 005 lifted the freeze, so the exemption goes with it and the check
// widens from one tab to everything a reader sees.
//
// `json` is deliberately absent: it is a data contract, encodes unknown as a
// negative number, and P12 exempts it.
func TestNoSecondSpellingOfUnknown(t *testing.T) {
	snap := sampleSnapshot()
	// A node with no CPU data at all: the case that used to print "n/a".
	snap.Nodes = append(snap.Nodes, model.NodeStatus{Node: "draco", Phase: "CrashLoopBackOff"})
	snap.Bitcoin = &model.BitcoinView{
		Scope: "all namespaces",
		Pools: []model.BitcoinPool{devicePool(), addressPool()},
	}
	snap.Summarize()

	var stream strings.Builder
	Stream(&stream, snap)

	for name, out := range map[string]string{
		"monero tab": tuiModel{snap: snap, updated: time.Now(), tab: tabMonero}.View(),
		"btc tab":    tuiModel{snap: snap, updated: time.Now(), tab: tabBitcoin}.View(),
		"stream":     stream.String(),
	} {
		if strings.Contains(stripANSI(out), "n/a") {
			t.Errorf("P7: %s spells unavailable as n/a; the one mark is %q\n---\n%s",
				name, unavailable, out)
		}
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
