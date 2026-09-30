<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Testing and Evolution

A Hatmax application is easiest to change when tests follow its ownership
boundaries. The goal is not one test per file; it is evidence that domain,
workflow, adapter, HTTP, lifecycle, and assembly contracts still agree.

## Test Each Boundary at Its Natural Size

Use the smallest test that can catch the relevant regression:

| Boundary | Useful evidence |
| --- | --- |
| Model | valid and invalid construction, transitions, stable errors |
| Service | workflow order, authorization, effects, error preservation |
| Handler | parsing, full pages, HTMX partials, form errors, status mapping |
| Postgres store | queries, constraints, transactions, row mapping, not found |
| Supporting adapter | provider encoding, cleanup, and failure translation |
| Composition | application build, startup order, route visibility |

Model tests are ordinary table-driven Go tests. Service and handler tests use
small consumer-owned fakes rather than replacing the complete application.
Keep each fake beside the test that owns its interface and record only the
calls needed by the assertion.

Hatmax `fake` implementations cover mail and telemetry when those exact
interfaces are needed. Scheduler supplies fake stores and clocks. Pubsub
offers no-op and memory brokers. `testhelper.SetupTestDB` creates an isolated
Postgres schema or test container and returns cleanup for integration tests.

## Test Server-Rendered Behavior

Handler tests should distinguish ordinary and HTMX requests. Assert the named
page or partial, safe validation feedback, redirect or HTMX response header,
and whether the service was called. Avoid asserting large generated HTML
strings when a stable semantic element or renderer call proves the contract.

Template parsing belongs in startup or focused template tests. A handler fake
that records `Render` and `RenderPartial` isolates HTTP decisions; a smaller
integration test can then prove that the real embedded template set parses and
executes.

## Use Real Postgres for Persistence Claims

An in-memory fake can prove that a service calls `Store.Create`; it cannot
prove SQL syntax, a unique constraint, row locking, nullable mapping, or
transaction rollback. Use real Postgres for those claims and run migrations
or the feature schema required by the test.

Keep integration data inside the isolated schema returned by `testhelper`.
Always invoke cleanup, and test stable feature errors rather than leaking raw
driver messages into higher layers.

## Evolve a Feature Across Every Surface

Before editing, state the behavioral change and enumerate its affected
surfaces. A stored field can require migration, SQLC queries and generated
types, model, store mappings, service input, handler parsing, templates,
wiring, fixtures, and tests. A new effect can require configuration, an
adapter, lifecycle order, failure policy, and operational monitoring.

Work from durable state outward, then verify the round trip in both
directions. Preserve compatibility when application instances and database
schema may be deployed at different times. Prefer additive migrations and
explicit transitions over rewriting applied history.

## Run Gates in Increasing Scope

Use focused package tests while changing one boundary, then run repository
gates before delivery. In this repository the normal progression is:

```text
go test ./path/to/changed/package
make docs-check
make check
```

`make check` formats, vets, tests, checks the coverage threshold, and runs the
strict lint rules. A consuming application can define additional SQLC,
template, browser, or deployment checks; those project-owned commands are part
of its acceptance contract.

Live provider checks and generator smoke tests are opt-in because they require
external services or authenticated runtimes. Do not replace deterministic
tests with them. Run them when the changed boundary depends on the real
integration, and record exactly which candidate they exercised.

## Review the Assembled Application Again

After a cross-cutting change, read `main.go` from top to bottom. Confirm that
constructors remain free of I/O, lifecycle components follow their
dependencies, plain services are not added to `app.Setup`, routes appear only
after successful startup, and shutdown order remains valid.

Then inspect the user-visible path from request to response and the durable
path from model to Postgres. Passing tests do not justify a second router,
validation framework, hidden dependency container, or client-side domain
model that contradicts the Hatmax application model.

For exact testing helpers, see [Fake](../../reference/fake/README.md) and
[Test Helper](../../reference/testhelper/README.md). Revisit
[Feature Anatomy](feature-anatomy.md) when planning a cross-surface change.

---

[Previous: Application Services](application-services.md) ·
[User Guide](README.md) ·
[Next: Assisted Generation](assisted-generation.md)
