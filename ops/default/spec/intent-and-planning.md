<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Generator Intent and Planning

Status: Approved
Kind: Subordinate specification
Umbrella: `ops/default/spec/interactive-hatmax-generator.md`
Book: `ops/default/spec/hatmax-book.md`

## Purpose

This specification defines how natural-language requests become schema-valid
Hatmax intents and deterministic, inspectable execution plans.

The model interprets user goals. The planner owns architectural consequences.
Neither free-form model text nor hidden reasoning is executable input.

## Lifecycle

1. Inventory the existing project or proposed application target and compute
   its fingerprint.
2. Select the compatible Hatmax Book.
3. Provide the interpreter with the request, relevant project observations,
   and selected Book context.
4. Require a schema-valid intent or a structured clarification result.
5. Apply semantic validation against the inventory and Book.
6. Deterministically expand the admitted intent into a plan.
7. Expose the plan before any edit.
8. Recheck the project fingerprint immediately before execution.
9. Invalidate the plan after execution, rejection, or relevant project drift.

## Intent Envelope

The intent is ephemeral and contains at least:

```yaml
schema_version: 2
operation: create_feature
project_fingerprint: sha256:...
hatmax_version: 0.4.0
book_version: 1
archetype: server_rendered_crud
feature: property
domain: {}
capabilities:
  - postgres_persistence
  - htmx_form
  - runtime_validation
documentation: not_requested
exceptions: []
```

Structured schemas define allowed fields, enums, names, and values. Unknown
fields are rejected unless the schema marks a specific extension map.

Intent schema version 2 supports:

- `create_feature`;
- `add_field`;
- `add_validation`.

Intent schema version 3 adds `create_application`, `source_fingerprint`, the
application identity object, and the proposed target object. For
existing-project operations, `source_fingerprint` carries the same project
fingerprint represented by `project_fingerprint` in schema version 2.

`create_application` uses an application identity object, proposed target,
selected compatible Hatmax version, and target fingerprint. Existing-project
operations use the current project identity and project fingerprint. Both are
represented by `source_fingerprint`; an intent never fabricates an existing
project merely to satisfy the schema.

Additional operations require Book support and a schema revision. Free-form
operation names do not degrade to generic coding requests.

## Domain Decisions

The `domain` object contains application-specific choices admitted by the
archetype, such as entity names, fields, routes, labels, ownership, and business
rules. It must not contain file paths, imports, wiring order, storage adapters,
or other architectural decisions owned by the Book.

The interpreter identifies semantic concepts without owning implementation
syntax. Before semantic validation, Hatmax deterministically normalizes
unambiguous ASCII names: features and fields use lower snake case, entities use
exported Go identifier form, and routes use absolute lower-kebab paths. For a
new feature, an omitted entity derives from the feature; an omitted route and
display label derive from the entity's deterministic plural. Explicit product
labels remain unchanged. Names containing materially ambiguous punctuation are
rejected instead of guessed.

The interpreter must ask one focused question when multiple materially
different product meanings remain.

## Documentation Intent

`documentation` is required after normalization and has one of these values:

- `not_requested`;
- `document_existing_behavior`;
- `document_planned_change`.

An omitted value normalizes to `not_requested`. Implementation size or user
visibility never implies documentation intent.

## Exceptions

Exceptions are empty by default. Each exception records the exact Book rule,
reason, bounded scope, and explicit user approval. An exception cannot admit
an unknown architecture, disable safety rules, or silently become a new
default.

The first delivery set parses and validates exceptions but does not execute
them. It reports `exception_execution_unsupported`.

## Semantic Validation

Semantic validation rejects an intent when:

- the applicable project or target fingerprint is absent;
- no compatible Hatmax version can be selected from the Book;
- the Book does not support the operation or archetype;
- a capability is unknown, incompatible, or missing a prerequisite;
- a domain value violates naming, type, or ownership rules;
- the requested operation conflicts with observed project state;
- documentation intent is invalid;
- an exception lacks explicit approval or cannot be bounded;
- material ambiguity remains.

Validation returns stable diagnostic codes and field paths. It does not return
only prose.

## Mandatory Plan

Every admitted intent expands to one plan. The minimal user-visible projection
is:

```yaml
intent: create_feature
archetype: server_rendered_crud
feature: property
capabilities:
  - postgres_persistence
  - htmx_form
  - runtime_validation
affected_surfaces:
  - migration
  - model
  - store
  - service
  - handler
  - templates
  - wiring
  - tests
documentation: not_requested
```

The complete internal plan also records:

- schema, Hatmax, and Book versions;
- source project or target fingerprint;
- selected rule IDs;
- ordered operations and their owning obligations;
- preconditions and expected observations;
- allowed file and dependency effects;
- validation and conformance checks;
- explicit exceptions;
- a deterministic plan digest.

The planner derives affected surfaces, operations, and checks from Book
obligations. The model cannot add or remove them directly.

## Determinism

Equivalent validated intents, project inventories, and Book versions must
produce the same semantic plan and digest. Serialization order is stable.

Application-specific source text need not be identical. Plan identity depends
on semantic operations, selected rules, declared inputs, and expected effects,
not whitespace or model wording.

## Inspection and Approval

The plan is always available before editing. A product surface may automatically
continue for operations whose approval policy permits it, but it must not hide
or skip plan production.

High-impact operation policy belongs to the interactive-surface specification.
The planning kernel exposes risk and effect data without deciding how a user
interface obtains approval.

## Drift and Expiry

A plan is valid only for its source fingerprint and selected Hatmax and Book
versions. Any relevant change causes `plan_stale` before editing.

Plans are single-use. A retained plan is an execution record only. It is not
read as current project state and is never required to build or run generated
applications.

## Failure Results

Planning returns one of:

- an admitted plan;
- `clarification_required` with focused questions and affected fields;
- `intent_invalid` with schema diagnostics;
- `intent_incompatible` with semantic diagnostics;
- `capability_unsupported` with the missing Hatmax capability;
- `plan_stale` with changed observations;
- `exception_required` with the conflicting rule.

No failure result authorizes partial edits.

## Acceptance Criteria

- Free-form model output cannot become an executable plan.
- The intent contains product decisions but not Book-owned architecture.
- Semantic validation is deterministic and diagnostic.
- Every admitted intent expands to a visible plan.
- Equivalent inputs produce the same semantic plan and digest.
- Project drift invalidates a plan before edits.
- Documentation remains `not_requested` unless explicitly activated.
- Intent and plan artifacts remain ephemeral and non-authoritative.
