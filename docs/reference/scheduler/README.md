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
`Metadata`. `Result.Failed` is true when `Err` is non-nil.

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
`Interval`. `Stop` closes the stop channel and waits.

A tick does nothing when `scheduler.paused` is true. A nil settings provider
is not paused. `Tick` lists due jobs up to `BatchSize`. One worker runs them
in order. More than one worker runs that many jobs at a time and waits for
the batch.

`process` creates a run, marks it running, and marks it failed for an unknown
task type or a failed result. Success JSON-encodes `Output`, or an empty
object when `Output` is nil. The runner does not call `UpdateNextRun` and
does not use `RetryAttempts` or `RetryBackoff`.

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
enabled jobs ordered by `next_run_at` with `FOR UPDATE SKIP LOCKED`. The store
implements `UpdateNextRun`, but the runner does not call it.
