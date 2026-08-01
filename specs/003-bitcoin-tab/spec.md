# Feature Specification: Bitcoin tab

**Feature Branch**: `003-bitcoin-tab`

**Created**: 2026-07-31

**Status**: Draft

**Input**: User description: "Add a box holding all the information with the title outside it, then tuples — one for Monero and another one for Bitcoin. Switch with m/b keys. The Bitcoin information should come from public-pool or ckpool, or could be both; the system needs to auto-detect this when connecting to the cluster."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the Bitcoin side of the homelab (Priority: P1)

An operator runs a Monero miner and a Bitcoin solo-mining pool on the same cluster. They
already watch Monero with `minepulse watch`. They press `b` and the same framed dashboard
now shows the Bitcoin side: which pool is running, its hashrate, how many workers are
connected, accepted/rejected shares and the best share seen. Pressing `m` goes back to
Monero.

**Why this priority**: This is the feature — one scope for both coins, in the tool the
operator already leaves running. Without it they need a second window and a second tool.

**Independent Test**: Run `minepulse watch --mock`, press `b`, and confirm a Bitcoin view
renders with pool identity, hashrate, workers and shares; press `m` and confirm the Monero
view returns unchanged.

**Acceptance Scenarios**:

1. **Given** the dashboard is open on the Monero tab, **When** the operator presses `b`, **Then** the box shows the Bitcoin view and the tab strip marks Bitcoin as active.
2. **Given** the dashboard is open on the Bitcoin tab, **When** the operator presses `m`, **Then** the Monero view returns with exactly the content it had before.
3. **Given** either tab is active, **When** the refresh interval elapses, **Then** both coins' data are refreshed, so switching tabs never shows stale-by-tab data.

---

### User Story 2 - The pool is found without being configured (Priority: P1)

The operator never tells minepulse where their pool is or which one they run. On connecting
to the cluster, minepulse finds the solo pool by itself, recognises whether it is
public-pool or ckpool, and reads that implementation's stats the way that implementation
exposes them.

**Why this priority**: Equal to US1 in importance — a Bitcoin tab that must be hand-wired
with namespace, port and implementation flags is not the feature the operator asked for.
Auto-detection is what makes it feel like one tool over one cluster.

**Independent Test**: Point minepulse at a cluster running the `bitcoin-stack` mining-pool
chart with `pool.implementation=public-pool`, then at one with `ckpool`, with no
Bitcoin-related flags set; confirm each is detected and its stats rendered, and that the
view names which implementation was found.

**Acceptance Scenarios**:

1. **Given** a cluster running public-pool, **When** minepulse gathers a snapshot, **Then** it identifies the pool as public-pool and reads its stats from the pool's own stats API.
2. **Given** a cluster running ckpool (which has no stats API), **When** minepulse gathers a snapshot, **Then** it identifies the pool as ckpool and derives its stats from what ckpool publishes to its log stream.
3. **Given** a cluster running both pools, **When** minepulse gathers a snapshot, **Then** both are listed in the Bitcoin view, each with its own stats, and neither hides the other.
4. **Given** a cluster with no Bitcoin pool at all, **When** minepulse gathers a snapshot, **Then** the Bitcoin tab states that no pool was found and the Monero tab is unaffected.

---

### User Story 3 - As much per-miner detail as the pool will give (Priority: P2)

Where the pool reports it, the operator sees a row per mining device — worker name,
hashrate, best difficulty and how long it has been connected — so a Bitaxe that has fallen
off or gone slow is visible at a glance. Where the pool reports only per-address figures,
they see a row per payout address instead; where it reports neither, they see the totals
and are told which detail is missing and why.

**Why this priority**: Valuable, but the pool-level totals in US1 already deliver the core
"is my Bitcoin mining alive" answer, and how much detail exists differs by implementation.

**Independent Test**: Against a pool reporting per-device data for a known payout address,
confirm one row per connected device; against a pool reporting only per-address data,
confirm one row per address plus a note naming what is unavailable; against neither,
confirm totals render with a note rather than an error.

**Acceptance Scenarios**:

1. **Given** a pool that reports per-device stats and a known payout address, **When** the Bitcoin tab renders, **Then** each connected device appears as its own row.
2. **Given** a pool that reports per-address but not per-device stats, **When** the Bitcoin tab renders, **Then** each address appears as a row (with its worker count) and a note states that per-device detail is unavailable from this source.
3. **Given** a pool reporting only aggregates, **When** the Bitcoin tab renders, **Then** the totals render and a note explains that no per-miner detail is available.
4. **Given** a worker that stops submitting shares, **When** the next snapshot is gathered, **Then** its row reflects the pool's own view of it (idle / last share) rather than disappearing silently.

---

### User Story 4 - Both coins in scripted and machine output (Priority: P3)

The operator pipes `minepulse snapshot` into a script, or runs `watch -o json` into a log
shipper, and gets the Bitcoin data alongside the Monero data in the same record.

**Why this priority**: Keeps the CLI contract whole (Constitution V), but the interactive
view is what was asked for.

**Independent Test**: Run `minepulse snapshot -o json --mock` and confirm the record has a
Bitcoin section with the same numbers the TUI shows; run `-o stream` and confirm a compact
Bitcoin block is printed.

**Acceptance Scenarios**:

1. **Given** `--output json`, **When** a snapshot is emitted, **Then** it contains a Bitcoin section, omitted entirely when no pool was found.
2. **Given** `--output stream`, **When** a snapshot is emitted, **Then** a compact Bitcoin block follows the Monero block in plain text with no cursor control.
3. **Given** an existing consumer that reads only the Monero fields, **When** it parses the new record, **Then** every field it relied on is unchanged and in place.

### Edge Cases

- Pool pod exists but is not Running (ImagePullBackOff, Init) → the Bitcoin view shows the pod's state and the reason, not a blank panel or an error.
- Pool is reachable but has no miners connected → zeros and "no workers connected", distinguishable from "pool not found".
- ckpool has only just started and has not yet published a status line → the view says stats are not available yet rather than reporting a false zero.
- The pool's stats source fails mid-session (API stops answering, log read denied) → last known values are shown marked stale, consistent with the Monero pool panel.
- minepulse lacks permission to look outside the miner's namespace → detection degrades to the namespaces it can read and says so, without failing the snapshot.
- The terminal is too narrow for the worker table → the box does not wrap into unreadable rows (same constraint the Monero table already honours).
- Pool detection matches something that is not a pool (a similarly named pod) → it must be identified by evidence (implementation fingerprint), not by name alone.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The dashboard MUST present its content in a single framed box, with the tool title and update status outside that box (already delivered; the tab strip joins them).
- **FR-002**: The dashboard MUST offer exactly two tabs — Monero and Bitcoin — with a visible strip marking which is active, and MUST start on Monero.
- **FR-003**: `m` MUST select the Monero tab and `b` the Bitcoin tab; existing keys (`q`, `p`, `r`) MUST keep their meaning on both tabs.
- **FR-004**: Switching tabs MUST NOT trigger a gather, change the refresh interval, or alter pause state — it only changes what is drawn.
- **FR-005**: Each refresh MUST gather both coins, so a tab shows data as fresh as the update timestamp claims.
- **FR-006**: minepulse MUST auto-detect the Bitcoin solo pool in the cluster with no Bitcoin-specific configuration, identifying which implementation it is (public-pool or ckpool) from evidence on the workload itself.
- **FR-007**: When more than one pool is present, all detected pools MUST be shown, each labelled with its implementation and workload identity.
- **FR-008**: For public-pool, minepulse MUST read pool-level statistics from the pool's own read-only stats API, reached without opening a local port-forward.
- **FR-009**: For ckpool, whose stats API is a local socket minepulse may not reach without exec, minepulse MUST derive statistics from ckpool's periodic status records when those records are present in the pod's log stream, labelling the view with that reduced-fidelity source — mirroring the existing XMRig API→logs fallback.
- **FR-009a**: When a ckpool pool is detected but publishes no readable status records (the stock deployment logs them to a file inside the container, not to its log stream), minepulse MUST still report the pool's identity, state and uptime, MUST state that no stats source is readable, and MUST give the operator an actionable remedy — never a zero hashrate and never a silently empty panel.
- **FR-010**: The Bitcoin view MUST show, where the source provides it: pool implementation and workload state, hashrate (with the averaging window named), connected workers/users, accepted and rejected shares, best share, and when the stats were last updated.
- **FR-011**: The Bitcoin view MUST show the finest per-miner detail its source provides, degrading one rung at a time and naming what is missing: per-device rows → per-payout-address rows (with each address's worker count) → pool totals only.
- **FR-011a**: Hashrates MUST be normalised to a single internal unit regardless of how the source states them — a raw number, or a magnitude-suffixed string such as `480G` or `1.23P` — so both tabs and all outputs agree.
- **FR-012**: Every Bitcoin data source MUST degrade rather than abort (Constitution III): a missing, unreachable or unparseable source produces a stated "unavailable"/stale view and a warning, never a failed snapshot and never a fabricated zero.
- **FR-013**: All Bitcoin collection MUST be strictly read-only (Constitution II) — list/get workloads, read logs, and read-only HTTP GETs through the API server's proxy; no exec, no writes, no new RBAC verbs beyond those already granted.
- **FR-014**: `--mock` MUST render both tabs fully populated with synthetic data, including a per-worker table, with no cluster access.
- **FR-015**: `stream` and `json` output MUST carry the Bitcoin data, with the Bitcoin section omitted when no pool was detected, and MUST leave the existing Monero fields unchanged (additive only).
- **FR-016**: The operator MUST be able to override detection — restrict or point the search, supply the payout address used for per-worker stats, and disable Bitcoin collection entirely — via flags that default to full auto-detection.
- **FR-017**: Bitcoin collection MUST NOT slow the refresh loop beyond its interval: sources are polled with bounded timeouts and a slow or hanging pool degrades that tab only.
- **FR-018**: `doctor` MUST report what Bitcoin detection found — which pools, which stats source, and the remedy when a pool is present but its stats cannot be read.
- **FR-019**: Hashrates MUST be rendered in the same human-readable form across both tabs, scaled to the magnitude of the coin (H/s through PH/s), never as raw unscaled numbers.
- **FR-020**: The payout address MUST be treated as sensitive-adjacent display data (Constitution VII): shown truncated in the interactive view, never logged, and only ever read from cluster metadata or a flag.

### Key Entities

- **Coin tab**: which of the two views is drawn — a UI selection only, never a data-collection switch.
- **Bitcoin pool**: one detected solo-mining workload — its implementation (public-pool/ckpool), workload identity (pod/node/state/uptime), stats source (API/logs/none) and freshness.
- **Bitcoin pool stats**: the pool-level aggregate — hashrate over a named window, users, connected and idle workers, accepted/rejected shares, best share, share of network difficulty, last update.
- **Bitcoin miner row**: one line of per-miner detail, at whichever granularity the source supports — a device (name, hashrate, best difficulty, connected since, last seen) or a payout address (hashrate, worker count, shares, best share, last share).
- **Bitcoin payout address**: the address workers authenticate with; the key per-worker stats are looked up by.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator running a supported solo pool sees Bitcoin hashrate, workers and shares within one refresh interval of starting minepulse, having set zero Bitcoin-related flags.
- **SC-002**: Switching between tabs is instantaneous — no gather, no visible redraw delay, no change in the data's age.
- **SC-003**: Both supported implementations are recognised: public-pool with pool-level totals and per-device rows; ckpool with pool-level totals and per-address rows when its status records are readable, and identity + state + a stated remedy when they are not. Each panel names its source and what that source cannot give.
- **SC-004**: With no pool, an unreachable pool, or a pool that has published nothing yet, the Bitcoin tab states which of those three it is, and the Monero tab and refresh loop keep working unchanged.
- **SC-005**: `--mock` renders both tabs completely on a machine with no cluster, making the whole feature demonstrable and testable offline.
- **SC-006**: Existing Monero consumers of `json` and `stream` output see no change to any field they already read.
- **SC-007**: Parsing of every Bitcoin stats format is covered by tests against recorded fixtures before the collectors ship (Constitution IV).

## Assumptions

- The Bitcoin pool is the one deployed by the sibling `bitcoin-stack` mining-pool chart — public-pool (a stats API on a ClusterIP service, one Deployment) or ckpool (stratum only, status published to its logs) — and runs in the same cluster minepulse already reads.
- Pool workloads are identified by the container image and workload labels the chart sets; name matching alone is never sufficient evidence.
- ckpool's own stats API is a Unix socket inside its container (the one `ckpmsg` speaks to), and its status files live on its volume; both would require exec to reach, so neither is available to minepulse under Constitution II. Its periodic status records — pool status plus one record per payout address — are therefore the only possible ckpool source, and they carry no per-device breakdown.
- Those records are logged at NOTICE, and ckpool sends NOTICE to its logfile only — its container log stream carries warnings and errors alone. A stock ckpool deployment therefore yields identity, state and uptime but no hashrate; full stats require the operator to route ckpool's logfile into the pod's log stream (a chart-side change they own). minepulse parses the records opportunistically and states the remedy when they are absent.
- public-pool exposes pool-wide totals without any address, and per-device detail only under a payout address; therefore per-device rows require the address to be supplied or discoverable, and their absence is a documented degradation, not a failure.
- The Bitcoin tab reports what the pool reports. minepulse does not query the Bitcoin node, chain state, price, or block explorers, and does not compute earnings — solo mining has no per-share payout to report.
- One box holds one tab's content at a time; a side-by-side two-coin layout is explicitly out of scope for this feature.
- The default refresh interval remains suitable for both coins; no separate Bitcoin interval is introduced.
