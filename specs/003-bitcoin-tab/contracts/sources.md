# Contract: external data sources (Bitcoin)

What minepulse consumes from each pool implementation. These are **observed** contracts —
upstream owns them — so each is pinned to the code that emits it and covered by a fixture
(Constitution IV). If upstream changes a shape, the parser degrades to `Unknown` + a warning
rather than mis-reporting (FR-012).

## S1 — public-pool (HTTP, via the API server's pod proxy)

Reached as `GET /api/v1/namespaces/{ns}/pods/{pod}:{apiPort}/proxy/api/{path}` — read-only,
no port-forward, no Service required. `apiPort` = the container port named `api`, else 3334.

### `GET /api/client/{address}` — live, uncached (primary when `--btc-address` is set)

```json
{
  "bestDifficulty": 4100000000,
  "workersCount": 2,
  "workers": [
    {
      "sessionId": "a1b2c3d4",
      "name": "bitaxe-01",
      "bestDifficulty": "1200000000.00",
      "hashRate": 480000000000,
      "startTime": "2026-07-25T09:12:00.000Z",
      "lastSeen": "2026-07-31T12:40:31.000Z"
    }
  ]
}
```

- `bestDifficulty` (top level) may be **absent/null** — the address has no settings row yet.
- `workers[].bestDifficulty` is a **string** with 2 decimals; `hashRate` is a number in H/s.
- `startTime`/`lastSeen` are ISO-8601.
- Mapping: → `DetailLevel = device`, one `BitcoinMiner` per entry, `Stats.Hashrate1m` = Σ
  `hashRate`, `Stats.Workers` = `workersCount`, `Stats.BestShare` = `bestDifficulty`.

### `GET /api/pool` — cached 5 minutes upstream

```json
{ "totalHashRate": 480000000000, "blockHeight": 907213, "totalMiners": 2, "blocksFound": [], "fee": 0 }
```

- Mapping: → `Stats.BlockHeight`, `Stats.BlocksFound` = `len(blocksFound)`, and — only when
  no address is configured — `Stats.Hashrate1m` = `totalHashRate`, `Stats.Workers` =
  `totalMiners`, with `Stats.TotalsAsOf` set so the view can mark the ≤5-minute cache.

### `GET /api/info` — cached 1 minute

Used only for liveness/uptime (`uptime`) and, when present, `blockData` length. It carries
no payout addresses (`highScores` deliberately omits the address column), so it is never a
discovery source.

### Failure modes

| Condition | Behaviour |
|---|---|
| Non-200 or unparseable body | `Source = api`, `Stats` = last known + `Stale`, warning added |
| Address unknown to the pool (`workers: []`, `bestDifficulty` null) | `DetailLevel = totals`, note: "no workers for this address" |
| Proxy forbidden (RBAC) | `Source = none`, note names the missing permission |

## S2 — ckpool (log records, via `pods/log`)

**Precondition — read this before expecting data.** ckpool logs these records at NOTICE, and
its `logmsg()` sends NOTICE to its **logfile only**; the container's log stream receives
warnings and errors alone (`ckpool.c:105-149`, `ckpool.c:1672`; research R1a). A stock
deployment therefore yields **no** records here, and the collector reports the pool with
`Source = none` plus a remedy. The records appear once the operator routes ckpool's logfile
into the pod's log stream (sidecar or wrapper running `tail -F
/var/lib/ckpool/logs/ckpool.log`). Everything below describes what the parser consumes when
that is in place.

Tail the pool container's log and take the **most recent** occurrence of each record. All
four appear once per status cycle (~20 s).

### Record 1 — pool counters

```text
[2026-07-31 12:40:38.421] ckpool: Pool:{"runtime": 512000, "lastupdate": 1785500438, "Users": 1, "Workers": 2, "Idle": 0, "Disconnected": 0}
```

### Record 2 — pool hashrates (suffixed strings)

```text
[2026-07-31 12:40:38.421] ckpool: Pool:{"hashrate1m": "480T", "hashrate5m": "472T", "hashrate15m": "470T", "hashrate1hr": "468T", "hashrate6hr": "465T", "hashrate1d": "460T", "hashrate7d": "455T"}
```

### Record 3 — shares

```text
[2026-07-31 12:40:38.421] ckpool: Pool:{"diff": 0.02, "accepted": 12483.0, "rejected": 3.0, "bestshare": 1200000000, "SPS1m": 1.4, "SPS5m": 1.3, "SPS15m": 1.3, "SPS1h": 1.2}
```

### Record 4 — one per payout address

```text
[2026-07-31 12:40:38.421] ckpool: User bc1qexample…:{"hashrate1m": "480T", "hashrate5m": "472T", "hashrate1hr": "468T", "hashrate1d": "460T", "hashrate7d": "455T", "lastshare": 1785500431, "workers": 2, "shares": 12483, "bestshare": 1200000000, "bestever": 4100000000, "authorised": 1785000720}
```

Parsing rules:

- Match on the marker (`Pool:` / `User <name>:`) and decode from the first `{` — the log
  prefix (timestamp, process name) is not part of the contract and must be tolerated in any
  form, including none.
- The three `Pool:` records are told apart by their **keys**, not their order.
- Hashrate strings follow `suffix_string()` with `sigdigits = 0`: `%.3g` + `K|M|G|T|P|E`
  above 1000, a bare integer with no suffix below it, and `1e+03T`-style exponents at decade
  edges (research R2).
- Unix seconds (`lastupdate`, `lastshare`, `authorised`) are `0` when unset → zero `time.Time`.
- Mapping: → `Source = logs`, `DetailLevel = address` (record 4 present) or `totals`,
  `Stats.HashrateWindow = "1m"`.

### Failure modes

| Condition | Behaviour |
|---|---|
| No `Pool:` record and the log stream carries other ckpool output (stock deployment) | `Source = none`, `Stats = nil`, note: "ckpool logs stats to its logfile, not stdout — tail it into the pod log to see hashrate"; identity/state/uptime still shown |
| No `Pool:` record and the log is empty (pool just started) | `Stats = nil`, note: "ckpool has not published a status cycle yet" |
| Log read forbidden/unavailable | `Source = none`, warning names it |
| Record present but a value unparseable | That field only becomes `Unknown`; the rest of the record still lands |

## S3 — Workload fingerprint (Kubernetes)

| Evidence | public-pool | ckpool |
|---|---|---|
| Container image reference contains | `public-pool` | `ckpool` |
| Container port named | `api` (+ `stratum`) | `stratum` only |
| Chart labels (corroboration) | `app.kubernetes.io/part-of: mining-pool` | same |

Image is the discriminator; ports and labels corroborate. A pool-shaped workload matching no
image fingerprint is reported as `impl: unknown` with a warning — never guessed (FR-006).
Reads used: `pods` list/get, `pods/log` get, `pods/proxy` get. Nothing else, ever
(Constitution II).
