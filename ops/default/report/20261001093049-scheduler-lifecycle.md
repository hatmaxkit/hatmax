<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Scheduler Lifecycle

Status: reviewing
Ticket: [TKT-20260930211809](../ticket/reviewing/20260930211809-bound-and-idempotently-stop-the-scheduler.md)
Branch: `fix/ticket-20260930211809-scheduler-lifecycle`
PR: pending

## Delivered Behavior

- An enabled runner owns one polling loop with synchronized idle, running, stopping, and stopped states. Repeated active starts are no-ops; concurrent starts cannot duplicate that loop.
- Canceled or expired startup returns the context error without consuming the idle instance. Cancellation of the first successful startup context, or an active Stop, cancels polling and execution. Later starts return ErrStopped once shutdown begins; restarting requires a new instance.
- Stop before startup is a no-op. Repeated and concurrent stops wait on the same completion signal. Each caller can cancel or bound its wait with a deadline; a timed-out shutdown can be joined again. Completed shutdown returns nil even with an expired wait context.
- Batch admission checks cancellation before querying, after the query, between sequential jobs, and while acquiring worker capacity. Already admitted handlers are joined. Direct Tick calls remain independent and caller-owned.

## Contracts and Ownership

The runner owns its background execution context and completion channel. A separate lifecycle mutex protects state transitions without conflating handler registration. Only one loop may close its completion signal. Stop waits directly on that signal; it does not launch a waiter goroutine or use a WaitGroup concurrently with Start.

The first successful Start's context owns the loop. Repeated starts do not replace it. Stop cancels active work even when the stop caller already has a canceled context; that caller's context controls only how long it waits. ErrStopped is the public restart rejection. Public method signatures, JobStore, schema, configuration, and worker/batch limits are unchanged.

Cancellation is cooperative. Go cannot forcibly terminate a handler or adapter that ignores its context. A Stop timeout is not proof that execution ended or a terminal result was saved. The caller must keep dependencies available and join shutdown before treating it as complete. Stores receive the canceled execution context, not a detached or unbounded cleanup context. PostgreSQL claims whose terminal write is interrupted remain unfinished and excluded from due batches under the existing slot contract; explicit repair remains application-owned.

Manual Tick calls are not background-loop children. Their callers must cancel and join them separately. Cancellation observed during admission prevents further claims, but does not undo an admitted handler's side effects or make them transactional.

The scheduler reference, background-job how-to, User Guide, package note, and Unreleased now state the lifecycle, deadline, restart, and interrupted-persistence boundaries.

## Validation

All Go checks used `GOTOOLCHAIN=go1.26.7`. Database tests used an isolated native PostgreSQL 18.6 cluster on loopback, stopped after validation. These are local results, not evidence that PostgreSQL 16 CI ran.

- Before the fix, `go test ./scheduler -run '^(TestStartCanceled|TestStopRepeat)$' -count=1 -timeout=15s` failed: canceled startup returned nil and repeated cleanup panicked with close of closed channel.
- `go test -race ./scheduler/... -count=20 -timeout=60s`: passed all 20 repetitions, including worker-panic, atomic slot-completion, and real PostgreSQL cancellation regressions.
- `make check`: passed licensing, formatting, vet, all tests, coverage, and strict lint. Total coverage: 80.3%; scheduler: 95.0%; scheduler/postgres: 93.5%.
- `make docs-check`: passed.
- `git diff --check`: passed.

Lifecycle tests cover cleanup before startup, disabled startup, rejected canceled/expired startup, concurrent and racing starts/stops, startup-context cancellation without Stop, restart rejection during and after shutdown, canceled stop callers, deadline-limited waits on non-cooperative handlers, completed-shutdown precedence, and joining an admitted batch after cancellation. Single-worker and parallel cases verify unadmitted slots remain unclaimed. PostgreSQL tests verify canceled execution cannot fabricate durable completion and interrupted claims do not block later work.

## Boundary

Only finding F10 is addressed. No forced handler termination, detached persistence cleanup, same-slot retry, abandoned-claim recovery, restart support on the same instance, or unrelated architecture-review correction is added.
