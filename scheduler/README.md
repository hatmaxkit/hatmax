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

    "github.com/hatmaxkit/hatmax/scheduler"
    "github.com/hatmaxkit/hatmax/scheduler/postgres"
)

func main() {
    db := connectDB()
    store := postgres.NewStore(db)

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

    sched.Start(ctx)
    defer sched.Stop(ctx)
}
```

## Configuration

### Static (Config struct)

| Field | Default | Description |
|-------|---------|-------------|
| Enabled | false | Enable/disable scheduler |
| Interval | 1m | Polling interval |
| BatchSize | 20 | Max jobs per tick |
| Workers | 1 | Concurrent workers |
| RetryAttempts | 3 | Max retry attempts |
| RetryBackoff | 1m | Base backoff duration |

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
import "github.com/hatmaxkit/hatmax/scheduler/postgres"

store := postgres.NewStore(db)

// Apply schema (or use migrations)
db.Exec(postgres.Schema)
```

Tables:
- `scheduled_jobs` - Job definitions
- `job_runs` - Execution history
