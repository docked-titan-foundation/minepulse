# Phase 1 Data Model: Bitcoin tab

All types are plain data in `internal/model` — no cluster clients, no HTTP — so aggregation
and rendering stay pure and unit-testable (Constitution IV). JSON tags are the `--output
json` contract; every addition is additive, so existing Monero consumers are untouched
(FR-015).

## Snapshot (extended)

```go
type Snapshot struct {
    Timestamp time.Time      `json:"timestamp"`
    Cluster   ClusterSummary `json:"cluster"`
    Nodes     []NodeStatus   `json:"nodes"`
    Pool      *PoolStats     `json:"pool,omitempty"`
    Bitcoin   *BitcoinView   `json:"bitcoin,omitempty"` // NEW — omitted when no pool detected
    Warnings  []string       `json:"warnings,omitempty"`
}
```

`Bitcoin == nil` means "no Bitcoin pool was detected". A detected-but-unreadable pool is a
non-nil view holding a pool whose `Source == "none"` — the three states of SC-004 are
distinguishable without inspecting strings.

## BitcoinView

The Bitcoin tab's whole payload: every pool found, plus what detection itself has to say.

| Field | JSON | Type | Meaning |
|---|---|---|---|
| Pools | `pools` | `[]BitcoinPool` | One entry per detected pool, sorted by namespace/name |
| Scope | `scope` | `string` | Namespaces detection actually searched (`"all"` or a list) — the honest scope when RBAC narrowed it |
| Note | `note,omitempty` | `string` | Why the view is thin (e.g. "no pool found", "cluster-wide search denied") |

## BitcoinPool

One solo-mining pool workload as minepulse sees it.

| Field | JSON | Type | Meaning |
|---|---|---|---|
| Impl | `impl` | `PoolImpl` | `public-pool` \| `ckpool` \| `unknown` |
| Namespace, Pod, Node | `namespace`, `pod`, `node` | `string` | Workload identity |
| Phase | `phase` | `string` | Display phase, same convention as `NodeStatus.Phase` (waiting/terminated reason when applicable) |
| Running | `running` | `bool` | Container is Running |
| Uptime | `uptime` | `time.Duration` | Since container start |
| Source | `stats_source` | `StatsSource` | `api` (public-pool) \| `logs` (ckpool) \| `none` |
| Detail | `detail_level` | `DetailLevel` | Finest per-miner rung this source reached: `device` \| `address` \| `totals` (FR-011) |
| Stats | `stats,omitempty` | `*BitcoinStats` | Pool-level aggregate; nil when nothing has been published yet |
| Miners | `miners,omitempty` | `[]BitcoinMiner` | Per-miner rows at whatever granularity `Detail` names |
| Stale | `stale` | `bool` | Stats are the last known values, not this tick's |
| AsOf | `as_of` | `time.Time` | When `Stats` was last successfully read |
| Note | `note,omitempty` | `string` | One line naming the degradation ("ckpool: per-device detail unavailable from logs") |

### PoolImpl / DetailLevel

```go
type PoolImpl string   // "public-pool" | "ckpool" | "unknown"
type DetailLevel string // "device" | "address" | "totals"
```

`unknown` is what a pool-shaped workload that matched no fingerprint gets (R5) — never a
guess.

## BitcoinStats

Pool-level aggregate. Every numeric field is `model.Unknown` (-1) when the source does not
provide it — never 0, which would read as "mining stopped" (FR-012).

| Field | JSON | Type | Source |
|---|---|---|---|
| Hashrate1m, Hashrate5m, Hashrate1h, Hashrate1d | `hashrate_1m`, `hashrate_5m`, `hashrate_1h`, `hashrate_1d` | `float64` (H/s) | ckpool: `Pool:{hashrate…}`; public-pool: summed worker `hashRate` |
| HashrateWindow | `hashrate_window` | `string` | Which window the headline figure is (e.g. `1m`) — FR-010 requires naming it |
| Users | `users` | `int` | ckpool `Users`; public-pool: n/a (`Unknown`) |
| Workers | `workers` | `int` | ckpool `Workers`; public-pool `workersCount` / `totalMiners` |
| WorkersIdle | `workers_idle` | `int` | ckpool `Idle` |
| Disconnected | `disconnected` | `int` | ckpool `Disconnected` |
| Accepted, Rejected | `accepted`, `rejected` | `float64` | ckpool accepted/rejected **difficulty** shares |
| BestShare | `best_share` | `float64` | ckpool `bestshare`; public-pool `bestDifficulty` |
| NetworkDiffPct | `network_diff_pct` | `float64` | ckpool `diff` — share of network difficulty found |
| SPS1m | `sps_1m` | `float64` | ckpool shares/second |
| BlockHeight | `block_height` | `int64` | public-pool `/api/pool` |
| BlocksFound | `blocks_found` | `int` | public-pool `/api/pool` (length of `blocksFound`) |
| TotalsAsOf | `totals_as_of,omitempty` | `time.Time` | Marks the ≤5-minute-cached `/api/pool` figures (R3) |
| LastUpdate | `last_update` | `time.Time` | ckpool `lastupdate`; public-pool: fetch time |
| Runtime | `runtime` | `time.Duration` | ckpool `runtime` |

## BitcoinMiner

One per-miner row. Which fields are populated follows `BitcoinPool.Detail`.

| Field | JSON | Type | device | address |
|---|---|---|---|---|
| Name | `name` | `string` | worker name | payout address (truncated for display only, FR-020) |
| Hashrate | `hashrate` | `float64` (H/s) | ✓ | ✓ |
| BestDifficulty | `best_difficulty` | `float64` | ✓ | ✓ (`bestshare`) |
| BestEver | `best_ever` | `float64` | — | ✓ |
| Workers | `workers` | `int` | — | ✓ (devices under that address) |
| Shares | `shares` | `float64` | — | ✓ |
| StartTime | `start_time` | `time.Time` | ✓ | ✓ (`authorised`) |
| LastSeen | `last_seen` | `time.Time` | ✓ | ✓ (`lastshare`) |
| SessionID | `session_id,omitempty` | `string` | ✓ | — |

## Validation rules

- **VR-1**: Hashrates are stored in H/s only. Suffixed source strings (`"480G"`, `"1.23P"`,
  `"1e+03T"`, bare `"999"`) are converted on parse (R2); a string that will not parse yields
  `Unknown` plus a warning, never 0.
- **VR-2**: An absent field is `Unknown` for numbers, the zero `time.Time` for times, and an
  empty slice for rows — renderers show `—` for all three.
- **VR-3**: `Detail` must be consistent with `Miners`: `device` and `address` require a
  non-empty `Miners`; `totals` requires it empty.
- **VR-4**: `Stale == true` requires non-nil `Stats` and an `AsOf` older than this tick —
  same contract as the Monero `PoolStats.Stale` panel.
- **VR-5**: Payout addresses are stored whole and truncated only at render time; they are
  never written to logs or diagnostics (FR-020, Constitution VII).
- **VR-6**: `Snapshot.Summarize()` does not fold Bitcoin figures into `ClusterSummary` —
  the Monero cluster summary keeps its exact current meaning (FR-015).

## State transitions

A pool entry moves between four states across ticks, and the view names which one it is in:

```text
not detected ──detect──> detected, no stats yet ──first read──> live ──read fails──> stale
      ^                          |                                |                    |
      └──────pod gone────────────┴────────────────────────────────┴────────────────────┘
```

- *detected, no stats yet* — ckpool that has not completed a status cycle, or public-pool
  still starting: `Stats == nil`, `Note` says so, nothing is zeroed.
- *live* — `Stale == false`, `AsOf == Timestamp`.
- *stale* — last known `Stats` retained, `Stale == true`, `AsOf` unchanged, warning added.
- *pod gone* — the entry drops out; if no pools remain, `BitcoinView.Note` explains and the
  view is kept (not nil) for that tick so the tab does not flicker to "never had a pool".
