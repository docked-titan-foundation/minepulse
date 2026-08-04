# Implementation Plan: Pool identity line

**Branch**: `007-pool-identity-line` | **Date**: 2026-08-03 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/007-pool-identity-line/spec.md`

## Summary

Open each tab with one line saying where the work goes: `[locality] URL - brand - mode - IP`.
Locality is decided by matching the pool address against Pod IPs and Service ClusterIPs
minepulse can read — not by inspecting address ranges, which cannot tell an in-cluster pool
from the operator's NAS. Brand and mining mode come from a table of known pools for Monero and
from the workload fingerprint for Bitcoin, and neither invents a mode it cannot support.

## Technical Context

**Language/Version**: Go 1.26 (pinned by `mise`)

**Primary Dependencies**: existing only — `client-go` gains Service and cluster-wide Pod
listing. No new modules.

**Testing**: `go test ./...`; brand/mode resolution table-tested with no cluster, locality
classification tested including the degraded path, both tabs re-pinned to new goldens.

**Constraints**: read-only (Constitution II); every new read degrades the line rather than the
snapshot (Constitution III); `json` gains fields but changes none.

**Scale/Scope**: 2 new packages/files, 9 modified, 3 new test files

## Standard impact

None. The identity line is a panel header line in the P2 grammar, its unavailable fields use
P7's mark, and P12 keeps it plain in `stream`. No rule added or amended; the standard stays at
1.1.0.

## Key decisions

**The miner's resolution is authoritative.** XMRig's `/1/summary` already reports the address
it connected to (`connection.ip`), from inside the cluster. minepulse resolving the hostname
itself would answer a different question — it usually runs on a laptop that cannot resolve
`*.svc.cluster.local` at all.

**A match is the claim; a range is the fallback.** `[internal]` means an address matched a
cluster object. When Services cannot be read the line falls back to range inspection and
records that it did, so a weak test is never passed off as a strong one (FR-003/FR-004).

**Divergence is surfaced, not resolved.** With one node on the donate pool the miners no longer
agree, and the line marks it. Silently reporting the majority would hide the exact failure the
DONATE⚠ state exists to catch.

**Mode is not guessed.** `public-pool` is solo by design. `ckpool` ships both ways, so it is
solo only when its image says so — it does not inherit this repo's solo-mining framing.

## Constitution Check

| Principle | Gate | Result |
|---|---|---|
| I. Spec-Driven & Constitution-Bound | Spec approved before code | ✅ spec.md written first |
| II. Observe, Never Mutate | No new write access | ✅ two new list verbs, both reads, added to `deploy/rbac.yaml` |
| III. Degrade Gracefully | Missing source still renders | ✅ unreadable Services degrade the marker to a named fallback; the snapshot is unaffected |
| IV. Test-First for Pure Logic | Pure logic fixture-tested | ✅ `poolid` and `classify` are pure and table-tested |
| V. CLI Contract: Text/JSON In-Out | Output modes stable | ✅ `json` gains fields only; `stream` carries the same line in plain text (P12) |
| VI. Single Signed Artifact | No new deps | ✅ zero |
| VII. Privacy & Least Data | Careful with identifiers | ⚠️ the line prints a pool IP. It is infrastructure the operator already runs or already connects to, never a payout address, and it is what makes the locality claim checkable. Recorded below |

**Gate result**: PASS, with one recorded deviation (pool IP on screen).

## Project Structure

```text
internal/poolid/          # NEW — brand + mining mode from an address, pure
internal/collect/
├── locality.go           # NEW — cluster address set and the classifier
├── endpoint.go           # NEW — endpoint builders for both tabs
├── btcdetect.go          # MODIFIED — capture pod IP, labels, stratum port
├── btc.go                # MODIFIED — endpoint per detected pool
├── cluster.go            # MODIFIED — one cluster-net read per tick, shared
└── mock.go               # MODIFIED — synthetic endpoints through the real chooser
internal/model/           # MODIFIED — Locality, MiningMode, PoolEndpoint, MiningStats.PoolIP
internal/xmrig/summary.go # MODIFIED — parse connection.ip
internal/render/          # MODIFIED — PoolLine, and it on both tabs and stream
deploy/rbac.yaml          # MODIFIED — services: get,list
```

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Constitution VII: the line prints a pool IP | It is the evidence for the locality claim — `[internal]` with no address is unverifiable | Omitting the IP (rejected — the operator asked for it, and it is the field that makes the marker checkable); truncating it (rejected — a half address identifies nothing and hides nothing) |
| New cluster-wide Service and Pod listing | Matching objects is the only honest way to distinguish "in this cluster" from "in a private range" | Range inspection alone (rejected — reports a NAS-hosted pool as internal, a false claim about where hashrate goes) |
