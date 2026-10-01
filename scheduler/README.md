<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# scheduler

Job scheduler with pluggable storage backends.

## Features

- **Pluggable storage**: Bring your own `JobStore` implementation (Postgres included)
- **Concurrent workers**: Configurable worker pool for parallel job execution
- **Dynamic configuration**: Override settings at runtime via `SettingsProvider`
- **Testable**: Fake clock, store, and logger for deterministic tests
- **Schedule types**: Daily, Weekly, and Interval schedules with timezone support

## Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "hatmax.adrianpk.com/scheduler"
    schedulerpostgres "hatmax.adrianpk.com/scheduler/postgres"
)

func main() {
    db := connectDB()
    store := schedulerpostgres.NewStore(db)

    cfg := scheduler.Config{
        Enabled:   true,
        Interval:  30 * time.Second,
        BatchSize: 20,
        Workers:   4,
    }

    sched := scheduler.New(store, cfg, logger)

    sched.Register("send-email", func(ctx context.Context, job scheduler.Job) scheduler.Result {
        // Process job
        return scheduler.Result{Output: map[string]any{"sent": true}}
    })

    err := sched.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }

    defer func() {
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        err := sched.Stop(shutdownCtx)
        if err != nil {
            log.Printf("scheduler shutdown: %v", err)
        }
    }()
}
```

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

Implement `SettingsProvider` to override at runtime:

```go
sched.SetSettings(settingsService)
```

Or wire from root config + settings in one constructor:

```go
sched := scheduler.NewWithConfig(store, settingsService, appCfg, logger)
```

Settings keys:
- `scheduler.enabled` - Override Enabled
- `scheduler.interval_seconds` - Override Interval
- `scheduler.paused` - Pause without stopping

## Schedule Types

```go
// Daily at 9:00 AM UTC
daily := scheduler.Daily{Hour: 9, Minute: 0}

// Weekly on Friday at 5:00 PM in New York
loc, _ := time.LoadLocation("America/New_York")
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

Use fakes for deterministic tests:

```go
store := scheduler.NewFakeStore()
clock := scheduler.NewFakeClock(baseTime)
log := &scheduler.FakeLogger{}

sched := scheduler.New(store, cfg, log)
sched.SetClock(clock)

store.AddJob(scheduler.Job{
    ID:           "test-job",
    TaskType:     "email",
    ScheduledFor: clock.Now(),
})

sched.Tick(ctx)
clock.Advance(time.Hour)
```

## Postgres Backend

```go
import schedulerpostgres "hatmax.adrianpk.com/scheduler/postgres"

store := schedulerpostgres.NewStore(db)

// Apply schema (or use migrations)
db.Exec(schedulerpostgres.Schema)
```

Tables:
- `scheduled_jobs` - Job definitions
- `job_runs` - Execution history
