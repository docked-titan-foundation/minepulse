# Feature Specification: Doctor command

**Feature Branch**: `002-doctor-command`

**Created**: 2026-07-30

**Status**: Draft

**Input**: User description: "Add a doctor command that checks if the XMRig API is enabled; if not, show a warning and recommend activating it."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Diagnose why mining stats are missing (Priority: P1)

An operator sees `0/2 mining · 0 H/s` in `watch` even though the miners are clearly
using CPU. They run `minepulse doctor` and get a short checklist that tells them the
XMRig HTTP API is not reachable on the miners, explains that minepulse is therefore
falling back to log parsing, and recommends enabling `httpApi.enabled: true` on the
`monero-idle-miner` chart.

**Why this priority**: This is the exact confusion the command exists to remove — the
gap between "miners are running" and "minepulse can't read their stats". It is the whole
feature.

**Independent Test**: Run `minepulse doctor` against a cluster whose miner has the HTTP
API disabled; confirm a WARN line names the disabled API and prints the remediation.

**Acceptance Scenarios**:

1. **Given** miner pods with the XMRig HTTP API disabled, **When** the operator runs `doctor`, **Then** it reports the API as unreachable with a WARN and recommends setting `httpApi.enabled: true`.
2. **Given** miner pods with the HTTP API enabled and reachable, **When** the operator runs `doctor`, **Then** the API check passes (OK) with no warning.
3. **Given** the API is reachable on some pods but not others, **When** the operator runs `doctor`, **Then** it reports the partial count and still recommends enabling it everywhere.

### User Story 2 - General preflight (Priority: P2)

The same command surfaces the other things that must be true for `watch` to work:
cluster reachability, that miner pods are found, and whether CPU metrics and the pool
API are available — each with a one-line remedy when not OK.

**Why this priority**: A "doctor" that only checks one thing is surprising; the adjacent
checks reuse the existing collectors and make the command a real preflight. Secondary to
the API check.

**Independent Test**: Run `doctor` with no cluster access → the connectivity check fails
first with a clear remedy; run against a healthy cluster → all checks OK.

**Acceptance Scenarios**:

1. **Given** no/invalid kubeconfig, **When** `doctor` runs, **Then** the connectivity check fails with a remedy and later checks are skipped gracefully.
2. **Given** the selector matches no pods, **When** `doctor` runs, **Then** a WARN names the selector/namespace and suggests checking them.
3. **Given** metrics-server is absent, **When** `doctor` runs, **Then** the CPU-metrics check WARNs (not fails), consistent with `watch` degrading.

### Edge Cases

- No miner pods running yet → API check is reported as not-applicable (info), not a false failure.
- `--no-pool` → the pool check is skipped, not failed.
- Non-interactive / scripted use → `doctor` supports machine-readable output and a meaningful exit code.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `minepulse doctor` MUST perform a bounded set of read-only checks and print a per-check result (OK / WARN / FAIL / INFO) with a short detail.
- **FR-002**: It MUST check XMRig HTTP API reachability across the running miner pods and, when not reachable on all of them, emit a WARN whose remedy is to set `httpApi.enabled: true` on the miner chart.
- **FR-003**: It MUST also check: cluster reachability (+ miner discovery), CPU metrics availability, and — unless `--no-pool` — pool API reachability.
- **FR-004**: A failed prerequisite (e.g. no cluster access) MUST stop dependent checks cleanly rather than crash (Constitution III).
- **FR-005**: It MUST be strictly read-only (Constitution II) — the checks only list pods and issue read/proxy/metrics/HTTP GETs.
- **FR-006**: It MUST offer machine-readable output and exit non-zero when any check FAILs (WARN alone does not fail the exit code).
- **FR-007**: Remediation text MUST be actionable and specific (the exact value to set, where).

### Key Entities *(include if feature involves data)*

- **Check**: name, status (ok/warn/fail/info), detail, and an optional remedy shown when not OK.
- **Report**: the ordered list of Checks plus an overall worst-status used for the exit code.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For the disabled-API case, `doctor` output names the XMRig HTTP API and contains the exact `httpApi.enabled: true` recommendation.
- **SC-002**: `doctor` exits 0 on a fully healthy cluster and non-zero when a hard prerequisite fails.
- **SC-003**: `doctor` completes in one bounded pass (no continuous loop) and touches each miner at most once.

## Assumptions

- Reuses the same read-only access and collectors as `watch` (`deploy/rbac.yaml`).
- "API enabled" is inferred operationally: the XMRig HTTP API endpoint answering via the
  pod proxy means enabled; a connection refusal means disabled/unreachable.
- v1 targets the `monero-idle-miner` chart's `httpApi.enabled` value for the remedy text.
