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

`Setup` collects `Startable` and `Stoppable` values into separate slices.
`Start` uses the position of a failing start function to select rollback stop
functions. Consequently, rollback corresponds to components only while the
startup and shutdown slices remain positionally aligned.

Applications that rely on rollback should make every ordered startup
component implement both `Startable` and `Stoppable`. A component with only
one of those capabilities shifts one slice and invalidates that assumption.
Normal shutdown still calls every collected stop function in reverse order.

This is a current API constraint, not a general transaction guarantee. The
[Application Lifecycle Reference](../../reference/application-lifecycle/README.md)
states the exact execution behavior.

## Constructors do not perform startup

Constructors should retain dependencies and configuration. Network access,
schema setup, template parsing, polling loops, and other fallible work belong
in `Start`. This keeps assembly deterministic and gives startup one visible
failure boundary.
