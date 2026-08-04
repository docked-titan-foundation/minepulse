# Feature Specification: Semantic color

**Feature Branch**: `008-semantic-color`

**Created**: 2026-08-04

**Status**: Draft

**Input**: User description: "can you use colors in the line of internal/external, add more colors
where it make sense."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read the locality without reading the word (Priority: P1)

The identity line renders in one flat colour, so establishing whether the work leaves the
cluster means reading the bracket. Colour makes it answerable at a glance: green inside, blue
outside.

**Why this priority**: This is the request, and locality is the one judgement that line makes.

**Acceptance Scenarios**:

1. **Given** a pool matching a cluster object, **When** the line renders, **Then** `[internal]` is the healthy colour.
2. **Given** a pool outside the cluster, **When** the line renders, **Then** `[external]` is an informational colour — *not* the warning colour, because for Monero that is the ordinary case.
3. **Given** a locality that could not be established, **When** the line renders, **Then** the marker is dim, so an unproven claim recedes rather than competing.
4. **Given** any locality, **When** the escapes are stripped, **Then** the text is exactly what `stream` prints.

---

### User Story 2 - The unhealthy value is the one that stands out (Priority: P1)

Colour goes only where there is a state to report: a node that should be mining and is not, a
rejected share, a worker connected but idle. Everything healthy stays uncoloured.

**Why this priority**: Equal to US1. Colouring every healthy value green leaves nothing for the
unhealthy one to stand out against, which is worse than no colour at all.

**Acceptance Scenarios**:

1. **Given** some nodes not mining, **When** the cluster line renders, **Then** the fraction is the degraded colour; with all mining it is healthy; with no nodes at all it is dim, not a fault.
2. **Given** a node with rejected shares, **When** its row renders, **Then** only the rejected half is coloured, and only when non-zero.
3. **Given** workers connected but idle, **When** the pool header renders, **Then** the idle count carries the warning colour.

### Edge Cases

- A monochrome terminal, a pipe, or `NO_COLOR` → every reading survives, because the text carries it too.
- `0/0` nodes → dim. Nothing is installed; that is not a failure to flag.
- `0✓/0✗` shares → uncoloured. A clean run should look calm, not green.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The identity line's locality marker MUST be coloured by value: healthy for internal, informational for external, dim for unknown.
- **FR-002**: External MUST NOT use the warning colour. It is a fact, not a fault.
- **FR-003**: Within the identity line, the address MUST be the brightest element and the separators and IP the dimmest, so attention lands on what identifies the pool.
- **FR-004**: Stripping colour from any line MUST leave exactly the text `stream` prints — colour may never be the only channel (Constitution; standard P14).
- **FR-005**: Colour MUST be applied only where a state is being reported, never to mark a category or to decorate.
- **FR-006**: A healthy or empty value MUST stay uncoloured, so the unhealthy one has contrast to stand out against.
- **FR-007**: The cluster mining fraction, non-zero rejected shares, and idle worker counts MUST carry state colour.
- **FR-008**: The standard MUST record what colour is allowed to mean, so the next panel inherits the rule instead of inventing one.
- **FR-009**: `stream` and `json` MUST NOT change.

### Key Entities

- **State colour**: green healthy, amber degraded, red broken, blue neutral-but-notable, dim supporting or unproven.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The three localities render in three distinct colours, proven by a test, and external is not the warning colour.
- **SC-002**: For every locality, the styled line stripped of escapes equals the plain line, proven by a test.
- **SC-003**: The cluster fraction renders differently for all-mining, partly-mining and no-nodes, with identical text.
- **SC-004**: Both tabs' goldens are unchanged — colour added no text and removed none.
- **SC-005**: `stream` and `json` are byte-identical before and after.

## Assumptions

- lipgloss already drops colour for non-TTY output, so `stream` through a pipe stays plain
  without special handling; FR-004 is what makes that safe rather than lossy.
- The palette is the one already in use (green 42, amber 214, red 203, dim 244) plus one
  informational blue. No existing colour changes meaning.
