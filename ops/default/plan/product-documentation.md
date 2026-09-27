# Product Documentation Delivery Plan

Status: Approved
Delivery set: product-documentation
Slice strategy: behavior-first
Reason: each slice publishes one reference subject or one User Guide chapter
that a reader can open on its own. The reference subjects land first so each
chapter can link to contracts that already exist. A layered split would leave
the guide and the contracts in separate unfinished states.
Spec: none
Tracker: `ops/default/tracker/product-documentation.md`
Base branch: `dev`
Planning base: `6d91be1`
Active slice: Slice 5, auth reference
Execution gate: Slice 5 pull request #8 is open against `dev`

## Objective

Publish two Diataxis surfaces for Hatmax:

- the User Guide, the progressive path for someone building an application
  with the toolkit;
- the reference quadrant, the factual contracts for those packages.

The shape follows Odra's documentation layout. The content is Hatmax's
current Go API and the current application wiring, not Odra's editor.

Package `readme.md` files stay the implementation notes next to the code.
This delivery does not rewrite them.

## Slicing Result

Slices 1–8 publish the reference. Slices 9–16 publish the User Guide in
learning order.

1. open the documentation entry, the User Guide entry, the reference index,
   and the shared vocabulary;
2. document application lifecycle, static configuration, runtime settings, and logging;
3. document the HTTP, HTMX, middleware, and render contracts;
4. document the UI kit and the presentation helpers it depends on;
5. document authentication and cryptographic contracts;
6. document persistence, model primitives, validation, seeding, and slugs;
7. document image storage and processing;
8. document mail, pubsub, scheduling, telemetry, and test doubles, then close
   the reference index;
9. teach the first running process;
10. teach adding Postgres;
11. teach serving one page;
12. teach accepting one form;
13. teach signing in;
14. teach saving one record;
15. teach work outside the request;
16. teach runtime settings, add the wiring appendix, and close the User Guide.

## Planning Decisions

### Quadrant boundary

This delivery publishes the User Guide and the reference quadrant.

The User Guide lives at `docs/tutorials/user-guide/`. Its visible name is
`User Guide`. `docs/tutorials/index.md` points at that guide and does not
add a second tutorial series.

How-to guides and explanation are not part of this delivery. `docs/index.md`
does not add empty bodies for them.

A reference page states current contracts: exported types, functions,
interfaces, options, errors, limits, and lifecycle or failure behavior. It
does not teach a setup path and does not argue why the design exists.

A User Guide chapter has one learning goal, the actions that reach it, and
the result the reader can check. It links to the reference pages for exact
contracts. It does not copy those contracts.

### Document shape

Use these paths:

```text
docs/index.md
docs/tutorials/index.md
docs/tutorials/user-guide/index.md
docs/tutorials/user-guide/<chapter>.md
docs/reference/index.md
docs/reference/<subject>/index.md
```

The User Guide index lists the chapters in learning order and links a chapter
only after that chapter's file exists. The reference index links every
reference subject from this delivery. A page that is not linked from its
index is not delivered.

Write in English. Do not put state in a title suffix. Package readmes are
linked as implementation notes, not copied.

### User Guide path

The reader is someone composing a Hatmax application. The path is:

1. Getting Started. Load configuration, create a logger and router, and reach
   `app.Start`. The process stays up.
2. Add Postgres. Start a database component and see startup stop cleanly when
   that component fails.
3. Serve a Page. Return one HTML page and complete one HTMX update.
4. Accept a Form. Reject invalid input and accept valid input, including
   same-origin protection.
5. Sign In. Reach a page that only a signed-in request can see.
6. Save a Record. Store one record that is still there after a restart.
7. Work Outside the Request. A saved change causes one background effect
   through pubsub, the scheduler, or the mailer.
8. Change Settings at Runtime. Read one settings value that is not the static
   configuration loaded at startup.

### Appendices

Appendices are links or one short page beside the learning path. They are
not chapters and they do not restate a reference contract.

The User Guide index gains an appendix only when its target already exists:

- Terminology. Link to `docs/reference/terminology/index.md` from Slice 1.
- Package contracts. Link to `docs/reference/index.md` from Slice 1.
- Wiring. One page, `docs/tutorials/user-guide/wiring.md`, written in Slice 16.
  It maps `app.Setup` order and the `Mailer`, `Publisher`, `Subscriber`, and
  `JobStore` interfaces, and it links to the lifecycle, database, pubsub,
  scheduler, and mailer reference instead of copying them.
- Image storage, telemetry, and crypto primitives. Links from Slice 16 to
  those reference pages. They are not chapters.
- Ticked. Link from Slice 16 to `examples/ticked/README.md` as a finished
  application. The guide does not reproduce its endpoints.

`testhelper` and `fake` do not get an appendix. They are test support, not
part of building an application.

Code samples use the current constructors and signatures.

### Source of truth

When `docs/features.md`, a package `readme.md`, and the exported code
disagree, both the reference and the User Guide follow the exported code and
name the disagreement. They do not preserve a stale summary to match an
older table.

### Existing documents

Leave these files in place:

```text
docs/features.md
docs/gallery.md
docs/CHANGELOG.md
```

Do not move them into a quadrant in this delivery set. `docs/index.md` may
link the gallery and the changelog as existing documents outside the four
quadrants.

Do not edit package `readme.md` files. Do not document `examples/ticked` as a
reference subject. Its README remains the example's own guide.

### Repository gate

Every slice in this set changes documentation only. The repository playbook
skips the runtime aggregate gate for a Markdown-only diff. The focused gate
for every slice is:

```sh
git diff --check
```

Also confirm that every reference page added or renamed by the slice is
linked from `docs/reference/index.md`, and that every User Guide chapter
added by the slice is linked from `docs/tutorials/user-guide/index.md`.

The delivery-set gate, after Slice 16 is on `dev`, is the same docs-only gate
for the User Guide and the reference tree. Do not run `make check` or
`make ci` unless a slice changes Go source, which this plan does not allow.

## Prerequisites

- Planning base `6d91be1` is the current `dev` commit. It is one commit ahead
  of `origin/dev`.
- No stable specification precedes this delivery. The User Guide and the
  reference pages are the product artifacts.
- Package `readme.md` files and exported Go declarations already exist for
  every package named below.
- This plan and tracker are approved and committed on `dev` before Slice 1
  starts.
- Slice 1 is the active slice only after that commit. Later slices start only
  after the previous slice is merged into `dev`.

## Ordered Slices

| Slice | Short Name | Branch | Pull Request Title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | documentation map | `docs/product-documentation` | `docs(slice-1): add the documentation map` | `ops/default/report/slices/product-documentation/slice-1-documentation-map.md` |
| Slice 2 | core reference | `docs/slice-2-core-reference` | `docs(slice-2): document lifecycle and configuration` | `ops/default/report/slices/product-documentation/slice-2-core-reference.md` |
| Slice 3 | web reference | `docs/slice-3-web-reference` | `docs(slice-3): document the HTTP and HTMX surface` | `ops/default/report/slices/product-documentation/slice-3-web-reference.md` |
| Slice 4 | UI reference | `docs/slice-4-ui-reference` | `docs(slice-4): document the UI kit` | `ops/default/report/slices/product-documentation/slice-4-ui-reference.md` |
| Slice 5 | auth reference | `docs/slice-5-auth-reference` | `docs(slice-5): document authentication and crypto` | `ops/default/report/slices/product-documentation/slice-5-auth-reference.md` |
| Slice 6 | data reference | `docs/slice-6-data-reference` | `docs(slice-6): document persistence and validation` | `ops/default/report/slices/product-documentation/slice-6-data-reference.md` |
| Slice 7 | media reference | `docs/slice-7-media-reference` | `docs(slice-7): document image storage and processing` | `ops/default/report/slices/product-documentation/slice-7-media-reference.md` |
| Slice 8 | infrastructure reference | `docs/slice-8-infrastructure-reference` | `docs(slice-8): document infrastructure contracts` | `ops/default/report/slices/product-documentation/slice-8-infrastructure-reference.md` |
| Slice 9 | getting started | `docs/slice-9-getting-started` | `docs(slice-9): add the getting started chapter` | `ops/default/report/slices/product-documentation/slice-9-getting-started.md` |
| Slice 10 | postgres chapter | `docs/slice-10-postgres-chapter` | `docs(slice-10): add the Postgres chapter` | `ops/default/report/slices/product-documentation/slice-10-postgres-chapter.md` |
| Slice 11 | page chapter | `docs/slice-11-page-chapter` | `docs(slice-11): add the page chapter` | `ops/default/report/slices/product-documentation/slice-11-page-chapter.md` |
| Slice 12 | form chapter | `docs/slice-12-form-chapter` | `docs(slice-12): add the form chapter` | `ops/default/report/slices/product-documentation/slice-12-form-chapter.md` |
| Slice 13 | sign-in chapter | `docs/slice-13-sign-in-chapter` | `docs(slice-13): add the sign-in chapter` | `ops/default/report/slices/product-documentation/slice-13-sign-in-chapter.md` |
| Slice 14 | record chapter | `docs/slice-14-record-chapter` | `docs(slice-14): add the record chapter` | `ops/default/report/slices/product-documentation/slice-14-record-chapter.md` |
| Slice 15 | background-work chapter | `docs/slice-15-background-work-chapter` | `docs(slice-15): add the background-work chapter` | `ops/default/report/slices/product-documentation/slice-15-background-work-chapter.md` |
| Slice 16 | settings chapter | `docs/slice-16-settings-chapter` | `docs(slice-16): finish the User Guide` | `ops/default/report/slices/product-documentation/slice-16-settings-chapter.md` |

## Slice 1: Documentation Map

### Purpose

Create the documentation entry, the User Guide entry, the reference index,
and the shared vocabulary. Do not document package contracts or teach a
chapter yet.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Add `docs/index.md`, `docs/tutorials/index.md`, `docs/tutorials/user-guide/index.md`, `docs/reference/index.md`, and `docs/reference/terminology/index.md`. The visible guide name is `User Guide`. State the reference rules, the package-readme boundary, and the vocabulary for component, transactional startup, static config, settings, and Postgres-backed services. List the eight chapter titles without linking to files that do not exist yet. Under Appendices, link Terminology and the reference index. | `docs(guide): add the documentation map` |
| T1.2 | Link the new entry from the root `README.md` Docs section. Leave `docs/features.md`, `docs/gallery.md`, and `docs/CHANGELOG.md` in place. | `docs(readme): link the documentation map` |

### Pages

```text
docs/index.md
docs/tutorials/index.md
docs/tutorials/user-guide/index.md
docs/reference/index.md
docs/reference/terminology/index.md
```

### Validation

```sh
git diff --check
```

## Slice 2: Core Reference

### Purpose

Document application lifecycle and the two configuration layers.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T2.1 | Write the lifecycle, configuration, and logging reference from `app`, `config`, `settings`, and `log`. | `docs(reference): document lifecycle and configuration` |
| T2.2 | Link the three subjects from `docs/reference/index.md`. | `docs(reference): list the core reference subjects` |

### Pages

```text
docs/reference/application-lifecycle/index.md
docs/reference/configuration/index.md
docs/reference/logging/index.md
```

`application-lifecycle` covers `app`, including `Startable`, `Stoppable`, and
transactional startup failure behavior.

`configuration` covers `config` and `settings` as separate layers: static
startup configuration, and runtime schema-validated settings with namespace,
store, and value contracts.

`logging` covers the `log` package.

### Validation

```sh
git diff --check
```

## Slice 3: Web Reference

### Purpose

Document the HTTP request and response surface.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T3.1 | Write the HTTP, HTMX, middleware, and render reference from `web`, `htmx`, `middleware`, and `render`. | `docs(reference): document the HTTP and HTMX surface` |
| T3.2 | Link the four subjects from `docs/reference/index.md`. | `docs(reference): list the web reference subjects` |

### Pages

```text
docs/reference/http/index.md
docs/reference/htmx/index.md
docs/reference/middleware/index.md
docs/reference/rendering/index.md
```

### Validation

```sh
git diff --check
```

## Slice 4: UI Reference

### Purpose

Document the UI kit and the presentation helpers that the kit exposes.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T4.1 | Write the UI, modal, format, pagination, and i18n reference from those packages. | `docs(reference): document the UI kit` |
| T4.2 | Link the five subjects from `docs/reference/index.md`. | `docs(reference): list the UI reference subjects` |

### Pages

```text
docs/reference/ui/index.md
docs/reference/modal/index.md
docs/reference/format/index.md
docs/reference/pagination/index.md
docs/reference/i18n/index.md
```

### Validation

```sh
git diff --check
```

## Slice 5: Auth Reference

### Purpose

Document authentication and cryptographic contracts.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T5.1 | Write the authentication and crypto reference from `auth` and `crypto`. | `docs(reference): document authentication and crypto` |
| T5.2 | Link both subjects from `docs/reference/index.md`. | `docs(reference): list the auth reference subjects` |

### Pages

```text
docs/reference/authentication/index.md
docs/reference/crypto/index.md
```

### Validation

```sh
git diff --check
```

## Slice 6: Data Reference

### Purpose

Document persistence primitives and the data helpers around them.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T6.1 | Write the database, model, validation, seed, and slug reference from those packages. | `docs(reference): document persistence and validation` |
| T6.2 | Link the five subjects from `docs/reference/index.md`. | `docs(reference): list the data reference subjects` |

### Pages

```text
docs/reference/database/index.md
docs/reference/model/index.md
docs/reference/validation/index.md
docs/reference/seed/index.md
docs/reference/slug/index.md
```

### Validation

```sh
git diff --check
```

## Slice 7: Media Reference

### Purpose

Document image storage and processing, including the local, S3, and standard
processor subpackages.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T7.1 | Write the image reference from `image`, `image/local`, `image/s3`, and `image/stdprocessor`. | `docs(reference): document image storage and processing` |
| T7.2 | Link the subject from `docs/reference/index.md`. | `docs(reference): list the media reference subject` |

### Pages

```text
docs/reference/image/index.md
```

### Validation

```sh
git diff --check
```

## Slice 8: Infrastructure Reference

### Purpose

Document the remaining service packages and close the reference index.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T8.1 | Write the mailer, pubsub, scheduler, telemetry, testhelper, and fake reference from those packages. | `docs(reference): document infrastructure contracts` |
| T8.2 | Link those subjects from `docs/reference/index.md` and confirm that every first-party package in this plan has a reference page. | `docs(reference): close the reference index` |

### Pages

```text
docs/reference/mailer/index.md
docs/reference/pubsub/index.md
docs/reference/scheduler/index.md
docs/reference/telemetry/index.md
docs/reference/testhelper/index.md
docs/reference/fake/index.md
```

### Validation

```sh
git diff --check
```

## Slice 9: Getting Started

### Purpose

Teach the first running Hatmax process. The reader loads configuration,
creates a logger and router, and reaches `app.Start`. Postgres, pages, and
accounts wait for later chapters.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T9.1 | Write `docs/tutorials/user-guide/getting-started.md` with one learning goal, the actions, and the result the reader can check. Link to the lifecycle, configuration, and logging reference. | `docs(guide): add the getting started chapter` |
| T9.2 | Link the chapter from `docs/tutorials/user-guide/index.md` as the first chapter. | `docs(guide): list the getting started chapter` |

### Pages

```text
docs/tutorials/user-guide/getting-started.md
```

### Validation

```sh
git diff --check
```

## Slice 10: Add Postgres

### Purpose

Teach adding a Postgres-backed database component. The reader can see the
application stop cleanly when that component fails to start.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T10.1 | Write `docs/tutorials/user-guide/postgres.md`. Link to the database and application-lifecycle reference. | `docs(guide): add the Postgres chapter` |
| T10.2 | Link the chapter from the User Guide index as chapter 2. | `docs(guide): list the Postgres chapter` |

### Pages

```text
docs/tutorials/user-guide/postgres.md
```

### Validation

```sh
git diff --check
```

## Slice 11: Serve a Page

### Purpose

Teach one HTML page and one HTMX update.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T11.1 | Write `docs/tutorials/user-guide/pages.md`. Link to the HTTP, HTMX, and rendering reference. | `docs(guide): add the page chapter` |
| T11.2 | Link the chapter from the User Guide index as chapter 3. | `docs(guide): list the page chapter` |

### Pages

```text
docs/tutorials/user-guide/pages.md
```

### Validation

```sh
git diff --check
```

## Slice 12: Accept a Form

### Purpose

Teach one form that rejects invalid input and accepts valid input.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T12.1 | Write `docs/tutorials/user-guide/forms.md`. Link to the validation, UI, and middleware reference. | `docs(guide): add the form chapter` |
| T12.2 | Link the chapter from the User Guide index as chapter 4. | `docs(guide): list the form chapter` |

### Pages

```text
docs/tutorials/user-guide/forms.md
```

### Validation

```sh
git diff --check
```

## Slice 13: Sign In

### Purpose

Teach a page that only a signed-in request can see.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T13.1 | Write `docs/tutorials/user-guide/sign-in.md`. Link to the authentication reference. Link to the crypto reference instead of walking through every primitive. | `docs(guide): add the sign-in chapter` |
| T13.2 | Link the chapter from the User Guide index as chapter 5. | `docs(guide): list the sign-in chapter` |

### Pages

```text
docs/tutorials/user-guide/sign-in.md
```

### Validation

```sh
git diff --check
```

## Slice 14: Save a Record

### Purpose

Teach storing one record that is still there after a restart.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T14.1 | Write `docs/tutorials/user-guide/records.md`. Link to the model, slug, seed, and database reference. | `docs(guide): add the record chapter` |
| T14.2 | Link the chapter from the User Guide index as chapter 6. | `docs(guide): list the record chapter` |

### Pages

```text
docs/tutorials/user-guide/records.md
```

### Validation

```sh
git diff --check
```

## Slice 15: Work Outside the Request

### Purpose

Teach one background effect caused by a saved change.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T15.1 | Write `docs/tutorials/user-guide/background-work.md`. Link to the pubsub, scheduler, and mailer reference. | `docs(guide): add the background-work chapter` |
| T15.2 | Link the chapter from the User Guide index as chapter 7. | `docs(guide): list the background-work chapter` |

### Pages

```text
docs/tutorials/user-guide/background-work.md
```

### Validation

```sh
git diff --check
```

## Slice 16: Change Settings at Runtime

### Purpose

Teach one runtime settings value that is distinct from static configuration,
add the wiring appendix, and close the User Guide.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T16.1 | Write `docs/tutorials/user-guide/settings.md`. Link to the configuration reference. | `docs(guide): add the settings chapter` |
| T16.2 | Write `docs/tutorials/user-guide/wiring.md`. Map `app.Setup` order and the `Mailer`, `Publisher`, `Subscriber`, and `JobStore` interfaces. Link the lifecycle, database, pubsub, scheduler, and mailer reference instead of copying their contracts. | `docs(guide): add the wiring appendix` |
| T16.3 | Link the settings chapter as chapter 8 and the wiring page under Appendices. Confirm that `docs/index.md` leads with the User Guide, that all eight chapters are linked in order, and that Appendices also links image storage, telemetry, crypto primitives, and `examples/ticked/README.md`. | `docs(guide): close the User Guide` |

### Pages

```text
docs/tutorials/user-guide/settings.md
docs/tutorials/user-guide/wiring.md
```

### Validation

```sh
git diff --check
```

## Completion Gates

- [x] This plan and tracker are approved and committed on `dev`.
- [x] Slice 1 is delivered through its recorded branch, report, pull request, review, and merge.
- [x] Slice 2 is delivered through its recorded branch, report, pull request, review, and merge.
- [x] Slice 3 is delivered through its recorded branch, report, pull request, review, and merge.
- [x] Slice 4 is delivered through its recorded branch, report, pull request, review, and merge.
- [ ] Slices 5 through 16 are each delivered through the recorded branch, report, pull request, review, and merge.
- [ ] The docs-only delivery-set gate passes on the integrated `dev` commit.

## Out of Scope

- How-to guides and explanation bodies.
- Moving `docs/features.md`, `docs/gallery.md`, or `docs/CHANGELOG.md`.
- Rewriting package `readme.md` files.
- Documenting `examples/ticked` as a reference subject or as a User Guide chapter.
- User Guide chapters for image storage, telemetry, `testhelper`, or `fake`.
- An appendix for `testhelper` or `fake`.
- Runtime code, tests, badge updates, and `make ci`.
