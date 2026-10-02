<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# User Guide

The User Guide explains how a Hatmax web application is composed. Read it in
order to build a working mental model of the application before consulting
individual package contracts or focused procedures. Use that model through
the recommended conversational TUI, the supported headless CLI, or direct Go
development.

This guide is not a step-by-step project tutorial. It uses small, connected
examples to explain responsibilities and assembly. A separate tutorial will
later build a complete Todo application.

## Audience

The guide assumes familiarity with Go, HTTP, HTML, and relational databases.
It does not assume previous Hatmax experience.

Hatmax is intended for server-rendered Go applications that use explicit
dependency wiring, HTMX for targeted interaction, and Postgres-first
infrastructure. The guide treats those choices as one application model, not
as a menu of interchangeable frameworks.

## Technical Journey

The journey follows the order in which an application is understood and
designed:

1. [Orientation](orientation.md) defines the kind of application Hatmax
   supports and the boundary between Hatmax and application-owned behavior.
2. [Application Anatomy](application-anatomy.md) maps the entrypoint,
   application components, features, infrastructure, templates, and tests.
3. [Lifecycle and Wiring](lifecycle-and-wiring.md) explains how dependencies
   become an ordered, running process.
4. [The Request Boundary](request-boundary.md) follows an HTTP request through
   middleware and routing.
5. [Pages and Partials](pages-and-partials.md) connects templates, rendering,
   and HTMX responses.
6. [Forms and Validation](forms-and-validation.md) places request, field,
   domain, and persistence validation at their correct boundaries.
7. [Presentation Primitives](presentation-primitives.md) covers UI helpers,
   modals, formatting, pagination, and internationalization.
8. [Feature Anatomy](feature-anatomy.md) connects model, store, service,
   handler, templates, wiring, and tests as one canonical feature.
9. [Persistence and Migrations](persistence-and-migrations.md) explains the
   Postgres lifecycle and schema ownership.
10. [Models and Data Flow](models-and-data-flow.md) follows application data
    across feature layers.
11. [Identity and Sessions](identity-and-sessions.md) adds authentication,
    authorization, and cryptographic support.
12. [Configuration and Runtime Settings](configuration-and-runtime-settings.md)
    separates startup configuration, logging, and mutable settings.
13. [Events and Background Work](events-and-background-work.md) introduces
    pubsub and scheduled work.
14. [Application Services](application-services.md) places mail, images,
    telemetry, and replaceable adapters.
15. [Testing and Evolution](testing-and-evolution.md) explains how to validate
    and safely extend the assembled application.
16. [Assisted Generation](assisted-generation.md) teaches the recommended
    conversational TUI and presents the headless CLI as a supported
    alternative for focused work.

## Journey Outcome

After this journey, you should be able to read a Hatmax composition root,
locate feature and infrastructure ownership, follow a request and its data
through every boundary, place new behavior in the canonical vertical slice,
and select tests that prove the resulting contracts.

The guide establishes that application model before assisted generation on
purpose. Generated changes remain reviewable ordinary Go, SQL, templates, and
explicit wiring; the same architectural understanding applies whether a
change is written manually or proposed by Hatmax.

## Supporting Documentation

- Use the [How-to Guides](../../how-to/README.md) for focused procedures.
- Use the [Reference](../../reference/README.md) for exact APIs, configuration,
  limits, and behavior.
- Use [Explanation](../../explanation/README.md) for design rationale and
  tradeoffs.
- Use the [Package Map](../../reference/package-map/README.md) to locate a
  specific Hatmax capability.
