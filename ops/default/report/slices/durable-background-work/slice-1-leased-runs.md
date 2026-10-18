<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Recover leased scheduled runs

Status: delivered
Delivery set: durable-background-work
Plan: [Delivery plan](../../../plan/durable-background-work.md)
Tracker: [Delivery tracker](../../../tracker/durable-background-work.md)
Branch: `feat/leased-scheduler-recovery`
PR: `#129`

## Purpose

Recover interrupted scheduled attempts with finite retained budgets and fenced
completion while preserving existing custom stores and legacy constructors.

## Delivered Behavior

- Explicit durable execution validates capability and policy before work starts.
  PostgreSQL creates the initial running attempt atomically and reclaims due retries
  or expired leases with the same run/slot and a fresh token.
- Runs freeze deadlines, lease duration, retry delay and total attempt budget.
  Terminal completion and recurrence/one-shot retirement commit together. Old
  ownership and legacy mutation methods cannot change leased rows.
- Cooperative handler deadlines retain bounded failure codes and JSON output;
  raw errors and panic values are not saved for durable work. Cancellation and
  shutdown retain the existing caller-owned lifetime boundaries.
- Inspection uses bounded keyset pages without payloads, tokens or raw diagnostics.
  Explicit pruning preserves unfinished work. Revision-matched legacy adoption
  requires quiesced workers and preserves consumed attempts and future waits.

Merged into `dev` through controller-owned PR #129 at
`80b183c0e50a6133598ca1abe04b3930b8c0573b`; the exact ordered Slice 1 checks
and whitespace check passed at that head.

## Implementation Notes

T1.1 is `fe26565b273132c904da05521fde56e176d4a0e8`.
T1.2 is `8fe8b7406b3dbf359db0e90edb2c24415b732b7a` and adds run operations,
operating guidance, Unreleased notes and the thin Slice 1 validation mode.
The controller owns Rebase + Fast-forward integration into `dev`.

The first claim persists the default policy on an unconfigured job. Explicit job
configuration applies to future slots; active runs retain their snapshot. Claims
lock job before run and use PostgreSQL time after obtaining those locks. Polling
closes finite rows before invoking handlers. No handler replacement or heartbeat
goroutine is introduced.

## Contracts Added or Changed

`ExecutionPolicy`, `WithDurableExecution`, `DurableStore`, `RunClaim`, classified
failure codes and capability/policy/stale-ownership errors supplement `JobStore`.
PostgreSQL adds conditional durable columns/constraints and lease/retry/terminal
indexes. Existing legacy rows are not automatically adopted or rewritten.

`ConfigureDurable`, `ListRuns`, `PruneRuns` and `AdoptLegacyRun` expose bounded
backend operations. Inspection defaults to 50 rows and caps at 100; pruning caps
at 1000 terminal rows per call. Applications own operator access and effect deduplication.

## Files of Interest

- `scheduler/durable.go`: policy, opt-in capability and cooperative invocation.
- `scheduler/postgres/durable.go`: atomic claims, fenced writes and recurrence.
- `scheduler/postgres/operations.go`: inspection, adoption and retention.
- `scheduler/postgres/schema.go`: additive migration and conditional constraints.
- `scheduler/postgres/durable_test.go`, `operations_test.go`: real PostgreSQL boundaries.
- `docs/how-to/run-background-jobs/README.md`: upgrade and operating procedure.
- `scripts/check-durable-background-work.sh`: selected Slice 1 mode and native fixture.

## Validation

Passed focused local checks:

- `scripts/durable-background-work/postgres-fixture.sh .tmp/durable-background-work/focused env GOWORK=off go test -race -count=1 -p=2 -timeout=60s ./scheduler ./scheduler/postgres -run '^(TestLeasedRuns|TestRetrySchema)$'`
- `scripts/durable-background-work/postgres-fixture.sh .tmp/durable-background-work/focused env GOWORK=off go test -race -count=1 -p=2 -timeout=60s ./scheduler/postgres -run '^TestWorkOperations$'`
- `GOWORK=off GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --fix --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./scheduler ./scheduler/postgres`
- `GOWORK=off make docs-check`
- `bash -n scripts/check-durable-background-work.sh scripts/durable-background-work/postgres-fixture.sh`
- `git diff --check`

The focused leased/retry packages reported 1.016 s and 2.559 s; operations reported
1.285 s. Native owned PostgreSQL shutdown/root removal succeeded after both runs.
These are focused checks, not the delivery-set gate.

The controller passed `GOWORK=off scripts/check-durable-background-work.sh slice 1`
and `git diff --check` at `ccc5fac4a704e186d8512a806b2b2ec00eac6e42` before this
evidence update. Measured native wall times were:

| Selected command | Seconds |
| --- | --- |
| `GOWORK=off go vet ./scheduler ./scheduler/postgres` | 0.31 |
| `GOWORK=off go test -race -count=1 -p=2 -timeout=180s ./scheduler ./scheduler/postgres` | 4.89 |
| `GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./scheduler ./scheduler/postgres` | 0.71 |
| `make docs-check` | 3.99 |

The owned native PostgreSQL fixture stopped and removed its temporary root.
These timings are below the estimated 1–3 minute warm-cache budget. The controller
regenerates the same ordered slice-command/whitespace receipt after this report
commit; only the receipt for the final clean PR HEAD authorizes integration.

BW01: legacy suites plus capability rejection and unfenced-write tests.
BW02 (scheduler): concurrent stores, an actually killed worker, lease reclamation
and stale-token/identity/attempt rejection. BW03: persisted waits, frozen policy,
exhaustion, atomic recurrence and cooperative shutdown. BW08 (scheduler): bounded
inspection, classified diagnostics, retention, repeated schema upgrade and explicit
legacy adoption. Pubsub and final integrated acceptance remain with their owning slices.

## Risks and Follow-ups

Handlers must honor cancellation and protect repeated business effects. An expired
lease cannot terminate a previous worker. Old workers must be quiesced before
migration/adoption. A timeout during shutdown leaves outstanding work honestly
running until return or lease recovery. No external effect is claimed exactly once.
