# Implementation Plan: Cluster Mining Scope

**Branch**: `001-cluster-mining-scope` | **Date**: 2026-07-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-cluster-mining-scope/spec.md`

## Summary

A single-binary Go CLI, `minepulse`, that continuously renders how the
`monero-idle-miner` DaemonSet is behaving across the cluster: per-node hashrate,
threads, shares and pool state; the miner's CPU vs each node's free-CPU over time
(the dynamic idle-CPU story); and pool-side earnings for the wallet. It polls
several read-only sources on an interval, folds them into a `Snapshot`, and renders
a Bubble Tea TUI (on a TTY) or plain-text/JSON snapshots (otherwise). A `--mock`
source runs the whole thing with no cluster.

## Technical Context

**Language/Version**: Go 1.23

**Primary Dependencies**: `k8s.io/client-go`, `k8s.io/metrics` (metrics.k8s.io clientset),
`spf13/cobra`, `charmbracelet/bubbletea` + `lipgloss` + `bubbles`; stdlib `net/http` for
the XMRig and pool APIs.

**Storage**: none (in-memory ring buffer of recent snapshots per node for sparklines).

**Testing**: `go test` with recorded JSON fixtures for all parsers + aggregation; `--mock`
for UI/E2E-without-cluster.

**Target Platform**: Linux/macOS, amd64/arm64; a container image for later in-cluster use.

**Project Type**: Single-project CLI.

**Performance Goals**: Negligible load on the observed cluster (Constitution II) — one
bounded gather per interval (default 3s); first paint within ~5s (SC-001).

**Constraints**: Strictly read-only; no telemetry; must not crash on any single
unavailable source; runnable offline via `--mock`.

**Scale/Scope**: Homelab-to-modest clusters (order tens of mining nodes) rendered readably.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Spec-Driven** — this plan derives from an approved spec; tasks will follow. ✅
- **II. Observe, Never Mutate** — collectors use only list/get/watch, `pods/log`,
  `pods/proxy`, and metrics reads; `deploy/rbac.yaml` grants exactly those verbs. No
  write client is constructed. ✅
- **III. Degrade Gracefully** — each collector returns partial results + an availability
  flag; the aggregator never fails a snapshot because one source is down; `--mock` source
  included. ✅
- **IV. Test-First for Pure Logic** — parsers and aggregation live in pure functions with
  fixture-based tests written first; TUI excluded, driven by `--mock`. ✅
- **V. CLI Contract** — `tui`/`stream`/`json` outputs; stdout data, stderr diagnostics. ✅
- **VI. Single Signed Artifact** — GoReleaser multi-arch binary + image, cosign + SBOM
  (wired in the scaffolding task). ✅
- **VII. Privacy & Least Data** — only egress is the cluster API + configured pool API;
  no telemetry; no secrets read. ✅

**Result**: PASS (no violations; Complexity Tracking empty).

## Project Structure

### Documentation (this feature)

```text
specs/001-cluster-mining-scope/
├── plan.md              # This file
├── research.md          # Decisions: XMRig reach, metrics source, pool API, donate detection
├── data-model.md        # Snapshot / NodeStatus / MiningStats / CPUSample / PoolStats / ring buffer
├── quickstart.md        # How to run (mock + real), and the dynamic-yield demo
├── contracts/           # Consumed external response shapes (XMRig, SupportXMR) + fixtures index
└── tasks.md             # /speckit-tasks output
```

### Source Code (repository root)

```text
minepulse/
├── main.go                      # thin entrypoint → cmd.Execute()
├── cmd/
│   ├── root.go                  # cobra root, global flags, output-mode + TTY detection
│   ├── watch.go                 # continuous loop (default)
│   └── snapshot.go              # one gather → print → exit (--once equivalent)
├── internal/
│   ├── config/config.go         # resolved options (namespace, selector, interval, wallet, ...)
│   ├── collect/
│   │   ├── collector.go         # Collector interface + Aggregator (fan-in → Snapshot)
│   │   ├── kube.go              # client-go: pods by label, node allocatable
│   │   ├── metrics.go          # metrics.k8s.io: pod container mCPU, node CPU usage
│   │   ├── xmrig.go            # XMRig API via pods/proxy (/1/summary, /2/backends)
│   │   ├── logparse.go         # fallback: parse pod logs for hashrate/shares/threads/pool
│   │   ├── pool.go             # SupportXMR /api/miner/{wallet}/stats
│   │   └── mock.go             # synthetic source (no cluster)
│   ├── model/
│   │   ├── snapshot.go         # Snapshot, NodeStatus, MiningStats, CPUSample, PoolStats
│   │   └── ring.go             # bounded per-node history for sparklines
│   ├── xmrig/                   # API client + typed responses (summary.go, backends.go)
│   ├── pool/                    # supportxmr client + typed responses
│   └── render/
│       ├── tui.go              # Bubble Tea model + Lip Gloss layout + sparklines
│       ├── stream.go           # compact text block per tick
│       └── json.go             # one JSON snapshot per line
├── deploy/rbac.yaml             # least-privilege Role/ClusterRole + example ServiceAccount
├── docs/images/minepulse-banner.svg
└── (module + tooling + CI, added in the scaffolding task)

tests live beside code as *_test.go, with fixtures under internal/*/testdata/
```

**Structure Decision**: Single Go project. `cmd/` holds the cobra surface; all logic is in
`internal/` split by concern (collect / model / render) so parsers and aggregation are pure
and unit-testable per Constitution IV. External API types are isolated in `internal/xmrig`
and `internal/pool` so their fixtures pin the contracts we depend on.

## Complexity Tracking

> No constitution violations — section intentionally empty.
