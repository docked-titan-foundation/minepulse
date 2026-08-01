# Quickstart: validating the Bitcoin tab

How to prove this feature works, from "no cluster at all" to "against your own pool". Each
scenario names what you should see and which requirement it exercises.

## Prerequisites

```bash
mise install          # pinned Go toolchain
mise run build        # -> bin/minepulse
mise run test         # go test ./... — parsers must be green before anything else
```

No cluster is needed for scenarios 1–3.

## 1. Both tabs, no cluster (FR-014, SC-005)

```bash
./bin/minepulse watch --mock
```

Expect:

- The framed box with the title and update time above it, and a tab strip showing
  `[monero] bitcoin`.
- `b` → the Bitcoin tab: two synthetic pools (one public-pool with per-device rows, one
  ckpool with per-address rows), each naming its stats source.
- `m` → the Monero tab, identical to what it showed before switching.
- `tab` toggles; `p` pauses both coins; `q` quits.
- Switching tabs does not change the `updated HH:MM:SS` stamp — no gather is triggered
  (FR-004).

## 2. Machine output carries both coins (FR-015, US4)

```bash
./bin/minepulse snapshot --mock -o json | jq '{monero: .cluster, btc: .bitcoin.pools[].impl}'
./bin/minepulse snapshot --mock -o stream
```

Expect: a `bitcoin` object alongside the untouched `cluster`/`nodes`/`pool` keys; hashrates
as plain H/s numbers; unknown values as `-1`, never `0`. In `stream`, a `bitcoin:` block
after the pool line.

Regression check that existing consumers are unaffected:

```bash
./bin/minepulse snapshot --mock -o json | jq 'del(.bitcoin)' > /tmp/new.json
# compare against a pre-feature snapshot of the same mock seed — must be identical
```

## 3. No pool is not an error (SC-004)

```bash
./bin/minepulse snapshot --mock --no-btc -o json | jq 'has("bitcoin")'   # false
```

Expect `false` — the key is absent, exactly as when no pool exists, because `--no-btc`
means "do not report on Bitcoin". In the TUI the tab reads `no Bitcoin pool detected`, with
a line naming `--no-btc` as one reason you might be seeing it; in `stream` it is the single
line `bitcoin: no pool detected`. A pool that exists but cannot be read is the distinct
case, and it names itself (scenario 5).

## 4. Against a real cluster, zero configuration (US2, SC-001)

```bash
./bin/minepulse watch            # no --btc-* flags at all
```

Expect within one interval:

- **public-pool**: the tab names it, shows pool totals, block height, and — with
  `--btc-address bc1…` — one row per device with its hashrate and best difficulty.
- **ckpool**: the tab names it, shows hashrate (1m), users/workers/idle, accepted/rejected,
  best share, and one row per payout address, with the note that per-device detail is not
  available from its logs.
- Both: `stats_source` visible in the panel (`api` vs `logs`).

Narrow or point the search when you prefer:

```bash
./bin/minepulse watch --btc-namespace bitcoin --btc-address bc1qexample…
```

## 5. Degradation paths (FR-012, SC-004)

| Scenario | How to produce it | Expected |
|---|---|---|
| Pool not Running | Scale the pool deployment to a bad image / catch it during Init | Panel shows the pod's phase and reason, no stats, no error |
| ckpool just started | Restart the pool pod and look within the first status cycle (~20 s) | "has not published a status cycle yet" — not a zero hashrate |
| Stats source dies mid-session | Deny `pods/proxy` or stop the API container | Last known values, marked stale with their age, plus a warning line |
| No permission to search cluster-wide | Run with a token bound only to the miner namespace | Detection narrows to readable namespaces and says so; snapshot still succeeds |
| Narrow terminal | Resize below the table width | The box stops stretching and the table does not wrap into unreadable rows |

## 6. Doctor reports the Bitcoin path (FR-018)

```bash
./bin/minepulse doctor
./bin/minepulse doctor -o json | jq '.checks[] | select(.name | test("bitcoin"))'
```

Expect a check naming what detection found (implementation, namespace/pod, stats source), a
WARN with an actionable remedy when a pool is present but unreadable, and INFO — not FAIL —
when no pool exists at all.

## 7. Re-record fixtures against your own pool (Constitution IV follow-up)

The shipped fixtures are derived from upstream's emitters (see research R7). Once you have a
live pool, replace them:

```bash
# ckpool — capture a couple of status cycles
kubectl logs -n <ns> deploy/mining-pool --tail=400 | grep -E 'Pool:|User ' \
  > internal/btc/testdata/ckpool_status.log

# public-pool — capture the two endpoints the collector reads
kubectl exec -n <ns> deploy/mining-pool -- true 2>/dev/null || true   # not needed; use the proxy:
kubectl get --raw "/api/v1/namespaces/<ns>/pods/<pod>:3334/proxy/api/pool" \
  > internal/btc/testdata/publicpool_pool.json
kubectl get --raw "/api/v1/namespaces/<ns>/pods/<pod>:3334/proxy/api/client/<address>" \
  > internal/btc/testdata/publicpool_client.json
```

Then `go test ./internal/btc/...` must stay green with no parser change — that is the proof
the derived shapes were right. Scrub the payout address from any fixture you commit.
