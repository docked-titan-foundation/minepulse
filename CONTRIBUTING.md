# Contributing to minepulse

## Requirements

- [mise](https://mise.jdx.dev) — pins the toolchain (Go, golangci-lint, goreleaser, cosign, syft)
- Docker only if you build the container image locally

## Setup

```bash
git clone https://github.com/docked-titan-foundation/minepulse.git
cd minepulse
mise install
mise run build   # -> bin/minepulse
mise run test    # go test ./...
mise run lint    # golangci-lint + go vet
mise run run     # the mock TUI (no cluster)
```

## Spec-driven (Spec Kit)

This repo follows [Spec Kit](https://github.com/github/spec-kit) and its
[constitution](.specify/memory/constitution.md). Non-trivial change flows through the
skills: `/speckit-specify` → `/speckit-plan` → `/speckit-tasks` → `/speckit-implement`.
Every spec/plan/tasks artifact must comply with the constitution or amend it (with a
Sync Impact Report and a version bump).

## Test-first for pure logic (NON-NEGOTIABLE)

Parsers (`internal/xmrig`, `internal/pool`, `internal/collect/logparse.go`) and
aggregation (`internal/model`) are covered by fixture-based `go test` before or
alongside the implementation. Fixtures are recorded payloads under `testdata/`. The TUI
is exercised through `--mock` and a `View()` render test.

## Presentation standard (before you write a renderer)

Every panel on every tab follows one grammar — flush left, a bold role label plus
` · `-separated metrics, identity dimmed beneath, upper-case column heads that name their
averaging window, one share format, one mark for "unavailable". The rules are P1–P12 in
[`specs/004-tab-presentation-standard/plan.md`](specs/004-tab-presentation-standard/plan.md),
and they are enforced by `internal/render/standard_test.go`.

Format values through the shared helpers in `internal/render/format.go` (`Hashrate`,
`Shares`, `Difficulty`, `Dur`, `Ago`, `ShortAddress`) rather than building the strings in
a panel: that is what keeps a new panel consistent by default. The Monero tab is the
reference and is pinned byte-for-byte by a golden test — changing it is a deliberate change
with its own spec, not a side effect.

## Conventional Commits

Messages follow [Conventional Commits](https://www.conventionalcommits.org); the
release scope is `mp`:

- `feat(mp): ...` → minor · `fix(mp): ...` / `perf(mp): ...` → patch
- `feat(mp)!:` or a `BREAKING CHANGE:` footer → major
- `docs:`, `ci:`, `chore(deps):`, `test:` → no release

Other allowed scopes: `ci`, `docs`, `deps`, `test`, `build`, `security`, `release`, `repo`.

## Pull requests

- Target the `beta` branch.
- `mise run lint` and `mise run test` pass; `gofmt` clean.
- Read-only invariant holds: no write verbs, no `exec`, no resource mutation
  (Constitution II).
