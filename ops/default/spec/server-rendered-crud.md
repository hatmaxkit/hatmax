# Canonical Server-Rendered CRUD

Status: Approved
Kind: Subordinate archetype specification
Umbrella: `ops/default/spec/interactive-hatmax-generator.md`
Book: `ops/default/spec/hatmax-book.md`
Intent: `ops/default/spec/intent-and-planning.md`

## Purpose

`server_rendered_crud` is the first canonical Hatmax feature archetype. It
defines one application structure for a Postgres-backed feature rendered by Go
templates and enhanced with HTMX.

The archetype is opinionated by design. A generator must not choose between a
repository layer, active record, JSON API, client-side application, alternate
router, or alternate validation framework.

## Current Hatmax Evidence

The archetype derives from these established Hatmax patterns:

- explicit ordered composition through `app.Setup` and `app.Start`;
- constructors retain dependencies while fallible startup belongs in `Start`;
- interfaces are owned by consumers and concrete infrastructure stays in
  adapters;
- `db.Database` and `db.Migrator` own Postgres lifecycle and migrations;
- `web.TemplateManager` owns template parsing and rendering;
- handlers parse requests, invoke services, and select full or partial views;
- Hatmax `validation`, `web`, `htmx`, `middleware`, `model`, and `log`
  primitives are used when applicable;
- `examples/ticked` demonstrates feature model, store, service, SQLC, HTMX,
  and explicit main wiring.

Examples remain evidence rather than authority. This specification resolves
their inconsistencies for generated features.

## Canonical Layout

For a feature named `<feature>`, generation uses:

```text
internal/feat/<feature>/
├── model.go
├── model_test.go
├── store.go
├── postgres_store.go
├── postgres_store_test.go
├── service.go
├── service_test.go
├── handler.go
└── handler_test.go

assets/migration/postgres/<sequence>-<feature>.sql
assets/templates/<feature>/page.html
assets/templates/<feature>/form.html
assets/templates/<feature>/row.html
db/queries/<feature>.sql
main.go
```

Files that have no admitted obligation are omitted. A feature can add more
templates or domain files only through a declared implementation slot.

The handler is feature-owned. A generated feature must not append unrelated
routes and dependencies to one application-wide handler. Cross-feature page
composition may use a separate application-owned web component.

## Model

The feature package owns domain types, invariants, and domain errors.

- IDs use Hatmax model primitives.
- Creation and update times use `model.Now` and the applicable Hatmax helpers.
- Constructors establish valid initial state.
- Domain methods enforce rules that must hold independently of HTTP or storage.
- Expected domain failures use stable sentinel or typed errors.
- Domain types do not depend on HTTP, templates, SQLC rows, or database handles.

Validation that protects a durable invariant must exist in the domain and, when
possible, as a Postgres constraint. Form validation may add user-facing detail
but cannot replace the invariant.

## Postgres Persistence

Generated CRUD persistence is Postgres-first and uses:

- an ordered reversible migration;
- explicit primary keys, foreign keys, nullability, uniqueness, checks, and
  indexes required by supported queries;
- SQLC query definitions and generated adapters;
- a consumer-facing store interface;
- a concrete `PostgresStore` adapter;
- transactions for multi-record invariants;
- contextual error wrapping and stable not-found translation.

The generator may use an existing project SQLC configuration. If the project
does not have the required SQLC pipeline, the plan must include its canonical
bootstrap obligations. It must not fall back to ad hoc SQL silently.

The store receives a database provider in its constructor. It acquires the
live connection in `Start` and implements `Stop` so lifecycle slices remain
aligned. Constructors do not perform I/O.

## Service

The service owns application workflows and depends on the feature store
interface plus explicitly selected Hatmax capabilities.

- Handlers do not call SQLC or the database directly.
- Persistence occurs only after domain validation succeeds.
- Events, mail, jobs, or other effects are capabilities, not implicit CRUD
  behavior.
- A nil optional dependency is allowed only when the Book marks that
  capability optional and defines the behavior.
- Errors returned to handlers preserve classification without exposing
  infrastructure details.

## Handler and Routes

The feature handler:

- owns consumer-side service interfaces;
- implements `RegisterRoutes(chi.Router)`;
- uses canonical REST-like paths declared by the intent and normalized by the
  Book;
- uses `web.ParseForm` for form data;
- normalizes and validates fields through Hatmax validation primitives;
- maps domain and service failures to explicit HTTP behavior;
- logs internal context without exposing it to users;
- renders full pages or named partials through `web.TemplateManager`;
- returns structured, user-safe field errors for invalid forms.

State-changing routes rely on the canonical same-origin middleware installed
at router composition. A handler must not introduce a second CSRF or origin
framework when Hatmax owns the protection.

## HTMX and Templates

HTML rendered by Go templates is authoritative. HTMX adds targeted requests
and replacement behavior without creating a client-side domain model.

- Templates use Hatmax HTMX helpers when a matching helper exists.
- Raw `hx-*` attributes require a Book-declared gap or extension.
- A non-HTMX request has a complete server-rendered response or explicit
  redirect behavior.
- Partial endpoints render the smallest stable named fragment.
- Targets and swaps use stable element IDs derived from feature identity.
- Client-side validation is expressed through admitted HTML or HTMX behavior;
  it does not replace server or domain validation unless the user explicitly
  requests a client-only rule that protects no durable invariant.
- Trusted `template.HTML` and `template.HTMLAttr` values come only from Hatmax
  helpers or reviewed application extension points.

## Wiring

`main.go` remains the visible composition root. A generated feature adds its
components after their dependencies:

1. database;
2. migrator;
3. template manager;
4. feature Postgres store;
5. feature service dependencies;
6. feature service;
7. feature handler.

Only lifecycle and route-registering components need to appear in the
`app.Setup` dependency list. Plain services are constructed explicitly and
passed to their consumers.

The generator must preserve existing order and insert components at the first
position that satisfies all declared dependencies. It must not infer or hide a
runtime dependency graph.

## Tests

The archetype requires:

- table-driven domain tests for valid and invalid transitions;
- service tests with small consumer-owned fakes;
- handler tests for full requests, HTMX partials, invalid forms, and mapped
  failures;
- Postgres integration tests for queries, constraints, transactions, and
  not-found behavior;
- migration and SQLC generation validation;
- a composition test or compile gate that covers main wiring.

Tests use Hatmax `testhelper` and `fake` support when applicable. A generated
test must catch a meaningful contract regression; compilation-only assertions
do not satisfy behavioral obligations.

## Operation Expansion

### `create_feature`

Creates every required surface for the admitted capabilities. It must not
leave unwired packages, unlinked templates, or placeholder methods.

### `add_field`

Expands to every affected representation: migration, SQLC queries and rows,
domain model, mapping, service inputs, form parsing, validation, templates,
fixtures, and tests. Optional or computed fields may remove obligations only
through Book rules.

### `add_validation`

Identifies the invariant owner and expands to applicable domain, persistence,
form, HTML or HTMX, and test surfaces. A request for client-only validation
must be represented explicitly and is rejected when the rule protects durable
state.

## Documentation Boundary

The archetype never adds documentation by default. When documentation intent
is active, the plan adds only the warranted Diataxis surfaces and index links.

## Prohibited Forms

The generator must reject or diagnose:

- direct database access from handlers;
- business rules implemented only in templates or JavaScript;
- an alternate router, ORM, validation framework, or client application model;
- hidden service construction inside handlers;
- startup I/O in constructors;
- unregistered migrations, routes, templates, or lifecycle components;
- raw HTMX markup where an applicable Hatmax helper exists;
- generated TODOs, stubs, or knowingly incomplete cross-surface changes.

## Acceptance Criteria

- Every generated CRUD feature has one feature-owned model, store, service,
  handler, template set, and explicit wiring path.
- Postgres schema, SQLC queries, domain mapping, forms, and templates agree.
- Hatmax primitives are used for lifecycle, HTTP, HTMX, validation, model,
  middleware, logging, rendering, and testing where applicable.
- `create_feature`, `add_field`, and `add_validation` expand deterministically.
- Generated features remain ordinary Go code without a runtime generator.
- Conformance can reject a compiling implementation that violates the
  canonical structure.
