# Feature Specification: Tab presentation standard

**Feature Branch**: `004-tab-presentation-standard`

**Created**: 2026-08-01

**Status**: Draft

**Input**: User description: "It's showing TH and workers being added, but we should have a standard about how we present data in each tab." Follow-up decision: the Monero tab is the reference and does not change; the Bitcoin tab conforms to it.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read either tab without relearning it (Priority: P1)

An operator who has watched the Monero tab for weeks presses `b` and can read the Bitcoin
tab immediately: the same line does the same job, the same column head means the same
thing, and the same glyph means the same thing. Nothing about the switch asks them to
re-learn where to look.

**Why this priority**: This is the whole feature. Two panels that report the same *kind* of
thing — miners, hashrate, shares — in two different dialects make the operator translate
between them, which is exactly the work a dashboard exists to remove.

**Independent Test**: Put both tabs side by side (`--mock`) and check every shared concept —
identity line, headline metrics, table columns, unknown values, notes — reads the same way
on both.

**Acceptance Scenarios**:

1. **Given** both tabs, **When** the operator compares the headline lines, **Then** each begins with a bold role label followed by ` · `-separated metrics, most important first.
2. **Given** both tabs, **When** the operator compares tables, **Then** identity columns are left-aligned, numeric columns right-aligned, and column heads are upper-case.
3. **Given** either tab, **When** a hashrate column reports an average, **Then** the column head names the averaging window (as `HASH/60s` does today).
4. **Given** either tab, **When** shares are shown, **Then** they use one format everywhere they appear — headline and table alike.
5. **Given** either tab, **When** a value is unavailable, **Then** it renders as the same single mark, never as a zero and never as a second spelling of "unknown".

---

### User Story 2 - The Monero tab does not move (Priority: P1)

The operator's existing muscle memory, screenshots and terminal width all keep working: the
Monero tab renders exactly as it does today.

**Why this priority**: Equal to US1 — a consistency change that reflows the tab someone
watches daily has cost them more than it gave. The reference is the thing that stays still.

**Independent Test**: Capture the Monero tab's rendered body before and after; they are
byte-identical.

**Acceptance Scenarios**:

1. **Given** an unchanged snapshot, **When** the Monero tab renders before and after this change, **Then** the output is identical byte for byte.
2. **Given** the Monero tab's existing spellings of unavailable data (`n/a` in the CPU columns), **When** the standard names one mark for the Bitcoin tab, **Then** the Monero tab keeps its own, and the standard records that exception explicitly rather than pretending it does not exist.

---

### User Story 3 - The standard survives the next panel (Priority: P2)

Whoever adds the third panel — another coin, another pool, a node view — has one document
to follow and shared helpers that make following it the path of least resistance.

**Why this priority**: Without this, the tabs re-diverge on the next feature and the work is
repeated. Lower than US1/US2 because the immediate inconsistency is what the operator sees.

**Independent Test**: The rules exist as a written reference, and the formatting of shares,
hashrates, durations and unknown values comes from shared functions rather than per-panel
string building.

**Acceptance Scenarios**:

1. **Given** a new panel, **When** its author formats a share count or a hashrate, **Then** a shared helper exists for it and is the obvious thing to call.
2. **Given** the standard, **When** a reviewer checks a panel against it, **Then** each rule is stated concretely enough to be answered yes or no.

### Edge Cases

- A metric the source does not report (public-pool reports no rejected shares) → the unavailable mark, never a fabricated `0✗`.
- A pool with no per-miner rows → the table is omitted entirely rather than rendered empty with headers.
- Terminal narrower than the table → unchanged behaviour: the box stops stretching, rows do not wrap.
- A panel that needs a note and a remedy → they are distinguishable from each other and from a metric line.
- Two panels stacked (both pools present) → the separation between them is the same as between sections of the Monero tab.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Monero tab's rendered output MUST NOT change as a result of this feature.
- **FR-002**: A written presentation standard MUST exist, stating each rule concretely enough that a panel can be checked against it.
- **FR-003**: Panel content MUST be flush left; the box's own padding is the only indentation. Sub-lines MUST NOT be indented relative to their panel's header.
- **FR-004**: Each panel MUST open with a header line of a bold role label followed by ` · `-separated headline metrics, ordered most important first — the grammar the Monero `cluster` and `pool` lines already use.
- **FR-005**: Identity and provenance (where the workload runs, its state, which source the numbers came from) MUST appear as dimmed context, not compete with the metrics for the header line.
- **FR-006**: Table column heads MUST be upper-case; identity columns left-aligned, numeric columns right-aligned.
- **FR-007**: A column of averaged hashrates MUST name its window in the column head (`HASH/60s`, `HASH/1m`); an instantaneous figure MUST NOT imply a window it does not have.
- **FR-008**: Shares MUST render in one format wherever they appear — accepted then rejected, with the same glyphs the Monero tab uses — and MUST omit the rejected half when the source does not report it, rather than showing a fabricated zero.
- **FR-009**: Unavailable values on the Bitcoin tab MUST use a single mark. The Monero tab's existing `n/a` in its CPU columns is a recorded exception (FR-001 outranks uniformity here).
- **FR-010**: A problem note MUST be visually distinct from a remedy and from a metric line, and MUST use the same marker and colour the Monero tab uses for warnings.
- **FR-011**: Panels MUST be separated the same way sections of the Monero tab are separated.
- **FR-012**: Identity strings too long for their column (worker names, payout addresses) MUST be truncated with the same mark the Monero tab uses for truncated node names.
- **FR-013**: The formatting of hashrates, share counts, difficulties, durations and unavailable values MUST come from shared helpers, so a new panel inherits the standard by default.
- **FR-014**: The `stream` output MUST follow the same rules where they apply to plain text (column heads, share format, unavailable mark); `json` is a data contract and is unaffected.

### Key Entities

- **Panel**: one framed section of a tab — header line, optional context line, optional gauge, optional table, optional notes.
- **Headline metric**: one ` · `-separated figure on a header line.
- **Column head**: the upper-case name of a table column, carrying the averaging window when there is one.
- **Unavailable mark**: the single glyph standing for "this source does not report it".

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator switching tabs finds the same concept in the same shape: every rule in the standard holds on both tabs, or is a recorded exception.
- **SC-002**: The Monero tab's rendered body is byte-identical before and after, proven by a test.
- **SC-003**: Shares, hashrates and unavailable values have exactly one rendering each across the Bitcoin tab's headline, table and stream output.
- **SC-004**: No panel builds those strings by hand; each calls the shared helper.
- **SC-005**: A reviewer can check a new panel against the standard in a single pass, rule by rule.

## Assumptions

- The Monero tab is the reference because it is the one in daily use; where it is internally inconsistent (`n/a` beside `—`), that inconsistency is documented rather than propagated, and is not fixed here.
- This feature changes presentation only: no new data is collected, no model type changes, and the `json` contract is untouched.
- Colour and glyph choices remain terminal-safe: single-width marks, no emoji, no reliance on colour alone to carry meaning.
- Adding trends (a Bitcoin hashrate sparkline) and an uptime column to the Monero table were considered and are out of scope — they change the reference tab.
