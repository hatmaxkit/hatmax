<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Durable Background Work Delivery Tracker

Date: 2026-10-18
Status: Completed
Approved: 2026-10-18
Completed: 2026-10-18
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
Execution gate: Closed; BW01-BW08 accepted at `337f4666714f9bdc3a1e08b3cd0cf33643ddcd1a`

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
- [x] Selected gate passed at the exact integrated candidate, with owned cleanup.
- [x] Acceptance/report metadata closed and published module identity verified.

| Criterion | Owning slices | Evidence status |
| --- | --- | --- |
| BW01 | Slice 1 | Passed: legacy compatibility, capability rejection and unfenced-write protection |
| BW02 | Slice 1, Slice 2 | Passed: competing replicas, killed workers, reclaimed leases and stale completion/ack rejection |
| BW03 | Slice 1 | Passed: persisted retry waits, frozen budgets, exhaustion, atomic recurrence and cooperative shutdown |
| BW04 | Slice 2 | Passed: domain commit/rollback, canonical identity retries and conflicting reuse |
| BW05 | Slice 2 | Passed: independent fan-out, retained/current history, late commits, finite retries and failure quarantine |
| BW06 | Slice 3 | Passed: observed revision, concurrent replay winner, acknowledged exclusions and finite lifetime grants |
| BW07 | Slice 2, Slice 3 | Passed: input/registration/capacity bounds, fan-out retention, retirement, rollback and replay/prune/publication races |
| BW08 | Slice 1, Slice 3 | Passed: bounded safe inspection, legacy adoption/admission/retention and linked operating guidance |

## Gate Evidence and Closure

The controller passed `GOWORK=off scripts/check-durable-background-work.sh` at
`337f4666714f9bdc3a1e08b3cd0cf33643ddcd1a` on 2026-10-18. This exact immutable
candidate accepts BW01–BW08 above. The retained controller receipt records that
candidate and command; its native log confirms all four package suites, all six
required roots, vet, zero lint issues, documentation, whitespace and owned
PostgreSQL shutdown/temporary-root removal. The intentional `TestPanicProcess`
subprocess helper skip is outside the required families.

Native timed commands totaled 32.40 seconds (vet 0.41, race tests 12.90, lint 12.48,
documentation 6.61), below the 1–3 minute estimate. See the
[Slice 3 report](../report/slices/durable-background-work/slice-3-work-operations.md#validation)
for the exact selected commands and measurements.

All three slice reports are delivered, all six task commits and PRs #129–#131
are integrated, and the specification/model implementation metadata is complete.
No active task or successor remains. The gate already stopped its owned fixture;
no persistent test fixture requires shutdown.

`GOWORK=off GOPROXY=direct go list -m -json hatmax.adrianpk.com@337f4666714f9bdc3a1e08b3cd0cf33643ddcd1a`
resolved `hatmax.adrianpk.com` to
`v0.6.1-0.20261018191056-337f4666714f`, with origin
`https://forge.adrianpk.com/hatmax/hatmax.git`, `refs/heads/dev` and the exact tested
commit. Consumers can pin that validated revision.

Only configured closure documents differ from the tested candidate. Closure uses
documentation and whitespace checks without rerunning unchanged runtime or the
controller-owned full gate.
