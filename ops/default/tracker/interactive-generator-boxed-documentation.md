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
Active slice: Slice 3
Active tasks: none
Execution gate: Slice 3 tasks complete, pull request pending

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Documentation contracts | delivered | `feat/generator-documentation-contracts` | `feat(slice-1): define boxed documentation contracts` | #41 | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-1-documentation-contracts.md` |
| Slice 2 | Documentation inventory and planning | delivered | `feat/generator-documentation-planning` | `feat(slice-2): plan boxed documentation changes` | #42 | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-2-documentation-planning.md` |
| Slice 3 | Diataxis rendering and conformance | active | `feat/generator-documentation-rendering` | `feat(slice-3): render boxed Diataxis documentation` | pending | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-3-documentation-rendering.md` |
| Slice 4 | Interactive documentation product | pending | `feat/generator-documentation-product` | `feat(slice-4): deliver interactive documentation generation` | pending | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-4-documentation-product.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | complete | `feat(generator): define documentation intent` | `f28a277` | `go test ./generator/intent/... ./generator/eval/...`; full slice gate passed |
| T1.2 | complete | `feat(generator): add boxed documentation rules` | `2cc00e8` | `go test ./generator/...`; `go test -race ./generator/intent/... ./generator/plan/... ./generator/eval/...`; full slice gate passed |
| T1.3 | complete | `test(generator): cover documentation contracts` | `0a4b866` | deterministic documentation corpus; `make vet`; `make lint-strict`; `git diff --check` passed |

## Slice 2 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T2.1 | complete | `feat(generator): inspect documentation surfaces` | `cee852d` | documentation inventory tests; full slice gate passed |
| T2.2 | complete | `feat(generator): collect documentation evidence` | `e871116` | generated-feature evidence and fingerprint binding tests; full slice gate passed |
| T2.3 | complete | `feat(generator): plan documentation effects` | `e3ec5fe` | pure and combined manifest, conflict, snapshot, and drift tests; full slice gate passed |

## Slice 3 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T3.1 | complete | `feat(generator): manage boxed documentation sections` | `7818924` | managed-section replacement and byte-preservation tests; full slice gate passed |
| T3.2 | complete | `feat(generator): render Diataxis documentation` | `8f24814` | all four quadrants, indexes, evidence, and combined-manifest tests; full slice gate passed |
| T3.3 | complete | `test(generator): enforce documentation conformance` | `5051d24` | conformance, rollback, documentation-gate, and ownership tests; full slice gate passed |

## Slice 4 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T4.1 | planned | `feat(generator): coordinate documentation generation` | pending | pending |
| T4.2 | planned | `feat(generator): interpret documentation requests` | pending | pending |
| T4.3 | planned | `test(generator): validate boxed documentation generation` | pending | pending |

## Completion Gates

- [x] Umbrella and prerequisite subordinate specifications are approved.
- [x] The boxed Diataxis documentation specification is approved.
- [x] Plan and tracker are committed on `dev`.
- [x] Slice 1 is delivered through its branch, report, pull request, and merge.
- [x] Slice 2 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 3 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 4 is delivered through its branch, report, pull request, and merge.
- [ ] The exact integrated `dev` candidate passes deterministic, repository,
  documentation, preservation, project-isolation, and authenticated Codex
  gates.

## Current Gate

Slices 1 and 2 were merged through PRs #41 and #42. Slice 3 implementation and
validation are complete on `feat/generator-documentation-rendering`; its
report is ready and the pull request remains to be opened. Slice 4 remains
blocked until Slice 3 is reviewed and merged into `dev`. The three generator
hardening tickets remain outside this delivery set.
