<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Interactive Generator Execution and Conformance Delivery Plan

Status: Approved
Delivery set: interactive-generator-execution-conformance
Slice strategy: layered
Reason: safe project mutation requires executable contracts, atomic file
application, canonical generation, incremental mutation, and independent
conformance in dependency order. Each layer remains testable without a model
provider or interactive product surface.
Umbrella spec: `ops/default/spec/interactive-hatmax-generator.md`
Specs:

- `ops/default/spec/hatmax-book.md`
- `ops/default/spec/intent-and-planning.md`
- `ops/default/spec/server-rendered-crud.md`
- `ops/default/spec/execution-and-conformance.md`

Tracker: `ops/default/tracker/interactive-generator-execution-conformance.md`
Base branch: `dev`
Planning base: `9b7e0d9e71da8297665e42695459e5eb33c739e0`
Active slice: Slice 1
Execution gate: satisfied by this approved planning commit on `dev`

## Objective

Execute an approved generator plan against a compatible Hatmax project and
produce canonical, validated, ordinary Go source without allowing the
implementer to expand scope or select non-Hatmax alternatives.

The delivery begins with deterministic execution. It does not integrate a
production model or expose a public interactive command. Its public outcome is
a reusable Go API that can prepare, apply, verify, and report one approved
plan.

## Scope

The delivery includes:

- complete domain input in sealed plans;
- typed executable edits derived from Book-owned obligations;
- path, scope, drift, dirty-overlap, and tool preconditions;
- atomic file application, rollback, and idempotent reapplication;
- canonical `server_rendered_crud` generation for `create_feature`;
- cross-surface `add_field` and `add_validation` mutation;
- deterministic Hatmax conformance checks and stable diagnostics;
- bounded repository commands and exact validation evidence;
- provider-independent end-to-end regression fixtures.

It excludes:

- a production model-provider adapter;
- a public CLI, service, or agent skill;
- generated documentation;
- live database migration, deployment, publication, or network mutation;
- archetypes other than `server_rendered_crud`;
- capabilities outside the current Book.

## Architecture Boundaries

Package ownership is:

```text
generator/book/          Canonical obligations and construction rules
generator/project/       Read-only inventory, content selection, and fingerprint
generator/intent/        Typed application decisions
generator/plan/          Sealed semantic plan and declared effects
generator/execute/       Manifest preparation, structured edits, atomic apply,
                         repository validation, and execution report
generator/conformance/   Independent post-edit Hatmax rule evaluation
generator/eval/          Provider-independent regression corpus
```

`execute` may consume the other generator packages but does not reinterpret
natural language or invent obligations. `conformance` evaluates selected Book
rules independently from the code that rendered or applied edits. Neither
package runs arbitrary model-produced shell text.

## Ordered Slices

| Slice | Short Name | Branch | Pull Request Title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | Execution manifest | `feat/generator-execution-manifest` | `feat(slice-1): define generator execution manifests` | `ops/default/report/slices/interactive-generator-execution-conformance/slice-1-execution-manifest.md` |
| Slice 2 | Atomic application | `feat/generator-atomic-application` | `feat(slice-2): apply generator edits atomically` | `ops/default/report/slices/interactive-generator-execution-conformance/slice-2-atomic-application.md` |
| Slice 3 | CRUD generation | `feat/generator-crud-generation` | `feat(slice-3): generate canonical Hatmax CRUD features` | `ops/default/report/slices/interactive-generator-execution-conformance/slice-3-crud-generation.md` |
| Slice 4 | Incremental mutations | `feat/generator-incremental-mutations` | `feat(slice-4): mutate canonical Hatmax features` | `ops/default/report/slices/interactive-generator-execution-conformance/slice-4-incremental-mutations.md` |
| Slice 5 | Execution conformance | `test/generator-execution-conformance` | `test(slice-5): validate generator execution conformance` | `ops/default/report/slices/interactive-generator-execution-conformance/slice-5-execution-conformance.md` |

## Slice 1: Execution Manifest

### Purpose

Turn a sealed semantic plan into a complete, inspectable set of typed edits
without modifying the project.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Preserve normalized domain input in the sealed plan and include it in validation, serialization, and digest identity. | `feat(generator): preserve executable plan inputs` |
| T1.2 | Add typed execution manifests, edit kinds, targets, preconditions, postconditions, implementation slots, and stable diagnostics under `generator/execute/`. | `feat(generator): define execution manifests` |
| T1.3 | Prepare deterministic manifests for the three admitted operations, resolve canonical project-relative targets, and reject undeclared, protected, escaping, generated, or overlapping paths before mutation. | `feat(generator): prepare canonical execution targets` |

### Validation

```sh
go test ./generator/plan/... ./generator/execute/...
go test ./generator/...
make vet
make lint-strict
git diff --check
```

## Slice 2: Atomic Application

### Purpose

Apply a prepared manifest as one recoverable filesystem transaction with no
partial success.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T2.1 | Add a bounded workspace that snapshots admitted targets and stages structured create, replace, and insertion edits before commit. | `feat(generator): stage bounded project edits` |
| T2.2 | Add drift revalidation, atomic commit, rollback, concurrent-change detection, and stable execution diagnostics. | `feat(generator): apply project edits atomically` |
| T2.3 | Add idempotent reapplication, semantic conflict detection, and tests proving failed execution leaves fixture projects unchanged. | `test(generator): cover atomic execution behavior` |

### Validation

```sh
go test ./generator/execute/...
go test -race ./generator/execute/...
go test ./generator/...
make vet
make lint-strict
git diff --check
```

## Slice 3: CRUD Generation

### Purpose

Generate and wire one complete canonical `server_rendered_crud` feature from a
`create_feature` plan.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T3.1 | Add Book-owned render recipes and deterministic rendering for the domain model, store contract, Postgres adapter, service, and required behavioral tests. | `feat(generator): render canonical CRUD packages` |
| T3.2 | Render reversible migration, SQLC queries, handler, full-page and HTMX templates, form validation, and related tests using current Hatmax primitives. | `feat(generator): render CRUD transport surfaces` |
| T3.3 | Insert explicit imports, constructors, lifecycle components, and route registration into the composition root; run only repository-declared generation and formatting commands. | `feat(generator): wire generated CRUD features` |

### Validation

```sh
go test ./generator/execute/...
go test ./generator/...
make vet
make lint-strict
git diff --check
```

## Slice 4: Incremental Mutations

### Purpose

Apply `add_field` and `add_validation` across every representation owned by the
canonical feature without duplicating existing structure.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T4.1 | Implement semantic discovery of generated feature declarations, schema, queries, forms, templates, wiring, fixtures, and tests. | `feat(generator): inspect canonical feature structure` |
| T4.2 | Apply `add_field` to every required domain, persistence, transport, template, fixture, and test surface. | `feat(generator): add fields across CRUD surfaces` |
| T4.3 | Apply durable or client-only `add_validation` at the admitted owners and reject conflicts or missing canonical structure. | `feat(generator): add layered feature validation` |

### Validation

```sh
go test ./generator/execute/...
go test -race ./generator/execute/...
go test ./generator/...
make vet
make lint-strict
git diff --check
```

## Slice 5: Execution Conformance

### Purpose

Independently prove that generated changes satisfy selected Hatmax rules and
produce honest execution evidence.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T5.1 | Evaluate required files, package ownership, imports, boundaries, wiring order, persistence consistency, HTMX behavior, validation ownership, and documentation scope with stable rule-attributed diagnostics. | `feat(generator): check Hatmax execution conformance` |
| T5.2 | Run bounded repository commands after conformance and produce an execution report containing plan identity, changes, conformance results, and exact validation evidence. | `feat(generator): report validated execution` |
| T5.3 | Add versioned end-to-end fixtures for create, reapply, add-field, add-validation, drift, rollback, semantic conflict, and compiling non-conformant output. | `test(generator): exercise execution conformance` |

### Validation

```sh
go test ./generator/...
go test -race ./generator/execute/... ./generator/conformance/...
make check
git diff --check
```

`make check` must run through a process with Docker group membership because
existing Hatmax integration tests use Testcontainers.

## Delivery-Set Gate

After all five slices are merged into `dev`:

1. capture the exact `origin/dev` commit;
2. run `make check` with Docker access;
3. run the execution corpus against the same commit;
4. verify rejected and failed executions leave no project changes;
5. verify reapplication is unchanged or returns a precise semantic conflict;
6. close the tracker only when the exact candidate is green.

No model-provider integration, `main` alignment, tag, or release is implied by
delivery-set closure.

## Completion Criteria

- Sealed plans contain every application decision needed for deterministic
  execution.
- Every target and command is derived from the Book, repository policy, and
  admitted plan.
- A valid `create_feature` plan produces one complete canonical Hatmax CRUD
  feature with explicit wiring and meaningful tests.
- `add_field` and `add_validation` update every required representation.
- Drift, dirty overlap, protected paths, undeclared effects, and missing tools
  fail before mutation.
- Execution is atomic and reapplication is idempotent or rejected precisely.
- Conformance can reject compiling code that violates Hatmax architecture.
- Reports distinguish conformance, focused checks, integration tests, and the
  aggregate repository gate.
- The full repository gate passes for the integrated `dev` candidate.
