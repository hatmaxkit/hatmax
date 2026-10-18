<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Durable Background Work Delivery Plan

Date: 2026-10-18
Status: Approved
Approved: 2026-10-18
Specification/model approved: 2026-10-18
Planning authorized: 2026-10-18
Delivery set: durable-background-work
Concern: [Durable background work](../spec/durable-background-work.md)
Model: [Durable work model](../spec/durable-background-work-model.md)
Tracker: [Delivery tracker](../tracker/durable-background-work.md)
Base branch: `dev`
Planning base: `b21af0b17c01ebb3544c37d7d200d2c87f4db0b1`
Execution authorized: 2026-10-18
Integration authority: Controller-owned Rebase + Fast-forward into `dev`
Active slice: Slice 1
Next slice: Slice 2
Execution gate: Approved artifacts published; controller-owned implementation active
Slice strategy: behavior-first
Reason: leased scheduled execution, bounded event delivery and retained-work operations are independently usable package capabilities; each includes its tests and documentation.

## Scope and Activation

Deliver BW01-BW08 inside the existing scheduler/pubsub packages and their Postgres
adapters. Preserve opt-in compatibility and the exact stored/claim contracts in
the approved model. Do not create another transport, workflow engine or domain
receipt abstraction. This delivery has no prerequisite in a consumer repository.

Before execution verify canonical `dev`, the committed/pushed artifacts and actual
PR/worktree/controller ownership. Keep the primary checkout on `dev`; use a dedicated
worktree and the recorded branch for each slice. The controller performs the
authorized Rebase + Fast-forward merges into `dev`. Necessary same-set corrections
are authorized; retain the selected checks and controller repair path. No main
alignment, tag, release or mirror publication is authorized.

## Ordered Slices

| Slice | Short name | Exact branch | Expected PR title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | Recover leased scheduled runs | `feat/leased-scheduler-recovery` | `feat(slice-1): recover interrupted scheduled work` | `ops/default/report/slices/durable-background-work/slice-1-leased-runs.md` |
| Slice 2 | Deliver bounded durable events | `feat/bounded-durable-pubsub` | `feat(slice-2): deliver bounded durable events` | `ops/default/report/slices/durable-background-work/slice-2-durable-events.md` |
| Slice 3 | Operate retained background work | `feat/durable-work-operations` | `feat(slice-3): operate retained background work` | `ops/default/report/slices/durable-background-work/slice-3-work-operations.md` |

## Tasks and Expected Commits

| Task | Deliverable and validation responsibility | Expected commit |
| --- | --- | --- |
| T1.1 | Implement opt-in durable policy/capability, atomic Postgres run claims, frozen budgets, lease recovery and fenced completion/recurrence. Preserve legacy stores and reject unfenced writes to leased rows; add focused real PostgreSQL concurrency/restart tests with short controlled time. | `feat(scheduler): recover leased job attempts` |
| T1.2 | Complete scheduled-run inspection/terminal pruning and quiesced legacy-adoption guidance. Update scheduler reference/how-to/tutorial material, verify cancellation/legacy behavior and introduce the thin gate's first mode with an owned native PostgreSQL fixture. | `feat(scheduler): expose durable run operations` |
| T2.1 | Implement bounded-mode admission, canonical idempotent publication and caller-owned transaction publication. Enforce payload/metadata/identity/topic/subscriber/message-capacity limits atomically, with additive schema and actual rollback/concurrent-admission coverage. | `feat(pubsub): publish bounded messages transactionally` |
| T2.2 | Implement fixed subscriber policy, history admission, leased per-subscriber delivery, retained attempts and terminal failures. Preserve late-commit/fan-out guarantees and bound shutdown. Update pubsub guidance and mode 2; verify two workers, stale ack, exhausted messages and later-message progress. | `feat(pubsub): fence durable subscriber attempts` |
| T3.1 | Add bounded delivery inspection, compare-and-set replay, subscriber retirement and fan-out-aware terminal pruning/capacity accounting. Verify pending preservation, lifetime budgets, migration compatibility and stale replay in existing real PostgreSQL fixtures. | `feat(postgres): inspect and retire durable work` |
| T3.2 | Finish linked four-quadrant operating/upgrade guidance, package contracts and BW01-BW08 evidence. Enable the selected integrated gate, measure its cost and complete the canonical slice report; no separate evidence-verification framework. | `docs(background): guide durable work recovery` |

Tests of changed behavior belong to its implementing task. Each slice includes
its usable package/API guidance and updates Unreleased for completed user-visible
behavior. Do not reserve correctness or compatibility for the final task.

## Validation Selection

Introduce a thin `scripts/check-durable-background-work.sh`. `slice 1`, `slice 2`
and `slice 3` execute only that slice's affected checks and direct regressions;
no-argument mode is the integrated gate and becomes available in Slice 3. Do not
chain earlier concern scripts or rerun all preceding slice modes.

Use the current `testhelper.SetupTestDB` schema isolation with an owned native
PostgreSQL instance. Set `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
explicitly so tests use that instance instead of their local container fallback.
Reuse the existing native fixture pattern, including a private socket/run root,
finite startup and owned cleanup. Do not invoke an older aggregate gate or depend
on its receipt tool. No production service is reused or stopped.

| Mode | Runtime check selection |
| --- | --- |
| Slice 1 | `GOWORK=off go test -race -count=1 -p=2 -timeout=180s ./scheduler ./scheduler/postgres` |
| Slice 2 | `GOWORK=off go test -race -count=1 -p=2 -timeout=180s ./pubsub ./pubsub/postgres` |
| Slice 3 | Selected retained-work/lease/delivery roots in the four affected packages; use the families below with explicit final names recorded before validation |

The Slice 3 selection is `^(TestWorkOperations|TestLeasedRuns|TestBoundedDeliveries)$`.
Introduce/extend these meaningful root families with their owning behavior; they are
planned names, not proof of implementation. A required absent/skipped family fails
validation. Reuse equivalent existing coverage rather than duplicating its cases.
Each mode also runs affected-package vet, strict lint, documentation and whitespace
checks. Verify modified Go formatting before commit.

The integrated wrapper executes, in order, against one owned PostgreSQL fixture:

1. `GOWORK=off go vet ./scheduler ./scheduler/postgres ./pubsub ./pubsub/postgres`
2. `GOWORK=off go test -race -count=1 -p=2 -timeout=180s ./scheduler ./scheduler/postgres ./pubsub ./pubsub/postgres`
3. `GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./scheduler ./scheduler/postgres ./pubsub ./pubsub/postgres`
4. `make docs-check`
5. Owned PostgreSQL cleanup and `git diff --check`.

This protects real schema/transaction/lease behavior and current direct package
compatibility. Use one batched race execution, short controlled clocks/expiry and
observable conditions instead of production-length sleeps or one process per case.
The wrapper may retain native command output and require named roots to pass; do
not add JSON evidence models, source-hash inventories or tests of operational prose.

Expected warm-cache cost is 1–3 minutes, an estimate requiring measurement on the
first run. Record actual per-command elapsed time in the slice report. Investigate
unexpected cost before increasing timeouts or expanding selection. Full repository
tests and `make check` are not this set's routine gate. No runtime test is required
for the current Markdown-only planning work.

## Integration, Repair and Closure

Use the exact recorded PR title and canonical Summary/Changes/Validation body, with
the immutable report-introduction commit link last under Changes. One PR per slice,
targeting `dev`; one meaningful task commit per recorded task where practical.

When execution is authorized, run the selected slice mode through the controller's
exact-HEAD validation and retain its native evidence. After the last integration,
the controller runs the unchanged no-argument gate. Necessary same-set repairs use
focused reproduction and the existing correction-PR path; approval of unrelated
scope is not implied. Respect controller ownership and frozen configuration.

Record BW01-BW08 acceptance in this tracker and the existing slice reports. Close
only the spec/model/plan/tracker/reports after the exact candidate passes. Validate
closure links and whitespace without rerunning unchanged runtime solely for prose.
Publish the delivered canonical `dev` revision and verify its Go pseudo-version for
consumers; main/tag publication is a separate operation, not a dependency prerequisite.
