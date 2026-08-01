# Phase 0 Research: Bitcoin tab

All decisions below were taken against the upstream sources of the two pool
implementations and the sibling `bitcoin-stack` chart that deploys them, not from memory.
File/line references are to the upstream repositories as of 2026-07-31.

## R1 — How ckpool exposes statistics

**Decision**: Read ckpool's periodic status records from the **pod log stream** (`pods/log`),
parsing `Pool:{…}` lines (pool aggregate) and `User <address>:{…}` lines (per payout
address).

**Rationale**: ckpool publishes the same status three ways, and only one is reachable
read-only from outside the container:

| Channel | Evidence | Verdict |
|---|---|---|
| Unix socket command API (`stats`, `poolstats`, `users`, `workers`, `clients`, `getuser`…) | `stratifier.c:4935-5047`, spoken by the `ckpmsg` binary; socket lives in `sockdir`, which the image sets to `/tmp/ckpool` (`Dockerfile` CMD `-B -s /tmp/ckpool -c /config/ckpool.conf`) | ❌ Local AF_UNIX only — no TCP, no HTTP. Reaching it needs `exec`, forbidden by Constitution II |
| Status files `logdir/pool/pool.status`, `logdir/users/<address>` | `stratifier.c:8689`, `8650` | ❌ On the pool's PVC; also needs `exec` |
| Log stream at NOTICE level | `stratifier.c:8707,8721,8738` (three `LOGNOTICE("Pool:%s", …)`) and `8594-8598` → `add_msg_entry` → `notice_msg_entries` → `LOGNOTICE` (`stratifier.c:513-523`) | ⚠️ Emitted, but see R1a — NOTICE does not reach the container's stdout in the stock image |

**Records available from the logs**, per status cycle:

1. `Pool:{"runtime","lastupdate","Users","Workers","Idle","Disconnected"}`
2. `Pool:{"hashrate1m","hashrate5m","hashrate15m","hashrate1hr","hashrate6hr","hashrate1d","hashrate7d"}`
3. `Pool:{"diff","accepted","rejected","bestshare","SPS1m","SPS5m","SPS15m","SPS1h"}`
4. `User <address>:{"hashrate1m","hashrate5m","hashrate1hr","hashrate1d","hashrate7d","lastshare","workers","shares","bestshare","bestever","authorised"}` — one per payout address

**Consequence for the spec**: ckpool reaches the per-address rung of FR-011 but never the
per-device rung — `workername` appears only in the `users/<address>` **file**
(`stratifier.c:8635-8650`), never in a log line.

## R1a — …but ckpool's NOTICE records do not reach `kubectl logs` by default

**Decision**: Treat the ckpool log parser as **opportunistic**. Parse the records when they
are present in the pod's log stream; when they are not, report the pool as detected with
`Source = none` and a remedy, rather than pretending it has no stats or inventing zeros.

**Rationale**: this contradicts the naive reading of R1, and it was verified rather than
assumed. `logmsg()` (`ckpool.c:105-149`) is documented as *"Log everything to the logfile,
but display warnings on the console as well"*, and implements exactly that:

- `ckpool.loglevel = LOG_NOTICE` (`ckpool.c:1672`) — the status records **are** generated.
- The console queue is fed only when `loglevel <= LOG_WARNING` (`ckpool.c:138`). NOTICE (5)
  is above WARNING (4), so the records go **only** to `logdir/<name>.log`
  (`ckpool.c:140-143`, filename built at `ckpool.c:1892-1894`).
- `console_log()` writes to **stderr** (`ckpool.c:58-67`), which is what a container's log
  stream captures — but only warnings and worse arrive there.
- Neither the hardened image (`ckpool/Dockerfile`, ENTRYPOINT `ckpool -B -s /tmp/ckpool -c
  /config/ckpool.conf`) nor the chart (`bitcoin-stack/charts/mining-pool`) tails that
  logfile to stdout or runs a sidecar that would.

There is no loglevel or config setting that redirects NOTICE to the console — the severity
gate is hard-coded, and with no logfile open the record is dropped entirely rather than
falling back to stderr.

**So, for a stock ckpool deployment, minepulse can see that the pool exists, what state it
is in, and how long it has run — but not its hashrate.** The honest remedy, surfaced in the
Bitcoin panel and in `doctor`, is one of:

1. tail ckpool's logfile into the pod's log stream (a sidecar or a wrapper that runs
   `tail -F /var/lib/ckpool/logs/ckpool.log` — a `bitcoin-stack` chart change, which the same
   operator owns);
2. run public-pool, which has a real API;
3. accept identity-only reporting for the ckpool panel.

The parser still ships: it costs little, it is fixture-tested, and it turns option 1 into a
one-line chart change rather than a minepulse release.

**Alternatives considered**: raising `-l/--loglevel` (rejected — the console gate is on
message severity, not the configured level, so it changes nothing); having minepulse read
the logfile (rejected — needs exec); declaring ckpool unsupported (rejected — detection,
state, uptime and the remedy are real value, and the operator can enable the rest).

**Alternatives considered**: an HTTP sidecar wrapping `ckpmsg` (rejected — requires
changing the `bitcoin-stack` chart, i.e. deploying something, which is outside minepulse's
scope and its read-only posture); mounting the pool's PVC (rejected — not available to a
CLI running outside the cluster).

## R2 — ckpool hashrate encoding

**Decision**: Parse magnitude-suffixed strings with a dedicated, fixture-tested function;
normalise everything to H/s as `float64`.

**Rationale**: ckpool emits hashrates through `suffix_string(val, buf, 16, 0)`
(`libckpool.c:2035-2087`). With `sigdigits == 0` the output is `%.3g` + a one-letter
suffix for values ≥ 1000 (`K`, `M`, `G`, `T`, `P`, `E`), and a bare `%d` with **no suffix**
below 1000. So the parser must accept `"0"`, `"999"`, `"1.5K"`, `"480G"`, `"1.23P"`, and —
because `%.3g` switches to exponent form at the top of a decade — `"1e+03T"`.

**Alternatives considered**: treating the strings as opaque display text (rejected —
FR-011a requires one internal unit so both tabs and all outputs agree, and totals cannot be
summed otherwise).

## R3 — public-pool's stats API

**Decision**: Read `GET /api/client/{address}` as the live source when a payout address is
known, and `GET /api/pool` for pool-wide totals; reach both through the API server's **pod
proxy** subresource, exactly as the XMRig collector already does.

**Rationale**: verified against upstream:

| Endpoint | Shape | Cache |
|---|---|---|
| `GET /api/pool` (`app.controller.ts` `pool()`) | `{ totalHashRate, blockHeight, totalMiners, blocksFound[], fee }`; `totalHashRate` is `SUM(client.hashRate)` over user agents (`client.service.ts:156-166`) | **5 minutes** |
| `GET /api/client/{address}` (`client.controller.ts:18-42`) | `{ bestDifficulty, workersCount, workers[{sessionId, name, bestDifficulty (string, 2dp), hashRate, startTime, lastSeen}] }` | none — live |
| `GET /api/info` | `{ blockData, userAgents, highScores, uptime }` | 1 minute |
| `GET /api/network` | bitcoind mining info (`blocks`, …) | none |

The chart puts this API on a ClusterIP service and the container port named `api`
(default 3334, `mining-pool/values.yaml` `pool.publicPool.api.port`; the image's own
healthcheck hits `/api/info` on `$API_PORT`). The pod proxy needs no service and no local
port-forward.

**Consequence**: `/api/pool` is up to 5 minutes stale by construction, so when an address is
known the per-worker sum from `/api/client/{address}` is the live figure and the `/api/pool`
numbers are labelled as pool-wide totals rather than presented as current hashrate.

**Alternatives considered**: `/api/info/chart` for a hashrate series (rejected for v1 —
10-minute cache, and the Monero sparkline's data comes from CPU metrics, not the pool);
reading public-pool's SQLite store (rejected — needs `exec`).

## R4 — Discovering the payout address

**Decision**: `--btc-address` flag; when absent, public-pool renders pool-wide totals only,
and ckpool renders its per-address rows (whose addresses come free in the log records).

**Rationale**: public-pool exposes no endpoint that enumerates addresses. The one candidate,
`/api/info` `highScores`, selects only `updatedAt, bestDifficulty, bestDifficultyUserAgent`
(`address-settings.service.ts:52-58`) — the address column is deliberately not returned. So
address discovery is impossible for public-pool without operator input, and unnecessary for
ckpool.

**Alternatives considered**: reading the address off the miners (rejected — the Bitaxe/S9
devices are not in the cluster and minepulse never talks to miners directly); parsing the
stratum connection logs of public-pool (rejected for v1 — format is not a stable contract
and log volume is high).

## R5 — Pool discovery and implementation fingerprinting

**Decision**: List pods (all namespaces by default, narrowing on RBAC denial), and classify
by **container image reference** — `public-pool` or `ckpool` — corroborated by the chart's
workload labels and container port names. Cache the result for the session; re-detect when
the cached pod disappears.

**Rationale**: FR-006 requires evidence, not name matching. The `bitcoin-stack` chart names
the workload `mining-pool` regardless of which implementation runs
(`mining-pool/_helpers.tpl` `mining-pool.name`), and sets `app.kubernetes.io/part-of:
mining-pool` plus a `stratum` port on both, and an `api` port on public-pool only. The image
is therefore the discriminator, the labels and port names the corroboration. A workload that
looks like a pool but matches no known implementation is reported as "unrecognised pool"
with a warning rather than guessed at.

**RBAC consequence**: cross-namespace discovery needs `pods` list and `pods/log`,
`pods/proxy` get beyond the miner's namespace. `deploy/rbac.yaml` grows a ClusterRole for
those reads (still read verbs only); when the binding is absent, detection falls back to the
namespaces it can read and says so in a warning (FR-012).

**Alternatives considered**: requiring `--btc-namespace` (rejected — US2 is explicitly
zero-config); discovering via Services rather than Pods (rejected — the pod proxy addresses
pods, and ckpool has no API service at all).

## R6 — Where the Bitcoin gather runs in the refresh loop

**Decision**: One `Gather` produces both coins. Bitcoin sources are polled with their own
bounded per-source timeout (min(interval, 5s)), and any failure degrades that pool's panel
only.

**Rationale**: FR-005 (a tab is never staler than the timestamp claims) and FR-017 (a hanging
pool must not stall the loop). The TUI already gathers off the UI goroutine, so no change to
the event loop is needed; the timeout is what keeps a wedged pool from eating the interval.

**Alternatives considered**: gathering Bitcoin only while its tab is visible (rejected —
violates FR-004/FR-005: switching tabs would show stale data and trigger work).

## R7 — Fixture provenance

**Decision**: Fixtures are generated from the upstream emitters' exact format strings and
field sets (cited above), recorded as files under `testdata/`, and replaced with live
captures the first time this runs against the operator's own cluster.

**Rationale**: Constitution IV asks for real recorded payloads. No live pool is reachable
from this development environment, so the honest position is: byte-shapes derived from the
upstream source that produces them (including its quirks — the suffix encoding of R2, the
`%.3g` exponent edge, `bestDifficulty` as a 2-decimal string), with a follow-up to re-record
against the operator's cluster. This is noted in the plan's Complexity Tracking as the one
place the constitution's letter is met by derivation rather than capture.
