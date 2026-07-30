<!--
Sync Impact Report
- Version change: (none) → 1.0.0 (initial ratification)
- Bump rationale: First constitution for the minepulse project.
- Added principles:
  - I. Spec-Driven & Constitution-Bound (NON-NEGOTIABLE)
  - II. Observe, Never Mutate (NON-NEGOTIABLE)
  - III. Degrade Gracefully / Runs Without a Cluster
  - IV. Test-First for Pure Logic (NON-NEGOTIABLE)
  - V. CLI Contract: Text/JSON In-Out
  - VI. Single Signed Artifact & Supply Chain
  - VII. Privacy & Least Data
- Added sections: Additional Constraints; Development Workflow & Quality Gates; Governance
- Templates / docs checked:
  - .specify/templates/plan-template.md — ✅ Constitution Check gate references this file
  - .specify/templates/spec-template.md — ✅ no contradictions
  - .specify/templates/tasks-template.md — ✅ test-first ordering compatible
- Follow-up TODOs: none
-->

# minepulse Constitution

This constitution captures the non-negotiable conventions of `minepulse`, a
read-only terminal tool that shows how the cluster's Monero miner
(`monero-idle-miner`) is behaving in real time. It is the source of truth that
every `/speckit-*` spec, plan, and tasks artifact must comply with. A spec that
contradicts a principle here must either be revised or amend this document (see
Governance).

## Core Principles

### I. Spec-Driven & Constitution-Bound (NON-NEGOTIABLE)
No implementation lands without an approved specification under `specs/NNN-*/`.
Work proceeds constitution → spec → plan → tasks → implement. Every spec, plan,
and tasks file MUST comply with this constitution; where they cannot, the
constitution is amended first (with a Sync Impact Report and a version bump), not
silently violated. Code without a governing spec is a defect.

### II. Observe, Never Mutate (NON-NEGOTIABLE)
minepulse is strictly read-only toward the cluster and the miner. It performs
only `get`/`list`/`watch`, `pods/log`, `pods/proxy`, and metrics reads. It MUST
NOT create, update, patch, delete, scale, evict, or exec-into any resource, and
ships RBAC that grants exactly those read verbs and nothing more. It must also
not perturb what it observes: polling is bounded (a configurable interval, never
a tight loop) and places negligible load on the miner and nodes. An observability
tool that changes the system it watches is a defect.

### III. Degrade Gracefully / Runs Without a Cluster
Every data source has a defined fallback and no optional source may crash the
tool: XMRig HTTP API unreachable → fall back to log parsing; metrics-server
absent → show CPU as "unavailable", not an error; pool API down → mark the panel
stale with the last-known value and a timestamp. A `--mock` source MUST render
the entire UI with synthetic data and no cluster access, so the tool is
demonstrable and testable anywhere.

### IV. Test-First for Pure Logic (NON-NEGOTIABLE)
All parsing and aggregation logic — XMRig `/1/summary` and `/2/backends`, log
lines, pool `stats`, and snapshot math — is specified by tests against captured
fixtures, written before or alongside the implementation, and `go test ./...` is
a CI gate. Fixtures are real recorded payloads, not hand-waved shapes. TUI
rendering is exempt from strict TDD but MUST be exercisable via `--mock`.

### V. CLI Contract: Text/JSON In-Out
minepulse is a well-behaved CLI. On a TTY it renders a live TUI; without a TTY it
defaults to a plain-text `stream`. It offers `stream` (human/agent-readable text)
and `json` (one JSON snapshot per line) output modes. Data goes to stdout,
diagnostics to stderr, and exit codes are meaningful. Output modes are stable and
composable so the tool can be scripted and consumed by other programs (and by an
agent watching it in the background).

### VI. Single Signed Artifact & Supply Chain
minepulse ships as a single static, multi-arch Go binary and a container image,
published by digest, SBOM-attested, and cosign-signed — the same supply-chain bar
as the rest of the organization. Dependencies are pinned and Renovate-managed. A
release that is unsigned or unattested is not a release.

### VII. Privacy & Least Data
The Monero wallet address is treated as public but is never shipped to any sink
other than the pool the operator configured. The only outbound network
connections are the configured pool API and the cluster's own API server; there
is no telemetry. minepulse never reads, requires, or logs secrets.

## Additional Constraints

- **Language & libraries**: Go 1.23; `k8s.io/client-go` and `k8s.io/metrics` for
  the cluster; `charmbracelet/{bubbletea,lipgloss,bubbles}` for the TUI;
  `spf13/cobra` for the CLI. No heavyweight framework beyond these.
- **Toolchain**: pinned via `mise` (go, golangci-lint, goreleaser, cosign, syft).
  `golangci-lint run` and `go vet ./...` MUST pass clean.
- **Commits**: Conventional Commits; the release scope is `mp`. PRs target `beta`.
- **License**: GPL-3.0.
- **Scope discipline**: the in-cluster deployment and web UI are explicitly out of
  scope for v1 (roadmap). v1 is a local CLI plus pool-side earnings.

## Development Workflow & Quality Gates

Features flow through the Spec Kit skills: `/speckit-constitution` →
`/speckit-specify` → (`/speckit-clarify`) → `/speckit-plan` → `/speckit-tasks` →
`/speckit-implement`. The plan's Constitution Check gate MUST be re-evaluated
after design. CI gates every change on: `go test ./...`, `golangci-lint run`,
`go vet`, a build of the binary, an image/dependency scan, and — on release — SBOM
attestation and a cosign signature. No merge to a release branch with a red gate.

## Governance

This constitution supersedes other practices for this repository. Amendments
require a versioned edit to this file with a Sync Impact Report header and a
rationale; the version follows semantic versioning:

- **MAJOR** — a principle is removed or redefined in a backward-incompatible way.
- **MINOR** — a new principle or section is added.
- **PATCH** — a clarification or wording refinement with no change in meaning.

All PRs and reviews verify compliance; added complexity must be justified against
these principles or the offending change is revised.

**Version**: 1.0.0 | **Ratified**: 2026-07-30 | **Last Amended**: 2026-07-30
