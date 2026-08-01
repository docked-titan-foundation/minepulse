# Contract: CLI surface (Bitcoin tab)

The interfaces minepulse exposes to its users for this feature: flags, keys, and the two
machine-readable outputs. Everything here is additive — no existing flag, key, or field
changes meaning (FR-015).

## Flags (persistent, all commands)

| Flag | Default | Meaning |
|---|---|---|
| `-t`, `--tab` | `monero` | Which coin the dashboard opens on (`monero`\|`bitcoin`). An unknown value is rejected before any cluster call. Keys still switch freely once running |
| `--btc-namespace` | *(empty = search every namespace the credentials allow)* | Restrict pool discovery to one namespace |
| `--btc-selector` | *(empty)* | Extra label selector for pool discovery; ANDed with the fingerprint check, never replacing it |
| `--btc-address` | *(empty)* | Payout address used for per-device rows on public-pool. Unset ⇒ pool-wide totals only |
| `--btc-api-port` | `0` (auto) | Override the public-pool API port; auto = the container port named `api`, else 3334 |
| `--no-btc` | `false` | Skip Bitcoin discovery and collection entirely |

Existing flags keep their exact current behaviour. `--mock` populates both coins.

## Keys (TUI)

| Key | Action |
|---|---|
| `m` | Show the Monero tab |
| `b` | Show the Bitcoin tab |
| `tab` | Toggle between the two |
| `q`, `ctrl+c`, `esc` | Quit (unchanged) |
| `p` | Pause refresh (unchanged, applies to both coins) |
| `r` | Refresh now (unchanged, refreshes both coins) |

Tab selection never gathers, never alters pause state, and never changes the interval
(FR-004). The active tab is drawn in a strip between the title and the box; the footer hint
becomes `q quit · p pause · r refresh · m/b coin`.

## `--output json`

One snapshot per line, as today, with one new optional top-level key:

```json
{
  "timestamp": "2026-07-31T12:40:42Z",
  "cluster": { "...": "unchanged" },
  "nodes": [ { "...": "unchanged" } ],
  "pool": { "...": "unchanged" },
  "bitcoin": {
    "scope": "all namespaces",
    "pools": [
      {
        "impl": "ckpool",
        "namespace": "bitcoin", "pod": "mining-pool-7c9f8b6d4-x2k9p", "node": "orion",
        "phase": "Running", "running": true, "uptime": 512000000000,
        "stats_source": "logs", "detail_level": "address",
        "stale": false, "as_of": "2026-07-31T12:40:38Z",
        "note": "per-device detail unavailable from ckpool logs",
        "stats": {
          "hashrate_1m": 480000000000, "hashrate_5m": 472000000000,
          "hashrate_1h": 468000000000, "hashrate_1d": -1,
          "hashrate_window": "1m",
          "users": 1, "workers": 2, "workers_idle": 0, "disconnected": 0,
          "accepted": 12483, "rejected": 3, "best_share": 1200000000,
          "network_diff_pct": 0.02, "sps_1m": 1.4,
          "block_height": -1, "blocks_found": -1,
          "last_update": "2026-07-31T12:40:38Z", "runtime": 512000000000
        },
        "miners": [
          {
            "name": "bc1qexampleaddress…", "hashrate": 480000000000,
            "best_difficulty": 1200000000, "best_ever": 4100000000,
            "workers": 2, "shares": 12483,
            "start_time": "2026-07-25T09:12:00Z", "last_seen": "2026-07-31T12:40:31Z"
          }
        ]
      }
    ]
  },
  "warnings": []
}
```

Contract rules:

- `bitcoin` is **absent** when no pool was detected, or when `--no-btc` is set.
- Unknown numbers are `-1` (`model.Unknown`), never `0` and never `null`.
- Hashrates are H/s, always — the suffixed forms ckpool emits are parsed away (FR-011a).
- Durations are nanoseconds (Go's `time.Duration` encoding), matching `nodes[].uptime`.
- Addresses appear **whole** in `json` (it is a data contract), and truncated only in the
  TUI/stream views (FR-020).

## `--output stream`

The Bitcoin block follows the Monero block, before the trailing rule. Plain text, no ANSI,
no cursor control:

```text
── minepulse 12:40:42 ──
cluster: 2/2 mining · 617 H/s · shares 42✓/0✗ · miner CPU 5000m · node free 12%
  NODE       STATE    HASH/60s  ...
pool: 662 H/s reported · due 0.003120 XMR · paid 0.184000 XMR · last share 4s ago
bitcoin: ckpool (logs) · bitcoin/mining-pool-7c9f8b6d4-x2k9p · Running 5d
  480 TH/s (1m) · 1 users · 2 workers (0 idle) · 12483✓/3✗ · best 1.20 G · 0.02% of net diff
  MINER                  HASH/1m   WORKERS   SHARES   BEST      LAST SHARE
  bc1qexampleaddress…    480 TH/s        2    12483   1.20 G    11s ago
  ! per-device detail unavailable from ckpool logs
────────────────────────────────────────
```

When no pool is detected the block is one line — `bitcoin: no pool detected` — so scripts
can distinguish "absent" from "broken" without parsing JSON.

## Exit codes

Unchanged. A Bitcoin source that cannot be read is a warning, never a non-zero exit for
`watch`/`snapshot`. `doctor` follows its existing rule: Bitcoin checks WARN (they do not
FAIL the exit code) unless the operator explicitly pointed at a pool with `--btc-namespace`
and it could not be read (FR-018).
