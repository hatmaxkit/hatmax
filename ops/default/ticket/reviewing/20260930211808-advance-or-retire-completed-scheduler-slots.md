---
id: TKT-20260930211808
title: Advance or retire completed scheduler slots
status: reviewing
kind: follow_up
severity: high
priority: high
scope: persistence
tags: architecture-review, persistence, hardening
source: review
reported_at: 2026-09-30T21:18:08Z
ready_at: 2026-10-01T07:54:14Z
started_at: 2026-10-01T07:54:14Z
reviewed_at: 2026-10-01T08:11:47Z
branch: fix/ticket-20260930211808-scheduler-slots
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/76
commits: 2517372dc878bf6b6b5bb0154c0caf4e6f183b05
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

A completed job remains due because process never calls UpdateNextRun or retires a one-shot job. PostgreSQL then rejects the same slot through UNIQUE(job_id, scheduled_for). The fake store instead permits the handler to run repeatedly.

Sources: `scheduler/runner.go:194-240, scheduler/postgres/store.go:25-72, scheduler/postgres/schema.go:40`.

Evidence: TestObservedPostgresScheduleStall ran two ticks: the handler ran once, but the job remained due. TestObservedRepeatedSchedule ran the same fake-store slot twice. The lack of automatic advancement is already documented.

Impact: The built-in runner/store pair stalls after one successful run and repeatedly polls the exhausted slot. The fake backend does not expose the production behavior.

## Expected Outcome

Make ownership of next-run advancement explicit and implement the corresponding terminal or recurring transition. Align fake slot uniqueness with PostgreSQL.

## Validation

Verify one-shot completion, two recurring slots, slot uniqueness, and agreement between fake and PostgreSQL behavior. Confirm exhausted slots no longer consume due batches.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f9).

## Implementation

JobStore terminal operations now own atomic result and schedule completion. One-shot jobs retire; recurring jobs advance from completion time using Daily, Weekly, or Interval rules. The PostgreSQL store locks and updates the job and run in one transaction. The fake follows the same transition and slot-uniqueness contract.

Due selection excludes all claimed slots, including pending or running claims, so exhausted work cannot starve later batches. Terminal writes are idempotent and preserve explicit manual rescheduling or disabling. Terminal persistence failures are logged, not reported as completed runs.

The existing PostgreSQL schedule_spec column stores a documented JSON rule; an empty specification is one-shot work. The database schema and JobStore method signatures are unchanged. Custom stores must implement the terminal-transition contract. No same-slot retry, lease recovery, cron parser, or catch-up replay was added.

Delivery: [Scheduler slot completion](../../report/20261001081147-scheduler-slot-completion.md).
