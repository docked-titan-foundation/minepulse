# Tasks: Cluster Mining Scope

**Feature**: `001-cluster-mining-scope` | **Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

Conventions: `[P]` = parallelizable (different files, no ordering dep). `[USn]` = serves User
Story n. Test-first for pure logic (Constitution IV): parser/aggregation tests precede their
implementation.

## Phase 0 — Setup

- **T001** Install Go via `mise` (`.mise.toml` pinning go 1.23 + golangci-lint, goreleaser, cosign, syft); `go mod init github.com/docked-titan-foundation/minepulse`.
- **T002** [P] Add deps: cobra, bubbletea, lipgloss, bubbles, client-go, k8s.io/metrics.
- **T003** [P] `.golangci.yaml`, `.gitignore`, `Makefile`/mise tasks (`build`, `test`, `lint`, `run`).

## Phase 1 — Model (pure, foundational)

- **T010** [P] `internal/model/snapshot.go`: `Snapshot`, `ClusterSummary`, `NodeStatus`, `MiningStats`, `CPUSample`, `PoolStats`, `StatsSource` per data-model.md (JSON tags = the `json` output contract).
- **T011** [P] `internal/model/ring.go`: fixed-capacity ring buffer + `Slice()`.
- **T012** Test: `ring_test.go` (wrap-around, ordering) and `snapshot_test.go` (aggregation math: totals, free-CPU %, donate rollup) — **written before T013 fills aggregation**.
- **T013** `internal/collect/collector.go`: `Collector` interface + `Aggregator` fan-in (errgroup) producing a `Snapshot`; make T012 pass.

## Phase 2 — CLI skeleton + mock + early renderers (US1 first light)

- **T020** `cmd/root.go`: cobra root, global flags (`-n/--namespace`, `--selector`, `--interval`, `--output`, `--kubeconfig`, `--context`, `--wallet`, `--no-pool`, `--xmrig-api`, `--mock`), TTY detection → default output.
- **T021** `cmd/watch.go` + `cmd/snapshot.go`: ticker loop vs one-shot; wire Aggregator → renderer.
- **T022** [P] [US1] `internal/collect/mock.go`: synthetic multi-node source incl. a donate-fallback node and a crashloop node; drives every field.
- **T023** [P] `internal/render/stream.go` + `internal/render/json.go`: compact text block; jsonl. (Fast first output; used by tests + background/agent.)
- **T024** [US1] `go test` for stream/json determinism against a fixed mock `Snapshot`.

## Phase 3 — Collectors (real data) + fixtures [US1, US2, US3]

- **T030** [P] `internal/xmrig/{summary,backends}.go`: typed responses + parse funcs. **Fixtures first**: `testdata/summary.json`, `backends.json` (recorded) + `*_test.go` asserting hashrate/threads/shares/pool/worker/version.
- **T031** [P] `internal/pool/supportxmr.go`: typed `stats` response + parse (atomic→XMR). Fixture `testdata/stats.json` + test. [US3]
- **T032** [P] `internal/collect/logparse.go`: regex `speed`, `accepted`, `READY threads`, `POOL #1`, donate detection. Fixture `testdata/xmrig.log` + test. [US1]
- **T033** [US1] `internal/collect/kube.go`: client-go pods-by-label → node/phase/restarts/image/uptime/worker(from `-u`/`-o` args); node allocatable CPU. (read verbs only)
- **T034** [US1] `internal/collect/xmrig.go`: reach pods via API-server pod proxy (`/1/summary`,`/2/backends`); `--xmrig-api auto` falls back to logparse. Donate-fallback flag (R5).
- **T035** [US2] `internal/collect/metrics.go`: metrics.k8s.io pod+node CPU → `CPUSample`; nil + warning when metrics absent.
- **T036** [US3] `internal/collect/pool.go`: fetch SupportXMR stats; wallet auto-detect from miner args; `Stale`+`AsOf` on failure; `--no-pool` skips.
- **T037** Aggregator wiring: history ring append per node; graceful-degrade behavior test (each source individually forced unavailable → snapshot still valid). [US1/US2/US3]

## Phase 4 — TUI [US1, US2]

- **T040** `internal/render/tui.go`: Bubble Tea model (`tickMsg` timer + async collect cmd), Lip Gloss layout — header (cluster totals + aggregate free-CPU gauge), node table, pool panel.
- **T041** [US2] Per-node CPU/free sparkline from the ring buffer; donate-fallback + reduced-fidelity badges; sort + `q`/`p`(pause)/`?` keys.
- **T042** Manual/`--mock` verification pass (SC-001/002/003/005 via mock scenarios).

## Phase 5 — Packaging & docs

- **T050** [P] `deploy/rbac.yaml`: least-privilege Role/ClusterRole (`pods` get/list/watch, `pods/log`, `pods/proxy`, `metrics.k8s.io` pods+nodes) + example ServiceAccount + RoleBinding.
- **T051** [P] `docs/images/minepulse-banner.svg` (Monero-palette ANSI banner) + `README.md` (quickstart, dynamic-yield demo), `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`, `LICENSE` (GPL-3).
- **T052** Org scaffolding + CI: `.goreleaser.yaml` (multi-arch binary + image + SBOM), `.releaserc`+`package.json` (scope `mp`), `.commitlintrc.json`, `renovate.json`, `.github/workflows/{pipeline,pr-pipeline,rebuild,renovate}.yml` (lint→test→build→scan→publish-by-digest→SBOM→attest→cosign→release). (May fast-follow once the binary works.)

## Verification (maps to spec Success Criteria)

- `go test ./...` green (T012, T024, T030–T032, T037) — Constitution IV gate.
- `golangci-lint run` + `go vet ./...` clean.
- `minepulse watch --mock` renders full UI offline (SC-005).
- Against a cluster: per-node status < ~5s (SC-001); `cpu-hog` demo shows dip/recover (SC-002); donate-fallback node flagged (SC-003); survives a source outage without crashing (SC-004).

## Dependencies / order

Phase 0 → 1 → 2 gives a runnable mock UI (US1 MVP). Phase 3 adds real data; T030/T031/T032
are `[P]` (independent files/fixtures). Phase 4 needs Phase 1 (ring) + Phase 3 (data). Phase 5
is independent of runtime and can proceed in parallel once the module exists (T001).
