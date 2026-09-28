# Interactive Generator Product Surface Tracker

Status: Approved
Delivery set: interactive-generator-product-surface
Plan: `ops/default/plan/interactive-generator-product-surface.md`
Umbrella spec: `ops/default/spec/interactive-hatmax-generator.md`
Specs:

- `ops/default/spec/hatmax-book.md`
- `ops/default/spec/intent-and-planning.md`
- `ops/default/spec/server-rendered-crud.md`
- `ops/default/spec/execution-and-conformance.md`
- `ops/default/spec/interactive-product-surface.md`

Base branch: `dev`
Planning base: `ec16a4df27c6bb3ee6004d6303195f4717da73de`
Active slice: Slice 3
Active tasks: none
Execution gate: Slice 3 tasks complete, pull request pending

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Interpreter contracts | delivered | `feat/generator-interpreter-contracts` | `feat(slice-1): define interactive interpreter contracts` | #32 (merged) | `ops/default/report/slices/interactive-generator-product-surface/slice-1-interpreter-contracts.md` |
| Slice 2 | Codex App Server runtime | delivered | `feat/generator-codex-app-server` | `feat(slice-2): connect the Codex App Server runtime` | #33 (merged) | `ops/default/report/slices/interactive-generator-product-surface/slice-2-codex-app-server-runtime.md` |
| Slice 3 | Codex interpreter | active | `feat/generator-codex-interpreter` | `feat(slice-3): interpret Hatmax requests with Codex` | pending | `ops/default/report/slices/interactive-generator-product-surface/slice-3-codex-interpreter.md` |
| Slice 4 | Interaction coordinator | pending | `feat/generator-interaction-coordinator` | `feat(slice-4): coordinate interactive generation` | pending | `ops/default/report/slices/interactive-generator-product-surface/slice-4-interaction-coordinator.md` |
| Slice 5 | Terminal product surface | pending | `feat/generator-terminal-surface` | `feat(slice-5): expose the Hatmax generator command` | pending | `ops/default/report/slices/interactive-generator-product-surface/slice-5-terminal-product-surface.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | complete | `feat(generator): define interactive interpreter contracts` | `cf8707c` | `go test ./generator/eval/...`; `go test -race ./generator/eval/...`; `go test ./generator/...`; `make vet`; `make lint-strict`; `git diff --check` passed |
| T1.2 | complete | `feat(generator): constrain interpreter output` | `2b38354` | `go test ./generator/eval/... ./generator/intent/...`; `go test -race ./generator/eval/...`; `go test ./generator/...`; `make vet`; `make lint-strict`; `git diff --check` passed |
| T1.3 | complete | `test(generator): cover interactive interpretation contracts` | `a7df81f` | `go test ./generator/eval/... ./generator/intent/... ./generator/plan/...`; `go test -race ./generator/eval/...`; `go test ./generator/...`; `make vet`; `make lint-strict`; `git diff --check` passed |

## Slice 2 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T2.1 | complete | `feat(generator): manage the Codex runtime` | `9c9e169` | `go test ./generator/backend/codex/...`; `go test -race ./generator/backend/codex/...`; `go test ./generator/...`; `make vet`; `make lint-strict`; `git diff --check` passed |
| T2.2 | complete | `feat(generator): speak the Codex App Server protocol` | `8bd4172` | `go test ./generator/backend/codex/...`; `go test -race ./generator/backend/codex/...`; `go test ./generator/...`; `make vet`; `make lint-strict`; `git diff --check` passed |
| T2.3 | complete | `feat(generator): manage isolated Codex threads` | `2262759` | `go test ./generator/backend/codex/...`; `go test -race ./generator/backend/codex/...`; `go test ./generator/...`; `make vet`; `make lint-strict`; `git diff --check` passed |

## Slice 3 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T3.1 | complete | `feat(generator): compile bounded Codex interpretation` | `64cff25` | `go test ./generator/backend/codex/...`; `go test -race ./generator/backend/codex/...`; `go test ./generator/...`; `make vet`; `make lint-strict`; `git diff --check` passed |
| T3.2 | complete | `feat(generator): add the Codex interpreter` | `e9d15c8` | `go test ./generator/backend/codex/...`; `go test -race ./generator/backend/codex/...`; `go test ./generator/...`; `make vet`; `make lint-strict`; `git diff --check` passed |
| T3.3 | complete | `test(generator): exercise Codex interpretation` | `457e6d7` | deterministic and race validation passed; `make generator-live-smoke` observed the missing standalone Codex daemon prerequisite before inference |

## Slice 4 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T4.1 | planned | `feat(generator): define interactive generation lifecycle` | pending | pending |
| T4.2 | planned | `feat(generator): coordinate intent and approval` | pending | pending |
| T4.3 | planned | `feat(generator): coordinate validated execution` | pending | pending |

## Slice 5 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T5.1 | planned | `feat(generator): add the Hatmax generate command` | pending | pending |
| T5.2 | planned | `feat(generator): report interactive generation results` | pending | pending |
| T5.3 | planned | `test(generator): validate the terminal product surface` | pending | pending |

## Completion Gates

- [x] Umbrella and prerequisite subordinate specifications are approved.
- [x] The interactive product-surface specification is approved on `dev`.
- [x] Plan and tracker are approved and committed on `dev`.
- [x] Slice 1 is delivered through its branch, report, pull request, and merge.
- [x] Slice 2 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 3 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 4 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 5 is delivered through its branch, report, pull request, and merge.
- [ ] The exact integrated `dev` candidate passes the deterministic, full
  repository, thread-reuse, project-isolation, and authenticated Codex gates.

## Current Gate

Slice 3 implementation and deterministic validation are complete on
`feat/generator-codex-interpreter`. Its report is ready and the pull request is
pending. The observational live smoke stopped before inference because the
installed `mise` binary lacks the standalone daemon installation. Slice 4
remains blocked until Slice 3 is reviewed and merged into `dev`.
