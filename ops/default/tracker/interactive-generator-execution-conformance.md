# Interactive Generator Execution and Conformance Tracker

Status: Approved
Delivery set: interactive-generator-execution-conformance
Plan: `ops/default/plan/interactive-generator-execution-conformance.md`
Umbrella spec: `ops/default/spec/interactive-hatmax-generator.md`
Specs:

- `ops/default/spec/hatmax-book.md`
- `ops/default/spec/intent-and-planning.md`
- `ops/default/spec/server-rendered-crud.md`
- `ops/default/spec/execution-and-conformance.md`

Base branch: `dev`
Planning base: `9b7e0d9e71da8297665e42695459e5eb33c739e0`
Active slice: Slice 1
Active tasks: T1.1, T1.2, T1.3
Execution gate: satisfied by this approved planning commit on `dev`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Execution manifest | ready | `feat/generator-execution-manifest` | `feat(slice-1): define generator execution manifests` | pending | `ops/default/report/slices/interactive-generator-execution-conformance/slice-1-execution-manifest.md` |
| Slice 2 | Atomic application | pending | `feat/generator-atomic-application` | `feat(slice-2): apply generator edits atomically` | pending | `ops/default/report/slices/interactive-generator-execution-conformance/slice-2-atomic-application.md` |
| Slice 3 | CRUD generation | pending | `feat/generator-crud-generation` | `feat(slice-3): generate canonical Hatmax CRUD features` | pending | `ops/default/report/slices/interactive-generator-execution-conformance/slice-3-crud-generation.md` |
| Slice 4 | Incremental mutations | pending | `feat/generator-incremental-mutations` | `feat(slice-4): mutate canonical Hatmax features` | pending | `ops/default/report/slices/interactive-generator-execution-conformance/slice-4-incremental-mutations.md` |
| Slice 5 | Execution conformance | pending | `test/generator-execution-conformance` | `test(slice-5): validate generator execution conformance` | pending | `ops/default/report/slices/interactive-generator-execution-conformance/slice-5-execution-conformance.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | pending | `feat(generator): preserve executable plan inputs` | pending | pending |
| T1.2 | pending | `feat(generator): define execution manifests` | pending | pending |
| T1.3 | pending | `feat(generator): prepare canonical execution targets` | pending | pending |

## Slice 2 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T2.1 | pending | `feat(generator): stage bounded project edits` | pending | pending |
| T2.2 | pending | `feat(generator): apply project edits atomically` | pending | pending |
| T2.3 | pending | `test(generator): cover atomic execution behavior` | pending | pending |

## Slice 3 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T3.1 | pending | `feat(generator): render canonical CRUD packages` | pending | pending |
| T3.2 | pending | `feat(generator): render CRUD transport surfaces` | pending | pending |
| T3.3 | pending | `feat(generator): wire generated CRUD features` | pending | pending |

## Slice 4 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T4.1 | pending | `feat(generator): inspect canonical feature structure` | pending | pending |
| T4.2 | pending | `feat(generator): add fields across CRUD surfaces` | pending | pending |
| T4.3 | pending | `feat(generator): add layered feature validation` | pending | pending |

## Slice 5 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T5.1 | pending | `feat(generator): check Hatmax execution conformance` | pending | pending |
| T5.2 | pending | `feat(generator): report validated execution` | pending | pending |
| T5.3 | pending | `test(generator): exercise execution conformance` | pending | pending |

## Completion Gates

- [x] Umbrella and subordinate specifications are approved.
- [x] Plan and tracker are committed on `dev`.
- [ ] Slice 1 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 2 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 3 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 4 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 5 is delivered through its branch, report, pull request, and merge.
- [ ] The exact integrated `dev` candidate passes the delivery-set gate.

## Current Gate

The plan and tracker are approved and committed on `dev`. Runtime
implementation has not started. Slice 1 is ready to begin from the planning
commit on `feat/generator-execution-manifest`.
