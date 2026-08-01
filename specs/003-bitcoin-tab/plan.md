# Implementation Plan: Bitcoin tab

**Branch**: `003-bitcoin-tab` | **Date**: 2026-07-31 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/003-bitcoin-tab/spec.md`

## Summary

Turn the framed dashboard into two tabs — Monero (today's view, unchanged) and Bitcoin —
switched with `m`/`b`, and feed the Bitcoin tab from whichever solo pool is running in the
cluster, discovered without configuration. public-pool is read over HTTP through the API
server's pod proxy (the same mechanism the XMRig collector already uses); ckpool, which has
no reachable API, is read from the status records it logs every cycle. Both normalise into
one plain-data model that the TUI, `stream`, and `json` all render, and every source
degrades to a stated "unavailable"/stale panel rather than a failed snapshot.

## Technical Context

**Language/Version**: Go 1.23 (`go.mod`), toolchain pinned by `mise`

**Primary Dependencies**: existing only — `k8s.io/client-go` (pods list/log/proxy),
`charmbracelet/bubbletea` + `lipgloss` (TUI), `spf13/cobra` (CLI), stdlib `encoding/json`.
**No new module dependencies** (Constitution VI: pinned, Renovate-managed, minimal)

**Storage**: N/A — snapshots are in-memory; per-node CPU history rings are session state

**Testing**: `go test ./...` with table + fixture tests under `internal/*/testdata/`;
`--mock` renders the whole UI with no cluster (Constitution III/IV)

**Target Platform**: Linux/macOS terminals; static multi-arch binary

**Project Type**: Single Go CLI (`cmd/` + `internal/`)

**Performance Goals**: a full gather (both coins) completes well inside the refresh interval
(default 3 s); Bitcoin sources are bounded at min(interval, 5 s) each

**Constraints**: strictly read-only cluster access (list/get pods, `pods/log`, `pods/proxy`,
metrics); no exec, no port-forward, no writes; no outbound network beyond the API server and
the configured Monero pool API; additive JSON contract

**Scale/Scope**: homelab clusters — a handful of nodes, 1–2 pool workloads, tens of workers.
~6 new files, ~4 modified, no schema or storage concerns

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Gate | Pre-design | Post-design |
|---|---|---|---|
| I. Spec-Driven & Constitution-Bound | Spec approved under `specs/003-bitcoin-tab/` before code | ✅ spec.md + requirements checklist written first | ✅ plan/research/data-model/contracts complete; no code yet |
| II. Observe, Never Mutate | Only `get`/`list`/`watch`, `pods/log`, `pods/proxy`, metrics reads | ✅ intended reads only | ✅ **Design consequence**: ckpool's socket API and status files are unreachable without `exec`, so the design uses its logs instead (research R1). RBAC grows read-only cross-namespace pod reads |
| III. Degrade Gracefully / No Cluster | Every source has a defined fallback; `--mock` renders everything | ✅ | ✅ Four-state pool lifecycle (data-model), per-source failure tables (contracts/sources.md), `--mock` populates both tabs |
| IV. Test-First for Pure Logic | Parsers/aggregation fixture-tested before/with implementation | ✅ | ⚠️ Fixtures are derived from upstream emitters rather than captured from a live pool — see Complexity Tracking |
| V. CLI Contract: Text/JSON In-Out | `stream` + `json` stable and composable | ✅ | ✅ Additive `bitcoin` key; Monero fields byte-identical; absent-vs-broken distinguishable in both outputs |
| VI. Single Signed Artifact & Supply Chain | No new deps; pinned toolchain | ✅ | ✅ Zero new modules — stdlib JSON and existing clients only |
| VII. Privacy & Least Data | No new sinks; no secrets; addresses handled carefully | ✅ | ✅ No new outbound hosts (pool traffic goes *through* the API server); payout address truncated in TUI/stream, never logged (VR-5) |

**Gate result**: PASS, with one documented deviation (fixture provenance) tracked below.

## Project Structure

### Documentation (this feature)

```text
specs/003-bitcoin-tab/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 — source & discovery decisions (R1–R7)
├── data-model.md        # Phase 1 — model types, validation rules, state transitions
├── quickstart.md        # Phase 1 — how to validate the feature end to end
├── contracts/
│   ├── cli.md           # Flags, keys, json/stream output contract
│   └── sources.md       # public-pool endpoints, ckpool log records, k8s fingerprint
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 — /speckit-tasks output (NOT created here)
```

### Source Code (repository root)

```text
cmd/
├── root.go              # MODIFIED — --btc-namespace/--btc-selector/--btc-address/--btc-api-port/--no-btc
└── doctor.go            # MODIFIED — surface Bitcoin detection results

internal/
├── config/
│   └── config.go        # MODIFIED — BTCNamespace, BTCSelector, BTCAddress, BTCAPIPort, NoBTC
├── model/
│   ├── snapshot.go      # MODIFIED — Snapshot.Bitcoin; Summarize() untouched for Monero
│   └── bitcoin.go       # NEW — BitcoinView/Pool/Stats/Miner, PoolImpl, DetailLevel
├── btc/                 # NEW — pure parsers, no cluster clients
│   ├── publicpool.go    # /api/client/{address}, /api/pool, /api/info → model
│   ├── ckpool.go        # Pool:{…} and User <addr>:{…} log records → model
│   ├── hashrate.go      # suffix_string() decoding ("480G", "1.23P", "1e+03T", "999")
│   └── testdata/        # Recorded payloads and log excerpts
├── collect/
│   ├── kube.go          # MODIFIED — namespace/port-aware proxy + log reads
│   ├── xmrigproxy.go    # MODIFIED — proxyGet/podLogs gain ns+port params (Monero callers unchanged)
│   ├── btcdetect.go     # NEW — discovery + implementation fingerprinting, session cache
│   ├── btc.go           # NEW — per-pool gather orchestration, timeouts, stale retention
│   ├── cluster.go       # MODIFIED — call the Bitcoin gather; attach Snapshot.Bitcoin
│   ├── doctor.go        # MODIFIED — Bitcoin checks
│   └── mock.go          # MODIFIED — synthetic Bitcoin (one public-pool, one ckpool)
└── render/
    ├── tui.go           # MODIFIED — tab state, m/b/tab keys, tab strip, footer hint
    ├── bitcoin.go       # NEW — the Bitcoin tab body (pool panel + miner table)
    ├── stream.go        # MODIFIED — bitcoin block
    ├── format.go        # MODIFIED — difficulty/short-address helpers
    └── doctor.go        # MODIFIED — render Bitcoin check rows

deploy/
└── rbac.yaml            # MODIFIED — cross-namespace read-only pod/log/proxy ClusterRole

README.md                # MODIFIED — Bitcoin tab, new flags, RBAC note
```

**Structure Decision**: The existing single-binary layout is kept exactly. Parsing lives in a
new pure package `internal/btc` (mirroring `internal/xmrig` and `internal/pool`), cluster
access stays confined to `internal/collect`, and rendering to `internal/render` — so the
test-first rule applies cleanly to the only logic that needs it, and the TUI keeps its
`--mock` escape hatch.

## Design decisions carried into tasks

1. **Detection is evidence-based and cached** — classify by container image, corroborate
   with chart labels and port names; cache per session and re-detect when the cached pod
   disappears (research R5).
2. **`proxyGet`/`podLogs` become namespace- and port-aware.** Today both hardcode the miner
   namespace and port 8080. Monero call sites keep their current behaviour through thin
   wrappers, so the change is additive.
3. **public-pool prefers the live endpoint.** `/api/client/{address}` is uncached and is the
   headline hashrate when an address is configured; `/api/pool` is 5-minute-cached upstream
   and is labelled as totals (research R3).
4. **ckpool reaches per-address, never per-device** — the `workername` array exists only in
   the file it writes, not in its logs (research R1). The view says so rather than implying
   the devices are missing.
4a. **ckpool log parsing is opportunistic.** ckpool sends NOTICE-level records to its logfile,
   not to stderr, so a stock deployment's pod log carries no stats at all (research R1a).
   The collector parses them when present and otherwise reports identity + state + a remedy
   (FR-009a). The parser ships either way — it is what makes the operator's one-line chart
   fix (tail the logfile to stdout) sufficient.
5. **Unknown is -1, absent is absent.** No source failure ever renders as a zero hashrate
   (data-model VR-1/VR-2), because a false zero reads as "mining stopped".
6. **Tabs are pure UI state.** Both coins gather every tick; `m`/`b` only change what is
   drawn (FR-004/FR-005).

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Constitution IV asks for fixtures that are *real recorded payloads*; the ckpool and public-pool fixtures are **derived from the upstream emitters' source** (exact format strings, field sets, and encoding quirks — `stratifier.c:8689-8738`, `libckpool.c:2035-2087`, `app.controller.ts`, `client.controller.ts`) rather than captured from a running pool | No Bitcoin pool is reachable from this development environment, and shipping the parsers untested would be the larger violation | Hand-waving the shapes (rejected — the very thing the principle forbids); deferring the feature until a live capture exists (rejected — the operator runs these pools, so the honest path is derived fixtures now plus a re-record task against their cluster, listed in tasks) |
| Cross-namespace pod discovery widens `deploy/rbac.yaml` from one namespaced Role to an additional read-only ClusterRole | Zero-config discovery (US2/FR-006) cannot know which namespace the pool lives in | Requiring `--btc-namespace` (rejected — US2 is explicitly zero-config; the flag remains available for operators who prefer to narrow it, and detection degrades gracefully when the wider read is denied) |
