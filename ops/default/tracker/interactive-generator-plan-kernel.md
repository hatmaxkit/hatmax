# Interactive Generator Plan Kernel Tracker

Status: Approved
Delivery set: interactive-generator-plan-kernel
Plan: `ops/default/plan/interactive-generator-plan-kernel.md`
Umbrella spec: `ops/default/spec/interactive-hatmax-generator.md`
Specs:

- `ops/default/spec/hatmax-book.md`
- `ops/default/spec/intent-and-planning.md`
- `ops/default/spec/server-rendered-crud.md`
- `ops/default/spec/execution-and-conformance.md`

Base branch: `dev`
Planning base: `28e2b943d2267d903561eb2c5d6f7ad13c31a51a`
Active slice: Slice 1
Active tasks: T1.1, T1.2, T1.3
Execution gate: satisfied by this approved planning commit on `dev`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Book core | ready | `feat/generator-book-core` | `feat(slice-1): add the Hatmax Book core` | pending | `ops/default/report/slices/interactive-generator-plan-kernel/slice-1-book-core.md` |
| Slice 2 | Project inventory | pending | `feat/generator-project-inventory` | `feat(slice-2): inventory Hatmax projects` | pending | `ops/default/report/slices/interactive-generator-plan-kernel/slice-2-project-inventory.md` |
| Slice 3 | Intent validation | pending | `feat/generator-intent-validation` | `feat(slice-3): validate generator intents` | pending | `ops/default/report/slices/interactive-generator-plan-kernel/slice-3-intent-validation.md` |
| Slice 4 | Deterministic planning | pending | `feat/generator-deterministic-planning` | `feat(slice-4): expand deterministic generator plans` | pending | `ops/default/report/slices/interactive-generator-plan-kernel/slice-4-deterministic-planning.md` |
| Slice 5 | Planning evaluation | pending | `test/generator-planning-evaluation` | `test(slice-5): validate generator planning` | pending | `ops/default/report/slices/interactive-generator-plan-kernel/slice-5-planning-evaluation.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | pending | `feat(generator): add the Hatmax Book loader` | pending | pending |
| T1.2 | pending | `feat(generator): define the initial Hatmax Book` | pending | pending |
| T1.3 | pending | `test(generator): validate Hatmax Book contracts` | pending | pending |

## Slice 2 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T2.1 | pending | `feat(generator): inventory Hatmax projects` | pending | pending |
| T2.2 | pending | `feat(generator): fingerprint planning inputs` | pending | pending |
| T2.3 | pending | `test(generator): cover project inventory states` | pending | pending |

## Slice 3 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T3.1 | pending | `feat(generator): define typed generator intents` | pending | pending |
| T3.2 | pending | `feat(generator): validate intent semantics` | pending | pending |
| T3.3 | pending | `test(generator): cover generator intent validation` | pending | pending |

## Slice 4 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T4.1 | pending | `feat(generator): define execution plans` | pending | pending |
| T4.2 | pending | `feat(generator): expand Hatmax obligations` | pending | pending |
| T4.3 | pending | `feat(generator): bind plans to project state` | pending | pending |

## Slice 5 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T5.1 | pending | `test(generator): add the planning corpus` | pending | pending |
| T5.2 | pending | `feat(generator): define interpreter evaluation` | pending | pending |
| T5.3 | pending | `test(generator): exercise the plan kernel` | pending | pending |

## Completion Gates

- [x] Umbrella specification is approved and committed on `dev`.
- [x] Subordinate specifications are approved.
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
commit on `feat/generator-book-core`.
