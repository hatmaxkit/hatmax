<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Hatmax Documentation Conformance Tracker

Date: 2026-10-08
Status: In progress
Approved: 2026-10-08
Delivery set: documentation-conformance
Concern: [Documentation conformance](../spec/documentation-conformance.md)
Plan: [Delivery plan](../plan/documentation-conformance.md)
Base branch: `dev`
Planning base: `3ec58ea2207f71b8eddae39501e69485d2673470`
Active slice: Slice 2
Next slice: Slice 2
Active tasks: T2.1, T2.2
Execution gate: Open — independent slice-loop execution authorized on 2026-10-08

## Slice Status

| Slice | Short name | Status | Branch | Expected PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Coverage and validation | delivered | `docs/hatmax-documentation-coverage` | `docs(slice-1): inventory Hatmax documentation coverage` | `#112` | `ops/default/report/slices/documentation-conformance/slice-1-coverage-and-validation.md` |
| Slice 2 | Runtime and presentation | implementing | `docs/hatmax-runtime-presentation` | `docs(slice-2): reconcile runtime and presentation guidance` | pending | `ops/default/report/slices/documentation-conformance/slice-2-runtime-and-presentation.md` |
| Slice 3 | Data and configuration | pending | `docs/hatmax-data-configuration` | `docs(slice-3): reconcile data and configuration guidance` | pending | `ops/default/report/slices/documentation-conformance/slice-3-data-and-configuration.md` |
| Slice 4 | Identity and recovery | pending | `docs/hatmax-identity-recovery` | `docs(slice-4): verify identity and recovery documentation` | pending | `ops/default/report/slices/documentation-conformance/slice-4-identity-and-recovery.md` |
| Slice 5 | Infrastructure and helpers | pending | `docs/hatmax-infrastructure-helpers` | `docs(slice-5): reconcile infrastructure and helper guidance` | pending | `ops/default/report/slices/documentation-conformance/slice-5-infrastructure-and-helpers.md` |
| Slice 6 | Generator and assisted workflows | pending | `docs/hatmax-generator-guidance` | `docs(slice-6): reconcile generator and assisted workflow guidance` | pending | `ops/default/report/slices/documentation-conformance/slice-6-generator-and-assisted-workflows.md` |
| Slice 7 | Integrated documentation acceptance | pending | `docs/hatmax-documentation-acceptance` | `docs(slice-7): establish complete documentation acceptance` | pending | `ops/default/report/slices/documentation-conformance/slice-7-integrated-documentation-acceptance.md` |

## Tasks

| Task | Status | Expected commit | Commit |
| --- | --- | --- | --- |
| T1.1 | completed | `docs: inventory Hatmax documentation surfaces` | `596cde4204b7596309760b3ffe8f819aff5b34be` |
| T1.2 | completed | `test(docs): establish Hatmax conformance checks` | `1fe3c2e40d7d58a476dd20e0e3adf64a6ac7bc3f` |
| T2.1 | pending | `docs: reconcile runtime and presentation guidance` | pending |
| T2.2 | pending | `test(docs): verify runtime and presentation examples` | pending |
| T3.1 | pending | `docs: reconcile data and configuration guidance` | pending |
| T3.2 | pending | `test(docs): verify data and configuration examples` | pending |
| T4.1 | pending | `docs: reconcile identity and recovery guidance` | pending |
| T4.2 | pending | `test(docs): verify identity and recovery examples` | pending |
| T5.1 | pending | `docs: reconcile infrastructure and helper guidance` | pending |
| T5.2 | pending | `test(docs): verify infrastructure and helper examples` | pending |
| T6.1 | pending | `docs: reconcile generator and assisted workflow guidance` | pending |
| T6.2 | pending | `test(docs): verify generator documentation workflows` | pending |
| T7.1 | pending | `docs: reconcile complete Hatmax documentation coverage` | pending |
| T7.2 | pending | `test(docs): bind Hatmax documentation acceptance` | pending |

## Prerequisites and Gates

- [x] Complete documentation-conformance planning requested on 2026-10-08.
- [x] Own repository scope, source baseline, four quadrants and current entrypoints inspected.
- [x] 7 exact slice identities and 14 task subjects recorded.
- [x] Repository-local coverage, example evidence and immutable final-gate contracts defined.
- [x] Specification and plan approved; implementation explicitly authorized on 2026-10-08.
- [x] Actual activation base, clean canonical dev/PR/worktree state and toolchain verified.
- [x] Fresh independent loop configuration/run/view identities frozen and reported.
- [x] Complete source/page inventory and example identities recorded.
- [ ] All mapped tasks integrated with actual immutable commit evidence.
- [ ] All canonical slice PRs integrated and reports delivered.
- [ ] All acceptance criteria passed with source-bound example/walkthrough evidence.
- [ ] Exact clean immutable dev candidate passed the full documentation gate.
- [ ] Frozen operational closure paths updated without changing tested documentation.

## Acceptance Tracking

| Criterion | Evidence status | Reference |
| --- | --- | --- |
| DC01 | inventory established; integrated acceptance pending | `ops/default/report/documentation-conformance-coverage.md` |
| DC02 | focused anchors/navigation passed; integrated acceptance pending | `ops/default/report/slices/documentation-conformance/slice-1-coverage-and-validation.md` |
| DC03 | pending | pending |
| DC04 | pending | pending |
| DC05 | pending | pending |
| DC06 | pending | pending |
| DC07 | pending | pending |
| DC08 | pending | pending |
| DC09 | pending | pending |
| DC10 | pending | pending |

## Execution Record

Execution authorized on 2026-10-08. Activation verified clean primary `dev`
and canonical `origin/dev` at `6715ab2912d27263a78f0bd1419db8b49c3d8089`,
no open canonical PR and no competing controller for this repository.
Go identity: `go version go1.27.1 linux/amd64`.

Independent configuration: `hatmax-documentation-conformance`.
Configuration SHA-256: `c21b0166b17341f7971349afb0d4ab1088b3e198d6e8178353a4cdcbee79f222`.
Launch: `a03d71a56a40`; run: `loop-cbe92c396a22-a03d71a56a40`;
worker conversation: `01a11aaa-acae-7bd0-854a-b1a71364ce3b`.
View command: `slice-watch hatmax-documentation-conformance`.
The controller owns all merge mutations.

T1.1 records 443 source/page/example identities at the activation revision,
including 101 product pages, 218 fenced blocks and 28 complete walkthroughs.
Content digests account for package tests, example assets/migrations/browser
fixtures, both embedded Book releases and dependency inputs. All behavioral
receipts remain pending for their owning slices. Focused source reconciliation
and four-quadrant navigation checks passed. T1.2 establishes the slice gate and
negative controls. Focused checker tests, shell syntax and strict lint passed.
The required slice command and whitespace checks will be recorded through the
controller against the final clean pushed head after the PR-reference commit.
The plan owns fixture isolation, exact command prefixes, acceptance and the
correction route.

Execution remains repository-local. A completed source/page inventory or links
check alone cannot close the set. Record runtime defects or missing prerequisites
as blockers; do not fill pending execution evidence with earlier planning checks.

Slice 1 canonical PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/112.
Report introduction: `1fe3c2e40d7d58a476dd20e0e3adf64a6ac7bc3f`.
Both task commits are complete. The PR-reference follow-up records the canonical
PR in this tracker and the report before final controller validation and push.

Slice 1 of 7 completed: canonical PR #112 merged through Rebase + Fast-forward
at `4edfea516b25656b2ab4797163ddbc6523c54e33`. Its exact clean-head controller
receipt passed `GOWORK=off scripts/check-documentation-conformance.sh slice 1`
followed by `git diff --check`. The report is delivered; integrated acceptance
remains pending. Slice 2/T2.1/T2.2 is active on its approved branch and task map.
