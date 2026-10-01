---
id: TKT-20260930211818
title: Resolve inert scheduler retry configuration
status: reviewing
kind: follow_up
severity: medium
priority: normal
scope: domain
tags: architecture-review, domain, correctness
source: review
reported_at: 2026-09-30T21:18:18Z
ready_at: 2026-10-01T14:10:39Z
started_at: 2026-10-01T14:10:39Z
reviewed_at: 2026-10-01T14:31:52Z
branch: fix/ticket-20260930211818-scheduler-retry
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/86
commits: 6a17dd58bc1228641d7fc1a6b88c745aa5bbf988
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

RetryAttempts and RetryBackoff are accepted and defaulted, but never used by process. CreateRun always records attempt 1. scheduler/README.md presents these as retry settings while the reference correctly states that retries are not implemented.

Sources: `scheduler/config.go:14-20, scheduler/runner.go:194-240, scheduler/postgres/store.go:64-72`.

Evidence: Source review found no retry execution path; docs/reference/scheduler/README.md:53 explicitly acknowledges the limitation.

Impact: The API advertises operational controls that have no effect. This is a documented capability gap, not a newly discovered hidden implementation promise.

## Expected Outcome

Choose one explicit retry contract: implement bounded retry/backoff and attempt persistence, or remove/deprecate unsupported controls and align package guidance. Keep this independent of recurring slot advancement.

## Validation

Verify the selected contract through fake-clock and PostgreSQL tests. If retries are implemented, cover maximum attempts, backoff, restart, and terminal failure.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f19).

## Selected Contract

Implement retries, as approved by the maintainer. RetryAttempts is the maximum total handler attempts per slot, including the first; 1 disables retries and the existing nonpositive default is 3. RetryBackoff is a fixed delay from a failed attempt's completion. Polling admits due retries without sleeping in a worker. A run retains its slot, attempt counter, and initial total budget while waiting, so a new runner can resume persisted waits even with changed configuration without resetting or changing that budget. Successful completion or exhausted failure advances or retires the slot under the existing completion contract. Unknown task types remain immediate terminal failures. Interrupted pending/running claims still require application repair; this change does not implement leases or automatic crash recovery.

## Delivery

The runner admits persisted retry waits through atomic ClaimRetry and records handler failures through MarkRetry while the slot's fixed budget remains. RunID stays stable, Attempt is one-based, and MaxAttempts is retained with the retry deadline. PostgreSQL and fake stores enforce deadline, sequence, budget, current-slot, and terminal-state guards. The PostgreSQL schema adds nullable retry_at and retry_limit columns idempotently without changing existing outcomes.

`make check`, `make docs-check`, `go test -race ./scheduler/... -count=1`, and 20 repetitions of the focused retry suite passed. Total coverage is 81.6%; scheduler is 93.4% and its PostgreSQL adapter is 92.7%. Store implementers must adopt the extended JobStore interface, and existing PostgreSQL installations must apply the additive schema upgrade before starting the new runner.

Report: [Scheduler Retry Contract](../../report/20261001143152-scheduler-retry-contract.md).
