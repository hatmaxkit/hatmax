<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Generator Execution and Conformance

Status: Approved
Kind: Subordinate specification
Umbrella: `ops/default/spec/interactive-hatmax-generator.md`
Book: `ops/default/spec/hatmax-book.md`
Intent: `ops/default/spec/intent-and-planning.md`
First archetype: `ops/default/spec/server-rendered-crud.md`

## Purpose

This specification defines project and pre-project target inventory, plan
execution, drift control, idempotency, Hatmax conformance, diagnostics, and
regression evaluation.

Execution is constrained transformation, not an unconstrained second model
request. Conformance evaluates the result independently from the implementer.

## Project Inventory

Inventory is read-only. For an existing project it records at least:

- repository root and current revision when Git is available;
- dirty paths relevant to the requested operation;
- Go module and Go version;
- Hatmax module version or local workspace replacement;
- Book compatibility;
- application entrypoints and composition roots;
- existing feature, asset, migration, template, query, and generated-code
  layouts;
- configured validation and generation commands;
- repository instructions and protected paths;
- declared dependencies relevant to the request.

Inventory uses semantic observations rather than sending the full repository
to a model. File content is loaded only when selected rules or edits need it.

For `create_application`, pre-project inventory instead records:

- the authorized parent directory and proposed target;
- whether the target is absent, empty, or contains preserved entries;
- Git metadata and an unambiguous repository remote when present;
- collisions, existing Go modules, and Hatmax compatibility evidence;
- filesystem capabilities needed for staging and atomic publication;
- applicable parent or repository instructions and protected paths;
- the Hatmax and Book versions selected for the proposed application.

Pre-project inventory does not require `go.mod`, an application entrypoint, or
an installed Hatmax version in the target. Their planned values come from the
selected application archetype and compatible Book.

## Fingerprint

The inventory produces a stable fingerprint over every observation that can
change planning or execution. The fingerprint includes file identity and
content for planned surfaces, relevant configuration, selected Hatmax version,
Book version, and repository rules. For application creation it also includes
the parent and target identities, every preserved target entry, and relevant
filesystem observations.

Unrelated file changes do not invalidate a plan. Any change to a precondition,
selected input, or planned surface does.

## Execution Preconditions

Execution begins only when:

- the intent and plan schemas validate;
- the plan digest matches its content;
- the current fingerprint matches the plan fingerprint;
- every operation has an owning Book obligation and rule;
- every target path is inside the existing project or the admitted new target
  below its authorized parent;
- no undeclared dependency or surface is required;
- the repository does not contain an overlapping uncommitted change;
- required tools and compatible versions are available.

Failure of any precondition produces no edits.

## Edit Operations

The execution engine supports named, schema-valid operations. Initial operation
families are:

- create a file from an archetype-owned structure;
- update a Go declaration or composition list;
- add or modify a SQL migration or query;
- add or modify a named template;
- update an admitted configuration structure;
- run a declared generator or formatter;
- add or update a test owned by the planned behavior.

Each operation declares inputs, target, preconditions, expected postconditions,
and rollback material. A model may supply content only for fields marked as
implementation slots by the Book.

The engine must not execute arbitrary shell text produced by the model.
Repository-owned commands are selected from Book or repository policy.

## Atomicity and Recovery

The engine captures original content for every target before the first edit.
If a structural operation fails, it restores changes made by the current plan
unless restoration would overwrite concurrent external changes. In that case
it stops and reports `execution_conflict` with recovery paths.

External effects such as database execution, network publication, deployment,
account mutation, Git initialization, branch creation, or remote changes are
outside initial generator execution. Creating migration source is allowed;
applying it to a live database is not implied.

For a new application, the engine prepares the complete tree in an isolated
staging location below the authorized parent. It publishes the tree to the
target only after structural preparation succeeds. An admitted target with
preserved entries uses explicit rollback material and receives only effects
listed in the plan. Cancellation before publication leaves the target
unchanged.

## Idempotency

Reapplying an already satisfied operation must either:

- produce no change and report `already_satisfied`; or
- reject a semantic conflict precisely.

It must not duplicate routes, fields, migrations, template blocks, wiring
entries, dependencies, or tests. Textual difference alone is not proof that an
operation is absent.

## Conformance

Conformance evaluates the selected Book rules against the post-edit inventory.
It is separate from compilation, tests, formatting, linting, and generated-file
checks, although all can contribute observations.

Conformance rules can inspect:

- paths and package ownership;
- Go imports, declarations, interfaces, constructors, and calls;
- application wiring and lifecycle order;
- migration, constraint, query, and mapping consistency;
- routes, middleware, forms, HTMX attributes, templates, and partials;
- dependency additions and substitutions;
- tests and fixtures required by an operation;
- documentation boundaries when activated.

A successful repository test suite does not override a conformance violation.

## Diagnostics

Every diagnostic contains:

```yaml
code: HMGEN-...
rule: hatmax....
severity: error
surface: wiring
location: main.go
observed: ...
expected: ...
repairable_within_plan: false
```

Messages are user-safe and do not expose hidden prompts, credentials, or
irrelevant source. Stable codes support tests and tooling.

The implementer may repair a failure only when the failing rule, surface, and
repair operation are already admitted by the plan. Otherwise it returns
`replan_required`.

## Repository Validation

After conformance passes, execution runs the commands selected by repository
policy and the Book. Validation evidence records the exact command, working
directory, exit status, and relevant result.

The generator distinguishes:

- focused checks;
- generated-source checks;
- integration tests;
- repository aggregate gates;
- CI or external validation not run locally.

It must not describe focused validation as a full gate.

## Regression Evaluation

The versioned corpus contains:

- semantically equivalent intent fixtures that must produce the same plan;
- invalid and incompatible intents with expected diagnostic codes;
- ambiguous natural-language prompts with expected clarification fields;
- unsupported-library and non-Hatmax requests that must fail closed;
- project inventories with expected archetype admission or rejection;
- positive and negative generated structures for each conformance rule;
- stale-plan and idempotent-reapplication cases.

Provider-independent tests validate schemas, expansion, execution, and
conformance deterministically. Provider evaluations later run natural-language
prompts through a selected model and compare structured results. Model output
quality is not inferred from deterministic fixture tests.

## Reporting

The final report identifies:

- admitted intent and plan digest;
- Book and Hatmax versions;
- changed surfaces and dependencies;
- conformance results;
- exact repository validation evidence;
- repairs performed within the plan;
- unresolved warnings, exceptions, or external validation.

The report never turns an execution record into project source of truth.

## Acceptance Criteria

- Inventory is read-only, bounded, and sufficient for planning.
- Fingerprints invalidate only relevant drift.
- Every edit is schema-valid and owned by a plan obligation.
- Model output cannot execute arbitrary commands or expand scope.
- Reapplication is idempotent or produces a precise semantic conflict.
- Conformance is independent from compilation and ordinary tests.
- Diagnostics have stable codes and rule attribution.
- Regression fixtures cover success, rejection, drift, and negative structure.
- Final reports distinguish every validation level honestly.
