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
Active slice: Slice 6
Next slice: Slice 6
Active tasks: None — Slice 6 PR awaiting controller validation and integration
Execution gate: Open — independent slice-loop execution authorized on 2026-10-08

## Slice Status

| Slice | Short name | Status | Branch | Expected PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Coverage and validation | delivered | `docs/hatmax-documentation-coverage` | `docs(slice-1): inventory Hatmax documentation coverage` | `#112` | `ops/default/report/slices/documentation-conformance/slice-1-coverage-and-validation.md` |
| Slice 2 | Runtime and presentation | delivered | `docs/hatmax-runtime-presentation` | `docs(slice-2): reconcile runtime and presentation guidance` | `#113` | `ops/default/report/slices/documentation-conformance/slice-2-runtime-and-presentation.md` |
| Slice 3 | Data and configuration | delivered | `docs/hatmax-data-configuration` | `docs(slice-3): reconcile data and configuration guidance` | `#114` | `ops/default/report/slices/documentation-conformance/slice-3-data-and-configuration.md` |
| Slice 4 | Identity and recovery | delivered | `docs/hatmax-identity-recovery` | `docs(slice-4): verify identity and recovery documentation` | `#115` | `ops/default/report/slices/documentation-conformance/slice-4-identity-and-recovery.md` |
| Slice 5 | Infrastructure and helpers | delivered | `docs/hatmax-infrastructure-helpers` | `docs(slice-5): reconcile infrastructure and helper guidance` | `#116` | `ops/default/report/slices/documentation-conformance/slice-5-infrastructure-and-helpers.md` |
| Slice 6 | Generator and assisted workflows | reviewing | `docs/hatmax-generator-guidance` | `docs(slice-6): reconcile generator and assisted workflow guidance` | `#117` | `ops/default/report/slices/documentation-conformance/slice-6-generator-and-assisted-workflows.md` |
| Slice 7 | Integrated documentation acceptance | pending | `docs/hatmax-documentation-acceptance` | `docs(slice-7): establish complete documentation acceptance` | pending | `ops/default/report/slices/documentation-conformance/slice-7-integrated-documentation-acceptance.md` |

## Tasks

| Task | Status | Expected commit | Commit |
| --- | --- | --- | --- |
| T1.1 | completed | `docs: inventory Hatmax documentation surfaces` | `596cde4204b7596309760b3ffe8f819aff5b34be` |
| T1.2 | completed | `test(docs): establish Hatmax conformance checks` | `1fe3c2e40d7d58a476dd20e0e3adf64a6ac7bc3f` |
| T2.1 | completed | `docs: reconcile runtime and presentation guidance` | `cc268ec177e4cdd23817d5bf363fe152ebf5edcf` |
| T2.2 | completed | `test(docs): verify runtime and presentation examples` | `79a9665a852ffd11f0383146ae745ada41f5d0da` |
| T3.1 | completed | `docs: reconcile data and configuration guidance` | `c48977e03c158214f6118312d8e331b4dec8bd40` |
| T3.2 | completed | `test(docs): verify data and configuration examples` | `22b7bd76e7d239185a1ac2bb06e2be11b7ce1442` |
| T4.1 | completed | `docs: reconcile identity and recovery guidance` | `f96565f609da32a7430cc617138d72d016376fcf` |
| T4.2 | completed | `test(docs): verify identity and recovery examples` | `1cb59fa18fb16e50b0a56c64b9e1fbdb808ee713` |
| T5.1 | completed | `docs: reconcile infrastructure and helper guidance` | `a94d6a41afd83e55c7cd02fe70b36238846c9e04` |
| T5.2 | completed | `test(docs): verify infrastructure and helper examples` | `38eb310994d7abf1731bd3aeb2ccb14d8a628ae4` |
| T6.1 | completed | `docs: reconcile generator and assisted workflow guidance` | `0a70f3b71cd69db25dfaae0a62224a0f6eef087d` |
| T6.2 | completed | `test(docs): verify generator documentation workflows` | `cdd3e8888c4a34bd253dc5503fe4f9bd37c599ba` |
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

Mandatory closure condition approved on 2026-10-08:

- [ ] Current setup, examples, validators and fixtures use Go/shell where sufficient; no gratuitous auxiliary runtime requirement remains.
- [ ] The final gate enforces this constraint and records successful relevant workflows with only their required native toolchain available.

DC09 and delivery-set completion remain pending until both conditions pass.
Historical execution records and product-owned browser technology retain their
existing meaning.

| Criterion | Evidence status | Reference |
| --- | --- | --- |
| DC01 | inventory established; integrated acceptance pending | `ops/default/report/documentation-conformance-coverage.md` |
| DC02 | focused anchors/navigation passed; integrated acceptance pending | `ops/default/report/slices/documentation-conformance/slice-1-coverage-and-validation.md` |
| DC03 | pending | pending |
| DC04 | pending | pending |
| DC05 | pending | pending |
| DC06 | pending | pending |
| DC07 | generator guidance reconciled; full journey blocked by runtime defects | `ops/default/report/slices/documentation-conformance/slice-6-generator-and-assisted-workflows.md` |
| DC08 | focused evidence classified; three generator rows blocked | `ops/default/report/documentation-conformance-coverage.md` |
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

T2.1 reconciles runtime/presentation package notes and supporting tutorial,
how-to and reference guidance. It corrects mixed Go/template blocks, middleware
installation order, function-map translation behavior, cleanup ownership and
server/response boundaries. The current bootstrap binds a local `dev` checkout
because published `v0.5.0` has the older server API, and resolves dependencies
after `main.go` exists. Its exact Go program built in an isolated source-bound
module; nine published HTML fragments parsed and executed in typed view context.
The repository docs/example-compilation check, strict lint and whitespace passed.
T2.2 now supplies durable source-bound contextual compilation, actual bootstrap
and Guide HTTP/process execution, classified receipts and the Slice 2 gate.
The canonical PR and final clean-head controller evidence are prepared after
this task commit; Slice 2 remains active until canonical merge verification.

Slice 2 canonical PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/113.
Report introduction: `79a9665a852ffd11f0383146ae745ada41f5d0da`.
Both task commits and PR references are complete. Final clean-head validation
is recorded through the controller before readiness; merge remains controller-owned.

Slice 2 repair rebased the existing candidate onto canonical dev
`064c5490f8f97f730cd37daaf80de6492d495cec`, preserving the approved native-tooling
closure condition. Task and report-introduction references identify the replayed
commits; the final pushed repair head requires fresh ordered controller checks.

Slice 2 of 7 completed: canonical PR #113 merged through Rebase + Fast-forward
at `f2d706af2f4afb467b17a8e9b489af9355af1407`. Its exact clean-head controller
receipt passed `GOWORK=off scripts/check-documentation-conformance.sh slice 2`
followed by `git diff --check`, including 448 identities and 141 runtime bindings.
The report is delivered. Slice 3/T3.1/T3.2 is active on its approved branch and
task map; integrated and native-tooling acceptance remain pending.

T3.1 reconciles static configuration loading/validation and safe diagnostics,
settings defaults and adapter ownership, database lifecycle and migration
history, seeder retry/tracking, model primitives and byte-based validation.
Published seed/model examples use current APIs; illustrative invoice updates
validate a candidate before mutation. Guide documents persistent notes and
memory-only greetings. Structural reconciliation covers 451 identities;
behavioral Slice 3 receipts remain pending for T3.2. The documentation check,
strict lint and whitespace check passed during this task.

T3.2 supplies fresh native PostgreSQL storage, exact published Go compilation,
real YAML/validator and SQLC execution, migration rollback/history checks,
settings persistence/absence/error behavior, seeder retry/tracking and Guide
note/greeting restart observations. Coverage records 89 Slice 3 bindings and
keeps future groups pending. The gate reruns Slice 2 and adds Slice 3 receipts,
tool binary identities, command diagnostics and owned-process/cluster cleanup.
The report is prepared for canonical PR creation; ordered final validation is
recorded by the controller on the clean pushed head after PR references.

Slice 3 canonical PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/114.
Report introduction: `22b7bd76e7d239185a1ac2bb06e2be11b7ce1442`.
Both task commits and canonical references are complete. The controller records
the ordered Slice 3 command and whitespace check after this reference commit
is pushed; merge remains controller-owned.

Slice 3 of 7 completed: canonical PR #114 merged through Rebase + Fast-forward
at `76fc39c8a7c0c40c1aca9020e753c079fe366514`. Its exact clean-head controller
receipt passed `GOWORK=off scripts/check-documentation-conformance.sh slice 3`
followed by `git diff --check`, including 451 identities, 141 runtime bindings
and 89 data bindings. Owned PostgreSQL and HTTP fixtures stopped. The report is
delivered. Slice 4/T4.1/T4.2 is active on its approved branch and task map;
integrated acceptance and the generic native-tooling closure requirement remain
pending.

T4.1 compares the auth/crypto service, policy, admission, session, factor,
recovery, observer, key and token contracts with current source. It retains
actual proof/freshness, atomic mutation, factor-preservation and lost-response
boundaries. Ticked setup now uses its actual database and startup migrations,
required private admission material, current Go toolchain and direct owned
foreground execution. Crypto guidance distinguishes the 64 MiB raw primitive,
PHC credentials, signed readable PASETO claims and application key ownership.
Fragments name their owning contexts and retain error handling. Documentation,
strict lint, whitespace and 451-identity/navigation checks passed; Slice 4
behavioral evidence remains pending for T4.2.

T4.2 supplies typed contexts for 15 exact Go fragments, including executed
crypto outcomes and private key-generation output. Go now owns Chromium/CDP,
profiles and bounded shutdown; JavaScript executes only actual browser journeys.
The published WebAuthn JSON-conversion fragment and all three uncached race
selectors passed against production handlers and owned PostgreSQL. Actual Ticked
entrypoint/Make workflows passed native role/database creation, configuration,
signup without a session, password sign-in, restart persistence, cookie retirement
and coordinated shutdown. The development run produced 50 classified bindings.
Exact-head cumulative runner/evidence checks follow the PR-reference commit;
DC05/DC08 integrated acceptance and the mandatory native-toolchain closure gate
remain pending for Slice 7. Historical interpreter-based evidence is unchanged.

Canonical PR #115 merged into dev at `ff63b06c67858b497691eece8d91969006999fee`.
The controller recorded the exact ordered Slice 4 command and whitespace check
on that clean pushed head: 451 identities, 141 cumulative runtime bindings,
89 data bindings and 50 identity bindings passed, including actual browser proof
and owned fixture shutdown. The foreground Make wrapper correction passed its
regression checks and the regenerated exact-head gate. Slice 4 is delivered;
Slice 5/T5.1/T5.2 is active with its original identities. DC09 and integrated
acceptance remain pending until the approved final native-toolchain gate.

T5.2 compiles 35 exact Go fragments and supplies 92 infrastructure/helper
bindings: 14 package suites, 11 compilation-only contexts, 38 executed outcomes
and 29 source inspections. Published mail/image/event/job/helper workflows use
owned SMTP capture, filesystem and PostgreSQL; named retry/restart, persistent
slot budgets, recurring slots, cancellation, telemetry drain and schema removal
passed development checks. The native runner binds fixture inputs, source/head,
tool identities, commands and diagnostics, then verifies owned shutdown.
Canonical PR references and ordered final clean-head controller checks follow
the task commit. Slice 5 remains active until verified canonical merge.

Slice 5 canonical PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/116.
Report introduction: `b1b77bb68e9e0c3c25b87f5b2ed2a64076dc8122`.
Both task commits and canonical references are complete. The controller records
`GOWORK=off scripts/check-documentation-conformance.sh slice 5` followed by
`git diff --check` against the clean pushed head after this reference commit.
The primary checkout remains on dev; merge remains controller-owned.

Slice 5 of 7 completed: canonical PR #116 merged through Rebase + Fast-forward
at `b29aa327d3b8a815b79c894571438ce2a6f44144`. Its exact clean-head controller
receipt passed the ordered Slice 5 command and whitespace check: 451 identities,
141 runtime, 89 data, 50 identity and 92 infrastructure bindings reconciled.
Owned fixtures stopped. The report is delivered; Slice 6/T6.1/T6.2 is active
with its approved identities. Integrated acceptance and generic native-toolchain
closure remain pending.

T6.1 corrects the current hm source-install route, CLI usage, playground tool
requirements, Book intervals and selection, generated ownership markers,
retention/privacy and post-commit validation outcomes. Focused documentation,
licensing, strict lint, 451-identity reconciliation and owned source installation
passed. All 63 Slice 6 workflow bindings remain pending for T6.2; report status
was drafting at that task commit; canonical PR preparation followed T6.2.

T6.2 at `cdd3e8888c4a34bd253dc5503fe4f9bd37c599ba` reconciles 63 bindings:
12 package suites, 14 executed bindings, 34 source inspections and three blocked
rows. Native CLI/TUI procedures, persisted conversation, published bare scaffold,
canonical feature creation and managed documentation execute with owned tools
and PostgreSQL. Actual failures are retained for fresh-scaffold composition
resolution, timestamp SQLC row mapping and generated validation-test lint.
Tickets `TKT-20261008134713`, `TKT-20261008135600` and `TKT-20261008135846`
own these runtime defects; no runtime fix is included in the documentation set.
Focused required checks and native fixtures passed their evidence assertions;
owned clusters stopped. Canonical PR #117 targets dev with the approved title
and immutable report-introduction link. The controller owns final exact-head
validation after this PR-reference follow-up is pushed. DC07/DC08 affected claims,
integrated acceptance and generic native-toolchain closure remain unresolved.
