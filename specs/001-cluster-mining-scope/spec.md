# Feature Specification: Cluster Mining Scope

**Feature Branch**: `001-cluster-mining-scope`

**Created**: 2026-07-30

**Status**: Draft

**Input**: User description: "A tool I run that keeps working and shows me how Monero is mining inside the cluster."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See mining health at a glance, continuously (Priority: P1)

An operator runs one command and gets a live, self-refreshing view of the
`monero-idle-miner` DaemonSet across the cluster: which nodes are mining, each
node's hashrate, how many threads are active, shares accepted/rejected, and
whether each miner is connected to the configured pool (and not silently mining
to the built-in donate pool). The view keeps updating until they quit.

**Why this priority**: This is the core need — today the only visibility is ad-hoc
`kubectl top`/`logs`. A single always-on view is the whole point of the tool and
is a viable product on its own.

**Independent Test**: Run the tool against a cluster (or with `--mock`); confirm it
lists one row per mining node with live hashrate, thread count, share counts, and
pool-connection status, and that the numbers update on the refresh interval.

**Acceptance Scenarios**:

1. **Given** the miner DaemonSet is running on N nodes, **When** the operator starts the tool, **Then** it shows N node rows, each with hashrate, active/total threads, accepted/rejected shares, and pool status, refreshing on the interval.
2. **Given** a miner has fallen back to the built-in donate pool, **When** the view refreshes, **Then** that node is flagged as NOT mining to the operator's configured pool.
3. **Given** a miner pod is not Running (pending/crashloop), **When** the view refreshes, **Then** that node shows the pod state instead of stale mining numbers.

---

### User Story 2 - Watch it take and yield idle CPU (Priority: P2)

The operator can see, over time, how much CPU each miner is using versus how much
the node has free — and watch the miner's usage fall when other workloads demand
CPU and rise again when they finish. This makes the "opportunistic, leaves
headroom, yields under load" behaviour visible rather than asserted.

**Why this priority**: It answers the question that motivated the whole miner
("is it really only using idle CPU?"). Valuable but depends on the P1 view existing.

**Independent Test**: With the tool running, start a CPU-hungry workload on a mining
node and confirm the displayed miner CPU drops and the node's free-CPU rises in the
session history, then recovers when the workload ends.

**Acceptance Scenarios**:

1. **Given** a node is otherwise idle, **When** the view refreshes, **Then** it shows the miner's current CPU and the node's free-CPU headroom.
2. **Given** a higher-priority workload starts consuming CPU on that node, **When** subsequent refreshes occur, **Then** the miner's shown CPU decreases and recent history reflects the dip, recovering after the workload stops.
3. **Given** the cluster has no metrics source available, **When** the view refreshes, **Then** CPU fields read "unavailable" and the rest of the view still works.

---

### User Story 3 - See pool-side earnings (Priority: P3)

The operator sees what the mining pool reports for their wallet: pool-side
hashrate, amount currently due (unpaid balance), and total paid out — so they know
whether the effort is actually accruing anything.

**Why this priority**: Rounds out "how is mining going" with the money view. Nice to
have on top of the in-cluster picture; independent of it.

**Independent Test**: With a wallet configured (or auto-detected), confirm the tool
shows a pool panel with reported hashrate, amount due, and total paid, and marks the
panel stale if the pool API cannot be reached.

**Acceptance Scenarios**:

1. **Given** a wallet is configured or auto-detected from the miner, **When** the pool responds, **Then** the panel shows reported hashrate, amount due, and total paid.
2. **Given** the pool API is unreachable, **When** the view refreshes, **Then** the panel shows the last-known values marked stale with a timestamp, and the tool keeps running.
3. **Given** the operator passes a flag to disable the pool view, **When** the tool runs, **Then** no pool query is made and no pool panel is shown.

---

### Edge Cases

- **No miner found**: the label selector matches nothing → the tool says so clearly and keeps polling (a miner may appear later), rather than erroring out.
- **Miner HTTP API disabled**: detailed hashrate/thread/share data is derived from miner logs instead, and the view notes the reduced-fidelity source.
- **Partial data**: a node reachable for CPU but not for miner stats (or vice-versa) shows what it has and marks the rest unavailable.
- **Non-interactive invocation** (piped / no TTY): the tool emits periodic text (or JSON) snapshots instead of a full-screen UI.
- **One-shot use**: the operator wants a single snapshot for a script or a quick check, not a running UI.
- **Large clusters**: many mining nodes remain readable (sorted, scannable) without overflowing the terminal.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST discover the miners by a configurable label selector in a configurable namespace and present one entry per mining node.
- **FR-002**: For each node it MUST show mining status: hashrate (short and longer windows), active/total threads, accepted/rejected shares, pool endpoint, and connection state.
- **FR-003**: It MUST detect and flag when a miner is connected to a pool other than the operator's configured pool (e.g. a donate-pool fallback).
- **FR-004**: It MUST show, per node, the miner's current CPU usage and the node's free-CPU headroom, and retain enough recent history to show the trend (rise/fall) over the session.
- **FR-005**: It MUST present pool-side earnings for the operator's wallet: reported hashrate, amount due, and total paid; the wallet MAY be auto-detected from the running miner and MUST be overridable, and the pool view MUST be disableable.
- **FR-006**: It MUST run continuously, refreshing on a configurable interval, AND support a single-snapshot mode that gathers once and exits.
- **FR-007**: It MUST render an interactive full-screen view when attached to a terminal, and emit periodic plain-text and machine-readable (line-delimited) snapshots when not, or when explicitly requested.
- **FR-008**: It MUST be strictly read-only toward the cluster and the miner — it never creates, modifies, deletes, scales, or execs into any resource (Constitution II).
- **FR-009**: It MUST degrade gracefully: any single unavailable source (miner API, metrics, pool API) results in that field/panel being marked unavailable or stale, never a crash (Constitution III).
- **FR-010**: It MUST provide a mock/offline mode that renders the full experience with synthetic data and no cluster access (Constitution III).
- **FR-011**: Aggregate/cluster-level figures (total hashrate, nodes mining, total accepted shares, aggregate CPU used vs free) MUST be shown alongside the per-node detail.
- **FR-012**: It MUST NOT emit telemetry and MUST make no outbound connections other than the cluster API server and the configured pool API (Constitution VII).

### Key Entities *(include if feature involves data)*

- **Miner instance**: one XMRig running on a node — node name, pod state, restarts, image, uptime, worker id.
- **Mining stats**: hashrate windows, active/total threads, accepted/rejected shares, best share, configured pool + connection state/ping.
- **CPU sample**: miner CPU used and node free-CPU at a point in time; a bounded recent history per node forms the trend.
- **Pool earnings**: reported hashrate, amount due, total paid, last-share time, for a wallet.
- **Snapshot**: a single moment's cluster summary + per-node entries + pool earnings; the unit of refresh, of one-shot output, and of a JSON record.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: From a cold start, an operator sees per-node mining status within ~5 seconds of launching the tool (or immediately, in mock mode).
- **SC-002**: When a competing workload starts or stops on a mining node, the change in that node's miner-CPU/free-CPU is visible in the tool within two refresh intervals.
- **SC-003**: A miner that has fallen back to the donate pool is flagged on the first refresh after it happens — no manual log reading required.
- **SC-004**: The tool runs unattended for at least an hour, refreshing continuously, without crashing when a node, the miner API, the metrics source, or the pool API becomes temporarily unavailable.
- **SC-005**: A new user can see the full experience with a single `--mock` command and no cluster access.

## Assumptions

- The operator has a kubeconfig with read access to the namespace where the miner runs (pods, pod logs, pod proxy, and metrics).
- The miner is the `monero-idle-miner` chart (DaemonSet, label `app.kubernetes.io/name=monero-idle-miner`), optionally exposing the XMRig HTTP API; when it is not exposed, log-derived stats are acceptable at reduced fidelity.
- Pool-side earnings assume a SupportXMR-compatible public stats API for v1; other pools are roadmap.
- The wallet address is public information and safe to display and to query the pool with.
- In-cluster deployment and a web UI are out of scope for v1 (roadmap).
