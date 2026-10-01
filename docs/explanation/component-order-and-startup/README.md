<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Component Order and Startup Rollback

Hatmax treats application assembly as an explicit ordered list. `app.Setup`
inspects each value for startup, shutdown, and route-registration capabilities.
`app.Start` starts the collected startup functions before it registers any
route. This prevents handlers from receiving traffic while their declared
startup dependencies are still unavailable.

## Order is dependency information

A component must appear after every component it uses during `Start`. A
database-backed store therefore follows the database connection. A subscriber
follows its broker. A route registrar follows the services used by its
handlers.

This order is visible in the application entrypoint. Hatmax does not infer a
dependency graph or hide startup in constructors.

## Startup and shutdown capabilities are independent

`Setup` pairs each start function with the same component's optional stop
function. A startup failure rolls back completed pairs in reverse order.
Start-only components do not borrow another component's stop function;
stop-only components participate in normal shutdown, not startup rollback.
Route-only components do not affect either sequence.

Components implement only the capabilities they genuinely own. No placeholder
stop function or positional alignment is needed. Normal shutdown still calls
every collected stop function in reverse component order.

This pairing preserves cleanup ownership, not a general transaction guarantee.
A failing start must clean up its own partial initialization, and a component
without a stop capability has no rollback operation. The
[Application Lifecycle Reference](../../reference/application-lifecycle/README.md)
states the exact execution behavior.

## Constructors do not perform startup

Constructors should retain dependencies and configuration. Network access,
schema setup, template parsing, polling loops, and other fallible work belong
in `Start`. This keeps assembly deterministic and gives startup one visible
failure boundary.
