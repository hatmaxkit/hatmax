<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Application Anatomy

A Hatmax application separates product behavior from infrastructure while
keeping their assembly visible. The goal of this chapter is to locate each
responsibility before examining individual packages.

## The Complete Shape

An idiomatic application commonly has this shape:

```text
main.go
config.yaml
assets/
  migration/postgres/
  static/
  templates/
db/
  queries/
internal/
  dal/
  feat/
    <feature>/
      model.go
      store.go
      postgres_store.go
      service.go
      handler.go
      *_test.go
```

Not every application needs every directory. The boundaries matter more than
the directory count: the entrypoint assembles, features own behavior, adapters
own external details, and assets remain explicit inputs.

## The Composition Root

`main.go` is the composition root. It contains one `main` function and no
feature behavior. That function performs the visible assembly sequence:

1. load and validate static configuration;
2. create the logger, context, and router;
3. construct infrastructure components;
4. construct feature stores, services, and handlers;
5. order lifecycle and route components;
6. start the application and serve HTTP;
7. coordinate cancellation and reverse-order shutdown.

Constructors receive dependencies and lightweight configuration. They do not
open connections, run migrations, parse templates, or start background work.
Those fallible actions belong to explicit lifecycle methods.

## Application Components

A value becomes an application component by implementing one or more Hatmax
lifecycle interfaces:

- `app.Startable` participates in ordered startup;
- `app.Stoppable` participates in rollback and shutdown;
- `app.RouteRegistrar` contributes routes after startup succeeds.

Plain domain services do not need to implement these interfaces. The
entrypoint constructs them explicitly and passes them to the components that
use them. Only lifecycle and route responsibilities enter `app.Setup`.

## Feature Ownership

A feature owns one cohesive application capability. Its normal dependency
direction is:

```text
handler -> service -> model
              |
            store interface <- Postgres adapter
```

The handler owns HTTP and rendering decisions. The service owns application
workflows. The model owns durable domain invariants. The store interface is
owned by its consumer, while the Postgres implementation owns queries and
mapping details.

Templates belong to the feature presentation surface, even when embedded from
the application-level `assets/` tree. Tests stay beside the layer whose
behavior they protect.

The later Feature Anatomy chapter develops this flow in detail. At this stage,
the important point is that a feature is a complete vertical responsibility,
not only a handler or database table.

## Shared Infrastructure

Infrastructure components serve features without absorbing their product
rules. Typical shared components include:

- the database connection and migrator;
- the template manager;
- authentication and session services;
- the event broker and scheduled job store;
- mail, image, and telemetry adapters.

Features depend on the smallest interfaces they need. Concrete adapters remain
visible at the composition root, which keeps substitution and startup order
reviewable.

## Reading an Application

To understand an unfamiliar Hatmax application, read it in this order:

1. inspect `main.go` to see the selected capabilities and their order;
2. inspect one feature from handler through service and model;
3. identify the store interface and concrete adapter;
4. inspect templates and routes for the user-visible behavior;
5. inspect tests for the protected contracts;
6. use the Hatmax reference for exact shared-package behavior.

This order reveals the product architecture without treating generated files,
configuration, or framework conventions as hidden control flow.

---

[Previous: Orientation](orientation.md) · [User Guide](README.md) ·
[Next: Lifecycle and Wiring](lifecycle-and-wiring.md)
