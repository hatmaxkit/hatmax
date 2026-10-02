<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Hatmax Book

Status: Approved
Kind: Subordinate specification
Umbrella: `ops/default/spec/interactive-hatmax-generator.md`

## Purpose

The Hatmax Book is the versioned, machine-validatable handbook that defines
Hatmax-native application construction. The generator uses it to select one
canonical archetype, expand obligations, restrict implementation choices, and
evaluate conformance.

The Book is not a prompt, a documentation search index, or a collection of
examples. It is an executable product contract. Prose can explain a rule, but
structured fields determine whether the rule applies and what it requires.

## Source Location

The normative Book source lives under `generator/book/` with this initial
layout:

```text
generator/book/
├── manifest.yaml
├── capabilities/
├── archetypes/
├── rules/
└── examples/
    ├── positive/
    └── negative/
```

The Go package in this directory owns loading, validation, selection, and
embedding. Generated human reference may live under
`docs/reference/generator/`, but it must derive from the Book. The generated
reference is not a second authority.

Operational specifications under `ops/default/spec/` govern Book design and
evolution. They are not loaded as runtime rules.

## Manifest

The manifest identifies the complete Book release:

```yaml
schema_version: 1
book_version: 1
hatmax:
  minimum: 0.4.0
  maximum_exclusive: 0.6.0
capabilities: []
archetypes: []
rules: []
```

Every referenced entry must exist exactly once. Unreferenced entries, duplicate
IDs, incompatible versions, and unknown schema versions are invalid.

The Book shipped by a tagged Hatmax release must validate against that exact
release. A newer generator may carry multiple compatible Book releases, but it
must select one explicitly. It must not use the latest rules implicitly for an
older project.

## Rule Taxonomy

Every rule has:

- a stable, namespaced `id`;
- a normative level: `required`, `recommended`, `optional`, or `prohibited`;
- a concise title and rationale;
- structured applicability conditions;
- required or prohibited observations;
- one or more diagnostic codes;
- references to capabilities or archetypes that own it;
- positive and negative example references when examples add evidence.

Rule IDs survive wording and implementation changes. A semantic change that
would make an old diagnostic misleading requires a new rule ID.

`required` and `prohibited` rules affect conformance. `recommended` rules
produce warnings unless an archetype promotes them. `optional` rules describe
declared extension points and never authorize arbitrary alternatives.

## Capabilities

A capability describes one supported technical behavior, such as
`postgres_persistence`, `htmx_form`, or `runtime_validation`. It defines:

- the user intent it satisfies;
- prerequisites and incompatible capabilities;
- obligations added to an archetype;
- affected logical surfaces;
- approved Hatmax packages and external dependencies;
- validation and conformance rules;
- unsupported variants and extension points.

Capabilities compose only when the Book declares the composition valid. The
planner must reject a capability set whose dependencies or exclusions cannot
be satisfied.

## Archetypes

An archetype defines one canonical assembly pattern. It owns:

- supported operations;
- required and optional capabilities;
- canonical logical surfaces and their relationships;
- obligation-expansion rules;
- implementation slots where application-specific generation is permitted;
- structural and behavioral conformance rules;
- operation-specific effects and validation.

The initial archetypes are:

- `server_rendered_hatmax_application`, governed by
  `ops/default/spec/canonical-application-scaffold.md`;
- `server_rendered_crud`, governed by
  `ops/default/spec/server-rendered-crud.md`.

The application archetype owns the root module, `internal/application`
composition package, process lifecycle, neutral web surface, root metadata,
and the transition that activates migration wiring with the first real
migration. Feature archetypes extend that composition package; they never turn
`main.go` into an application assembly surface.

Book release 1 remains the delivered `server_rendered_crud` contract. The
application archetype enters Book release 2 together with the compatible
composition rules for CRUD and intent schema version 3. A generator must not
advertise `create_application` while it has selected Book release 1.

An archetype must not offer equivalent architecture variants. A materially
different architecture requires a different named archetype and explicit
product approval.

## Dependency Policy

Each capability and archetype declares its dependency allowlist. Dependencies
fall into three categories:

- Hatmax packages that must be used when they provide the required behavior;
- admitted standard-library packages;
- explicitly admitted tools or third-party packages with a stated purpose.

The generator must not add an undeclared dependency. A transitive dependency
does not become admitted merely because it is already present in `go.sum`.

When Hatmax owns a primitive, a third-party package or local replacement is
prohibited unless a Book rule defines a bounded compatibility adapter.

## Selection and Context

Book selection uses the project inventory, selected Hatmax version, admitted
intent, archetype, and capabilities. The context compiler supplies the model
only the selected entries plus their direct prerequisites and examples.

The full Book may be used by deterministic validation, but the model should
not receive unrelated rules. Smaller relevant context reduces contradictory
instructions and makes rule attribution inspectable.

Every plan records the selected Book identity and applicable rule IDs. Every
diagnostic refers to at least one rule ID.

## Examples

Positive examples demonstrate one compliant property. Negative examples
demonstrate one specific violation, including code that compiles but is not
Hatmax-conformant.

Examples are evidence for rules, not implicit rules themselves. An example
cannot introduce a dependency, layout, or exception absent from its owning
structured entry.

The current `examples/guide` and `examples/ticked` applications are survey
inputs. They are not automatically canonical. Where they differ, the Book
must make one explicit choice before generation is allowed.

## Validation

Book validation must reject:

- invalid YAML or unsupported schemas;
- missing, duplicate, or cyclic references;
- incompatible Hatmax version ranges;
- archetypes without supported operations;
- obligations without owning rules;
- rule diagnostics without codes;
- dependencies outside the declared policy;
- examples that reference unknown rules;
- affected surfaces that the planner cannot represent.

Validation runs in unit tests, repository checks, and before the generator
uses an embedded Book.

## Failure Behavior

If no compatible Book exists, the generator returns
`book_version_unsupported`. If selected entries are invalid, it returns
`book_invalid`. If an intent requires an unknown capability or archetype, it
returns `capability_unsupported` or `archetype_unsupported`.

These failures occur before plan admission or project mutation.

## Acceptance Criteria

- The Book has one structured source of truth under `generator/book/`.
- Every Book release declares compatible Hatmax versions.
- Rules, capabilities, archetypes, and diagnostics have stable IDs.
- Dependency admission and substitution rules are explicit.
- A planner can select a minimal complete rule set deterministically.
- Invalid or incompatible Book content fails before project edits.
- Human reference can be generated without duplicating normative content.
