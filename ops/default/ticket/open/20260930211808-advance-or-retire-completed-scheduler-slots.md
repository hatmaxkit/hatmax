---
id: TKT-20260930211808
title: Advance or retire completed scheduler slots
status: open
kind: follow_up
severity: high
priority: high
scope: persistence
tags: architecture-review, persistence, hardening
source: review
reported_at: 2026-09-30T21:18:08Z
commits:
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
