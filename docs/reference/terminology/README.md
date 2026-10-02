<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Terminology

These names are the vocabulary for the User Guide and the reference. The
contracts they point at are documented in later reference pages. The
implementation notes remain in the package readmes.

## Hatmax

Hatmax is a composable Go toolkit. An application wires Hatmax packages
together. Hatmax does not require a hidden global registry for that wiring.

## Component

A component is a value passed to `app.Setup`. Setup keeps the capabilities
the value actually implements:

- `Startable` has `Start(context.Context) error`. `app.Start` calls it during
  startup.
- `Stoppable` has `Stop(context.Context) error`. Shutdown calls every
  collected stop function. Startup rollback calls a stop function only for
  the same component's successfully completed startup step.
- `RouteRegistrar` has `RegisterRoutes`. `app.Start` calls it after every
  start function has succeeded.

A component may implement any subset of these interfaces. The implementation
note is [app/readme.md](../../../app/readme.md).

## Ordered startup rollback

`app.Start` runs startup steps in the order `app.Setup` collected them. If
one start function returns an error, Hatmax walks completed steps in reverse
order and calls each step's optional stop function, then returns the original
start error. Routes are registered
only after every start function succeeds.

Each startup step pairs capabilities from the same component. Start-only
components have no rollback operation; stop-only components are stopped only
during normal shutdown. A failing start owns its own partial cleanup. Hatmax
does not provide a database transaction or an unconditional all-or-nothing
startup guarantee.

## Static configuration

Static configuration is the `config.Config` value loaded at process start.
`config.Load` reads a file, an environment prefix, and arguments.
`Config.Validate` checks that value before the application uses it. The
loaded value does not change while the process is running.

The implementation note is [config/readme.md](../../../config/readme.md).

## Settings

Settings are runtime key-value configuration. A `Schema` declares the key,
the type, and the validation rule. A `Registry` holds those schemas. A
`Store` reads and writes values. `Service` returns the stored value, or the
schema default when the store has no value.

Settings are not `config.Config`. Changing a setting does not reload the
static configuration.

The implementation note is [settings/readme.md](../../../settings/readme.md).

## Postgres-backed services

The database pool, the Postgres pubsub implementation, and the Postgres
scheduler backend are components passed through `app.Setup`. Auth sessions
are persisted through the auth `Queries` interface. The usual implementation
of that interface is Postgres.

Those services do not require Redis or a separate message broker. Replacing
one of them means passing a different component, or a different `Queries`
implementation, into application setup.

## Package readme

A package `readme.md` sits next to the package. It shows usage for that
package. It is the implementation note.

The reference states the contract without the usage walkthrough. The User
Guide teaches one step of an application and links to the reference for the
exact contract.
