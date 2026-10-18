<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Durable Background Work Delivery Tracker

Date: 2026-10-18
Status: Approved
Approved: 2026-10-18
Specification/model approved: 2026-10-18
Planning authorized: 2026-10-18
Delivery set: durable-background-work
Concern: [Durable background work](../spec/durable-background-work.md)
Model: [Durable work model](../spec/durable-background-work-model.md)
Plan: [Delivery plan](../plan/durable-background-work.md)
Base branch: `dev`
Planning base: `b21af0b17c01ebb3544c37d7d200d2c87f4db0b1`
Execution authorized: 2026-10-18
Integration authority: Controller-owned Rebase + Fast-forward into `dev`
Active slice: Slice 1
Next slice: Slice 2
Active tasks: None (Slice 1 awaiting controller integration)
Execution gate: Approved artifacts published; controller-owned implementation active

## Slice Status

| Slice | Short name | Status | Branch | Expected PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Recover leased scheduled runs | reviewing | `feat/leased-scheduler-recovery` | `feat(slice-1): recover interrupted scheduled work` | [#129](https://forge.adrianpk.com/hatmax/hatmax/pulls/129) | `ops/default/report/slices/durable-background-work/slice-1-leased-runs.md` |
| Slice 2 | Deliver bounded durable events | pending | `feat/bounded-durable-pubsub` | `feat(slice-2): deliver bounded durable events` | pending | `ops/default/report/slices/durable-background-work/slice-2-durable-events.md` |
| Slice 3 | Operate retained background work | pending | `feat/durable-work-operations` | `feat(slice-3): operate retained background work` | pending | `ops/default/report/slices/durable-background-work/slice-3-work-operations.md` |

## Tasks

| Task | Status | Expected commit | Commit |
| --- | --- | --- | --- |
| T1.1 | completed | `feat(scheduler): recover leased job attempts` | `fe26565b273132c904da05521fde56e176d4a0e8` |
| T1.2 | completed | `feat(scheduler): expose durable run operations` | `8fe8b7406b3dbf359db0e90edb2c24415b732b7a` |
| T2.1 | pending | `feat(pubsub): publish bounded messages transactionally` | pending |
| T2.2 | pending | `feat(pubsub): fence durable subscriber attempts` | pending |
| T3.1 | pending | `feat(postgres): inspect and retire durable work` | pending |
| T3.2 | pending | `docs(background): guide durable work recovery` | pending |

The plan owns each task's deliverables and selected gate. The controller owns
implementation activation, exact-HEAD validation and integration.

## Prerequisites and Acceptance

- [x] Specification and durable model approved on 2026-10-18.
- [x] Three-slice/six-task plan and matching tracker prepared.
- [x] Approved artifacts committed/pushed and execution explicitly started.
- [ ] Three PRs and six tasks implemented and integrated into `dev`.
- [ ] Selected gate passed at the exact integrated candidate, with owned cleanup.
- [ ] Acceptance/report metadata closed and published module identity verified.

| Criterion | Owning slices | Evidence status |
| --- | --- | --- |
| BW01 | Slice 1 | focused capability/legacy/fencing checks passed; exact-HEAD slice receipt required |
| BW02 | Slice 1, Slice 2 | scheduler replicas/killed-worker/stale-claim checks passed; pubsub pending |
| BW03 | Slice 1 | focused retry/exhaustion/recurrence/shutdown checks passed; exact-HEAD slice receipt required |
| BW04 | Slice 2 | pending |
| BW05 | Slice 2 | pending |
| BW06 | Slice 3 | pending |
| BW07 | Slice 2, Slice 3 | pending |
| BW08 | Slice 1, Slice 3 | scheduler inspection/migration/diagnostics checks passed; final operations pending |

## Gate Evidence and Closure

The final command will be `GOWORK=off scripts/check-durable-background-work.sh`.
Its selected commands, native PostgreSQL ownership and estimated cost are defined
in the plan. Current gate status: not run. A required family with no execution or
skipped evidence is not acceptance. Observed runtime cost: not measured.

Use the existing controller command receipts and slice reports for the actual
candidate, elapsed time and BW01-BW08 results; add no parallel ledger or receipt
parser. Closure changes only approved delivery documents, with focused links and
whitespace checks. No successor or public release is started by planning.
