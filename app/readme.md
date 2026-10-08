<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# app

Lifecycle management for application components.

## Usage

These are contextual declarations and assembly statements, not a standalone
program. Use the [bootstrap procedure](../docs/how-to/bootstrap-application/README.md)
for a complete process with imports, configuration, serving and signal handling.
`dbPool`, `myService` and `anotherService` denote constructed application
components; `ctx`, `router`, `logger` and `srv` belong to that composition root.
The API block below contains declarations for comparison with the exported API.

```go
// Components implement optional interfaces
type MyService struct{}

func (s *MyService) Start(ctx context.Context) error { return nil }
func (s *MyService) Stop(ctx context.Context) error { return nil }
func (s *MyService) RegisterRoutes(r chi.Router) {
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
}

// Setup discovers capabilities automatically
starts, stops, registrars := app.Setup(ctx, router,
    dbPool,
    &myService,
    &anotherService,
)

// Start executes in order and rolls back completed steps by component identity.
if err := app.Start(ctx, logger, starts, stops, registrars, router); err != nil {
    logger.Errorf("cannot start: %v", err)
    return
}

// Shutdown in reverse order (LIFO)
app.Shutdown(srv, logger, stops)
```

## API

```go
type Startable interface {
    Start(context.Context) error
}

type Stoppable interface {
    Stop(context.Context) error
}

type RouteRegistrar interface {
    RegisterRoutes(chi.Router)
}

type StartupStep struct {
    Start func(context.Context) error
    Stop  func(context.Context) error // nil for start-only components
}
```

Components implement the interfaces they need. Setup inspects and groups them.

`Setup` pairs each startup function with the same component's optional stop
function. Startup rollback stops only completed steps, in reverse order, and
preserves the original start error. The separate stop list belongs to normal
shutdown, including stop-only components; `Start` retains that argument for
existing calls but does not use it. Manual startup lists now use
`[]app.StartupStep` instead of function slices. See the
[Application Lifecycle Reference](../docs/reference/application-lifecycle/README.md)
for the exact failure behavior.

`Serve` accepts a caller-owned `*http.Server`. Construct it with the address
and handler, configure it before serving, and pass that same instance to
`Shutdown`. Missing header and idle timeouts default to 5 and 60 seconds;
positive caller values are preserved and negative limits are rejected.
`ReadTimeout` and `WriteTimeout` remain unchanged, including zero values for
streaming. `Serve` normalizes `http.ErrServerClosed` to nil; wait for `Shutdown`
to finish before exiting. Old `Serve(router, port)` calls must migrate to
`Serve(&http.Server{Addr: port, Handler: router})`, retaining the server pointer.
