# User Guide

The User Guide follows the current Hatmax repository API. It takes a reader
from the first running process through web interaction, persistence,
authentication, background work, and runtime settings.

## Before You Begin

Use Go 1.24 or later and a checkout of this repository. Chapters that use
Postgres require a reachable Postgres server and the `psql` client. Run all
commands from the repository root unless a chapter changes directory.

The small `examples/guide` application is the cumulative companion for most
chapters. `examples/ticked` demonstrates the complete authentication and
application structure.

## Start Here

Begin with [Run the First Process](getting-started.md). It requires no database
and produces both a health response and an HTML page.

## Learning Path

### Foundation

1. [Run the First Process](getting-started.md) starts the companion application
   without infrastructure.
2. [Add Postgres](postgres.md) enables the database-backed components and
   verifies startup ordering.
3. [Serve a Page](pages.md) traces embedded templates and an HTMX partial.
4. [Accept a Form](forms.md) adds same-origin protection and validation.

### Application Workflows

5. [Sign In](sign-in.md) runs the Ticked example and follows the session
   boundary.
6. [Save a Record](records.md) verifies durable todo-list state.
7. [Work Outside the Request](background-work.md) follows an event from a
   request to the durable audit subscriber.
8. [Change Settings at Runtime](settings.md) changes a schema-checked setting
   without reloading static configuration.

## Direct Routes

- To start a separate module, use
  [Bootstrap a Hatmax Application](../../how-to/bootstrap-application/index.md).
- To add one known capability, use the
  [How-to Guides](../../how-to/index.md).
- To inspect exact behavior, use the
  [Reference](../../reference/index.md).
- To understand package boundaries and tradeoffs, use
  [Explanation](../../explanation/index.md).
- To inspect component order and swappable boundaries, use
  [Wiring](wiring.md).

The [Package Map](../../reference/package-map/index.md) lists capabilities
that are not required chapters, including image storage, mail delivery,
telemetry, crypto primitives, and test helpers.
