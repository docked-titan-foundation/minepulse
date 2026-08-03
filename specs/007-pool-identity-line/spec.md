# Feature Specification: Pool identity line

**Feature Branch**: `007-pool-identity-line`

**Created**: 2026-08-03

**Status**: Draft

**Input**: User description: "in each [tab] mark in the first line inside the box if the pool it
is pointing to is internal (from the cluster) or external (from outside), in this format
`[internal] Pool URL - Pool type - IP`." Clarified: pool type means the *brand* of the pool,
and the line should also carry the type of mining — solo or shared.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Know at a glance where the work is going (Priority: P1)

The operator opens a tab and reads one line: which pool, whose pool, how it shares work,
whether it lives in this cluster, and at what address. Today that takes reading a truncated
host:port out of the last table column and knowing from memory what runs there.

**Why this priority**: This is the request, and "internal or external" is the part no existing
line answers at all — a pool address alone does not say whether the traffic leaves the cluster.

**Acceptance Scenarios**:

1. **Given** either tab, **When** it renders, **Then** its first line inside the box is `[locality] URL - brand - mode - IP`.
2. **Given** a pool whose address resolves to a Pod IP or Service ClusterIP minepulse can read, **When** the line renders, **Then** it reads `[internal]`.
3. **Given** a pool on a routable public address, **When** the line renders, **Then** it reads `[external]`.
4. **Given** an address minepulse can neither match nor classify, **When** the line renders, **Then** the locality is the unavailable mark and the line still renders every field it does know.

---

### User Story 2 - A mis-wired miner is visible on the summary line (Priority: P1)

When one node has fallen back to XMRig's donate pool, the miners no longer agree on where the
work goes. The line must not silently report the majority's pool as though it were the whole
truth.

**Why this priority**: Catching a silent donate-pool fallback is the tool's headline feature.
A summary line that averages it away actively hides what the tab exists to show.

**Acceptance Scenarios**:

1. **Given** nodes pointing at different pools, **When** the line renders, **Then** it names the pool most nodes are on and marks that the miners disagree.
2. **Given** every node on the same pool, **When** the line renders, **Then** no divergence marker appears.

---

### User Story 3 - The claim is only as strong as the evidence (Priority: P2)

`[internal]` means "this address is an object in your cluster", not "this address looks
private". A pool on the operator's NAS is private and external, and saying otherwise would be
a false claim about where their hashrate goes.

**Why this priority**: The distinction only bites in the setups most likely to be
misconfigured, and a confidently wrong marker is worse than an honest unknown.

**Acceptance Scenarios**:

1. **Given** Services are readable, **When** an address matches a Pod IP or ClusterIP, **Then** the line reads `[internal]` and records which object matched.
2. **Given** Services are not readable, **When** classification falls back to address ranges, **Then** the line still answers, and records that the weaker test was used.

### Edge Cases

- A pool minepulse has never heard of → brand is the address's registrable domain, mode is the unavailable mark. Never a guess.
- A miner that has not connected yet → no IP to classify; the line reports the configured URL and marks locality unavailable.
- Two Bitcoin pools in the cluster → each panel carries its own identity line, since one line cannot describe two pools.
- A bare IP as the pool address → no brand to derive; the mark, not the IP repeated as a name.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Each tab MUST render a pool identity line as the first line inside the box, formatted `[locality] URL - brand - mode - IP`.
- **FR-002**: Locality MUST be one of internal, external, or the unavailable mark, and MUST be decided by matching the address against Pod IPs and Service ClusterIPs minepulse can read.
- **FR-003**: When Services cannot be read, classification MUST fall back to address-range inspection, and the basis of the decision MUST be recorded so it can be shown.
- **FR-004**: `[internal]` MUST NOT be claimed on range inspection alone when a cluster-object match was possible and failed.
- **FR-005**: Brand MUST come from a table of known pools keyed on the address, falling back to the registrable domain, and to the unavailable mark for a bare IP.
- **FR-006**: Mining mode MUST be solo, shared, or the unavailable mark — never inferred where the source cannot support it. `public-pool` is solo by design; `ckpool` is solo only when its image says so.
- **FR-007**: The Monero line MUST describe the pool most miners are connected to, and MUST mark divergence when they disagree.
- **FR-008**: On the Bitcoin tab, each pool panel MUST carry its own identity line as that panel's first line.
- **FR-009**: The pool IP MUST be taken from the miner's own view (XMRig reports the address it resolved), not from minepulse's, which usually runs outside the cluster and cannot resolve in-cluster DNS.
- **FR-010**: New cluster reads MUST be read-only and MUST be added to the shipped RBAC (Constitution II), and their absence MUST degrade the line rather than fail the snapshot (Constitution III).
- **FR-011**: `json` MAY gain fields but MUST NOT change or remove existing ones. `stream` MUST carry the same identity line in plain text (standard P12).

### Key Entities

- **Pool endpoint**: URL, resolved IP, brand, mining mode, locality, and the basis on which locality was decided.
- **Locality**: internal, external, or unknown.
- **Mining mode**: solo, shared, or unknown.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Both tabs open with an identity line carrying all five fields, or the unavailable mark in the fields that cannot be filled.
- **SC-002**: A pool matching a ClusterIP reads `[internal]`; a public address reads `[external]`; neither is decided by a coin flip when Services are unreadable — the fallback is recorded.
- **SC-003**: Divergent miner pools produce a marked line, proven by a test with one node on the donate pool.
- **SC-004**: Brand and mode resolution is a pure function of the address, tested against a table including an unknown pool and a bare IP.
- **SC-005**: Removing Services from RBAC degrades the line and never fails the snapshot.
- **SC-006**: Existing `json` fields are unchanged; the tabs are re-pinned to new goldens.

## Assumptions

- The miner's resolution is authoritative for the Monero tab because the miner is the thing
  actually connecting. minepulse's own resolution would answer a different question.
- Brand is cosmetic: a wrong or missing brand misleads nobody about where hashrate goes, which
  is why a fallback to the domain is acceptable where a fallback for locality is not.
- Bitcoin pools are internal by construction — detection finds pods in the cluster — so the
  marker's value on that tab is confirming the address, not discovering the locality.
