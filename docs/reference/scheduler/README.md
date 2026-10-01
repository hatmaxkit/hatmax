<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Scheduler

`scheduler` polls a `JobStore` and runs a handler for each due job.

## Public boundaries

| Boundary | Methods |
| --- | --- |
| `Scheduler` | `Register`, `Start`, `Stop`, `Tick` |
| `JobStore` | `ListDue`, `CreateRun`, `MarkRunning`, `MarkSuccess`, `MarkFailed`, `UpdateNextRun` |
| `SettingsProvider` | `GetBool`, `GetInt` |
| `Clock` | `Now` |
| `Logger` | `Info`, `Infof`, `Error`, `Errorf` |

`Handler` is `func(context.Context, Job) Result`. `Schedule` has
`Next(time.Time) time.Time`.

## Jobs and schedules

`Job` has `ID`, `Name`, `TaskType`, `Payload`, `ScheduledFor`, `Attempt`, and
`Metadata`, and an optional `Schedule`. A nil schedule is a one-shot job.
`Result.Failed` is true when `Err` is non-nil.

`Daily.Next` and `Weekly.Next` use `TZ`, or UTC when it is nil, and return
UTC. A time that is not after `from` moves to the next day or the next week.
`Interval.Next` adds `Every` to `from`.

## Runner

`ConfigFromRoot` copies the root scheduler config. `WithDefaults` fills a
non-positive interval or retry backoff with one minute, batch size `20`, one
worker, and three retry attempts.

`New` and `NewWithConfig` build a runner. `Register` stores one handler per
task type and replaces an existing type. `Start` returns nil without a
goroutine when `Enabled` is false. Otherwise it ticks immediately and then on
`Interval`.

### Lifecycle

An enabled runner has one lifecycle: idle, running, stopping, stopped. `Start`
rejects an already canceled or expired context with its context error, without
consuming the idle instance. Repeated or concurrent `Start` calls on a running
instance return nil and do not create another loop. Only the first successful
start's context owns that loop. `Start` after shutdown begins returns
`ErrStopped`; construct a new runner to restart.

`Stop` before startup is a no-op, including for a disabled runner. On an active
runner, it cancels polling and the context supplied to stores, settings, and
handlers. Cancellation of the original start context has the same effect.
Shutdown stops admitting further jobs when cancellation is observed and joins
the admitted batch before marking the runner stopped.

Repeated and concurrent `Stop` calls wait on the same completion signal. The
caller's context bounds only that wait: an expired deadline returns
`context.DeadlineExceeded`, cancellation returns `context.Canceled`. Shutdown
continues, and a later call can wait again. Completed shutdown returns nil even
with a canceled wait context. No extra waiter goroutine is created per call.

Handlers and adapters must honor their execution context. Go cannot forcibly
terminate a handler that ignores cancellation; a timed-out `Stop` does not mean
the handler exited or its result was saved. Persistence uses the canceled
execution context rather than an unbounded cleanup context. An interrupted
claim or terminal write can leave an unfinished run requiring the explicit
repair described below. Keep the job store available until shutdown completes.

Direct `Tick` calls remain caller-owned, independent of background lifecycle.
`Tick` rejects a canceled context before querying, checks cancellation after
the query and during admission, and returns the context error after admitted
handlers finish. Worker admission observes cancellation rather than waiting
indefinitely to launch more work. `Stop` does not join independent `Tick` calls;
their callers must cancel and join them separately.

A tick does nothing when `scheduler.paused` is true. A nil settings provider
is not paused. `Tick` lists due jobs up to `BatchSize`. One worker runs them
in order. More than one worker runs that many jobs at a time and waits for
the batch.

`process` creates a run, marks it running, and marks it failed for an unknown
task type or a failed result. Success JSON-encodes `Output`, or an empty
object when `Output` is nil. Stores own the terminal transition: `MarkSuccess`
and `MarkFailed` record the result and advance or retire the slot atomically.
The runner logs terminal-write errors; it does not claim completion when a
write fails. It does not use `RetryAttempts` or `RetryBackoff`.

Completion retires a one-shot job. A recurring job schedules `Schedule.Next`
from the finish time, not the old due time. The next slot must be strictly
after both finish time and the current slot. This skips missed occurrences
rather than replaying a backlog. Success, handler errors, panics, and unknown
task types consume the current slot and use the same schedule transition.
The next recurring occurrence is not a retry of the failed slot.

Handler panics are converted to a failed result with `handler panic:` detail on
the goroutine invoking the handler. The run is marked failed, and processing
continues with the next job; concurrent workers release their slot as usual.
If `MarkFailed` returns an error, the runner logs that persistence failure and
the run may remain running. `Tick` still returns a `ListDue` error unchanged.

Recovery is limited to the synchronous handler invocation, not the whole tick.
Panics in stores, clocks, settings, or logging are not suppressed. Panics in
goroutines started by a handler, `runtime.Goexit`, process exits, and fatal
runtime failures are outside this boundary. Applications remain responsible
for diagnosing handler defects and scheduling any later attempt; recording a
failed run does not roll back application side effects.

`WithClock` and `WithSettings` are constructor options. `SetClock` and
`SetSettings` replace those dependencies after construction. Callers must not
mutate them concurrently with scheduler work.

The runtime reads only `scheduler.paused`. `scheduler.enabled` and
`scheduler.interval_seconds` are exported setting keys but are not consulted
by the current runner after construction.

## Stores

`JobStore` is `ListDue`, `CreateRun`, `MarkRunning`, `MarkSuccess`,
`MarkFailed`, and `UpdateNextRun`. `NoopStore` returns nil from each method
and no jobs. `postgres.NewStore` implements the same methods against the
package schema. `NewFakeStore` records jobs and runs in memory. `NewFakeClock`
returns the time set by `Set` and `Advance`.

The Postgres schema contains `scheduled_jobs` and `job_runs`. `ListDue` selects
enabled jobs ordered by `next_run_at` with `FOR UPDATE SKIP LOCKED`, excluding
any slot already present in `job_runs`. Pending and running claims are also
excluded. The unique `(job_id, scheduled_for)` key enforces one run per slot;
the fake store enforces the same rule and due-job ordering.

Terminal writes lock the run and job and update both within one transaction.
A one-shot job becomes `enabled = false`; a recurring job keeps its enabled
state and receives the next UTC timestamp. `last_run_at` records the consumed
slot's scheduled time. A repeated terminal write is a no-op. Explicit manual
rescheduling or disabling while a handler runs is preserved rather than
overwritten by that handler's completion.

`UpdateNextRun` remains an explicit administrative operation. A zero next-run
time disables the job; a non-zero time enables it and replaces its next slot.
The runner does not need to call it after recording a terminal outcome.

If a claim or terminal write is interrupted, the claimed slot remains excluded
from due batches. Inspect and repair the run before scheduling a later slot.
There is no lease, automatic recovery, or same-slot retry. Slot uniqueness does
not make handler side effects transactional or guarantee exactly-once effects.

## Stored schedule format

The existing `schedule_spec` text column is empty for one-shot jobs. Recurring
jobs use one JSON object describing an existing schedule type:

| Type | `schedule_spec` example | Next occurrence |
| --- | --- | --- |
| Interval | `{"type":"interval","every":"30m"}` | Positive Go duration after completion |
| Daily | `{"type":"daily","hour":9,"minute":0}` | Next calendar day/time after completion |
| Weekly | `{"type":"weekly","day":5,"hour":17,"minute":0}` | Next weekday/time after completion |

Calendar schedules use `schedule_tz`, defaulting to UTC. Weekdays use Go's
numbering: Sunday `0` through Saturday `6`. Omitted calendar fields are zero.
Hours must be `0`–`23`, minutes `0`–`59`; intervals must be positive. Non-zero
calendar fields are invalid on intervals, and a non-zero weekday is invalid
on daily schedules. Calendar rules cannot include an interval duration.

Unknown fields, unsupported types, invalid zones, malformed JSON, and multiple
JSON values return a `ListDue` error before any handler receives that batch.
Existing non-empty opaque specifications must be converted to this format;
there is no cron expression parser. No schema migration is required.
