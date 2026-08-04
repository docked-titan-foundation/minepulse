# Feature Specification: Ruled table grid

**Feature Branch**: `005-table-grid`

**Created**: 2026-08-03

**Status**: Draft

**Input**: User description: "In the mine pulse project, i want better table organization and visual
feeling." Follow-up decision: full box-drawing grid — columns that size themselves to their
content, dim vertical rules between columns, a crossing rule under the column heads.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find the column you want without counting spaces (Priority: P1)

An operator scanning eight nodes wants the FREE column. Today the columns are held apart by
hardcoded widths, so the eye has to track a run of blank space from the head down to the row.
With a ruled grid the column is bounded on both sides and the value is found by following a
line rather than by estimating a distance.

**Why this priority**: This is the request. A table that is only alignment is legible; a table
that is structure is *scannable*, which is what a dashboard glanced at every few seconds needs.

**Acceptance Scenarios**:

1. **Given** any tab with a table, **When** it renders, **Then** every column is separated from the next by a vertical rule, and the heads are separated from the rows by a crossing horizontal rule.
2. **Given** a table, **When** the operator compares two rows, **Then** each rule sits at the same display column on both.
3. **Given** a styled cell (a coloured state, a sparkline), **When** the row renders, **Then** its rule still lands where the plain rows put it — widths are measured in display columns, never bytes.

---

### User Story 2 - Nothing is truncated to pay for padding (Priority: P1)

A node in `CrashLoopBackOff` reads as `CrashLoopBackOff`, not `CrashLoopBack…`, and the
`HASH/60s` column stops reserving four columns no row ever uses.

**Why this priority**: The hardcoded widths are simultaneously too small for the values that
matter most (a failing state) and too large for the ones that do not. Both are fixed by the
same change, and the truncation is the one that costs the operator information.

**Acceptance Scenarios**:

1. **Given** a table, **When** it renders, **Then** each column is exactly as wide as its widest cell or its head, whichever is larger.
2. **Given** a pathologically long identity (a fully-qualified node name, a payout address), **When** it renders, **Then** it truncates with `…` at a per-column bound so one value cannot widen the table without limit.

---

### User Story 3 - An empty cell reads as "nothing recorded", not as a broken table (Priority: P2)

A node with no CPU history yet leaves the `CPU-FREE ~2m` cell blank today. Inside a ruled
grid a blank cell looks like a rendering fault.

**Why this priority**: Only visible in the first seconds after start or when metrics-server is
absent — but that is exactly when the operator is deciding whether the tool works.

**Acceptance Scenarios**:

1. **Given** a node with no sparkline, **When** the table renders, **Then** the cell carries a dimmed dotted placeholder as wide as the traces beside it.

### Edge Cases

- Terminal narrower than the table → unchanged behaviour: the box stops framing, rows do not wrap.
- A table with no rows → omitted entirely, heads and rule included (unchanged from today).
- A cell containing multi-byte glyphs (`✓`, `⚠`, block-drawing sparklines) → measured in columns.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every table in the TUI MUST render as a grid: one dimmed vertical rule between adjacent columns, one dimmed horizontal rule between the head row and the first data row, crossing the verticals.
- **FR-002**: Column widths MUST be derived from content — the widest of the head and the column's cells — not from per-table constants.
- **FR-003**: All width arithmetic MUST be in display columns, so styled and multi-byte cells align with plain ones.
- **FR-004**: Identity columns MUST stay left-aligned and numeric columns right-aligned; heads MUST stay upper-case (standard P4, unchanged).
- **FR-005**: Identity values MUST truncate with `…` at a bound stated per column, so a single long value cannot widen the table without limit (standard P10, unchanged).
- **FR-006**: A cell with no value to show MUST render the column's unavailable mark or a dimmed placeholder, never empty space inside the grid.
- **FR-007**: Both tabs MUST use one shared table renderer; a panel MUST NOT lay out columns itself (the P11 rule, extended from value formatting to table structure).
- **FR-008**: This feature supersedes 004-FR-001 ("the Monero tab MUST NOT change") — the operator has asked for the reference tab to move. The byte-equality guard is replaced by a new golden, not deleted: the tab is still pinned, to its new shape.
- **FR-009**: `json` MUST NOT change, and `stream` MUST keep its structure, columns and every figure it reports. The grid is a TUI affordance; `stream` stays plain text for pipes (standard P12). Its unavailable mark is the one exception, per FR-011.
- **FR-010**: No data may change: same columns, same values, same order. This feature is presentation only.
- **FR-011**: Unavailable MUST render as one mark in every output mode. The Monero tab's `n/a` — kept only because 004-FR-001 froze that tab, and superseded here with it — becomes `—`, in the `MINER` and `FREE` columns, in the free-CPU gauge, and in `stream`. Standard P7 then admits no exceptions.
- **FR-012**: The standard MUST live in one document outside any feature's plan, so a rule cannot be true in one spec and amended in another. Both 004 and 005 reference it; neither restates it.

### Key Entities

- **Table**: an ordered set of columns plus rows of already-formatted cells.
- **Column**: a head, an alignment, and a width computed at render time.
- **Rule**: the dimmed line separating columns from each other and heads from rows.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In both tabs, every table's vertical rules sit at identical display columns on every line of that table, proven by a test over styled rows.
- **SC-002**: No table column is wider than its widest cell, and no cell is truncated while its column has spare width.
- **SC-003**: The Monero and Bitcoin tables are built by the same renderer, called with columns and cells; neither panel contains a column-width constant.
- **SC-004**: `stream` keeps every column head, row and figure it had; the unavailable mark is the only difference, and it moves toward P7 rather than away.
- **SC-005**: The Monero tab is pinned to a new golden, so the next unintended reflow still fails a test rather than reaching a terminal.
- **SC-006**: `n/a` appears nowhere in rendered output, in any mode, and a test asserts it for both tabs rather than just the Bitcoin one.
- **SC-007**: A reviewer checking a panel opens exactly one document, and every rule in it is currently true.

## Assumptions

- Box-drawing characters (`│ ─ ┼`) and the middle dot are safe: the tab strip, gauge and
  sparklines already rely on the same class of glyph, and all are single-width.
- Colour still carries no meaning alone — the rules are structure, and the dim style they use
  degrades to plain text on a monochrome terminal without losing the layout.
- The reference-tab freeze from 004 was a constraint of that feature, not a permanent
  property of the Monero tab; it is superseded here explicitly rather than quietly broken.
- Anything the freeze was propping up falls with it. The `n/a`/`—` split is the whole of that
  list: it was the only exception 004 recorded, and 004's own US1 already called a second
  spelling of "unknown" the defect it was there to remove.
- Row striping, per-column colour and a Bitcoin sparkline were considered and remain out of
  scope: this feature changes table *structure*, not the palette.
