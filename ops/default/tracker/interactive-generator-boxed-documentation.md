# Interactive Generator Boxed Documentation Tracker

Status: Approved
Delivery set: interactive-generator-boxed-documentation
Plan: `ops/default/plan/interactive-generator-boxed-documentation.md`
Umbrella spec: `ops/default/spec/interactive-hatmax-generator.md`
Specs:

- `ops/default/spec/hatmax-book.md`
- `ops/default/spec/intent-and-planning.md`
- `ops/default/spec/execution-and-conformance.md`
- `ops/default/spec/interactive-product-surface.md`
- `ops/default/spec/boxed-diataxis-documentation.md`

Base branch: `dev`
Planning base: `1549e663d0d5648e00cdcc7e5fae1c50c5dd29c3`
Active slice: Slice 1
Active tasks: T1.1-T1.3
Execution gate: merge the approved planning artifacts into `dev`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Documentation contracts | ready | `feat/generator-documentation-contracts` | `feat(slice-1): define boxed documentation contracts` | pending | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-1-documentation-contracts.md` |
| Slice 2 | Documentation inventory and planning | pending | `feat/generator-documentation-planning` | `feat(slice-2): plan boxed documentation changes` | pending | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-2-documentation-planning.md` |
| Slice 3 | Diataxis rendering and conformance | pending | `feat/generator-documentation-rendering` | `feat(slice-3): render boxed Diataxis documentation` | pending | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-3-documentation-rendering.md` |
| Slice 4 | Interactive documentation product | pending | `feat/generator-documentation-product` | `feat(slice-4): deliver interactive documentation generation` | pending | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-4-documentation-product.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | pending | `feat(generator): define documentation intent` | pending | pending |
| T1.2 | pending | `feat(generator): add boxed documentation rules` | pending | pending |
| T1.3 | pending | `test(generator): cover documentation contracts` | pending | pending |

## Slice 2 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T2.1 | pending | `feat(generator): inspect documentation surfaces` | pending | pending |
| T2.2 | pending | `feat(generator): collect documentation evidence` | pending | pending |
| T2.3 | pending | `feat(generator): plan documentation effects` | pending | pending |

## Slice 3 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T3.1 | pending | `feat(generator): manage boxed documentation sections` | pending | pending |
| T3.2 | pending | `feat(generator): render Diataxis documentation` | pending | pending |
| T3.3 | pending | `test(generator): enforce documentation conformance` | pending | pending |

## Slice 4 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T4.1 | pending | `feat(generator): coordinate documentation generation` | pending | pending |
| T4.2 | pending | `feat(generator): interpret documentation requests` | pending | pending |
| T4.3 | pending | `test(generator): validate boxed documentation generation` | pending | pending |

## Completion Gates

- [x] Umbrella and prerequisite subordinate specifications are approved.
- [x] The boxed Diataxis documentation specification is approved.
- [ ] Plan and tracker are committed on `dev`.
- [ ] Slice 1 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 2 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 3 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 4 is delivered through its branch, report, pull request, and merge.
- [ ] The exact integrated `dev` candidate passes deterministic, repository,
  documentation, preservation, project-isolation, and authenticated Codex
  gates.

## Current Gate

The specification, plan, and tracker are approved. Implementation starts only
after these planning artifacts merge into `dev`. The three open generator
hardening tickets remain intentionally outside this delivery set and are
scheduled after boxed documentation generation closes.
