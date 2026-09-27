# Application Lifecycle

`app` collects component capabilities, starts them, registers routes, serves
HTTP, and stops them. The implementation note is
[app/readme.md](../../../app/readme.md).

## Interfaces

| Interface | Method | When it runs |
| --- | --- | --- |
| `Startable` | `Start(context.Context) error` | During `Start`, in setup order |
| `Stoppable` | `Stop(context.Context) error` | From the rollback stop slice and during `Shutdown` |
| `RouteRegistrar` | `RegisterRoutes(chi.Router)` | After every start function succeeds |

A component may implement any subset of these interfaces.

## Setup

`Setup(ctx, router, components...)` walks the components in argument order
and builds three lists:

- start functions, one for each `Startable`;
- stop functions, one for each `Stoppable`;
- route registrars, one for each `RouteRegistrar`.

The three lists are independent. A component that implements only one
interface appears in only one list. `Setup` does not call `Start`, `Stop`, or
`RegisterRoutes`.

## Start

`Start` calls the start functions in order. When a start function returns an
error at index `i`, `Start` logs `error starting component #<index>` and then
calls `stops[j]` for each `j` from `i-1` down to `0`. Rollback uses
`context.Background()`. A stop error during rollback is logged and does not
replace the original start error. `Start` then returns the start error.

Routes are registered only after every start function returns nil.

The stop list is indexed independently from the start list. The indexes refer
to the same components only while every relevant component contributes to
both lists in the same order. A component that is `Startable` but not
`Stoppable`, or `Stoppable` but not `Startable`, shifts the later indexes.

An application that relies on startup rollback must keep those slices aligned:
each ordered startup component should implement both interfaces. `Start` does
not validate alignment and can panic when a failure index requires a stop
entry that does not exist. This constraint does not affect the reverse walk
performed by normal `Shutdown`.

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
