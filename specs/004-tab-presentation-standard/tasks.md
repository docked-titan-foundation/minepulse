# Tasks: Tab presentation standard

**Feature**: `004-tab-presentation-standard` | **Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

**Input**: Design documents from `/specs/004-tab-presentation-standard/`

**Tests**: Required. The pure helper is table-tested, and FR-001 (the Monero tab does not
move) is only credible if a test pins it.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on incomplete work)
- **[Story]**: US1 (both tabs read alike), US2 (Monero unchanged), US3 (standard survives)
- Rule references (P1–P12) are to the standard table in plan.md

---

## Phase 1: Foundational — pin the reference before touching anything

**Purpose**: FR-001 is the constraint every later task works under, so the guard comes first

- [X] T001 [US2] Create `internal/render/standard_test.go` with a byte-equality guard: render the Monero tab body from `sampleSnapshot()` against a captured golden string, so any accidental change to the reference tab fails the build
- [X] T002 [P] [US3] Add `Shares(good, bad float64) string` to `internal/render/format.go` per P6 — `N✓/M✗`, the accepted half alone when rejects are unavailable, `—` when neither is known — with table cases in `internal/render/format_test.go` covering all three

**Checkpoint**: the Monero tab is pinned and the shared share-formatter exists

---

## Phase 2: User Story 1 - Both tabs read alike (Priority: P1) 🎯 MVP

**Goal**: The Bitcoin panel adopts the Monero tab's grammar.

**Independent test**: `minepulse watch --mock`, press `b`, and check each rule P1–P10 holds
against what the Monero tab does for the same concept.

- [X] T003 [US1] Flush the Bitcoin panel left in `internal/render/bitcoin.go` (P1): drop the two-space indent from every sub-line, table head, table row, note and remedy
- [X] T004 [US1] Restructure the panel header in `internal/render/bitcoin.go` (P2/P3): header line is the bold implementation label plus ` · `-separated metrics, most important first; identity, state, uptime and stats source move to a dimmed context line beneath
- [X] T005 [US1] Name the hashrate window in both miner tables in `internal/render/bitcoin.go` (P5): `HASH/1m` for ckpool's averaged figure, `HASH` for public-pool's instantaneous one, driven by the pool's `HashrateWindow`
- [X] T006 [US1] Use `Shares()` for every share figure in `internal/render/bitcoin.go` (P6/P11) — headline and table cell alike — so public-pool's unreported rejects never render as `0✗`
- [X] T007 [US1] Style notes and remedies in `internal/render/bitcoin.go` (P8): `! ` in the warning colour like the Monero tab's warnings, `→ ` dimmed for remedies
- [X] T008 [P] [US1] Update `internal/render/bitcoin_test.go` to assert the standard's shapes rather than the old ones: header carries the metrics, context line carries identity, column heads name the window, shares render `N✓/M✗`

**Checkpoint**: the two tabs are readable as one dashboard

---

## Phase 3: User Story 1 (cont.) - Plain-text output follows too (Priority: P2)

- [X] T009 [US1] Apply P4–P7 and P10 to the Bitcoin block in `internal/render/stream.go`: window-named hashrate heads, `Shares()` in the table cell, truncated identities, `—` for unavailable
- [X] T010 [P] [US1] Update `internal/render/stream_test.go` for the share format and column heads, keeping the existing no-ANSI and address-truncation assertions

**Checkpoint**: `stream` and the TUI agree; `json` untouched

---

## Phase 4: Verification & cross-tab invariants

- [X] T011 [US1] Extend `internal/render/standard_test.go` with the cross-tab invariants a reviewer would check by eye: no Bitcoin panel line starts with an indent (P1), no `n/a` appears on the Bitcoin tab (P7), and every hashrate column head that shows an average names its window (P5)
- [X] T012 [US2] Confirm FR-001 end to end: `go test ./internal/render/` passes with the golden guard from T001 untouched, and `minepulse snapshot --mock -o json` is byte-identical to before the change
- [X] T013 Run the full gate: `gofmt -l .`, `go vet ./...`, `go test ./...`, `golangci-lint run` (Go 1.23 toolchain, matching CI), plus a `--mock` run of `watch` on both tabs and `snapshot -o stream`
- [X] T014 [P] [US3] Link the standard from `CONTRIBUTING.md` so the next panel's author finds it before writing a renderer, not during review

---

## Dependencies

**Phase order**: Foundational (T001–T002) → US1 TUI (T003–T008) → US1 stream (T009–T010) →
Verification (T011–T014)

- **T001 blocks everything**: without the guard, "the Monero tab did not move" is a claim
  rather than a fact.
- **T002 blocks T006 and T009** — both consume `Shares()`.
- T003–T007 all edit `bitcoin.go` and are therefore sequential; T008 follows them.
- T009 depends on T002 only; it may run alongside T003–T008.
- T014 depends on nothing and may run at any point.

## Parallel execution examples

- **Foundational**: T002 alongside T001
- **US1**: T009/T010 (stream) alongside T003–T008 (TUI) — different files
- **Verification**: T014 alongside T011–T013

## Implementation strategy

**MVP** = Phase 1 + Phase 2: the tab the operator actually looks at now reads like its
sibling, with the reference provably unmoved. Phase 3 brings the piped output along; Phase 4
turns the standard from prose into tests that keep it true.
