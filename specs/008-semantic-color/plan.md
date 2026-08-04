# Implementation Plan: Semantic color

**Branch**: `008-semantic-color` | **Date**: 2026-08-04 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/008-semantic-color/spec.md`

## Summary

Give the identity line's locality marker a color that means something — green inside the
cluster, blue outside, dim when unproven — and rank the rest of the line so the address leads
and the supporting detail recedes. Then apply color elsewhere only where a state is being
reported, and write down in the standard what color is allowed to mean, so the next panel
inherits the rule rather than inventing one.

## Technical Context

**Language/Version**: Go 1.26 (pinned by `mise`)

**Primary Dependencies**: existing only — `lipgloss`. No new modules.

**Testing**: `go test ./...`; the strip-to-plain invariant and the distinctness of the three
locality colors are both tests rather than assumptions, and both tabs' goldens are unchanged.

**Constraints**: presentation-only — no model change, no collector change, no `stream` or
`json` change; color never the sole carrier of meaning (Constitution).

**Scale/Scope**: 4 files modified, 1 test file extended, standard amended

## Standard impact

Takes [the standard](../../.specify/memory/presentation-standard.md) to **1.2.0** by adding
**P14**: color encodes state, never decoration or category; green healthy, amber degraded, red
broken, blue neutral-but-notable, dim supporting or unproven; a calm state stays uncolored; and
whatever color says, the text must say too. No existing rule is amended.

P14 exists because this feature would otherwise be a set of one-off choices. The constitution
already forbade relying on color alone; what was missing was a statement of what the colors
*mean*, which is the part a reviewer needs to check a new panel against.

## Key decisions

**External is blue, not amber.** Mining to a pool outside the cluster is the ordinary case for
Monero. Giving it the warning color would fire on every correctly configured install, and a
warning that is always on is not a warning. Blue says "notable, not wrong". A test pins this
so a future tidy-up cannot quietly reassign it.

**A healthy value stays uncolored.** Coloring every good state green leaves nothing for the bad
one to stand out against. So `0✓/0✗` is plain and only a non-zero reject count turns red; only
the rejected half of the pair takes the color at all.

**`0/0` nodes is dim, not amber.** Nothing installed is not a failure, and the cluster fraction
is the one place that distinction is visible.

**The strip-to-plain invariant is a test, not a convention.** `PoolLine` stays plain for
`stream`; `poolLineStyled` is the TUI's. Keeping two renderings of one line honest by review
would have failed eventually, so a test asserts that for every locality the styled line minus
its escapes equals the plain one.

## Constitution Check

*GATE: Must pass before implementation.*

| Principle | Gate | Result |
|---|---|---|
| I. Spec-Driven & Constitution-Bound | Spec approved before code | ✅ spec.md written before the renderer changed; P14 recorded in the standard rather than in this plan |
| II. Observe, Never Mutate | No new cluster access | ✅ render-only |
| III. Degrade Gracefully | Missing data still renders | ✅ unchanged; an unproven locality is dim rather than absent |
| IV. Test-First for Pure Logic | Pure logic fixture-tested | ✅ `localityStyle` and `miningFraction` are pure and tested directly |
| V. CLI Contract: Text/JSON In-Out | Output modes stable | ✅ `stream` and `json` byte-identical; lipgloss already drops color for non-TTY, and FR-004 is what makes that lossless rather than lossy |
| VI. Single Signed Artifact | No new deps | ✅ zero |
| VII. Privacy & Least Data | Careful with identifiers | ✅ unchanged |

**Gate result**: PASS, no deviations.

## Project Structure

```text
internal/render/
├── format.go         # MODIFIED — poolLineStyled + localityStyle beside the plain PoolLine
├── tui.go            # MODIFIED — infoSt; miningFraction; rejected-share coloring
├── bitcoin.go        # MODIFIED — idle worker count takes the warning color
└── render_test.go    # MODIFIED — strip-to-plain, color distinctness, fraction states

.specify/memory/presentation-standard.md   # MODIFIED — P14, standard → 1.2.0
```

**Structure Decision**: The styled variant lives next to the plain one in `format.go` rather
than in the panels, so the two cannot drift apart unnoticed and the panels keep calling one
function each. The palette stays in `tui.go` with the other styles; `infoSt` is the only
addition, and no existing color changes meaning.

## Release note

Committed as `style(mp)` at the operator's direction. `.releaserc` releases only on
`feat`/`fix`/`perf` with scope `mp`, so **this change ships no release of its own** and reaches
users with whatever `feat` or `fix` lands next. That is the intended trade: the dashboard's
colors are not worth a version bump on their own.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Two renderings of the identity line (`PoolLine`, `poolLineStyled`) | `stream` must stay plain for pipes (P12) while the TUI carries color | Styling `PoolLine` itself and relying on lipgloss's TTY detection (rejected — it makes the plain output an accident of the environment rather than a property of the code, and nothing would catch it if that detection changed) |
