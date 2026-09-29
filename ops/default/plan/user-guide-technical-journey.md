# User Guide Technical Journey Delivery Plan

Status: Approved
Delivery set: user-guide-technical-journey
Slice strategy: behavior-first
Reason: each slice replaces one consecutive part of the reader's journey with
a coherent, independently useful section. The guide remains navigable after
every merge, while the central application model expands from foundation to
web behavior, data, identity, supporting services, and operation.
Ticket: `ops/default/ticket/reviewing/20260929090213-rebuild-user-guide-technical-journey.md`
Spec: none
Tracker: `ops/default/tracker/user-guide-technical-journey.md`
Base branch: `dev`
Planning base: `4f783c75d9e9417d35ce4cc2e8a4118a10a89c19`
Active slice: Slice 4
Execution gate: satisfied when this approved activation is committed to `dev`

## Objective

Rebuild the Hatmax User Guide as an ordered technical journey through the
composition of a server-rendered web application. A reader must finish the
guide with a working mental model of the complete application, the role of
each Hatmax primitive, and the idiomatic way those primitives are assembled.

The guide is not a cumulative application tutorial. It may use small,
coherent code fragments and one consistent application vocabulary, but it
does not instruct the reader to build a project chapter by chapter. A separate
future tutorial may build the Todo application from `examples/ticked`.

## Editorial Contract

### Reader and outcome

The reader knows Go and ordinary web concepts but does not know Hatmax. The
guide begins with the kind of application Hatmax supports, shows its complete
shape early, and then revisits that shape as responsibilities are added.

Each chapter must:

- have one technical learning goal;
- state where the subject belongs in the application;
- explain how it connects to already introduced responsibilities;
- use current public Hatmax APIs and idiomatic project structure;
- link to reference for exact contracts, how-to guides for focused procedures,
  and explanation for design rationale;
- leave the reader with a concrete understanding that can be checked without
  requiring a cumulative local build.

### Diataxis boundary

The User Guide remains at `docs/tutorials/user-guide/` with the visible name
`User Guide`. It is progressive guidance, but it is not the future practical
Todo tutorial.

- The User Guide teaches how the parts of a Hatmax application fit together.
- The future Todo tutorial will teach by building one complete application.
- How-to guides retain focused procedures with known goals.
- Reference retains exact contracts and stable behavior.
- Explanation retains rationale, boundaries, and tradeoffs.

Do not reproduce full reference tables or turn chapters into command-driven
acceptance scripts. Commands are appropriate only when they clarify normal
product use.

### Existing material

Treat the current User Guide as source material, not as the editorial
foundation. Reuse verified facts, examples, recovery guidance, and links from:

- the current User Guide;
- `docs/reference/`;
- `docs/how-to/`;
- `docs/explanation/`;
- `examples/guide` and `examples/ticked`.

Remove or replace a current chapter only when its useful content is retained
in the new journey or deliberately delegated to another Diataxis quadrant.
Update all inbound links in the same slice that removes or renames a page.

### Narrative spine

The guide follows this technical progression:

1. Hatmax's application model and intended server-rendered web architecture.
2. Project anatomy, entrypoint, lifecycle, dependencies, and wiring.
3. HTTP request boundaries, middleware, templates, rendering, and HTMX.
4. Forms, validation, presentation primitives, and a complete feature shape.
5. Postgres, migrations, domain models, stores, services, and data helpers.
6. Authentication, sessions, cryptography, configuration, logging, and runtime
   settings.
7. Pubsub, scheduled work, mail, images, telemetry, and replaceable adapters.
8. Testing, operational evolution, and the experimental generator.

The complete feature shape is the organizing bridge between web behavior and
data. It must show the canonical flow:

```text
model -> store -> service -> handler -> templates -> wiring -> tests
```

The guide revisits this flow when later chapters add persistence, identity,
events, and supporting services.

### Primitive coverage

Every public package must have an intentional place in the journey. A package
does not require its own chapter when a concise treatment and a reference link
are sufficient.

| Journey Area | Hatmax Packages and Concepts |
| --- | --- |
| Foundation | `app`, `config`, `log`, `web` |
| Request and presentation | `middleware`, `render`, `htmx`, `ui`, `modal`, `format`, `pagination`, `i18n` |
| Feature and data | `model`, `validation`, `db`, `seed`, `slug` |
| Identity and runtime behavior | `auth`, `crypto`, `settings` |
| Supporting services | `pubsub`, `scheduler`, `mailer`, `image`, `telemetry` |
| Testing and substitution | `testhelper`, `fake`, public interfaces and adapters |
| Assisted evolution | generator CLI and its current product boundary |

Before closure, compare this matrix with the repository package map and record
any intentionally excluded internal-only surface.

### Generator boundary

Generation appears only at the end of the journey. The chapter must state the
current facts without expanding into generator design documentation:

- the generator is experimental;
- it currently exposes a CLI and is expected to gain a conversational TUI;
- it changes an existing compatible Hatmax application and does not create the
  application itself;
- it interprets requests within the Hatmax Book and requires inspection and
  approval before mutation;
- manual understanding of application composition remains necessary.

Detailed operations, schemas, daemon behavior, and exit statuses remain in the
generator reference.

## Planned Guide Shape

The exact filenames may be refined during a slice when link preservation
requires it, but the chapter responsibilities and order are fixed by this
plan:

1. Orientation.
2. Application Anatomy.
3. Lifecycle and Wiring.
4. The Request Boundary.
5. Pages and Partials.
6. Forms and Validation.
7. Presentation Primitives.
8. Feature Anatomy.
9. Persistence and Migrations.
10. Models and Data Flow.
11. Identity and Sessions.
12. Configuration and Runtime Settings.
13. Events and Background Work.
14. Application Services.
15. Testing and Evolution.
16. Assisted Generation.

The index must read as one route through these chapters. It may provide direct
links to other quadrants after the main route, but those links must not compete
with the reading order.

## Ordered Slices

| Slice | Short Name | Branch | Pull Request Title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | journey foundation | `docs/user-guide-journey-foundation` | `docs(slice-1): establish the User Guide journey` | `ops/default/report/slices/user-guide-technical-journey/slice-1-journey-foundation.md` |
| Slice 2 | web interaction | `docs/user-guide-web-interaction` | `docs(slice-2): explain the Hatmax web interaction model` | `ops/default/report/slices/user-guide-technical-journey/slice-2-web-interaction.md` |
| Slice 3 | feature anatomy | `docs/user-guide-feature-anatomy` | `docs(slice-3): establish canonical feature anatomy` | `ops/default/report/slices/user-guide-technical-journey/slice-3-feature-anatomy.md` |
| Slice 4 | data lifecycle | `docs/user-guide-data-lifecycle` | `docs(slice-4): explain the Hatmax data lifecycle` | `ops/default/report/slices/user-guide-technical-journey/slice-4-data-lifecycle.md` |
| Slice 5 | identity and runtime | `docs/user-guide-identity-runtime` | `docs(slice-5): explain identity and runtime configuration` | `ops/default/report/slices/user-guide-technical-journey/slice-5-identity-runtime.md` |
| Slice 6 | application services | `docs/user-guide-application-services` | `docs(slice-6): explain Hatmax application services` | `ops/default/report/slices/user-guide-technical-journey/slice-6-application-services.md` |
| Slice 7 | testing and evolution | `docs/user-guide-testing-evolution` | `docs(slice-7): complete the User Guide journey` | `ops/default/report/slices/user-guide-technical-journey/slice-7-testing-evolution.md` |

## Slice 1: Journey Foundation

### Purpose

Replace the example-driven entry with the reader contract, complete journey,
Hatmax application model, project anatomy, lifecycle, and wiring. Establish
wiring as a recurring concern rather than an appendix.

### Tasks

- T1.1: Rewrite the User Guide index around the complete technical journey and
  its boundary from tutorials, how-to, reference, and explanation.
- T1.2: Add orientation and application-anatomy chapters that show the complete
  application shape before individual primitives.
- T1.3: Integrate lifecycle and wiring into the main path, preserving accurate
  startup, rollback, shutdown, and component-order material.
- T1.4: Retire or redirect superseded foundation pages and update every inbound
  documentation link.

### Validation

- `make docs-check`
- Confirm the index exposes one unambiguous reading order.
- Confirm application anatomy includes the entrypoint and complete component
  flow without prescribing a cumulative build.
- `git diff --check`

## Slice 2: Web Interaction

### Purpose

Explain how an HTTP request crosses Hatmax boundaries and becomes a full page
or HTMX response. Connect middleware, rendering, presentation helpers, forms,
and validation as one interaction model.

### Tasks

- T2.1: Add the request-boundary and pages-and-partials chapters.
- T2.2: Add the forms-and-validation chapter, including same-origin protection
  and server-owned validation behavior.
- T2.3: Add the presentation-primitives chapter covering `ui`, `modal`,
  `format`, `pagination`, and `i18n` at the appropriate level.
- T2.4: Replace superseded page and form exercises while preserving useful
  recovery guidance and links.

### Validation

- `make docs-check`
- Confirm every request and presentation package in the coverage matrix is
  introduced or deliberately linked.
- `git diff --check`

## Slice 3: Feature Anatomy

### Purpose

Give the guide its central composition model: one idiomatic Hatmax feature
from domain model through tests. Show responsibilities, dependency direction,
and assembly without building a specific application step by step.

### Tasks

- T3.1: Add the feature-anatomy chapter with the canonical model, store,
  service, handler, templates, wiring, and tests flow.
- T3.2: Show how validation, HTMX responses, errors, and dependencies cross
  feature layers without leaking alternate framework patterns.
- T3.3: Reconcile the chapter with current examples and the Hatmax Book, then
  link exact contracts without copying generator-owned rules into the guide.

### Validation

- `make docs-check`
- Confirm the feature can be understood independently of the generator and
  without following a cumulative tutorial.
- Confirm `main.go` is presented only as the application entrypoint and
  assembly root.
- `git diff --check`

## Slice 4: Data Lifecycle

### Purpose

Follow application data from migration and connection startup through model,
store, service, query results, pagination, seed data, and stable identifiers.

### Tasks

- T4.1: Add the persistence-and-migrations chapter with Postgres-first startup,
  failure containment, migration ownership, and transaction boundaries.
- T4.2: Add the models-and-data-flow chapter covering `model`, stores,
  services, `seed`, `slug`, and `pagination` in the feature flow.
- T4.3: Replace the isolated Postgres and record exercises while retaining
  accurate durable-state and restart behavior.

### Validation

- `make docs-check`
- Confirm the guide separates Hatmax primitives from application-owned schema,
  queries, and domain behavior.
- `git diff --check`

## Slice 5: Identity and Runtime

### Purpose

Explain how identity and runtime policy enter the application without breaking
the feature and wiring model established earlier.

### Tasks

- T5.1: Add the identity-and-sessions chapter covering authentication,
  request context, authorization boundaries, and cryptographic support.
- T5.2: Add the configuration-and-runtime-settings chapter covering static
  configuration, logging, runtime settings, and their distinct lifetimes.
- T5.3: Replace the sign-in and settings exercises while preserving useful
  failure and recovery material.

### Validation

- `make docs-check`
- Confirm authentication, cryptography, configuration, logging, and settings
  have distinct responsibilities and explicit wiring points.
- `git diff --check`

## Slice 6: Application Services

### Purpose

Extend the same application model to asynchronous work and replaceable
services without turning the guide into a catalog of unrelated packages.

### Tasks

- T6.1: Add the events-and-background-work chapter covering pubsub, scheduled
  work, delivery boundaries, durability, and failure handling.
- T6.2: Add the application-services chapter covering mail, images, telemetry,
  and the interfaces that keep adapters replaceable.
- T6.3: Replace the isolated background-work exercise and connect every service
  to feature and application wiring.

### Validation

- `make docs-check`
- Confirm `pubsub`, `scheduler`, `mailer`, `image`, and `telemetry` appear as
  application responsibilities rather than a package inventory.
- `git diff --check`

## Slice 7: Testing and Evolution

### Purpose

Close the journey with testing, substitution, operational evolution, and a
brief factual introduction to assisted generation.

### Tasks

- T7.1: Add the testing-and-evolution chapter covering `testhelper`, `fake`,
  interface substitution, representative validation, and safe application
  evolution.
- T7.2: Rewrite the generation chapter as a short experimental capability at
  the end of the guide, preserving detailed contracts in reference.
- T7.3: Complete the primitive-coverage audit, remove obsolete guide pages,
  repair navigation, and perform a full editorial consistency pass.

### Validation

- `make docs-check`
- Confirm every public package in the package map is intentionally covered or
  linked from the journey.
- Confirm the guide does not contain or impersonate the future Todo tutorial.
- Confirm the generator is last, experimental, existing-project-only, CLI
  today, and TUI-directed without speculative implementation detail.
- `git diff --check`

## Delivery-Set Gate

After Slice 7 is merged into `dev`:

1. Run `make docs-check` against the exact integrated commit.
2. Read the User Guide from its README entrypoint in order and record the
   editorial consistency result in the final slice report.
3. Compare the primitive-coverage matrix with `docs/reference/package-map/`
   and the root Go packages.
4. Confirm the future practical Todo tutorial remains separate and was not
   started by this delivery set.
5. Close the tracker only when the exact `dev` candidate passes these gates.

## Prerequisites

- The ticket, plan, and tracker are reviewed and committed on `dev`.
- The maintainer approved this plan and the order of delivery.
- The documentation-layout-normalization delivery set is closed on `dev` at
  `32038ac34893`.
- The existing User Guide, reference, how-to, explanation, examples, and
  exported APIs remain the evidence base for each slice.
- Each slice starts only after the preceding slice is merged into `dev`.
