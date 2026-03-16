# telemetry

Request counting and crash collection for telemetry reporting.

## Usage

```go
import (
    "hatmax.adrianpk.com/hatmax/telemetry"
    "hatmax.adrianpk.com/hatmax/middleware"
    "hatmax.adrianpk.com/hatmax/settings"
)

// Create collectors
counter := telemetry.NewCounter()
crashes := telemetry.NewCrashCollector()

// Register settings schemas with your registry
registry := settings.NewRegistry()
telemetry.RegisterSchemas(registry)

// Apply middleware to your router
router.Use(middleware.TelemetryRecovery(crashes))
router.Use(middleware.TelemetryCounter(counter))

// Periodically collect and send data
mode, _ := settingsService.GetString(ctx, telemetry.KeyMode)
if mode != telemetry.ModeOff {
    requests := counter.GetAndResetRequests()
    crashEvents := crashes.GetAndResetCrashes()
    // Send to your telemetry endpoint...
}
```

## API

### Counter

```go
type RequestCounter interface {
    IncrementRequests()
    GetAndResetRequests() int64
}
```

- `NewCounter() *AtomicCounter` - creates a thread-safe counter
- `IncrementRequests()` - increments by one (atomic)
- `GetAndResetRequests() int64` - returns count and resets to zero (atomic swap)

### CrashCollector

```go
type CrashRecorder interface {
    RecordPanic(panicValue any, endpoint, method string)
}

type CrashEvent struct {
    Type      string   // "panic"
    Message   string   // sanitized, max 200 chars
    Stack     []string // max 10 frames
    Endpoint  string
    Method    string
    Count     int64    // deduplicated count
    FirstSeen string   // RFC3339
    LastSeen  string   // RFC3339
}
```

- `NewCrashCollector() *CrashCollector` - creates a collector
- `RecordPanic(value any, endpoint, method string)` - records with deduplication by message+endpoint
- `GetAndResetCrashes() []CrashEvent` - returns all events and clears

### Settings

Predefined schemas for telemetry configuration:

```go
// Key constants
telemetry.KeyMode       // "telemetry.mode"
telemetry.KeyInstanceID // "telemetry.instance_id"

// Mode constants
telemetry.ModeOff   // "off" - no telemetry
telemetry.ModeBasic // "basic" - ping only (default)
telemetry.ModeFull  // "full" - ping + request count
telemetry.ModeDebug // "debug" - full + crash reports
```

- `Schemas` - slice of `settings.Schema` for registration
- `RegisterSchemas(r *settings.Registry)` - helper to register all schemas

## Middleware

The telemetry middleware is in the `middleware` package:

```go
// Count every request
router.Use(middleware.TelemetryCounter(counter))

// Catch panics and record them (also returns 500)
router.Use(middleware.TelemetryRecovery(crashes))
```

Both middleware functions accept nil safely (no-op).
