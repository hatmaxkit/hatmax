---
id: TKT-20260930211806
title: Contain scheduler handler panics in each worker
status: solved
resolution: fixed
kind: bug
severity: high
priority: high
scope: domain
tags: architecture-review, domain, hardening
source: review
reported_at: 2026-09-30T21:18:06Z
ready_at: 2026-10-01T06:11:32Z
started_at: 2026-10-01T06:11:32Z
reviewed_at: 2026-10-01T06:20:17Z
closed_at: 2026-10-01T06:33:50Z
branch: fix/ticket-20260930211806-scheduler-panics
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/74
commits: c397833d36532468cbd60929d2859da9d2720193, 9ec0623a4b0422b37bab4ddf650cbe1bade45ccb
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

The recover in tick protects only its own goroutine. With Workers greater than one, a handler runs in a child goroutine and its panic terminates the process.

Sources: `scheduler/runner.go:124-137, scheduler/runner.go:175-189`.

Evidence: TestObservedWorkerPanic ran a two-worker scheduler in a subprocess and confirmed process termination with the handler panic.

Impact: One background job can crash the complete single-binary application. A failed run can also remain marked running.

## Expected Outcome

Put the failure boundary around each handler invocation. Persist a failed run and release worker resources without masking normal store failures.

## Validation

Use a subprocess regression for concurrent workers. Verify single-worker behavior, failed run state, worker-slot release, and execution of a subsequent healthy job.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f7).

## Implementation

Each synchronous handler invocation now converts its own panic into a failed
result, on the invoking goroutine. The normal failed-run path persists the panic
detail and logs a failed-state persistence error when the store cannot complete
that write. The enclosing tick no longer suppresses infrastructure panics.

Subprocess regressions exercise one and two workers with failed jobs occupying
the pool before a healthy job. PostgreSQL integration checks failed status,
timestamps, panic detail, and subsequent success. No retry or lifecycle policy
was added.

Delivery: [Scheduler handler panic containment](../../report/20261001062017-scheduler-handler-panic-containment.md).

Merged into `dev` through PR #74 at
`9ec0623a4b0422b37bab4ddf650cbe1bade45ccb`.
