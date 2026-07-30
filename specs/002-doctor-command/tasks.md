# Tasks: Doctor command

**Feature**: `002-doctor-command` | **Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

- **T001** [US1] `internal/collect/doctor.go`: `CheckStatus`, `Check`, `Report` (+`Add`,`Worst`), and the **pure** `apiReachabilityCheck(running, reachable int) Check`.
- **T002** [US1] Test-first: `internal/collect/doctor_test.go` — table test of `apiReachabilityCheck` (running==0 → info; all reachable → ok; none → warn+remedy; partial → warn) and `Report.Worst()`.
- **T003** [US1/US2] `RunDoctor(ctx, cfg)` in `doctor.go`: kubeconfig → cluster/miners → metrics → per-pod XMRig API → pool; graceful stop on hard prerequisite failure (Constitution III).
- **T004** [US1/US2] `internal/render/doctor.go`: `RenderReport(w, *Report)` (status glyphs + colored + indented remedy); JSON via the existing encoder.
- **T005** [US1] `internal/render/doctor_test.go`: render a synthetic report; assert the disabled-API WARN and the exact `httpApi.enabled: true` remedy appear (SC-001).
- **T006** `cmd/doctor.go`: cobra command; run `RunDoctor`, render per `--output`, exit 1 when `Worst()==fail` (FR-006); register in `cmd/root.go`.
- **T007** Verify: `go test ./...`, `golangci-lint run`, `go vet`, `gofmt`; manual `doctor` run.

Order: T001→T002 (test-first) → T003; T004→T005; T006 wires it; T007 gates.
