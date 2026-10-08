<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# scheduler

Job scheduler with pluggable storage backends.

## Features

- **Pluggable storage**: Bring your own `JobStore` implementation (Postgres included)
- **Concurrent workers**: Configurable worker pool for parallel job execution
- **Runtime pause**: Consult `scheduler.paused` through `SettingsProvider`
- **Testable**: Fake clock, store, and logger for deterministic tests
- **Schedule types**: Daily, Weekly, and Interval schedules with timezone support

## Usage

This complete function runs until its startup context is canceled. Its caller
opens and closes the database, inserts jobs, and supplies a `scheduler.Logger`:

```go
package example

import (
    "context"
    "database/sql"
    "time"

    "hatmax.adrianpk.com/scheduler"
    schedulerpostgres "hatmax.adrianpk.com/scheduler/postgres"
)

func run(ctx context.Context, database *sql.DB, logger scheduler.Logger) error {
    if _, err := database.ExecContext(ctx, schedulerpostgres.Schema); err != nil {
        return err
    }
    store := schedulerpostgres.NewStore(database)

    cfg := scheduler.Config{
        Enabled:   true,
        Interval:  30 * time.Second,
        BatchSize: 20,
        Workers:   4,
    }

    sched := scheduler.New(store, cfg, logger)

    sched.Register("send-email", func(ctx context.Context, job scheduler.Job) scheduler.Result {
        // Demonstration output only; replace with the application workflow.
        return scheduler.Result{Output: map[string]any{"sent": true}}
    })

    if err := sched.Start(ctx); err != nil {
        return err
    }

    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    return sched.Stop(shutdownCtx)
}
```

After successful shutdown the caller may close its database. If `Stop` times
out, retain the store and wait again before assuming the handler has exited.
The example handler reports a JSON flag; it does not send mail.

## Lifecycle

An enabled runner starts one polling loop. Repeated active `Start` calls are
no-ops. Canceling the first startup context or calling `Stop` cancels polling
and active execution. Handlers and adapters must honor that execution context.
`Stop` before startup is a no-op; repeated and concurrent stops are safe.

Pass a fresh timeout context to `Stop`. Its deadline bounds the wait, not the
lifetime of a handler that ignores cancellation. A later stop can wait again;
keep the store alive until it succeeds. Interrupted persisted runs may require
explicit repair. After shutdown begins, `Start` returns `ErrStopped`; create a
new runner to restart. Manual `Tick` calls are caller-owned and are not joined
by `Stop`. See the [lifecycle contract](../docs/reference/scheduler/README.md#lifecycle).

## Configuration

### Static (Config struct)

| Field | Default | Description |
|-------|---------|-------------|
| Enabled | false | Enable/disable scheduler |
| Interval | 1m | Polling interval |
| BatchSize | 20 | Max jobs per tick |
| Workers | 1 | Concurrent workers |
| RetryAttempts | 3 | Maximum total attempts per slot, including the first; 1 disables retries |
| RetryBackoff | 1m | Fixed delay from failed-attempt completion |

### Dynamic (SettingsProvider)

In `run`, before startup, supply an existing `SettingsProvider` for pause checks:

```go
sched.SetSettings(settingsService)
```

Alternatively replace the constructor with root `*config.Config` plus settings:

```go
sched := scheduler.NewWithConfig(store, settingsService, appCfg, logger)
```

Only `scheduler.paused` is read by the runner at each tick. The exported
`scheduler.enabled` and `scheduler.interval_seconds` keys are not consulted.
Enabled, interval, workers, batch size and retry policy come from construction.
Do not mutate clock or settings dependencies concurrently with scheduler work.

## Schedule Types

In a function returning an error, calculate the next times and handle timezone
lookup failure. Daily and weekly results are UTC and strictly after `from`;
intervals require a positive duration chosen by the application:

```go
// Daily at 9:00 AM UTC
daily := scheduler.Daily{Hour: 9, Minute: 0}

// Weekly on Friday at 5:00 PM in New York
loc, err := time.LoadLocation("America/New_York")
if err != nil {
    return err
}
weekly := scheduler.Weekly{Day: time.Friday, Hour: 17, Minute: 0, TZ: loc}

// Every 30 minutes
interval := scheduler.Interval{Every: 30 * time.Minute}

// Get next run time
next := daily.Next(time.Now())
```

Stores own completion of a slot. `MarkSuccess` and `MarkFailed` record the
result and advance a recurring schedule from finish time, or retire a one-shot
job, atomically. Errors and panics wait for retry while budget remains, without
advancing the schedule. Exhausted failures consume the slot just like success;
recurrence is separate from retries. Missed occurrences are skipped.

Retry waits persist the attempt count, deadline, and initial total budget. A
new runner resumes them without resetting the budget or holding a worker idle.
Handlers receive one-based `Job.Attempt`, stable `RunID`, and `MaxAttempts`.
Custom stores must implement atomic `ClaimRetry` and durable `MarkRetry`.
Existing PostgreSQL installations must apply the schema's additive retry-column
upgrade before starting the new runner. See the
[retry contract](../docs/reference/scheduler/README.md#retries) for the method
signatures, restart policy, and failure boundaries.

For fake-store jobs, set `Job.Schedule` to one of these schedule values. Nil
means one-shot work. In PostgreSQL, use the existing `schedule_spec` column:
empty for one-shot work, or JSON such as `{"type":"interval","every":"30m"}`.
Calendar rules use `schedule_tz`. See the
[Scheduler Reference](../docs/reference/scheduler/README.md#stored-schedule-format)
for the complete format and interrupted-run recovery boundary.

## Testing

Inside a test with `ctx := context.Background()` and a fixed `baseTime`, execute
a registered one-shot task without a polling goroutine:

```go
store := scheduler.NewFakeStore()
clock := scheduler.NewFakeClock(baseTime)
log := &scheduler.FakeLogger{}

sched := scheduler.New(store, scheduler.Config{Workers: 1}, log)
sched.SetClock(clock)
sched.Register("email", func(ctx context.Context, job scheduler.Job) scheduler.Result {
    return scheduler.Result{Output: map[string]any{"sent": true}}
})

store.AddJob(scheduler.Job{
    ID:           "test-job",
    TaskType:     "email",
    ScheduledFor: clock.Now(),
})

if err := sched.Tick(ctx); err != nil {
    t.Fatal(err)
}
clock.Advance(time.Hour)
```

## Postgres Backend

With an existing `*sql.DB`, use the import from `run` and apply the schema before
polling. This fragment belongs in a function returning an error:

```go
store := schedulerpostgres.NewStore(database)

// Apply schema (or use migrations)
if _, err := database.ExecContext(ctx, schedulerpostgres.Schema); err != nil {
    return err
}
```

Tables:
- `scheduled_jobs` - Job definitions
- `job_runs` - One row per scheduled slot, retaining the latest attempt outcome
