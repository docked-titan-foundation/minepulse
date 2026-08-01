# Tasks: Bitcoin tab

**Feature**: `003-bitcoin-tab` | **Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

**Input**: Design documents from `/specs/003-bitcoin-tab/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Required, not optional — Constitution IV makes parsing and aggregation test-first,
and SC-007 restates it. Test tasks precede the parser they cover.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on incomplete work)
- **[Story]**: US1 (two-tab dashboard), US2 (zero-config detection), US3 (per-miner detail),
  US4 (stream/json parity)
- Paths are repo-relative, per the plan's source layout

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Make the new package and its fixtures exist before anything imports them

- [X] T001 [P] Create the pure-parser package skeleton `internal/btc/` with a package doc comment stating it holds no cluster clients (mirrors `internal/xmrig`, `internal/pool`)
- [X] T002 [P] Create `internal/btc/testdata/` and record the derived fixtures per contracts/sources.md: `ckpool_status.log` (all four record types, ≥2 status cycles, one line with an unparseable value), `publicpool_client.json`, `publicpool_pool.json`, `publicpool_client_empty.json` (no workers, null bestDifficulty)

**Checkpoint**: `go build ./...` still green; fixtures on disk for the test-first tasks

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Model types, cluster-access plumbing, and config that every user story needs

**⚠️ CRITICAL**: No user story work starts until this phase is complete

- [X] T003 Add the Bitcoin model types in `internal/model/bitcoin.go` per data-model.md: `BitcoinView`, `BitcoinPool`, `BitcoinStats`, `BitcoinMiner`, `PoolImpl`, `DetailLevel`, with the exact JSON tags from contracts/cli.md
- [X] T004 Add `Bitcoin *BitcoinView \`json:"bitcoin,omitempty"\`` to `Snapshot` in `internal/model/snapshot.go`; leave `Summarize()` untouched so `ClusterSummary` keeps its current meaning (VR-6, FR-015)
- [X] T005 [P] Test-first: `internal/model/bitcoin_test.go` — VR-2/VR-3 invariants (absent numbers are `Unknown` not 0; `Detail=totals` ⟺ empty `Miners`) and that a `Snapshot` with `Bitcoin == nil` marshals without a `bitcoin` key
- [X] T006 Make pod reads namespace- and port-aware in `internal/collect/xmrigproxy.go`: `proxyGetIn(ctx, ns, pod, port, path)` and `podLogsIn(ctx, ns, pod, container, tail)`, keeping `proxyGet`/`podLogs` as thin wrappers so every Monero call site is unchanged
- [X] T007 Add the pool-discovery read to `internal/collect/kube.go`: list pods across all namespaces, plus a narrowing fallback used when the wide list is denied (returns the namespaces actually searched, for `BitcoinView.Scope`)
- [X] T008 [P] Extend `internal/config/config.go` with `BTCNamespace`, `BTCSelector`, `BTCAddress`, `BTCAPIPort`, `NoBTC`, and defaults that mean "auto-detect everything"
- [X] T009 [P] Register the five flags in `cmd/root.go` exactly as specified in contracts/cli.md (`--btc-namespace`, `--btc-selector`, `--btc-address`, `--btc-api-port`, `--no-btc`)

**Checkpoint**: model + plumbing compile; no behaviour change yet in any output

---

## Phase 3: User Story 2 - Zero-config pool detection (Priority: P1) 🎯 MVP foundation

**Goal**: Find the pool(s) in the cluster and read pool-level stats from whichever
implementation is running, without any Bitcoin flags.

**Independent test**: Point minepulse at a cluster running the `bitcoin-stack` mining-pool
chart with `pool.implementation=public-pool`, then with `ckpool`, with no `--btc-*` flags;
each is detected, named, and its pool-level stats appear in `-o json`.

- [X] T010 [P] [US2] Test-first: `internal/btc/hashrate_test.go` — `ParseSuffixed` table per research R2: `"0"`, `"999"`, `"1.5K"`, `"480G"`, `"1.23P"`, `"1e+03T"`, `""` and `"garbage"` → `Unknown`
- [X] T011 [US2] Implement `internal/btc/hashrate.go`: `ParseSuffixed(string) float64` returning H/s, `Unknown` on failure, never 0 for unparseable input (VR-1)
- [X] T012 [P] [US2] Test-first: `internal/btc/ckpool_test.go` against `testdata/ckpool_status.log` — the three `Pool:` records are told apart by keys not order, the newest cycle wins, an unparseable value degrades that field only, and a log with no `Pool:` line yields nil stats (not zeros)
- [X] T013 [US2] Implement `internal/btc/ckpool.go`: `ParseStatusLog([]byte) (*model.BitcoinStats, []model.BitcoinMiner, error)` — tolerate any log prefix, decode from the first `{`, map fields per contracts/sources.md S2, `HashrateWindow = "1m"`
- [X] T014 [P] [US2] Test-first: `internal/btc/publicpool_test.go` against `testdata/publicpool_*.json` — `bestDifficulty` as a 2-decimal string, a null top-level `bestDifficulty`, empty `workers`, and ISO-8601 times
- [X] T015 [US2] Implement `internal/btc/publicpool.go`: `ParseClient([]byte)` and `ParsePool([]byte)` → `BitcoinStats`/`[]BitcoinMiner` per contracts/sources.md S1, setting `TotalsAsOf` on the cached `/api/pool` figures
- [X] T016 [P] [US2] Test-first: `internal/collect/btcdetect_test.go` — pure fingerprint function: image containing `public-pool` → public-pool, `ckpool` → ckpool, pool-shaped-but-unmatched → `unknown` + warning, non-pool pod → not detected
- [X] T017 [US2] Implement `internal/collect/btcdetect.go`: fingerprint by container image, corroborate with chart labels and the `api`/`stratum` port names, resolve the API port (named port → `--btc-api-port` → 3334), session cache invalidated when the cached pod disappears (research R5)
- [X] T018 [US2] Implement `internal/collect/btc.go`: per-pool gather — public-pool via `proxyGetIn`, ckpool via `podLogsIn` (tail 400) — each bounded by min(interval, 5s), retaining last-known stats marked `Stale` on failure, and recording `Scope`/`Note`/warnings (FR-012, R6)
- [X] T018a [US2] Handle the stock-ckpool case in `internal/collect/btc.go` (FR-009a, research R1a): logs with no `Pool:` record ⇒ `Source = none`, stats nil, identity/state/uptime kept, and a note carrying the tail-the-logfile remedy — distinct from the "not published a cycle yet" note
- [X] T019 [US2] Wire it into `internal/collect/cluster.go`: gather Bitcoin after the nodes, attach `snap.Bitcoin`, skip entirely when `--no-btc`, and never let a Bitcoin failure fail the snapshot
- [X] T020 [US2] Add synthetic Bitcoin data to `internal/collect/mock.go`: one public-pool (device rows) and one ckpool (address rows), so both branches render with no cluster (FR-014)

**Checkpoint**: `minepulse snapshot --mock -o json | jq .bitcoin` shows both pools; a real
cluster is detected with zero flags

---

## Phase 4: User Story 1 - The two-tab dashboard (Priority: P1) 🎯 MVP

**Goal**: The framed box becomes two tabs, `m`/`b` switch between them, and the Bitcoin tab
renders what US2 collected.

**Independent test**: `minepulse watch --mock`, press `b` → Bitcoin view with pool identity,
hashrate, workers, shares; press `m` → the Monero view exactly as before; the `updated`
stamp does not change on either press.

- [X] T021 [US1] Add tab state to `internal/render/tui.go`: a `coinTab` field defaulting to Monero, `m`/`b`/`tab` key handling in `Update` that only changes state (no gather, no pause change, no interval change — FR-004)
- [X] T022 [US1] Render the tab strip between the title and the box in `internal/render/tui.go`, marking the active tab, and update the footer hint to `q quit · p pause · r refresh · m/b coin`
- [X] T023 [US1] Implement `internal/render/bitcoin.go`: the Bitcoin tab body — one panel per pool (impl, namespace/pod/node, phase+uptime, stats source, stale age) plus the pool-level line (hashrate with its window named, users/workers/idle, accepted/rejected, best share, network-diff %), rendering `—` for `Unknown` (FR-010, VR-2)
- [X] T024 [US1] Handle the four lifecycle states in `internal/render/bitcoin.go` with distinct text: no pool detected / detected-but-nothing-published / live / stale, plus `--no-btc` disabled (SC-004)
- [X] T025 [P] [US1] Add `Difficulty()` (SI-suffixed, e.g. `1.20 G`) and `ShortAddress()` (truncating, FR-020) to `internal/render/format.go`, with cases in `internal/render/format_test.go`
- [X] T026 [US1] Extend `internal/render/render_test.go`: `View()` on the Bitcoin tab renders pool identity, hashrate and source for both impls; `m`/`b` switch the rendered body; the no-pool view says so

**Checkpoint**: the feature is demonstrable end to end via `--mock` — this is the MVP

---

## Phase 5: User Story 3 - Per-miner detail (Priority: P2)

**Goal**: Show the finest per-miner rung each source reaches, and name what is missing.

**Independent test**: with `--btc-address` against public-pool → one row per device; against
ckpool → one row per payout address plus the per-device note; neither available → totals with
a note.

- [X] T027 [US3] Populate `DetailLevel` and `Miners` in `internal/collect/btc.go`: public-pool + `--btc-address` → `device`; public-pool without an address → `totals` with the reason; ckpool → `address` (FR-011, R4)
- [X] T028 [US3] Render the miner table in `internal/render/bitcoin.go` with columns per granularity — device: name, hashrate, best difficulty, connected-since, last seen; address: short address, hashrate, workers, shares, best share, last share
- [X] T029 [US3] Keep the table inside the box's width contract in `internal/render/bitcoin.go` — same non-wrapping rule the Monero node table honours
- [X] T030 [P] [US3] Extend `internal/render/render_test.go`: a device-detail pool renders one row per worker; an address-detail pool renders the note naming what ckpool cannot provide; a totals-only pool renders no table

**Checkpoint**: all three rungs of FR-011 visible and labelled

---

## Phase 6: User Story 4 - Stream and JSON parity (Priority: P3)

**Goal**: Both machine outputs carry the Bitcoin data, additively.

**Independent test**: `snapshot --mock -o json` has `bitcoin` alongside untouched Monero
keys; `-o stream` prints the Bitcoin block; deleting `.bitcoin` reproduces the pre-feature
record byte for byte.

- [X] T031 [US4] Add the Bitcoin block to `internal/render/stream.go` exactly as contracts/cli.md specifies, including the one-line `bitcoin: no pool detected` form
- [X] T032 [P] [US4] Extend `internal/render/render_test.go`: the stream block appears with both impls, the no-pool line is emitted when `Bitcoin == nil`, and no ANSI escapes leak into stream output
- [X] T033 [US4] Verify the JSON contract in `internal/render/render_test.go`: unknown numbers serialize as `-1`, hashrates are plain H/s, addresses appear whole in JSON, and a `Bitcoin == nil` snapshot has no `bitcoin` key

**Checkpoint**: scripted consumers see the new data; existing ones see no change

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T034 [P] Add Bitcoin checks to `internal/collect/doctor.go` and their rows to `internal/render/doctor.go`: what detection found (impl, namespace/pod, source), WARN + remedy when a pool is present but unreadable, INFO when none exists (FR-018)
- [X] T035 [P] Extend `internal/collect/doctor_test.go` and `internal/render/doctor_test.go` for the new checks, keeping the existing rule that Bitcoin WARNs do not change the exit code
- [X] T036 [P] Widen `deploy/rbac.yaml` with a read-only ClusterRole for cross-namespace `pods` list and `pods/log`, `pods/proxy` get, with a comment tying it to zero-config discovery (plan Complexity Tracking)
- [X] T037 [P] Update `README.md`: the Bitcoin tab, the `m`/`b` keys, the five flags, which stats each implementation can and cannot give, and the RBAC note
- [X] T038 Run the full gate: `mise run test`, `golangci-lint run`, `go vet ./...`, `gofmt -l .`, and a `--mock` run of `watch`, `snapshot -o json`, `snapshot -o stream`, `doctor`
- [X] T039 Walk quickstart.md scenarios 1–3 and 6 (no cluster needed) and correct any drift between the doc and the built binary
- [ ] T040 Re-record `internal/btc/testdata/` from a live pool per quickstart.md §7 and confirm the parsers stay green unchanged — closes the Constitution IV deviation in plan.md (requires cluster access; do last)

---

## Dependencies

**Phase order**: Setup (T001–T002) → Foundational (T003–T009) → US2 (T010–T020) → US1
(T021–T026) → US3 (T027–T030) → US4 (T031–T033) → Polish (T034–T040)

**Story dependencies**:

- **US2 before US1** — the Bitcoin tab has nothing to draw until detection and collection
  exist. This inverts the spec's P1 ordering deliberately: both are P1, and the data side is
  the blocking half.
- **US3 depends on US2** (populates `Miners`/`Detail`) **and US1** (renders the table).
- **US4 depends on US2** only — `stream`/`json` read the model, not the TUI.
- Polish depends on everything; T040 additionally needs a live cluster.

**Within-phase**: test tasks precede the implementation they cover (T010→T011, T012→T013,
T014→T015, T016→T017). T018 needs T011/T013/T015/T017. T019 needs T018. T023 needs T003.

## Parallel execution examples

- **Setup**: T001, T002 together
- **Foundational**: T005, T008, T009 together once T003/T004 land
- **US2 parsers**: T010, T012, T014, T016 (four independent test files) together; then
  T011, T013, T015, T017 together
- **US1**: T025 alongside T021–T024
- **Polish**: T034, T036, T037 together; T035 after T034

## Implementation strategy

**MVP** = Phase 1 → 2 → 3 (US2) → 4 (US1): the box gains a Bitcoin tab that finds the pool
by itself and shows real pool-level numbers, demonstrable offline with `--mock`. Stop there
and the feature is already worth shipping.

**Increment 2** = US3: per-miner rows at whatever granularity the source allows.

**Increment 3** = US4 + Polish: machine-output parity, doctor coverage, RBAC and docs.

Each increment ends at a checkpoint that is independently testable via `--mock`, so no
increment needs a cluster to demonstrate — only T040 does.
