---
id: TKT-20260930211806
title: Contain scheduler handler panics in each worker
status: open
kind: bug
severity: high
priority: high
scope: domain
tags: architecture-review, domain, hardening
source: review
reported_at: 2026-09-30T21:18:06Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
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
