<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 3: Data and Configuration

Status: delivered
Integrated candidate: `090e260378cbb47734f6202be360923a9db1c3ac`
Integrated acceptance: [Passed controller evidence](../../documentation-conformance-acceptance.md)
Delivery set: documentation-conformance
Plan: [Delivery plan](../../../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../../../tracker/documentation-conformance.md)
Branch: `docs/hatmax-data-configuration`
PR: `#114`
Report introduction: `22b7bd76e7d239185a1ac2bb06e2be11b7ce1442`
Activation revision: `0c6817af71c895ce2d1e7b4d436c6c290124ba99`
Documentation task: `c48977e03c158214f6118312d8e331b4dec8bd40`
Merged revision: `76fc39c8a7c0c40c1aca9020e753c079fe366514`

## Purpose

Reconcile configuration, settings, persistence, model, validation, seeding and
logging guidance with current source and reproducible local data workflows.

## Delivered Behavior

Reference, learning-path, task and package guidance states effective loading
precedence, underscore mapping, explicit validator boundaries and safe logging.
Settings documentation distinguishes missing, empty, invalid and failed reads,
application-owned persistence and secret metadata. Database and seeder guidance
preserves existing migration identities, per-file transactions, retry/tracking
behavior and partial-start resource ownership. Published seed/model snippets use
current APIs; the illustrative invoice update preserves state on invalid input.

The fresh data workbench loads published YAML through real validators, runs the
published migrations and SQLC queries, maps persisted invoice rows, and executes
the exact constructor/update/seeder fragments. It verifies applied-file skipping,
per-file rollback, earlier-file persistence, unexecuted Down, failed schema/ping
cleanup, setting defaults and error propagation, and seeder tracking-loss retries.
Ticked's published SQLC configuration generates and compiles its DAL; non-empty
Ticked store suites exercise its current persistence adapter.

Guide database mode accepts a matching-origin note submission and retains its
title/slug after restart. Greeting updates return 303, read back as Welcome and
reset to Hello after restart because the adapter is in memory. Actual HTTP checks
observe name-validation feedback, HTML escaping and rejected empty notes.

Coverage reconciles 451 identities. The 89 Slice 3 bindings distinguish eight
non-empty package suites, 27 compiled contexts, 23 executed bindings and 31
source-inspected pages or notation. Thirty exact Go fragments compile; four
also execute in the workbench. Future groups retain pending evidence.

## Implementation Notes

Go and shell own the validators and fixtures. Tools are Go 1.27.1, SQLC v1.30.0
matching the published DAL, and the invocation's native PostgreSQL tools. The
development fixture used PostgreSQL 18.6. The runner resolves SQLC from PATH,
an explicit override, or its ignored worktree tool cache; PostgreSQL supports an
explicit binary-directory override. Version and binary digests bind data tools.

Each invocation creates a cluster and disposable role/database, distinct schemas
and a short unique Unix socket with TCP disabled. The fixture utility stops only
its own cluster after checks finish. Diagnostics stay in ignored storage. Module
and build commands have five-minute bounds, tests three-minute bounds, cluster
startup/stop 20-second bounds, and owned HTTP readiness/shutdown finite bounds.

The published psql command uses owned socket/port/schema substitutions. Guide
build plus direct launch replaces the Go run supervisor and uses fresh HTTP
ports; its main exits by interrupt rather than a graceful coordinator. Receipt
checking verifies stopped processes, configuration identities and cluster stop
evidence. Browser JavaScript is not exercised by these HTTP checks.

## Contracts Added or Changed

The slice runner accepts slices 1 through 3, reruns earlier runtime evidence for
Slice 3, and rejects future/integrated modes. Data receipts bind exact HEAD,
source/snippet and harness inputs, compiler/tool identities, actual commands,
diagnostic digests, outcomes and verification methods. Missing, changed,
duplicate or misclassified bindings and failed/empty execution cannot pass.

Application-owned fixture stores, invoice validation, handlers and DAL shapes
supply explicit contexts. Compiled fragments do not establish invoice CRUD,
application authorization or external deployment. The workbench's SQL adapter
is an execution fixture, not a new toolkit settings store. Toolkit runtime
code and module dependencies are unchanged.

## Files of Interest

- [Configuration reference](../../../../../docs/reference/configuration/README.md)
- [Database reference](../../../../../docs/reference/database/README.md)
- [Persistence learning path](../../../../../docs/tutorials/user-guide/persistence-and-migrations.md)
- [Guide companion](../../../../../examples/guide/README.md)
- [Coverage record](../../documentation-conformance-coverage.md)
- [Data workbench](../../../../../scripts/documentation-conformance/data_workbench.go)
- [Evidence reconciliation](../../../../../scripts/documentation-conformance/data_evidence.go)
- [Owned PostgreSQL fixture](../../../../../scripts/documentation-conformance/data-fixture.sh)

## Validation

Focused implementation checks passed:

- `GOWORK=off GOFLAGS=-p=2 go test -count=1 -timeout=2m ./scripts/documentation-conformance` — passed, including missing/changed/misclassified SQL evidence, failed commands, malformed contexts, local-name import handling and rejected unowned fixture storage.
- `GOWORK=off go run ./scripts/documentation-conformance check` — passed for 451 source/page/example identities and navigation.
- `bash -n scripts/check-documentation-conformance.sh scripts/documentation-conformance/data-fixture.sh` — passed.
- `GOWORK=off make lint-strict` — passed with zero issues.

The development data invocation produced 89 bindings after real owned
PostgreSQL/SQLC/Guide execution and stopped its fixtures. Its receipts are
historical implementation evidence. Final ordered checks are recorded through
the controller after the PR-reference follow-up is pushed, on the exact clean
head:

- `GOWORK=off scripts/check-documentation-conformance.sh slice 3`
- `git diff --check`

The slice command includes required docs, source-license and strict-lint checks,
earlier runtime checks, active data workflows and receipt reconciliation. The
report introduction precedes that final head; canonical PR/controller evidence
owns final exact-head results.

The controller recorded both ordered commands successful on clean pushed head
`76fc39c8a7c0c40c1aca9020e753c079fe366514`: 451 identities, 141 cumulative runtime
bindings and 89 data bindings passed. Cluster and HTTP cleanup evidence passed.
Canonical PR #114 merged at that same revision; this closure changes only
operational delivery state.

The first controller invocation exposed an ownership-test path error: the gate's
owned TMPDIR made `t.TempDir()` a permitted fixture path. The negative test now
creates its directory explicitly outside owned storage; final evidence is
regenerated on the corrected head.

A direct `GOWORK=off golangci-lint run --fix ./scripts/documentation-conformance`
attempt failed in extra static-analysis passes with a nil build-IR panic.
Explicit spacing corrections and the repository's required lint mode passed;
no toolchain downgrade or repository lint-policy change was made.

## Risks and Follow-ups

Integrated DC04/DC08 acceptance and the approved native-tooling closure condition
passed in the controller's full documentation gate on the candidate above.
Slice 4 supplied identity/browser proof and the complete Ticked composition.
Conceptual guidance and compiled application adapters retain the precise
assurance limits stated above. Changed inputs require fresh evidence.
