<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# telemetry

Request counting and crash collection. Collectors do not send data or enforce
telemetry modes. This complete program exercises the middleware locally:

## Usage

```go
package main

import (
    "fmt"
    "net/http"
    "net/http/httptest"

    "github.com/go-chi/chi/v5"
    "hatmax.adrianpk.com/middleware"
    "hatmax.adrianpk.com/settings"
    "hatmax.adrianpk.com/telemetry"
)

func main() {
    counter := telemetry.NewCounter()
    crashes := telemetry.NewCrashCollector()
    registry := settings.NewRegistry()
    telemetry.RegisterSchemas(registry)
    router := chi.NewRouter()
    router.Use(middleware.TelemetryRecovery(crashes))
    router.Use(middleware.TelemetryCounter(counter))
    router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
        panic("fixture failure")
    })
    response := httptest.NewRecorder()
    router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))
    requests := counter.GetAndResetRequests()
    crashEvents := crashes.GetAndResetCrashes()
    fmt.Println(response.Code, requests, len(crashEvents))
}
```

Output: `500 1 1`. Both reads clear their accumulated data. The application owns
when to read, how to redact and export, and whether its selected mode permits
export. Schema registration defines setting values; it does not schedule any
collection or transmission.

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
    RecordPanic(message, endpoint, method string)
}

type CrashEvent struct {
    Type      string   // "panic"
    Message   string   // truncated to at most 200 bytes; not redacted
    Stack     []string // max 10 frames
    Endpoint  string
    Method    string
    Count     int64    // deduplicated count
    FirstSeen string   // RFC3339
    LastSeen  string   // RFC3339
}
```

- `NewCrashCollector() *CrashCollector` - creates a collector
- `RecordPanic(message, endpoint, method string)` - records with deduplication by message+endpoint
- `GetAndResetCrashes() []CrashEvent` - returns all events and clears

### Settings

Predefined schemas for telemetry configuration:

In the application-owned exporter, use these keys and supported values:

```go
modeKey := telemetry.KeyMode             // "telemetry.mode"
instanceKey := telemetry.KeyInstanceID   // "telemetry.instance_id"
modes := []string{
    telemetry.ModeOff,   // "off"
    telemetry.ModeBasic, // "basic" (schema default)
    telemetry.ModeFull,  // "full"
    telemetry.ModeDebug, // "debug"
}
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

Both accept nil collectors safely. A nil counter passes requests through;
recovery still catches panics and attempts a 500 response without recording.
If a handler has already committed its response, recovery cannot replace that
status; the fresh panic route above has written no response before panicking.

Crash aggregation has no cap on distinct message/endpoint keys. Messages are
truncated by bytes, which can split UTF-8, and neither endpoints nor messages
are scrubbed for secrets. Up to ten function names are stored as stack detail.
Group identity uses the original message plus endpoint; HTTP method is not part
of that key. Periodic draining and safe inputs remain application-owned. See
the [Telemetry Reference](../docs/reference/telemetry/README.md).
