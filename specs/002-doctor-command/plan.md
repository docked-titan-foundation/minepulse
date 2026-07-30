# Implementation Plan: Doctor command

**Branch**: `002-doctor-command` | **Date**: 2026-07-30 | **Spec**: [spec.md](./spec.md)

## Summary

Add `minepulse doctor`: a one-pass, read-only preflight that runs a handful of checks and
prints a per-check OK/WARN/FAIL/INFO result with actionable remedies. The headline check is
**XMRig HTTP API reachability** across the miner pods; when it is not reachable, WARN and
recommend `httpApi.enabled: true`. Reuses the existing `kubeClient` collectors.

## Technical Context

Go 1.23; reuses `internal/collect` (kubeClient: `listMiners`, `proxyGet`, `nodeCPUUsed`,
`podLogs`) and `internal/pool`. New pure classification + rendering, unit-tested.

## Constitution Check

- **II. Observe, Never Mutate** — checks are list/get/proxy/metrics/HTTP GETs only. ✅
- **III. Degrade Gracefully** — a failed prerequisite stops dependent checks cleanly; no
  crash. ✅
- **IV. Test-First for pure logic** — the status-classification helpers and the report
  renderer are pure and fixture/table tested. ✅
- **V. CLI Contract** — text by default, `-o json`, exit non-zero only on FAIL. ✅

**Result**: PASS.

## Design

- `internal/collect/doctor.go`
  - `type CheckStatus` (`ok|warn|fail|info`), `type Check{Name,Status,Detail,Remedy}`,
    `type Report{Checks []Check}` with `Add` and `Worst()`.
  - `apiReachabilityCheck(running, reachable int) Check` — **pure**, the P1 classifier:
    running==0 → INFO (n/a); reachable==running → OK; reachable==0 → WARN (+remedy);
    else → WARN partial (+remedy). Remedy names `httpApi.enabled: true`.
  - `RunDoctor(ctx, cfg) (*Report, error)` — orchestrates: kubeconfig → cluster/miners →
    metrics → XMRig API per running pod (`proxyGet(pod,"1/summary")`) → pool. Each step
    appends a Check; a hard failure returns the partial report.
- `internal/render/doctor.go` — `RenderReport(w, *Report)` (glyphs + colored status +
  indented remedy) and reuse `render.JSON`-style for `-o json`.
- `cmd/doctor.go` — cobra command; runs `RunDoctor`, renders per `--output`, exits 1 if
  `Worst()==fail`.

## Tasks

See [tasks.md](./tasks.md).

## Verification

- `go test ./...`: table test for `apiReachabilityCheck` (all cases in FR-002/SC-001) and a
  render test asserting the disabled-API WARN + `httpApi.enabled: true` remedy appear.
- Manual: `minepulse doctor -n monero-idle-miner` before/after enabling the API.
