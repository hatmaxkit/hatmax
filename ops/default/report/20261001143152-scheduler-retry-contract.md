<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Scheduler Retry Contract

Status: reviewing
Ticket: [TKT-20260930211818](../ticket/reviewing/20260930211818-resolve-the-scheduler-retry-contract.md)
Branch: `fix/ticket-20260930211818-scheduler-retry`
PR: [#86](https://forge.adrianpk.com/hatmax/hatmax/pulls/86)
Implementation: `6a17dd58bc1228641d7fc1a6b88c745aa5bbf988`

## Delivered Behavior

- RetryAttempts bounds total handler attempts per scheduled slot, including the first. The existing default is 3; 1 disables retries. Handler errors and recovered handler panics enter retry_wait while budget remains; unknown task types fail immediately.
- RetryBackoff is a fixed delay after failed-attempt completion. Polling admits due waits without sleeping in a worker, changing the original scheduled time, or advancing the recurring schedule. Terminal success or exhausted failure uses the existing atomic completion and advancement boundary.
- A retry retains one run ID, an increasing attempt counter, the initial total budget, and its deadline. New runners and PostgreSQL store instances resume waits without resetting or changing the budget, even when the new configured limit is higher or lower. New limits apply to new slots.
- PostgreSQL, fake, and no-op stores implement ClaimRetry and MarkRetry. Deadline, sequence, persisted-budget, current-slot, and state guards prevent early, stale, duplicate, disabled, rescheduled, or terminal retry claims. Concurrent claimants admit only one execution of the next attempt.
- Public references, package guidance, configuration help, the relevant User Guide paragraph, and Unreleased now describe the implemented behavior and required adapter/schema changes. Diataxis keeps the full contract in reference and the User Guide change limited to product-use guidance.

## Contracts and Ownership

The runner owns retry policy; stores own durable waits and atomic admission. JobStore gains ClaimRetry(ctx, runID, attempt, startedAt) and MarkRetry(ctx, runID, maxAttempts, finishedAt, retryAt, errMsg). ListDue identifies initial work with an empty RunID and attempt 1; retry work carries its existing run ID, next attempt, and MaxAttempts. Handlers receive the stable run ID after either claim.

MarkRetry records a failed attempt's finish time, error, future deadline, and fixed budget without advancing the job. ClaimRetry conditionally moves that exact due wait to running and increments its attempt. MarkRunning admits only pending initial runs, preventing that method from bypassing retry ownership or reviving terminal runs. Retry admission is ordered by its deadline rather than an old slot timestamp, so delayed failures do not jump ahead of earlier ready work.

The PostgreSQL schema adds nullable retry_at TIMESTAMPTZ and retry_limit INT columns, including idempotent upgrades for existing tables. Apply the schema or equivalent migrations before starting the upgraded runner. Existing terminal outcomes and interrupted pending/running claims are preserved. Custom JobStore implementations must adopt both new methods. The existing unique slot key remains unchanged: a run row records the latest outcome and attempt count, not separate history rows for every attempt. The legacy scheduled_jobs.max_retries column remains outside this runner's policy.

## Validation

All Go checks used Go 1.26.7. Database tests used an isolated native PostgreSQL 18.6 cluster, not Docker or CI PostgreSQL 16.

- Before implementation, `go test ./scheduler -run '^TestRetrySteps$' -count=1` failed in all five cases: handlers received attempt 0 rather than the initial attempt 1. The same test now verifies successive attempts, fixed waits, restart budgets, disabled retries, eventual success, exhausted errors/panics, stable identity, one run per slot, and healthy work admission during waits.
- `make check`: passed source licensing, formatting, vet, all tests, the 80% coverage threshold, and strict lint. Total coverage: 81.6%; scheduler: 93.4%; scheduler/postgres: 92.7%.
- `make docs-check`: passed.
- `go test -race ./scheduler/... -count=1`: passed.
- `go test ./scheduler/... -run '^(TestRetrySteps|TestRetryWrite|TestRetryRun|TestRetryClaim|TestRetrySchema|TestRetryOrder)$' -count=20`: passed all 20 repetitions.
- `git diff --check`: passed.

PostgreSQL tests verify restart through new store/runner instances, fixed delays measured from completion, original slot preservation, unchanged recurrence during waits, terminal retirement/advancement, successful JSON output, failure detail, fresh budgets for later slots, canceled writes/claims, invalid or changed budgets, early/stale/skipped attempts, sixteen competing claimants, disabled and terminal claims, additive upgrade idempotence, preserved older terminal runs, and ready-work ordering. A fault-injected retry write remains observable and leaves the claimed run running rather than fabricating durable retry state. Existing panic, lifecycle, slot-uniqueness, manual-rescheduling, and atomic-completion regressions pass; terminal-only tests now explicitly disable retries.

## Boundary

Only F19 is implemented. Interrupted pending/running claims still require explicit application repair; there is no lease or automatic crash recovery. Handler side effects must tolerate repetition. Retry waits do not provide rollback or exactly-once effects. No dependency, new worker loop, recurring schedule algorithm, dynamic setting behavior, or other finding was changed.
