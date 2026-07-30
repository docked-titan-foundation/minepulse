# Research: Cluster Mining Scope

Decisions that de-risk the plan. Each: **Decision → Rationale → Alternatives rejected.**

## R1. How to reach each miner's XMRig HTTP API

**Decision**: Use the Kubernetes API-server **pod proxy** subresource —
`GET /api/v1/namespaces/{ns}/pods/{pod}:8080/proxy/1/summary` and `/2/backends` — via
`client-go`'s `RESTClient().Get().Resource("pods").SubResource("proxy")`.

**Rationale**: No local ports to manage, one auth path (the kubeconfig), works the same
for every pod, and it is a read verb (`pods/proxy`) consistent with Constitution II.

**Alternatives rejected**: `kubectl port-forward`/client-go portforward (per-pod local
ports, teardown complexity, SPDY setup for every node each interval); requiring a Service
(the miner intentionally exposes no Service for the API).

## R2. What to do when the XMRig API is disabled

**Decision**: Fall back to parsing `pods/log`. Regex the XMRig lines: `speed 10s/60s/15m … H/s`,
`accepted (G/T)`, `READY threads N/M`, and the `POOL #1 <host:port>` banner (+ donate-pool
detection). Mark the node's stats source as `logs` (reduced fidelity).

**Rationale**: The miner chart defaults the API off; the tool must still be useful with zero
chart changes (Constitution III). Logs always exist and are a read.

**Alternatives rejected**: Requiring `httpApi.enabled` (couples the tool to a chart change);
`kubectl exec` into the pod to hit localhost (violates Constitution II — exec is not read-only).

## R3. CPU usage source

**Decision**: `metrics.k8s.io` via the `k8s.io/metrics` clientset — PodMetrics for the miner
container's CPU (millicores) and NodeMetrics for node CPU usage; node capacity from
`Node.status.allocatable.cpu`. Compute node used%/free% and the miner's share.

**Rationale**: Standard, already present in most clusters (metrics-server), read-only, cheap.

**Alternatives rejected**: cAdvisor/Prometheus scrape (extra dependency, not always present);
reading cgroup files via exec (violates Constitution II). If metrics-server is absent → CPU
fields "unavailable" (Constitution III), not an error.

## R4. Pool-side earnings API

**Decision**: SupportXMR public API `GET https://supportxmr.com/api/miner/{wallet}/stats`
(`hash`, `amtDue`, `amtPaid`, `totalHashes`, `lastHash`). Base URL configurable; disable with
`--no-pool`. Wallet auto-detected from the miner pod's `-u <wallet>` container arg, overridable
with `--wallet`.

**Rationale**: The deployed miner points at SupportXMR; its API is public, unauthenticated, and
matches v1 scope. Auto-detect means zero config in the common case.

**Alternatives rejected**: Generic multi-pool abstraction (roadmap — pools differ enough that a
premature abstraction is guesswork); on-chain balance lookup (impossible for Monero — the
address reveals nothing, see project background).

## R5. "Is it mining to the right pool?" detection

**Decision**: Treat any connected pool whose host matches `*.xmrig.com` (the built-in donate
pool, e.g. `donate.v2.xmrig.com`) — or any host other than the operator's configured pool — as
a **donate-pool fallback** and flag the node. Configured pool is the miner's `-o` arg (or a
flag).

**Rationale**: This is the exact failure the miner chart already had once (rig-id discarding the
pool); surfacing it is a headline value of the tool (SC-003).

**Alternatives rejected**: Trusting share submission alone (a miner can submit shares to the
donate pool too).

## R6. TUI vs non-TTY output

**Decision**: Detect a TTY on stdout. TTY → Bubble Tea TUI. Non-TTY (piped, CI, background) →
default to `stream` (compact text block per tick). `--output tui|stream|json` forces a mode;
`json` emits one JSON `Snapshot` per line (jsonl).

**Rationale**: Constitution V; lets a human get the rich view and lets an agent/script consume
snapshots without ANSI parsing.

**Alternatives rejected**: TUI-only (unscriptable, unreadable in background); logging library
framing (not a stable data contract).

## R7. Refresh model

**Decision**: A single ticker at `--interval` (default 3s) triggers one concurrent gather across
collectors (errgroup); results merged into a `Snapshot`; a bounded ring buffer keeps the last N
per node for sparklines. `snapshot` subcommand does exactly one gather and exits.

**Rationale**: Bounded, predictable load (Constitution II); simple, testable aggregation.

**Alternatives rejected**: Kubernetes `watch` streams (push complexity for little gain at this
cadence; metrics/pool are poll-only anyway).
