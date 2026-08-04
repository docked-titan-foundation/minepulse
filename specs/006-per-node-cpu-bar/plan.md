# Implementation Plan: Per-node CPU-free bar

**Branch**: `006-per-node-cpu-bar` | **Date**: 2026-08-03 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/006-per-node-cpu-bar/spec.md`

## Summary

Move the free-CPU gauge off the cluster header and onto every node row, drop the sparkline
column it now stands in for, and fold the numeric `FREE` column into the bar's cell so the
table ends up one column narrower rather than one wider. The cluster percentage the gauge
carried moves to the header line as a metric, where `stream` already reports it.

## Technical Context

**Language/Version**: Go 1.26 (pinned by `mise`)

**Primary Dependencies**: existing only — `lipgloss`. No new modules.

**Testing**: `go test ./...`; the Monero tab re-pinned by byte equality, the bar's thresholds
and its unavailable case tested directly, `stream` pinned unchanged.

**Constraints**: presentation-only — no model change, no collector change, no `stream` or
`json` change; single-width glyphs only, colour never the sole carrier of meaning

**Scale/Scope**: 2 files modified, 2 test files updated

## Standard impact

None. The bar is a cell value like any other, so [the standard](../../.specify/memory/presentation-standard.md)
covers it as written: P7 gives it its unavailable mark, P11 requires it come from a shared
helper rather than being assembled inline, P12/P13 keep it out of `stream`, and its
percentage beside it is what keeps colour from being the sole carrier (a constitutional
constraint, not a presentation one). No rule is added or amended, and the version stays at
1.1.0.

## What changes on screen

```text
before                                          after
──────────────────────────────────────────────────────────────────────────────────────
cluster  3/4 mining · 840 H/s · 2 shares✓ · …   cluster  3/4 mining · 840 H/s · 2 shares✓ · … · node free 10%

node CPU free ██░░░░░░░░░░░░░░░░░░░░░░ 10%      NODE      │ … │ MINER │ NODE CPU FREE │ POOL
                                                ──────────┼───┼───────┼───────────────┼──────
NODE      │ … │ MINER │ FREE │ CPU-FREE ~2m │   andromeda │ … │ 5313m │ █░░░░░░░ 10%  │ pool…
andromeda │ … │ 5313m │  10% │ ▁            │   draco     │ … │     — │ —             │ —
```

Two columns become one, and the gauge line and the blank line under it both go — so the
panel is three lines shorter and the table is narrower.

## Constitution Check

*GATE: Must pass before implementation.*

| Principle | Gate | Result |
|---|---|---|
| I. Spec-Driven & Constitution-Bound | Spec approved before code | ✅ spec.md written before the renderer changed |
| II. Observe, Never Mutate | No new cluster access | ✅ render-only |
| III. Degrade Gracefully | Missing data still renders | ✅ FR-006: no metrics is the unavailable mark, never a zero-length bar reading as "0% free" |
| IV. Test-First for Pure Logic | Pure logic fixture-tested | ✅ the bar is a pure function of one percentage and tested as one |
| V. CLI Contract: Text/JSON In-Out | Output modes stable | ✅ `stream` and `json` untouched and pinned |
| VI. Single Signed Artifact | No new deps | ✅ zero |
| VII. Privacy & Least Data | Addresses handled carefully | ✅ unchanged |

**Gate result**: PASS, no deviations.

## Project Structure

```text
internal/render/
├── tui.go            # MODIFIED — gauge becomes cpuBar, called per row; two columns fold into one
├── standard_test.go  # MODIFIED — golden re-pinned
└── render_test.go    # MODIFIED — gauge test becomes the bar's

README.md             # MODIFIED — the feature list still promised a sparkline
```

**Structure Decision**: `gauge` is not deleted but narrowed into `cpuBar` — same thresholds,
same colours, no label (the column head is the label now). One helper, one set of thresholds,
which is what keeps the colours meaning the same thing they meant before the move.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| `Sparkline()` and `model.Ring` survive with nothing rendering them | FR-010 keeps "stop collecting CPU history" a separate, reversible decision; this spec changes a layout | Deleting both now (rejected — it turns a column swap into a collector change, and the operator may want the trace back in another column); leaving the column in place (rejected — it is what was asked to go) |
