# Implementation Plan: Ruled table grid

**Branch**: `005-table-grid` | **Date**: 2026-08-03 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/005-table-grid/spec.md`

## Summary

Replace the hand-rolled `%-12s %11s …` format strings in both tabs with one table renderer
that sizes each column to its content and draws dim box rules between columns and under the
heads. The Monero tab moves for the first time since it became the reference — deliberately,
at the operator's request — and its byte-equality guard is re-pinned to the new shape rather
than dropped. Lifting that freeze also removes the only justification the standard's one
exception ever had, so `n/a` becomes `—` everywhere. `json` is untouched.

## Technical Context

**Language/Version**: Go 1.26 (pinned by `mise`)

**Primary Dependencies**: existing only — `lipgloss` for styling and display-width measurement.
No new modules.

**Testing**: `go test ./...`; the Monero tab re-pinned by byte equality, rule alignment
asserted structurally over styled rows, `stream` pinned to its columns and figures.

**Target Platform**: terminals of any width, with and without colour

**Constraints**: presentation-only — no model change, no collector change, no `json` change,
and no change to which figures any mode reports; single-width glyphs only, colour never the
sole carrier of meaning

**Scale/Scope**: 2 new files, 5 modified, 3 test files updated

## The standard, amended

The rules live in
[`.specify/memory/presentation-standard.md`](../../.specify/memory/presentation-standard.md),
extracted by this feature from 004's plan — a living standard inside a finished feature's
plan is half-true the moment the next feature touches it, which is exactly what happened
here. This feature takes it to **v1.1.0**:

- **P13 added** — tables are ruled grids, drawn by one renderer.
- **P4 extended** — columns size themselves to content; a panel never states a width.
- **P1 extended** — a table's first column starts flush left, like every other line in a panel.
- **P7's exception removed** — see below.

### Superseding 004-FR-001, and what falls with it

004 froze the Monero tab byte-for-byte, and `TestMoneroTabIsUnchanged` enforced it. The
operator has now asked for the tables to change, which necessarily moves the reference tab —
it holds the primary table. The freeze is therefore **superseded, not violated**: the golden
in `standard_test.go` is regenerated against the grid, so the tab remains pinned and the next
*unintended* reflow still fails.

The freeze was also the sole justification for P7's exception — the Monero tab spelling
unavailable as `n/a` while every other panel spelled it `—`. With FR-001 gone the exception
has no basis, and the grid makes the cost visible: a single row read

```text
draco │ CrashLoopBackOff │  — │  — │  — │ n/a │ n/a │ · │ —
```

three dashes, two `n/a`, one dash. So this feature normalises the mark (FR-011). P7 now
admits no exceptions, and 004's Complexity Tracking entry for it is resolved rather than
inherited.

### What changes on screen

```text
before                                            after
──────────────────────────────────────────────────────────────────────────────────────────
NODE         STATE             HASH/60s    THR     NODE      │ STATE            │ HASH/60s │ THR
andromeda    Running            377 H/s    5/8    ───────────┼──────────────────┼──────────┼────
draco        CrashLoopBack…           —      —     andromeda │ Running          │  377 H/s │ 5/8
                                                   draco     │ CrashLoopBackOff │        — │   —
```

- `STATE` stops truncating a phase to make room for padding a hashrate.
- `HASH/60s` gives back the four columns no row used.
- The `CPU-FREE ~2m` cell of a node with no history becomes `······`, not a hole in the grid.
- `n/a` becomes `—` in the `MINER` and `FREE` columns, in the gauge, and in `stream`.

## Constitution Check

*GATE: Must pass before implementation.*

| Principle | Gate | Result |
|---|---|---|
| I. Spec-Driven & Constitution-Bound | Spec approved before code | ✅ spec.md written and agreed before the renderer changed; the 004 supersession is recorded above rather than left implicit |
| II. Observe, Never Mutate | No new cluster access | ✅ render-only |
| III. Degrade Gracefully | Missing data still renders | ✅ strengthened: an empty cell now says "nothing recorded" instead of looking broken |
| IV. Test-First for Pure Logic | Pure logic fixture-tested | ✅ the table renderer is pure and tested directly; the TUI stays exercisable via `--mock` |
| V. CLI Contract: Text/JSON In-Out | Output modes stable | ⚠️ `stream`'s unavailable mark changes with P7 (FR-011); its structure, columns and every reported figure are untouched, and `json` — the machine contract — is byte-identical. Recorded in Complexity Tracking |
| VI. Single Signed Artifact | No new deps | ✅ zero |
| VII. Privacy & Least Data | Addresses handled carefully | ✅ unchanged — `ShortAddress` still truncates payout addresses |

**Gate result**: PASS, with one recorded deviation (the `stream` unavailable mark).

## Project Structure

```text
internal/render/
├── table.go          # NEW — the grid: columns, alignment, widths, rules
├── tui.go            # MODIFIED — node table built from columns; width helpers consolidated
├── bitcoin.go        # MODIFIED — both miner tables built from columns
├── format.go         # MODIFIED — Pct() returns the standard's mark (P7)
├── stream.go         # MODIFIED — same, for the plain-text mode (P12)
├── standard_test.go  # MODIFIED — golden re-pinned; rule alignment asserted structurally
└── table_test.go     # NEW — the renderer's own tests (sizing, alignment, styled cells)

.specify/memory/
└── presentation-standard.md  # NEW — the standard, extracted from 004's plan and amended
```

**Structure Decision**: The renderer lives beside `format.go`, which already owns the
standard's value vocabulary. `format.go` says how a *value* is written; `table.go` says how a
*table* is built. Together they are what P11 and P13 mean by "the panel inherits the standard".

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| 004-FR-001 (the Monero tab does not move) is superseded | The operator asked for better table organisation; the Monero tab holds the primary table, so the request cannot be honoured without moving it | Applying the grid to the Bitcoin tab only (rejected — it recreates exactly the two-dialect problem 004 existed to remove, in the opposite direction) |
| Constitution V calls the output modes stable, and this changes `stream`'s unavailable mark | P7 is what makes the mark one mark; leaving `stream` on `n/a` would have re-created the exception in the output mode P12 explicitly binds to P7 | Normalising the TUI only (rejected — two spellings again, one tab deep); leaving both on `n/a` (rejected — it is the Bitcoin tab and every other panel that would then be the exception) |
