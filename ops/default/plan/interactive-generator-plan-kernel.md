<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Interactive Generator Plan Kernel Delivery Plan

Status: Approved
Delivery set: interactive-generator-plan-kernel
Slice strategy: layered
Reason: the first delivery establishes versioned contracts, project
observation, typed input, deterministic expansion, and evaluation in dependency
order. Each layer has an independently testable API, while no layer edits user
projects.
Umbrella spec: `ops/default/spec/interactive-hatmax-generator.md`
Specs:

- `ops/default/spec/hatmax-book.md`
- `ops/default/spec/intent-and-planning.md`
- `ops/default/spec/server-rendered-crud.md`
- `ops/default/spec/execution-and-conformance.md`

Tracker: `ops/default/tracker/interactive-generator-plan-kernel.md`
Base branch: `dev`
Planning base: `28e2b943d2267d903561eb2c5d6f7ad13c31a51a`
Active slice: Slice 1
Execution gate: satisfied by this approved planning commit on `dev`

## Objective

Deliver a provider-independent planning kernel that reads a Hatmax project,
validates a typed intent, and emits a deterministic, inspectable plan without
modifying the project.

The delivery proves the closed-construction half of the generator before model
integration or source mutation. Its public outcome is a reusable Go API and
evaluation harness, not a production CLI.

## Scope

The delivery includes:

- the initial structured Hatmax Book and loader;
- read-only project inventory and relevant-state fingerprinting;
- typed intents and semantic validation;
- deterministic obligation expansion and plan serialization;
- regression fixtures and a provider-neutral evaluation boundary.

It excludes:

- source edits, patch application, and automatic repair;
- a production model-provider adapter;
- a public CLI, service, or agent skill;
- generated application documentation;
- operations outside `create_feature`, `add_field`, and `add_validation`;
- archetypes other than `server_rendered_crud`.

These exclusions prevent the kernel delivery from silently becoming the
mutation or interactive-product delivery sets.

## Architecture Boundaries

Initial package ownership is:

```text
generator/book/       Book data, loading, validation, and selection
generator/project/    Read-only inventory and fingerprint
generator/intent/     Typed intents and semantic validation
generator/plan/       Deterministic expansion, digest, and serialization
generator/eval/       Fixtures and provider-neutral evaluation contracts
```

Packages depend inward in that order. `book` does not inspect projects;
`project` does not interpret natural language; `intent` does not enumerate
technical obligations; `plan` does not edit files; `eval` owns no production
behavior.

## Ordered Slices

| Slice | Short Name | Branch | Pull Request Title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | Book core | `feat/generator-book-core` | `feat(slice-1): add the Hatmax Book core` | `ops/default/report/slices/interactive-generator-plan-kernel/slice-1-book-core.md` |
| Slice 2 | Project inventory | `feat/generator-project-inventory` | `feat(slice-2): inventory Hatmax projects` | `ops/default/report/slices/interactive-generator-plan-kernel/slice-2-project-inventory.md` |
| Slice 3 | Intent validation | `feat/generator-intent-validation` | `feat(slice-3): validate generator intents` | `ops/default/report/slices/interactive-generator-plan-kernel/slice-3-intent-validation.md` |
| Slice 4 | Deterministic planning | `feat/generator-deterministic-planning` | `feat(slice-4): expand deterministic generator plans` | `ops/default/report/slices/interactive-generator-plan-kernel/slice-4-deterministic-planning.md` |
| Slice 5 | Planning evaluation | `test/generator-planning-evaluation` | `test(slice-5): validate generator planning` | `ops/default/report/slices/interactive-generator-plan-kernel/slice-5-planning-evaluation.md` |

## Slice 1: Book Core

### Purpose

Create the validated, embedded Book foundation and the minimal entries needed
to plan the first archetype.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Add typed Book models, YAML loading, manifest resolution, stable IDs, version compatibility, and structural validation under `generator/book/`. | `feat(generator): add the Hatmax Book loader` |
| T1.2 | Add the initial `server_rendered_crud` archetype, the `postgres_persistence`, `htmx_form`, and `runtime_validation` capabilities, dependency policy, rule diagnostics, and positive and negative fixtures. | `feat(generator): define the initial Hatmax Book` |
| T1.3 | Test invalid references, duplicate IDs, incompatible versions, dependency violations, selection, and deterministic entry ordering. | `test(generator): validate Hatmax Book contracts` |

### Validation

```sh
go test ./generator/book/...
make vet
make lint-strict
git diff --check
```

## Slice 2: Project Inventory

### Purpose

Produce a bounded semantic inventory and relevant-state fingerprint without
modifying or fully ingesting the project.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T2.1 | Add inventory types and read-only discovery for Go module data, Hatmax version, entrypoints, layouts, repository rules, dependencies, commands, and dirty overlap. | `feat(generator): inventory Hatmax projects` |
| T2.2 | Add relevant-state fingerprinting and change classification for selected inputs and planned surfaces. | `feat(generator): fingerprint planning inputs` |
| T2.3 | Add fixture projects for supported, incompatible, incomplete, dirty, and local-workspace states. | `test(generator): cover project inventory states` |

### Validation

```sh
go test ./generator/project/...
go test ./generator/book/... ./generator/project/...
make vet
make lint-strict
git diff --check
```

## Slice 3: Intent Validation

### Purpose

Represent and semantically validate the first typed operations without calling
a model or enumerating implementation obligations.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T3.1 | Add strict intent schemas for `create_feature`, `add_field`, and `add_validation`, including documentation and exception fields. | `feat(generator): define typed generator intents` |
| T3.2 | Validate intents against project inventory and selected Book entries with stable diagnostic codes and focused clarification fields. | `feat(generator): validate intent semantics` |
| T3.3 | Add valid, invalid, ambiguous, unsupported, and incompatible intent fixtures. | `test(generator): cover generator intent validation` |

### Validation

```sh
go test ./generator/intent/...
go test ./generator/book/... ./generator/project/... ./generator/intent/...
make vet
make lint-strict
git diff --check
```

## Slice 4: Deterministic Planning

### Purpose

Expand admitted intents through Book obligations into stable, visible plans.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T4.1 | Add plan types, rule attribution, ordered logical operations, validation obligations, and stable serialization. | `feat(generator): define execution plans` |
| T4.2 | Add deterministic expansion for the three initial operations and first archetype capabilities. | `feat(generator): expand Hatmax obligations` |
| T4.3 | Add plan digests, fingerprint checks, single-use lifecycle state, and stale-plan diagnostics without an executor. | `feat(generator): bind plans to project state` |

### Validation

```sh
go test ./generator/plan/...
go test ./generator/book/... ./generator/project/... ./generator/intent/... ./generator/plan/...
make vet
make lint-strict
git diff --check
```

## Slice 5: Planning Evaluation

### Purpose

Prove deterministic convergence and failure behavior and define the boundary
for later model-provider evaluation.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T5.1 | Add a versioned corpus for equivalent typed intents, ambiguity, unsupported requests, incompatible projects, and stale plans. | `test(generator): add the planning corpus` |
| T5.2 | Add a provider-neutral interpreter evaluation interface and natural-language cases with expected structured results; run deterministic tests through a fixture interpreter only. | `feat(generator): define interpreter evaluation` |
| T5.3 | Add end-to-end tests from inventory and typed intent through plan digest and diagnostics, and document that no production model result is validated yet. | `test(generator): exercise the plan kernel` |

### Validation

```sh
go test ./generator/...
make check
git diff --check
```

`make check` must run on a Docker-capable host or CI because existing Hatmax
integration tests use Testcontainers.

## Delivery-Set Gate

After all five slices are merged into `dev`:

1. capture the exact `origin/dev` commit;
2. run `make check` on a Docker-capable host or CI;
3. run the full generator corpus against the same commit;
4. verify the kernel performs no project writes;
5. close the tracker only when the exact candidate is green.

No `main` alignment or release is implied by delivery-set closure.

## Completion Criteria

- A compatible Book is selected and validated deterministically.
- A project inventory and relevant fingerprint can be reproduced.
- The three initial typed operations admit or reject with stable diagnostics.
- Equivalent admitted inputs produce the same plan and digest.
- Plans enumerate rules, surfaces, operations, and validation without editing.
- Evaluation fixtures distinguish deterministic kernel coverage from future
  live-model coverage.
- The full repository gate passes for the integrated `dev` candidate.
