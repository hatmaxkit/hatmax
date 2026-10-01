<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Scheduler Slot Completion

Status: reviewing
Ticket: [TKT-20260930211808](../ticket/reviewing/20260930211808-advance-or-retire-completed-scheduler-slots.md)
Branch: `fix/ticket-20260930211808-scheduler-slots`
PR: pending

## Delivered Behavior

- Terminal store operations record the run and its schedule transition together. PostgreSQL uses a transaction and row locks; the fake uses its existing mutex.
- One-shot jobs retire. Recurring jobs advance from completion time using the existing Daily, Weekly, and Interval schedules, including after handler failure, panic, or an unknown task type. Missed occurrences are skipped, not replayed.
- Due batches exclude already claimed slots in any run state. The fake now enforces both run-ID and job-slot uniqueness and matches PostgreSQL's due-time ordering.
- Duplicate completion is a no-op. Explicit manual rescheduling or disabling while a handler runs is preserved. Terminal write errors are logged and do not falsely assert completion.

## Contracts and Ownership

Slot progression belongs to JobStore, through MarkSuccess and MarkFailed, because the result and schedule must be one durable transition. The runner invokes those terminal operations rather than issuing a separate UpdateNextRun that could fail after a successful result write. Custom stores must implement the same terminal contract; method signatures are unchanged.

Job now carries an optional Schedule for in-memory and custom stores. The PostgreSQL adapter decodes the existing schedule_spec text and schedule_tz fields into the existing schedule types. Empty text means one-shot work; supported JSON rules are documented in the reference. Non-empty opaque specifications must be converted. Invalid specifications fail due selection before handler execution. No schema migration, new dependency, or compatibility layer is required.

The current batch and worker limits remain unchanged. Terminal completion locks one job and one run; due exclusion uses the existing unique slot index. No worker loops, retries, leases, queues, or catch-up batches were added. Fake-store selection scans its retained in-memory runs and sorts the eligible jobs before applying the batch limit; that fake history remains caller-owned test data.

Interrupted claims remain excluded rather than being retried automatically. Applications must inspect and repair them before rescheduling later work. A next recurring occurrence is not a retry of a failed slot. Handler side effects are not transactional and this change does not promise exactly-once effects. RetryAttempts and RetryBackoff remain inactive.

The scheduler reference, background-job how-to, User Guide, package note, and Unreleased describe completion ownership, recurrence, the stored format, and these limits.

## Validation

All Go checks used `GOTOOLCHAIN=go1.26.7`. Database tests used an isolated native PostgreSQL 18.6 cluster on loopback, stopped after validation. These are local results, not a claim that PostgreSQL 16 CI ran.

- Before the fix, `go test ./scheduler -run '^(TestOneShotSlot|TestSlotUnique)$' -count=1` failed: one-shot work executed twice and duplicate job-slot claims were accepted.
- `go test -race ./scheduler/... -count=1 -timeout=60s`: passed.
- `go test -race ./scheduler/... -count=20 -timeout=60s`: passed all 20 repetitions, including the existing worker-panic regressions and real PostgreSQL slot transitions.
- `make check`: passed, including licensing, formatting, vet, all tests, coverage, and strict lint. Total coverage: 80.3%; scheduler: 94.4%; scheduler/postgres: 92.5%.
- `make docs-check`: passed.
- `git diff --check`: passed.

The shared fake/PostgreSQL matrix covers one-shot, interval, daily, and weekly jobs after success, error, panic, and unknown-task outcomes. Recurring cases execute two distinct slots and reject replay of the first. Concurrent claims allow one winner; a claimed slot does not starve the next limited batch. Additional cases verify duplicate completion, explicit rescheduling, disabling, malformed schedules, timezone decoding, terminal error visibility, and rollback of both database changes when result persistence fails.

## Boundary

Only finding F9 is addressed. Same-slot retries, abandoned-claim recovery, catch-up policies, scheduler shutdown and cancellation, JSON output encoding failures, and other architecture-review findings remain unchanged.
