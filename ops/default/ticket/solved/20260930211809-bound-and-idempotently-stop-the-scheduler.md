---
id: TKT-20260930211809
title: Make scheduler shutdown bounded and idempotent
status: solved
resolution: fixed
kind: bug
severity: medium
priority: normal
scope: domain
tags: architecture-review, domain, correctness
source: review
reported_at: 2026-09-30T21:18:09Z
ready_at: 2026-10-01T09:20:01Z
started_at: 2026-10-01T09:20:01Z
reviewed_at: 2026-10-01T09:30:49Z
closed_at: 2026-10-01T09:37:23Z
branch: fix/ticket-20260930211809-scheduler-lifecycle
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/77
commits: f07bb73b603d7a4fd16838e9f18dba3e9bf8f192, f1927241605ac578f159e1dc4c7e84b418668208
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

Stop closes the same channel on every call, so a second call panics. It ignores its context while waiting. The run loop does not select ctx.Done, and repeated Start calls launch additional loops.

Sources: `scheduler/runner.go:82-121`.

Evidence: TestObservedRepeatedStopPanic reproduced the closed-channel panic. Cancellation and repeated-start behavior are source-confirmed.

Impact: Lifecycle callers cannot safely retry shutdown or enforce a shutdown deadline. Cancellation alone leaves polling active.

## Expected Outcome

Define runner lifecycle states, make repeated shutdown safe, and honor cancellation and shutdown deadlines. Prevent duplicate active loops.

## Validation

Cover Stop before Start, repeated Stop, repeated Start, canceled startup context, and a blocked handler with a short shutdown deadline.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f10).

## Implementation

Runner lifecycle state is synchronized independently from handler registration. The first successful Start owns one cancellable polling loop; repeated active starts are no-ops. Canceled startup is rejected without consuming the instance. Once shutdown begins, Start returns ErrStopped, including when the original startup context is canceled while a handler remains blocked. Restart requires a new runner.

Stop before startup is a no-op. Active Stop cancels execution and waits on one shared completion signal; repeated and concurrent stops are safe. Each caller's context bounds its own wait without creating an extra waiter goroutine. A timed-out shutdown can be joined again. Completed shutdown returns nil even with a canceled wait context.

Tick stops admitting jobs when cancellation is observed, including while waiting for worker capacity, and joins already admitted handlers. Direct Tick calls remain caller-owned and are not joined by Stop. Handlers and adapters must cooperate with cancellation; there is no forced termination or unbounded cleanup context. Interrupted PostgreSQL run claims remain excluded and require explicit repair, rather than being falsely completed or automatically retried.

Delivery: [Scheduler lifecycle](../../report/20261001093049-scheduler-lifecycle.md).

Merged into `dev` through PR #77 at `f1927241605ac578f159e1dc4c7e84b418668208`.
