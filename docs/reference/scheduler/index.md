# Scheduler

`scheduler` polls a `JobStore` and runs a handler for each due job. The
implementation note is [scheduler/readme.md](../../../scheduler/readme.md).

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

## Stores

`JobStore` is `ListDue`, `CreateRun`, `MarkRunning`, `MarkSuccess`,
`MarkFailed`, and `UpdateNextRun`. `NoopStore` returns nil from each method
and no jobs. `postgres.NewStore` implements the same methods against the
package schema. `NewFakeStore` records jobs and runs in memory. `NewFakeClock`
returns the time set by `Set` and `Advance`.
