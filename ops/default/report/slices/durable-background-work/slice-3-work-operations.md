<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 3: Operate retained background work

Status: delivered
Delivery set: durable-background-work
Plan: [Delivery plan](../../../plan/durable-background-work.md)
Tracker: [Delivery tracker](../../../tracker/durable-background-work.md)
Branch: `feat/durable-work-operations`
PR: `#131`

## Purpose

Operate retained event deliveries through bounded inspection, revision-checked
replay, explicit subscriber retirement and fan-out-aware message retention.
Complete the linked operating guidance and enable the selected integrated gate.

## Delivered Behavior

- Delivery inspection returns stable keyset pages of pending, running, expired,
  retry-waiting and terminal work without payloads, metadata or ownership tokens.
  Unclaimed work has zero consumed attempts and no invented policy history.
- Replay matches the observed failed attempt and update timestamp, rejects
  acknowledged or stale work and grants 1–8 additional attempts within the
  lifetime maximum of 32. Concurrent requests cannot multiply one grant.
- Retirement refuses unfinished entitlement unless explicitly discarded. It
  fences claims, preserves consumed attempts and reserves the original identity
  without inventing success acknowledgements or unclaimed attempts.
- Bounded pruning preserves unfinished fan-out and recent terminal outcomes.
  Message deletion, tracking deletion and capacity release commit together;
  concurrent replay cannot turn a selected failure into deleted active work.
- Linked tutorial, how-to, reference, explanation and package guidance cover
  inspection, replay, retirement, retention, migration and cooperative handlers.
- Slice 3 selects the recorded leased-run, bounded-delivery and operations roots
  across four packages. No-argument mode enables the integrated gate and rejects
  absent or skipped required families using native verbose test output.

## Implementation Notes

T3.1 is `e688a1c11d9b5224efae4f8cc1a37cef05de29ff`.
T3.2 is `a9b2ee74bb2a8f8e53e287b22ad8dfe7a3f76438` and introduces this report,
operating guidance and the final gate modes. Merged Slice 2 PR #130 is closed in
its existing report/tracker entries at `ebba2ebaf991d3b49681f835e2629b191059a4f2`.
Merged into `dev` through controller-owned PR #131 at
`889805514800e1550bdb3a3564b98b136b5eea67`. The required ordered Slice 3 checks and whitespace
check passed at that head. The controller also passed the final no-argument gate
at `337f4666714f9bdc3a1e08b3cd0cf33643ddcd1a`.

Replay follows the claim lock order: subscriber, message, delivery. Pruning locks
candidate messages and rechecks eligibility in a fresh READ COMMITTED snapshot
before deletion, because replay can change a delivery while a candidate waits for
its message lock. Entitlement changes serialize with pruning. Retirement also
serializes its boundary with publication. Message deletion releases the existing
capacity slot transactionally rather than maintaining another counter.

Retired subscribers remain tombstones with their original topic and count toward
the 64-name bound. Legacy subscribers awaiting explicit admission still protect
their retained entitlement; unbounded legacy messages are not pruned. Additive
nullable retirement state leaves existing subscriptions active.

## Contracts Added or Changed

`ListDeliveries`, `DeliveryQuery`, `DeliveryCursor` and `DeliveryInfo` provide
50-row default/100-row maximum diagnostic pages. `ReplayDelivery` and
`ReplayRequest` require an observed revision, a finite grant and a classified
reason (`operator_retry`, `dependency_restored` or `data_repaired`). Replay
preserves consumed counters and frozen timing policy.

`RetireSubscriber` and `RetireOptions.DiscardPending` make unfinished entitlement
an explicit decision. Retired names cannot reopen or change configuration.
`PruneMessages` accepts a nonzero cutoff and batch limit 1–1000. It removes only
eligible bounded history and excludes active/retry work. Publication identity
conflicts and idempotency remain defined over retained identities; applications
own longer-lived business deduplication and operator access.

## Files of Interest

- `pubsub/postgres/work_operations.go`: inspection, replay, retirement and retention.
- `pubsub/postgres/work_operations_test.go`: real PostgreSQL operations and races.
- `pubsub/postgres/schema.go`: additive retirement state and retention lookup index.
- `pubsub/postgres/subscription.go`: reserved retired identities and live claims.
- `docs/how-to/use-pubsub/README.md`: concrete operator and upgrade procedures.
- `docs/reference/pubsub/README.md`: exact bounds and retained-work contracts.
- `scripts/check-durable-background-work.sh`: selected and integrated gate modes.

## Validation

Passed focused checks before the final controller receipt:

- `scripts/durable-background-work/postgres-fixture.sh .tmp/durable-background-work/focused env GOWORK=off go test -race -count=1 -p=2 -timeout=90s ./pubsub/postgres -run '^TestWorkOperations$'`
- `GOWORK=off GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --fix --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./pubsub ./pubsub/postgres`
- `GOWORK=off make docs-check`
- `bash -n scripts/check-durable-background-work.sh`
- `git diff --check`

The focused PostgreSQL race execution reported 2.014 seconds. Its owned native
PostgreSQL fixture stopped and removed its temporary root. The documentation
check caught a file-update error; restoring each page's original content and
applying its intended addition resolved the broken local links.

The controller passed `GOWORK=off scripts/check-durable-background-work.sh slice 3`
and `git diff --check` at `a9b2ee74bb2a8f8e53e287b22ad8dfe7a3f76438` before this
PR-reference/evidence update. All six required roots passed with no skipped
required families. Native measured wall times were:

| Selected command | Seconds |
| --- | --- |
| `GOWORK=off go vet ./scheduler ./scheduler/postgres ./pubsub ./pubsub/postgres` | 0.37 |
| `GOWORK=off go test -v -race -count=1 -p=2 -timeout=180s -run '^(TestWorkOperations|TestLeasedRuns|TestBoundedDeliveries)$' ./scheduler ./scheduler/postgres ./pubsub ./pubsub/postgres` | 8.00 |
| `GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./scheduler ./scheduler/postgres ./pubsub ./pubsub/postgres` | 1.99 |
| `make docs-check` | 6.00 |

The owned native PostgreSQL fixture stopped and removed its temporary root.
Selected commands totaled 16.36 seconds, below the estimated warm-cache budget.
The controller regenerated the same ordered receipt for the final clean PR HEAD
`889805514800e1550bdb3a3564b98b136b5eea67` before integration.

The controller passed `GOWORK=off scripts/check-durable-background-work.sh` at
immutable integrated candidate `337f4666714f9bdc3a1e08b3cd0cf33643ddcd1a`.
Its retained native output confirms all four full package suites and all six
required roots passed, with zero lint issues, documentation and whitespace success.
The intentional `TestPanicProcess` helper skip does not skip a required family.
Final native wall times were:

| Integrated command | Seconds |
| --- | --- |
| `GOWORK=off go vet ./scheduler ./scheduler/postgres ./pubsub ./pubsub/postgres` | 0.41 |
| `GOWORK=off go test -v -race -count=1 -p=2 -timeout=180s ./scheduler ./scheduler/postgres ./pubsub ./pubsub/postgres` | 12.90 |
| `GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./scheduler ./scheduler/postgres ./pubsub ./pubsub/postgres` | 12.48 |
| `make docs-check` | 6.61 |

Timed commands totaled 32.40 seconds, below the 1–3 minute estimate. Owned
PostgreSQL stopped and removed its temporary root; no fixture remains to stop.

`GOWORK=off GOPROXY=direct go list -m -json hatmax.adrianpk.com@337f4666714f9bdc3a1e08b3cd0cf33643ddcd1a`
resolved `v0.6.1-0.20261018191056-337f4666714f` from the canonical Forgejo repository,
with the exact tested hash on `dev`. Closure changes only configured documentation;
the tested runtime and gate remain unchanged.

BW01: Slice 1 protects legacy construction and unfenced writes. BW02: the recorded
scheduler/pubsub families exercise competing replicas, killed workers and stale
claims. BW03: scheduler retries, exhaustion and cancellation retain their budgets.
BW04: publication commit/rollback and identity conflicts remain covered by Slice 2.
BW05: independent subscribers preserve late commits and progress beyond failures.
BW06: operations tests exercise stale/concurrent replay, acknowledged exclusions
and actual attempts through the lifetime maximum. BW07: operations tests extend
admission evidence with fan-out retention, recent-terminal protection, capacity
rollback, concurrent publication, retirement and replay/prune races. BW08: bounded
inspection, safe diagnostics and legacy upgrade/retention match the linked
operating guidance. Final integrated BW01–BW08 acceptance passed at the candidate above.

## Risks and Follow-ups

Handler cancellation is cooperative; fencing cannot undo an external effect.
Applications own effect deduplication, operator authority and retention cutoffs.
Explicit retirement can discard unfinished entitlement and permanently reserves
the subscriber identity. Quiesce legacy workers before admission upgrades.
