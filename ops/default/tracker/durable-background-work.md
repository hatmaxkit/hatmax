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
Active slice: None (all recorded slices integrated)
Next slice: None
Active tasks: None
Execution gate: Awaiting controller-owned integrated validation

## Slice Status

| Slice | Short name | Status | Branch | Expected PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Recover leased scheduled runs | delivered | `feat/leased-scheduler-recovery` | `feat(slice-1): recover interrupted scheduled work` | [#129](https://forge.adrianpk.com/hatmax/hatmax/pulls/129) | `ops/default/report/slices/durable-background-work/slice-1-leased-runs.md` |
| Slice 2 | Deliver bounded durable events | delivered | `feat/bounded-durable-pubsub` | `feat(slice-2): deliver bounded durable events` | [#130](https://forge.adrianpk.com/hatmax/hatmax/pulls/130) | `ops/default/report/slices/durable-background-work/slice-2-durable-events.md` |
| Slice 3 | Operate retained background work | delivered | `feat/durable-work-operations` | `feat(slice-3): operate retained background work` | [#131](https://forge.adrianpk.com/hatmax/hatmax/pulls/131) | `ops/default/report/slices/durable-background-work/slice-3-work-operations.md` |

## Tasks

| Task | Status | Expected commit | Commit |
| --- | --- | --- | --- |
| T1.1 | completed | `feat(scheduler): recover leased job attempts` | `fe26565b273132c904da05521fde56e176d4a0e8` |
| T1.2 | completed | `feat(scheduler): expose durable run operations` | `8fe8b7406b3dbf359db0e90edb2c24415b732b7a` |
| T2.1 | completed | `feat(pubsub): publish bounded messages transactionally` | `ab080c841c2b372a1b6e51a7f2180448bc91b70d` |
| T2.2 | completed | `feat(pubsub): fence durable subscriber attempts` | `c52d6cb2eb8e7c9a6247fcee1faacb74037ae4b3` |
| T3.1 | completed | `feat(postgres): inspect and retire durable work` | `e688a1c11d9b5224efae4f8cc1a37cef05de29ff` |
| T3.2 | completed | `docs(background): guide durable work recovery` | `a9b2ee74bb2a8f8e53e287b22ad8dfe7a3f76438` |

The plan owns each task's deliverables and selected gate. The controller owns
implementation activation, exact-HEAD validation and integration.

## Prerequisites and Acceptance

- [x] Specification and durable model approved on 2026-10-18.
- [x] Three-slice/six-task plan and matching tracker prepared.
- [x] Approved artifacts committed/pushed and execution explicitly started.
- [x] Three PRs and six tasks implemented and integrated into `dev`.
- [ ] Selected gate passed at the exact integrated candidate, with owned cleanup.
- [ ] Acceptance/report metadata closed and published module identity verified.

| Criterion | Owning slices | Evidence status |
| --- | --- | --- |
| BW01 | Slice 1 | Slice 1 controller checks passed and PR #129 integrated at `80b183c0e50a6133598ca1abe04b3930b8c0573b` |
| BW02 | Slice 1, Slice 2 | scheduler and pubsub replicas/killed-worker/stale-claim checks passed; PRs #129/#130 integrated |
| BW03 | Slice 1 | Slice 1 controller checks passed and PR #129 integrated at `80b183c0e50a6133598ca1abe04b3930b8c0573b` |
| BW04 | Slice 2 | Slice 2 controller checks passed and PR #130 integrated at `ebba2ebaf991d3b49681f835e2629b191059a4f2` |
| BW05 | Slice 2 | Slice 2 controller checks passed and PR #130 integrated at `ebba2ebaf991d3b49681f835e2629b191059a4f2` |
| BW06 | Slice 3 | Slice 3 controller checks passed and PR #131 integrated at `889805514800e1550bdb3a3564b98b136b5eea67`; replay revision, concurrent winner, acknowledgement exclusion and lifetime limits covered; integrated gate pending |
| BW07 | Slice 2, Slice 3 | focused admission, fan-out retention, retirement, capacity rollback and concurrent prune/publication checks passed; integrated gate pending |
| BW08 | Slice 1, Slice 3 | bounded scheduler/delivery inspection and legacy upgrade/retention checks passed; linked operating guidance complete; integrated gate pending |

## Gate Evidence and Closure

The final command will be `GOWORK=off scripts/check-durable-background-work.sh`.
Its selected commands, native PostgreSQL ownership and estimated cost are defined
in the plan. Current gate status: not run. A required family with no execution or
skipped evidence is not acceptance. Integrated runtime cost: not measured. Slice 3 selected commands totaled 16.36 seconds at `a9b2ee74bb2a8f8e53e287b22ad8dfe7a3f76438`; see the Slice 3 report for native per-command timings.

All recorded slices are integrated; PR #131 merged at
`889805514800e1550bdb3a3564b98b136b5eea67`. No active slice or task remains.
The specification/model and delivery-set acceptance remain open until the
controller records the integrated gate result.

Gate prerequisites use the installed Go toolchain, strict linter and native
PostgreSQL tools. The wrapper creates its own private cluster/socket, sets every
required database variable, verifies readiness and stops only that owned fixture.
Its run logs and lint cache are ignored in the primary checkout. No persistent
fixture or production service is required between preparation and gate execution.

Use the existing controller command receipts and slice reports for the actual
candidate, elapsed time and BW01-BW08 results; add no parallel ledger or receipt
parser. Closure changes only approved delivery documents, with focused links and
whitespace checks. No successor or public release is started by planning.
