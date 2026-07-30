# Data Model: Cluster Mining Scope

All types live in `internal/model`. They are plain data (no cluster clients) so the
aggregation and rendering logic is pure and unit-testable (Constitution IV). JSON tags
define the `--output json` contract.

## Snapshot

The unit of refresh, of one-shot output, and of one `json` line.

| Field | Type | Notes |
|---|---|---|
| `Timestamp` | `time.Time` | when the gather completed |
| `Cluster` | `ClusterSummary` | aggregates |
| `Nodes` | `[]NodeStatus` | one per discovered mining node, sorted by node name |
| `Pool` | `*PoolStats` | nil when `--no-pool` or no wallet resolved |
| `Warnings` | `[]string` | non-fatal issues (e.g. "metrics unavailable") |

## ClusterSummary

| Field | Type | Notes |
|---|---|---|
| `NodesMining` | `int` | pods Running with an active miner |
| `NodesTotal` | `int` | pods discovered by selector |
| `TotalHashrate` | `float64` | H/s, sum of per-node 60s hashrate |
| `AcceptedShares` | `int64` | sum |
| `RejectedShares` | `int64` | sum |
| `MinerCPUMilli` | `int64` | sum of miner container CPU (mCPU) |
| `NodeCPUFreePct` | `float64` | aggregate free-CPU % across mining nodes (avg weighted by cores); `-1` = unavailable |

## NodeStatus

| Field | Type | Notes |
|---|---|---|
| `Node` | `string` | node name |
| `Pod` | `string` | miner pod name |
| `Phase` | `string` | Running / Pending / CrashLoopBackOff / ... |
| `Restarts` | `int32` | container restarts |
| `Image` | `string` | miner image ref |
| `Uptime` | `time.Duration` | since container start |
| `Mining` | `*MiningStats` | nil if not Running or stats unavailable |
| `CPU` | `*CPUSample` | nil if metrics unavailable |
| `History` | `*Ring` | bounded recent (`CPUSample`) for sparklines (not serialized) |
| `StatsSource` | `enum{api,logs,none}` | fidelity indicator |
| `Note` | `string` | e.g. "pod pending", "API unreachable — using logs" |

## MiningStats

| Field | Type | Notes |
|---|---|---|
| `Hashrate10s/60s/15m` | `float64` | H/s; `-1` when a window is unknown |
| `ThreadsActive` | `int` | started mining threads |
| `ThreadsTotal` | `int` | logical CPUs the miner sees |
| `SharesGood` | `int64` | accepted |
| `SharesTotal` | `int64` | accepted + rejected submissions |
| `Pool` | `string` | host:port the miner is connected to |
| `Connected` | `bool` | live connection |
| `PingMs` | `int` | pool ping; `-1` unknown |
| `DonateFallback` | `bool` | true if `Pool` is a `*.xmrig.com` donate pool / not the configured pool (R5) |
| `WorkerID` | `string` | rig-id |
| `Version` | `string` | XMRig version |

## CPUSample

| Field | Type | Notes |
|---|---|---|
| `T` | `time.Time` | sample time |
| `MinerMilli` | `int64` | miner container CPU (mCPU) |
| `NodeUsedMilli` | `int64` | node CPU used (mCPU) |
| `NodeCapacityMilli` | `int64` | node allocatable CPU (mCPU) |
| `FreePct` | `float64` | (capacity − used) / capacity × 100 |

Derived: `MinerPctOfNode = MinerMilli / NodeCapacityMilli × 100`.

## PoolStats

| Field | Type | Notes |
|---|---|---|
| `Wallet` | `string` | address (public) |
| `ReportedHashrate` | `float64` | H/s reported by the pool |
| `AmountDueXMR` | `float64` | unpaid balance (atomic units → XMR) |
| `AmountPaidXMR` | `float64` | lifetime paid |
| `TotalHashes` | `int64` | lifetime |
| `LastShare` | `time.Time` | last accepted share time |
| `Stale` | `bool` | pool API unreachable this tick → last-known shown |
| `AsOf` | `time.Time` | when these values were last fetched successfully |

## Ring (`ring.go`)

Fixed-capacity circular buffer of `CPUSample` (and hashrate) per node — capacity ≈ enough
for the on-screen sparkline (e.g. 120 samples). O(1) append; `Slice()` returns oldest→newest.
Not serialized in `json` output (it is session UI state).

## Enumerations

- `StatsSource`: `api` (XMRig HTTP API), `logs` (log-parse fallback), `none` (no data).
- Availability is represented by nil pointers + `-1` sentinels + `Warnings`, never by errors
  that abort a snapshot (Constitution III).
