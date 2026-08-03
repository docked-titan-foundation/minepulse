# Feature Specification: Per-node CPU-free bar

**Feature Branch**: `006-per-node-cpu-bar`

**Created**: 2026-08-03

**Status**: Draft

**Input**: User description: "from monero tab remove node CPU free progress bar but add that
progress bar by node in the table, also remove the CPU-free 2m column"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See which node has headroom, not just the cluster (Priority: P1)

The gauge under the cluster header answers "is there spare CPU somewhere", which is the
question the operator has already answered by the time they are looking at the table. The
question the table exists for is *which node* is tight — and that one needed reading a
percentage out of a numeric column and holding four of them in mind at once. A bar on each
row answers it by shape: the short bars are the busy nodes.

**Why this priority**: This is the request, and it moves the gauge to where the comparison
actually happens.

**Acceptance Scenarios**:

1. **Given** the Monero tab, **When** it renders, **Then** there is no gauge line between the cluster header and the node table.
2. **Given** a node with CPU metrics, **When** its row renders, **Then** the node's free CPU appears as a bar with its percentage beside it, in the same colours the cluster gauge used (green with headroom, amber when tight, red when starved).
3. **Given** four nodes at different loads, **When** the operator scans the column, **Then** the busiest node is identifiable from bar length alone, without reading a number.

---

### User Story 2 - The table stops carrying a column nobody reads (Priority: P1)

`CPU-FREE ~2m` held a sparkline of each node's recent free-CPU history. It goes.

**Why this priority**: Same change, opposite direction — the row gains a bar and loses a
trace, so the table does not get wider.

**Acceptance Scenarios**:

1. **Given** the Monero tab, **When** the table renders, **Then** no `CPU-FREE ~2m` column is present.
2. **Given** the same table, **When** it renders, **Then** it is not wider than before this change.

---

### User Story 3 - The cluster-wide figure survives its gauge (Priority: P2)

Removing the gauge must not remove the fact it carried. Cluster free CPU joins the header
line as a headline metric, where `stream` has always reported it.

**Why this priority**: The operator asked to remove a *bar*, not a *number*. Silently
dropping the aggregate would be a scope error, and it would leave the TUI reporting less than
`stream` does from the same snapshot.

**Acceptance Scenarios**:

1. **Given** the Monero tab, **When** the cluster header renders, **Then** it carries the cluster's free-CPU percentage as a ` · `-separated metric.
2. **Given** metrics-server is absent, **When** the header renders, **Then** the figure is the unavailable mark, not a fabricated zero.

### Edge Cases

- A node with no CPU metrics → the unavailable mark in the cell, not an empty or zero-length bar, which would read as "0% free".
- A node at 100% free (idle) or 0% free (saturated) → a full or empty trough, never a bar that overflows its cell.
- Terminal narrower than the table → unchanged behaviour: the box stops framing, rows do not wrap.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The cluster free-CPU gauge MUST be removed from the Monero tab body.
- **FR-002**: Each node row MUST show that node's free CPU as a bar plus its percentage, in one column.
- **FR-003**: The bar MUST use the cluster gauge's existing thresholds and colours, so the colour means what it has always meant; colour MUST NOT be the sole carrier of the reading — the percentage stays beside it.
- **FR-004**: The `CPU-FREE ~2m` sparkline column MUST be removed.
- **FR-005**: The separate numeric `FREE` column MUST be folded into the bar's column rather than duplicated beside it — the bar already carries its percentage.
- **FR-006**: A node without CPU metrics MUST render the unavailable mark in that column (standard P7), never a zero-length bar.
- **FR-007**: The cluster's free-CPU percentage MUST appear as a headline metric on the `cluster` line (standard P2).
- **FR-008**: The bar MUST be built by a shared helper, not assembled in the row builder (standard P11).
- **FR-009**: `stream` and `json` MUST NOT change. The bar is a TUI affordance, as the grid is (standard P12/P13).
- **FR-010**: No collection may change: the CPU history ring stays as it is. This feature removes a *view* of it, and the decision to stop gathering it is a separate one.

### Key Entities

- **CPU-free bar**: a fixed-width trough, filled in proportion to a 0–100 percentage, followed by that percentage.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The Monero tab body contains no gauge line, and no `CPU-FREE ~2m` column.
- **SC-002**: The node table is no wider than before this change, at the same fixture.
- **SC-003**: Cluster free CPU is still reported on the Monero tab, and matches what `stream` reports from the same snapshot.
- **SC-004**: A node with no CPU metrics renders `—`, proven by a test, in the column where the bar would be.
- **SC-005**: `stream` and `json` output are byte-identical before and after.
- **SC-006**: The Monero tab is re-pinned to its new golden, so the next unintended reflow still fails a test.

## Assumptions

- The sparkline's value was trend, and the bar's is comparison across nodes at a glance. The
  operator has judged the second more useful in a table of four rows; this spec does not
  relitigate that.
- `Sparkline()` and the history ring stay in the tree though nothing renders them. They are
  exported, tested, and cheap, and FR-010 keeps their removal a separate decision — an
  unused-view cleanup is not something to smuggle into a layout change.
- The bar is 8 cells wide: enough to rank four nodes by eye, narrow enough that the column
  (bar, space, percentage) is no wider than the two columns it replaces.
