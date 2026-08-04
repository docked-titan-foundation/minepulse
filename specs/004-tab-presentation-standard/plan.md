# Implementation Plan: Tab presentation standard

**Branch**: `004-tab-presentation-standard` | **Date**: 2026-08-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/004-tab-presentation-standard/spec.md`

## Summary

Write down the grammar the Monero tab already speaks, then make the Bitcoin tab speak it:
flush-left panels, a bold role label followed by ` · `-separated metrics, dimmed identity,
upper-case column heads that name their averaging window, one share format, one unavailable
mark, and warnings that look like warnings. The Monero renderer is not touched — a test
pins its output byte for byte — and the shared formatting helpers become the only place
these strings are built, so the next panel inherits the standard instead of inventing one.

## Technical Context

**Language/Version**: Go 1.23 (pinned by `mise`; CI lints with golangci-lint 1.61)

**Primary Dependencies**: existing only — `lipgloss` for styling, `text/tabwriter` for the
stream tables. No new modules.

**Storage**: N/A

**Testing**: `go test ./...`; the Monero tab pinned by a byte-equality test, the Bitcoin tab
by content assertions, both driven from the existing fixtures in `internal/render`

**Target Platform**: terminals of any width, with and without colour

**Project Type**: single Go CLI — this feature touches only `internal/render`

**Performance Goals**: unchanged; rendering is trivial next to collection

**Constraints**: presentation-only — no model change, no collector change, no `json` change;
single-width glyphs only, and colour never the sole carrier of meaning

**Scale/Scope**: 3 files changed, 1 new test file, 1 helper added

## The standard

This feature ratified the rule set FR-002 requires. It has since been extracted to
[`.specify/memory/presentation-standard.md`](../../.specify/memory/presentation-standard.md)
and amended by [005-table-grid](../005-table-grid/plan.md) — a living standard cannot keep
living inside a finished feature's plan, where a later reader finds it half-true.

**Check a new panel against that document, not this section.** What it held when this feature
shipped (standard v1.0.0): P1–P12, with P7 carrying one exception — the Monero tab kept `n/a`
in its CPU columns, because FR-001 froze it. Both the freeze and the exception are gone; the
amendment history in the standard records when and why.

### Applying it to the Bitcoin tab

What changes, concretely:

```text
before                                     after
────────────────────────────────────────────────────────────────────────────────
ckpool  on orion · ns/pod · Running · logs ckpool  480.00 TH/s (1m) · 2 workers (1 idle) · 12483✓/3✗ · best 1.20 G
  480.00 TH/s (1m) · 1 users · …           bitcoin-solo/mining-pool-xyz on orion · Running 5d22h · via logs
  0.02% of network diff                    0.02% of network diff · 1.4 shares/s
  ADDRESS      HASHRATE  WORKERS  SHARES   ADDRESS         HASH/1m  WORKERS     SHARES     BEST  LAST SHARE
  bc1q…xxxx  480.00 TH/s      2   12483    bc1qexam…xxxx  480.00 TH/s      2  12483✓/3✗  1.20 G      1m ago
  ! per-device detail unavailable          ! per-device detail unavailable from ckpool logs (addresses only)
```

- P1: the two-space indent on every sub-line and table row goes away.
- P2/P3: metrics move up to the header line beside the implementation name; identity drops
  to a dimmed context line.
- P5: `HASHRATE` becomes `HASH/1m` (ckpool) or `HASH` (public-pool's instantaneous figure).
- P6: the table's bare share count becomes `12483✓/3✗`; public-pool, which reports no
  rejects, shows the accepted half alone rather than inventing `0✗`.
- P8: notes move from dim to the warning colour; remedies stay dim with `→`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Gate | Pre-design | Post-design |
|---|---|---|---|
| I. Spec-Driven & Constitution-Bound | Spec approved before code | ✅ spec.md + checklist written first, after the operator settled scope and doc home | ✅ standard recorded here; tasks derive from it |
| II. Observe, Never Mutate | No new cluster access | ✅ render-only | ✅ unchanged |
| III. Degrade Gracefully | Missing data still renders | ✅ | ✅ P6/P7 strengthen it: a missing figure gets the unavailable mark, never a fabricated zero |
| IV. Test-First for Pure Logic | Parsing/aggregation fixture-tested | ✅ n/a — no parsing changes; the TUI is exempt from strict TDD but must be exercisable via `--mock` | ✅ the new `Shares` helper is pure and table-tested; the Monero tab gains a byte-equality guard |
| V. CLI Contract: Text/JSON In-Out | Output modes stable | ⚠️ the `stream` Bitcoin block changes shape | ✅ acceptable — that block shipped in the same release train and has no consumers yet; `json`, the machine contract, is untouched. Recorded in Complexity Tracking |
| VI. Single Signed Artifact | No new deps | ✅ | ✅ zero |
| VII. Privacy & Least Data | Addresses handled carefully | ✅ | ✅ the truncation rule (P10) is now written down rather than incidental |

**Gate result**: PASS, with one recorded deviation (stream shape).

## Project Structure

### Documentation (this feature)

```text
specs/004-tab-presentation-standard/
├── plan.md              # This file — ratified the standard, now extracted to
│                        #   .specify/memory/presentation-standard.md
├── spec.md              # Requirements
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # /speckit-tasks output
```

No `research.md`, `data-model.md` or `contracts/`: nothing external is being learned, no
model type changes, and the observable contract is the standard plus the untouched `json`
schema.

### Source Code (repository root)

```text
internal/render/
├── bitcoin.go        # MODIFIED — panel conforms to P1-P10
├── format.go         # MODIFIED — Shares() helper: the standard's shared vocabulary
├── stream.go         # MODIFIED — Bitcoin block follows P4-P7, P10
├── bitcoin_test.go   # MODIFIED — assertions become standard-shaped
├── stream_test.go    # MODIFIED — share format in the stream table
└── standard_test.go  # NEW — pins the Monero tab byte-for-byte (FR-001) and checks the
                      #       cross-tab invariants a reviewer would otherwise check by eye
```

**Structure Decision**: Everything stays inside `internal/render`. Enforcement lives in
`format.go` (shared helpers) rather than in either panel, which is what makes P11 true
rather than aspirational.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Constitution V calls the output modes stable, and this reshapes the `stream` Bitcoin block (column heads, share format) | The block is days old, shipped in the same release train as the feature itself, and inconsistency is cheaper to fix now than after it has consumers | Freezing the block as-is (rejected — it would contradict P12 permanently and leave two share formats in one output); versioning the stream format (rejected — out of proportion for a human-readable mode whose `json` sibling is the actual contract) |
| The standard records an inconsistency (`n/a` vs `—`) instead of fixing it | FR-001: the operator ranked "the Monero tab does not move" above uniformity | Normalising both tabs (rejected — explicitly declined); leaving it unmentioned (rejected — an undocumented exception is how the drift started) |

**Resolved by [005](../005-table-grid/plan.md)**: both deviations above were consequences of
the FR-001 freeze. 005 lifted it, normalised the mark and removed the exception, so neither
survives into the current standard.
