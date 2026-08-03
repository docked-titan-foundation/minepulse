<!--
Sync Impact Report
- Version change: (none) → 1.1.0
- Bump rationale: extracted from specs/004's plan (v1.0.0, the rule set as shipped) and
  amended by specs/005 — P4 extended, P13 added, P7's exception removed. MINOR: rules were
  added and relaxed, none reversed.
- Amended rules: P4 (columns size themselves), P7 (one unavailable mark, no exceptions)
- Added rules: P13 (tables are ruled grids)
- Templates / docs checked:
  - .specify/memory/constitution.md — ✅ no contradiction; this document is subordinate to it
  - specs/004-tab-presentation-standard/plan.md — ✅ inline copy replaced by a pointer here
  - specs/005-table-grid/plan.md — ✅ inline copy replaced by a pointer here
- Follow-up TODOs: none
-->

# minepulse Presentation Standard

How every panel in the TUI presents data. It exists so an operator who has watched one tab
for weeks can read a new one immediately: the same line does the same job, the same column
head means the same thing, the same glyph means the same thing.

This is a **living document**, not a feature record. A spec that changes how the dashboard
presents anything amends this file — with a Sync Impact Report and a version bump — rather
than restating the rules in its own plan. The constitution
([`constitution.md`](./constitution.md)) outranks it.

**Version**: 1.1.0 | **Ratified**: 2026-08-01 (specs/004) | **Last amended**: 2026-08-03
(specs/005)

## The rules

Each is written to be answerable yes/no when reviewing a panel.

| # | Rule |
|---|---|
| P1 | Panel content is flush left. The box's padding is the only indent; sub-lines never indent relative to their panel header, and a table's first column starts at the same display column as the header above it. |
| P2 | A panel opens with `<bold role label>  <metric> · <metric> · …`, most important metric first, each figure labelled as the Monero header labels its own (`42✓/2✗ shares`, `miner 5000m`). A figure the source does not report is **omitted** from the list, not rendered as `—`: a bare dash says nothing in prose, while in a table cell it holds the column open. The headline hashrate is the exception — it is the panel's primary metric and appears even when unavailable. When a panel has no metrics at all, the label stands alone. |
| P3 | Identity and provenance (namespace/pod, node, state, uptime, stats source) go on a dimmed context line under the header, never mixed into the metrics. The provenance segment turns the warning colour when the numbers are stale or absent, and that fact is stated **once** — the header never repeats it. |
| P4 | Column heads are upper-case. Identity columns left-aligned, numeric columns right-aligned. Columns size themselves to the wider of head and widest cell; a panel never states a column width. |
| P5 | An averaged hashrate column names its window: `HASH/60s`, `HASH/1m`. An instantaneous figure is `HASH` — it never borrows a window. |
| P6 | Shares are `N✓/M✗` everywhere — headline and table. When the source does not report rejects, `N✓` alone; never `N✓/0✗`. |
| P7 | Unavailable is `—`, everywhere, in every output mode. One mark, no exceptions. |
| P8 | A problem is `! text` in the warning colour; a remedy is `→ text` dimmed. Neither looks like a metric line. |
| P9 | Panels are separated by one blank line. |
| P10 | Over-long identities truncate with `…`, at a bound the panel states per column so one value cannot widen a table without limit. |
| P11 | Hashrates, shares, difficulties, durations, ages and the unavailable mark come from `internal/render/format.go`. Building them inline in a panel is the defect. |
| P12 | `stream` follows P4–P7 and P10 in plain text; it has no rules (P13) because it is meant for pipes and greps. `json` is a data contract and follows none of this. |
| P13 | TUI tables are grids, drawn by `internal/render/table.go`: a dimmed `│` between adjacent columns, a dimmed `─`/`┼` rule under the heads, one space of gutter on each side of a rule. A cell with nothing to show carries the unavailable mark or a dimmed `·` placeholder as wide as the values beside it — never blank space inside the grid. |

## Amendment history

| Version | Date | Spec | Change |
|---|---|---|---|
| 1.0.0 | 2026-08-01 | [004-tab-presentation-standard](../../specs/004-tab-presentation-standard/spec.md) | Ratified P1–P12, written from the Monero tab's existing grammar and applied to the Bitcoin tab. P7 carried an exception: the Monero tab kept `n/a` in its CPU columns, because 004-FR-001 froze that tab byte-for-byte. |
| 1.1.0 | 2026-08-03 | [005-table-grid](../../specs/005-table-grid/spec.md) | P13 added (ruled grids). P4 extended: columns size themselves to content. P1 extended to cover a table's first column. P7's exception removed — 005 lifted the freeze that was its only justification, so `n/a` became `—` in the Monero tab and in `stream`. |

## Terms

- **Panel**: one framed section of a tab — header line, optional context line, optional gauge, optional table, optional notes.
- **Headline metric**: one ` · `-separated figure on a header line.
- **Column head**: the upper-case name of a table column, carrying the averaging window when there is one.
- **Unavailable mark**: `—`, the glyph standing for "this source does not report it".
- **Rule**: the dimmed line separating a table's columns from each other and its heads from its rows.
