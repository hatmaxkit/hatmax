<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Orientation

Hatmax is a composable Go toolkit for server-rendered web applications. It
provides a deliberate application model: ordinary Go packages, explicit
dependencies, HTML rendered on the server, targeted HTMX interaction, and
Postgres-first infrastructure.

The goal of this chapter is to identify which decisions Hatmax establishes and
which decisions remain owned by the application.

## One Application Model

A Hatmax application is one Go process assembled from explicit components.
The process has a visible entrypoint, an ordered lifecycle, feature-owned
behavior, and infrastructure adapters selected by the application.

```text
request
   |
middleware -> handler -> service -> model
                  |          |
              templates     store -> Postgres
                  |
              HTML or HTMX response
```

This flow is not a runtime framework hidden behind configuration. The
application constructs each dependency and makes the order visible in Go.
Hatmax packages contribute the common capabilities used along the flow.

## What Hatmax Establishes

Hatmax provides an idiomatic path for responsibilities that web applications
repeat:

- application startup, route registration, rollback, and shutdown;
- configuration, logging, HTTP routing, and middleware;
- server-side templates and HTMX response behavior;
- form parsing and validation primitives;
- Postgres connections, migrations, models, and data helpers;
- authentication and cryptographic support;
- events, scheduled work, mail, images, telemetry, and test support.

These capabilities are designed to compose. When Hatmax provides a primitive
for a responsibility, the application uses that primitive instead of adding a
parallel framework for the same job.

## What the Application Owns

Hatmax does not supply the product domain. The application still owns:

- domain names, rules, states, and invariants;
- feature boundaries and application workflows;
- database schema and queries;
- routes, page content, templates, and visual identity;
- authorization policy and user-visible failure behavior;
- operational configuration for its selected services.

The distinction is important. Hatmax standardizes how these responsibilities
are expressed and assembled; it does not decide what the product means.

## Deliberate Constraints

The primary Hatmax path is server-rendered HTML enhanced by HTMX. Postgres is
the canonical persistence and durable-service foundation. Dependencies are
constructed explicitly, and fallible startup occurs in lifecycle methods
rather than constructors.

These constraints reduce architectural variation. They make feature layout,
startup behavior, validation, testing, and future assisted generation
predictable across Hatmax applications.

## How to Read the Guide

The next chapters begin with the whole application and then examine each
responsibility in context. Reference pages remain authoritative for exact
functions, interfaces, defaults, and errors. How-to guides remain the place
for focused procedures.

Continue with [Application Anatomy](application-anatomy.md).

---

[User Guide](README.md) · [Next: Application Anatomy](application-anatomy.md)
