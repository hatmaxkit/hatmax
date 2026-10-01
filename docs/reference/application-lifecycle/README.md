<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Application Lifecycle

`app` collects component capabilities, starts them, registers routes, serves
HTTP, and stops them. The implementation note is
[app/readme.md](../../../app/readme.md).

## Interfaces

| Interface | Method | When it runs |
| --- | --- | --- |
| `Startable` | `Start(context.Context) error` | During `Start`, in setup order |
| `Stoppable` | `Stop(context.Context) error` | During rollback of its completed startup step and during `Shutdown` |
| `RouteRegistrar` | `RegisterRoutes(chi.Router)` | After every start function succeeds |

A component may implement any subset of these interfaces.

## Setup

`Setup(ctx, router, components...)` walks the components in argument order
and builds three lists:

- `StartupStep` values, one for each `Startable`, pairing its start function
  with the same component's stop function when it implements `Stoppable`;
- stop functions, one for each `Stoppable`;
- route registrars, one for each `RouteRegistrar`.

`StartupStep` has `Start` and `Stop` fields of type
`func(context.Context) error`. `Start` must be non-nil; `Stop` is nil for a
start-only component. Stop-only components appear only in the shutdown list.
`Setup` does not call `Start`, `Stop`, or `RegisterRoutes`.

## Start

`Start(ctx, logger, starts, stops, registrars, router)` calls the startup steps in order. When a start function returns an
error at index `i`, `Start` logs `error starting component #<index>` and then
walks completed steps from `i-1` down to `0`, calling each non-nil `Stop`.
Rollback uses
`context.Background()`. A stop error during rollback is logged and does not
replace the original start error. `Start` then returns the start error.

Routes are registered only after every start function returns nil.

The failing step and later steps are not stopped. A failing `Start` must clean
up its own partial initialization before returning an error. Start-only steps
have no rollback operation; stop-only and route-only components do not enter
the startup sequence. No positional alignment with the shutdown list is required.

The `stops` argument is retained so existing `Setup`-based calls keep their
shape, but `Start` does not read it. Pass the separate stop list to `Shutdown`.
Code that manually assembled `[]func(context.Context) error` for startup must
instead use `[]app.StartupStep`, assigning each start function its own optional
stop function. Calls using the outputs of `Setup` need no source change.

## Serve

`Serve(router, port)` listens on `port` with the given router and blocks.
`http.ErrServerClosed` is returned as a nil error. Any other listen error is
returned.

## Shutdown

`Shutdown(server, logger, stops)` gives the HTTP server five seconds to shut
down. A server shutdown error is logged. `Shutdown` then calls the stop
functions from the last index to the first, using `context.Background()`. A
stop error is logged and does not stop the remaining calls.

## Router options

`NewRouter(logger, options...)` creates a chi router and applies each option.
An option error is logged and `NewRouter` continues with the remaining
options.

`ApplyRouterOptions(router, options...)` applies the same options and returns
the first error.

| Option | Route |
| --- | --- |
| `WithMiddleware(middleware...)` | Installs the middleware. Returns nil. |
| `WithPing()` | `GET /ping` responds `200` with `{"status":"ok"}`. |
| `WithDebugRoutes()` | `GET /debug/routes` lists the routes registered on the router that is current when the request arrives. A walk error responds `500`. |
