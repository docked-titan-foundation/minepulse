## Description

<!-- What does this PR do and why? Link the spec under specs/ if applicable. -->

## Type of change

- [ ] `feat(mp)` → minor release
- [ ] `fix(mp)` / `perf(mp)` → patch release
- [ ] `feat(mp)!` / `BREAKING CHANGE` → major release
- [ ] `docs` / `ci` / `chore` / `test` — no release

## Checklist

- [ ] PR targets the `beta` branch
- [ ] Complies with the [constitution](../.specify/memory/constitution.md); spec/plan updated if behaviour changed
- [ ] `mise run lint` and `mise run test` pass; `gofmt` clean
- [ ] Read-only invariant preserved (no write verbs, no exec, no mutation)
- [ ] New parsing/aggregation logic has fixture-based tests
