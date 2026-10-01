<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Scheduler Handler Panic Containment

Status: reviewing
Ticket: [TKT-20260930211806](../ticket/reviewing/20260930211806-contain-scheduler-worker-panics.md)
Branch: `fix/ticket-20260930211806-scheduler-panics`
PR: pending

## Delivered Behavior

- Each synchronous handler call has a private recovery boundary on the invoking
  goroutine. String, error, and nil panic values become a failed `Result` with
  `handler panic:` detail; normal results remain unchanged.
- The existing failure path marks the run failed and records completion time.
  Later healthy jobs execute in both single-worker and concurrent batches;
  existing semaphore and wait-group defers release worker resources.
- Errors from the failed-state write are logged rather than hidden. If the store
  cannot persist that transition, the run can remain running; the runner does
  not claim durable completion or fabricate a successful run.
- The enclosing tick no longer uses broad recovery. Store panics remain visible
  and `Tick` still returns ordinary `ListDue` errors unchanged.

## Contracts and Ownership

Recovery belongs only to the application handler boundary, not scheduler state
or infrastructure operations. It handles the call on the same goroutine whether
`Tick` invokes it serially or dispatches it to a worker. Completion tracking also
distinguishes a legacy nil panic value from a normal return.

The recovery adds no goroutines, shared state, retries, queues, or dependencies.
Existing batch and worker bounds remain unchanged. It does not roll back a
handler's partial effects. Panics in handler-created goroutines, process exits,
`runtime.Goexit`, and fatal runtime failures are outside this boundary; stores,
clocks, settings, and logging are not covered by the handler recovery either.

Public signatures, schema, settings, and scheduling behavior are unchanged.
Handler defects require application diagnosis; failed-run recording does not
implement `RetryAttempts` or `RetryBackoff`. The scheduler reference, how-to,
User Guide, and `Unreleased` describe the behavior and persistence limit.

## Validation

All Go commands used `GOTOOLCHAIN=go1.26.7`. Database tests used an isolated
native PostgreSQL 18.6 cluster on loopback, stopped after validation. These are
local results, not a claim that the PostgreSQL 16 CI job ran.

- Before the fix, `go test ./scheduler -run '^TestWorkerPanic$' -count=1` failed
  both named cases. One worker recorded only the interrupted run; two workers
  terminated the child process with an unhandled handler panic.
- `go test -race ./scheduler/... -count=1 -timeout=60s`: passed, including real
  PostgreSQL failed/successful run persistence with one and two workers.
- `go test -race ./scheduler -run '^(TestWorkerPanic|TestHandlerPanic|TestStorePanic|TestStoreError|TestFailedRunWrite)$' -count=50 -timeout=60s`:
  passed all 50 repetitions, including subprocess crash/continuation checks.
- `GODEBUG=panicnil=1 go test ./scheduler -run '^TestHandlerPanic/nil$' -count=1`:
  passed the legacy nil-panic regression.
- `make check`: passed, including source licensing, formatting, vet, all tests,
  coverage, and strict lint. Total coverage: 80.2%; `scheduler`: 95.7%;
  `scheduler/postgres`: 81.8%.
- `make docs-check`: passed.
- `git diff --check`: passed.

Subprocess tests fill the worker slots with panicking jobs and require a later
healthy job to finish. Unit cases preserve ordinary success/failure results,
check arbitrary panic values, inject failed-state write errors, and confirm
store errors are not converted to job failures. A separate child process must
still terminate for a store panic. Database checks verify status, error detail,
started/finished timestamps, healthy JSON output, and absence of success output
on failed runs. Subprocesses have explicit deadlines and race instrumentation
when the parent test binary is race-enabled.

## Boundary

Only finding F7 is addressed. Scheduler retries, recurring-job advancement,
lifecycle changes, unrelated persistence error paths, Pubsub handler panics,
and other architecture-review findings are unchanged.
