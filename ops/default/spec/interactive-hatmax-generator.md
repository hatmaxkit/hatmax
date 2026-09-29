# Interactive Hatmax Generator

Status: Approved
Kind: Umbrella specification

## Purpose

The Interactive Hatmax Generator creates and evolves Hatmax-based Go
applications from natural-language intent. It gives users an interactive way
to request features, models, services, validation, wiring, tests, and
documentation without making application architecture an open-ended model
choice.

The generator is not a general coding assistant with Hatmax documentation in
context. It is an opinionated Hatmax product. Users who want a different
architecture, substitute libraries, or a different documentation system must
use tooling outside this generator.

This specification defines the shared product direction, invariants, and
behavioral boundaries. Detailed intent schemas, feature layouts, editing
algorithms, and conformance rules belong to subordinate specifications.

The terms **must**, **must not**, **should**, and **may** describe normative
requirements in this specification.

## Background

Hatmax originally explored deterministic source generation. That approach
could produce precise idiomatic Go, but template combinations froze decisions
early and made generated behavior expensive to evolve. The current Hatmax
toolkit instead provides small composable packages, explicit wiring, and
documented patterns.

Modern language models make a different boundary practical: natural language
can be the flexible input while an internal typed representation and Hatmax
rules retain precise output. The model interprets what the user wants; Hatmax
determines what a correct implementation means.

The historical direction is recorded in
[Project Direction](https://github.com/hatmaxkit/hatmax-legacy/blob/main/docs/project-direction.md).

## Product Thesis

The generator has open intent and closed construction:

- users express goals conversationally;
- the model normalizes those goals into a bounded Hatmax intent;
- Hatmax selects the canonical archetype and expands its obligations;
- implementation uses Hatmax primitives and patterns;
- deterministic checks decide whether the result conforms.

The model may improvise product details that vary legitimately between
applications. It does not improvise application architecture, substitute an
existing Hatmax primitive, or introduce a competing pattern.

Hatmax permits product flexibility, not architectural flexibility.

## Actors

### User

The user describes desired application behavior, answers material ambiguity,
reviews plans when required, and owns the resulting Go project.

### Intent interpreter

The intent interpreter uses a language model to understand conversational
requests, identify missing decisions, and produce a typed Hatmax intent. It
does not directly define Hatmax architecture.

### Hatmax planner

The planner validates the typed intent against the selected Hatmax version,
chooses the canonical archetype, and deterministically expands all required
implementation surfaces and invariants.

### Hatmax implementer

The implementer applies the admitted plan. It may use model generation within
bounded implementation slots, but structural changes remain governed by the
plan and the Hatmax book.

### Conformance checker

The checker verifies source layout, dependency choices, wiring, lifecycle,
security boundaries, persistence obligations, tests, and other applicable
Hatmax rules. Successful compilation alone is not Hatmax conformance.

## Hatmax Book

The Hatmax book is the versioned normative definition of Hatmax-native
application construction. It contains:

- supported capabilities and the intent each capability satisfies;
- canonical feature and component archetypes;
- obligations introduced by each archetype or capability;
- required, recommended, optional, and prohibited patterns;
- approved composition and extension points;
- dependency and substitution rules;
- positive and negative examples;
- conformance rules and diagnostics;
- behavior when a requested capability is unavailable.

The book is not inferred exclusively from source code, prose documentation, or
examples. Those artifacts support the book, but explicit normative rules are
required wherever multiple valid Go designs would otherwise be possible.

The book is versioned with Hatmax. A generator must not silently apply rules
from one Hatmax version to a project using another.

## Control Architecture

The generator constrains model behavior through independent control layers.
No prompt alone is treated as sufficient enforcement.

1. **Source inventory** reads either the current application or an admitted
   pre-project target, including relevant module, version, layout,
   configuration, migration, template, test, and repository observations.
2. **Book context** supplies only the rules, archetypes, examples, and
   capability definitions that apply to the detected project and request.
3. **Structured interpretation** requires schema-valid output from the model.
   Free-form reasoning is not an executable instruction.
4. **Deterministic planning** expands the admitted intent into all obligations
   owned by the selected archetype and capabilities.
5. **Constrained editing** permits changes only within the declared plan and
   approved Hatmax extension points.
6. **Conformance checking** verifies the resulting project independently of
   the model that generated the change.

The generator must fail closed when any layer cannot establish its required
input or result. Model confidence, plausible code, compilation, and passing
unit tests do not override a violated Hatmax rule.

The control inputs have this precedence:

1. repository safety and authorization rules;
2. the Hatmax book for the project's selected Hatmax version;
3. the current inspected project state;
4. admitted user intent;
5. canonical defaults defined by the selected archetype.

User intent can select product behavior and declared extension points. It
cannot authorize the generator to contradict the Hatmax book. A conflicting
request produces an unsupported or exception-required result instead of a
different architecture.

## Canonical Construction

Every supported operation maps to a canonical Hatmax archetype. A feature is
not assembled from arbitrary architectural alternatives selected by the
model. The archetype owns the required structure, component relationships,
wiring, and verification obligations.

Canonical construction applies to at least:

- initial application identity, module layout, process entrypoint, and
  composition root;
- feature boundaries and source layout;
- models and persistence;
- stores and services;
- HTTP handlers, routes, forms, templates, and HTMX behavior;
- configuration and runtime settings;
- application wiring and lifecycle participation;
- pubsub subscribers and scheduled work;
- tests and fixtures.

Application-specific fields, names, rules, page content, workflows, and
integrations may vary within the extension points declared by the selected
archetype.

When Hatmax already provides a capability, the generator must use it. A
third-party library or local implementation cannot replace that capability
merely because it would also work.

## Intent and Planning Lifecycle

The generator follows this lifecycle:

1. Capture the user's natural-language request and the relevant project or
   proposed-target state.
2. Classify the requested operation and candidate Hatmax archetype.
3. Ask only for decisions required to remove material ambiguity.
4. Produce a schema-valid typed Hatmax intent.
5. Validate the intent against the project and Hatmax versions.
6. Deterministically expand the intent into an inspectable execution plan.
7. Apply the plan within its declared scope.
8. Run Hatmax conformance and repository validation.
9. Repair admitted conformance failures or report a precise blocker.
10. Report the resulting behavior, validation, and any approved exception.

The generator must not edit a project from an intent that is syntactically
invalid, semantically inconsistent, incompatible with the project, or still
materially ambiguous.

## Typed Intent and Deterministic Plan

The typed intent records user and product decisions. It is an ephemeral
internal target language, not a user-facing DSL, a checked-in project model,
or the application's source of truth.

The typed intent describes concepts such as:

- the requested operation;
- application identity and target for application creation;
- the selected Hatmax archetype;
- domain names, fields, rules, and requested behavior;
- required Hatmax capabilities;
- explicitly requested documentation;
- explicitly approved exceptions.

The intent must not rely on the model to enumerate every affected file or
technical obligation. The deterministic planner derives those consequences
from the archetype and book. For example, adding a persistent field may imply
a migration, model change, storage mapping, form handling, fixtures, and tests
without requiring the interpreter to remember each surface.

The execution plan is mandatory and inspectable before application. It
identifies the intended effects, required invariants, affected surfaces, and
validation. The generator must emit it even when the request appears simple.
An implementation step must not depend on an obligation absent from the plan.

A minimal plan has this shape:

```yaml
intent: add_feature
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

The interpreter selects intent-level concepts. It must not manually enumerate
technical consequences that the book owns. In the example, the selected
archetype and capabilities deterministically introduce the affected surfaces.
The planner, not model recall, guarantees that the migration, wiring, and
tests are present.

The plan exists for one proposed operation. It becomes invalid when relevant
project state or the selected Hatmax version changes. A tool may retain it as
an execution log, but retained plans have no authority over the current
project and are never required to build or run the application.

The exact approval policy for low-risk and high-risk plans remains a
subordinate product-surface decision. Regardless of that policy, the plan must
be available for inspection before the first edit.

## Constrained Editing

The implementer must:

- change only surfaces admitted by the plan;
- use the canonical Hatmax primitive, dependency, layout, and composition
  rule for each obligation;
- preserve application-specific code outside the admitted change;
- revalidate the relevant project state immediately before editing;
- stop when an unexpected structure would require a new intent or plan;
- prefer structured edits and canonical tools where they exist;
- make repeated application safe or reject a duplicate operation explicitly;
- expose every new dependency, migration, route, configuration field, and
  lifecycle registration in the resulting report.

Model-generated code is permitted only inside slots whose variability is
declared by the archetype. The model may produce domain names, business rules,
copy, and application-specific transformations. It may not invent feature
boundaries, bypass wiring, select substitute infrastructure, or conceal
cross-surface work inside an apparently local edit.

## Conformance and Regression Evaluation

Conformance runs after edits and is independent from ordinary repository
validation. It checks the Hatmax properties applicable to the admitted plan,
including:

- required files, components, and relationships;
- approved imports and prohibited substitutes;
- explicit application wiring and lifecycle ownership;
- model, migration, query, and form consistency;
- handler, HTMX, template, and validation behavior;
- security and configuration obligations;
- required positive, negative, and integration tests;
- documentation boundaries and Diataxis placement when requested.

Diagnostics identify the violated rule, affected surface, observed state, and
expected Hatmax form. The implementer may repair a failure only when the
repair remains inside the admitted plan. Otherwise it must stop and produce a
new plan or blocker.

The generator also requires a versioned evaluation corpus. It must include:

- paraphrased requests that should converge to the same typed intent and
  archetype;
- ambiguous requests that must trigger clarification;
- requests for competing libraries or non-Hatmax patterns that must be
  rejected;
- incomplete project states that must not be edited speculatively;
- representative generated changes checked against expected obligations;
- negative examples that compile but violate Hatmax architecture.

Model or prompt changes cannot be accepted solely through subjective output
review. They must preserve the required intent, plan, and conformance results
for this corpus.

## Precision and Ambiguity

The generator targets different forms of precision through different
mechanisms:

- structured model output provides syntactic precision;
- schema and semantic validation provide valid intents;
- deterministic obligation expansion provides architectural completeness;
- canonical tools and transforms provide repeatable structural changes;
- conformance checks provide enforceable Hatmax correctness.

Natural-language intent can remain ambiguous even when the model is capable.
The generator must expose unresolved choices rather than inventing a product
decision. It may infer a value only when the book defines a single canonical
default or the project already establishes that value unambiguously.

Textually identical generated code is not required. Architecturally
equivalent requests must nevertheless converge to the same Hatmax archetype,
obligations, and invariants.

## Unsupported Intent and Hatmax Gaps

If the user requests a capability that Hatmax does not provide, the generator
must not silently substitute another framework, library, or architectural
pattern.

It must produce an explicit unsupported-capability result and identify the
missing Hatmax primitive. A future subordinate specification will define when
the generator may:

- stop and report the gap;
- use a declared application extension point;
- propose a Hatmax capability addition;
- continue after explicit approval of a bounded exception.

No exception may be inferred from convenience or model preference.

## Documentation Generation

Documentation generation is an explicit, boxed capability. It is not an
automatic consequence of generating or modifying application code.

- A request to add or modify behavior does not authorize documentation edits.
- A request to document behavior activates the documentation capability.
- A combined request may authorize implementation and documentation together.
- The generator does not reorganize user-authored material outside its managed
  documentation surface unless explicitly requested.

Generated documentation uses the canonical Diataxis structure:

- `docs/tutorials/` for learning paths;
- `docs/how-to/` for task-oriented procedures;
- `docs/reference/` for exact contracts and behavior;
- `docs/explanation/` for rationale and tradeoffs.

The generator does not offer an alternative handbook, cookbook, wiki, or free
documentation architecture. It creates only the Diataxis artifacts warranted
by the documentation request and does not mechanically create one document in
every quadrant.

Documentation must be reconciled with the implemented code, linked from the
appropriate README entrypoints, and validated with the project documentation
gate.

Documentation intent is represented explicitly as one of:

- `not_requested`;
- `document_existing_behavior`;
- `document_planned_change`.

The absence of documentation intent is equivalent to `not_requested`. The
model must not infer documentation authorization from the size, novelty, or
user-visible nature of a code change.

## Project Ownership and Source of Truth

Generated projects contain ordinary Go, SQL, templates, configuration, tests,
and documentation. Users own and may edit those artifacts normally. The
generator does not require a runtime DSL or hidden framework state for the
application to function.

The project source remains authoritative. A retained intent or plan, if the
eventual product stores one, is an execution record rather than an independent
definition from which the project must always be regenerated.

The generator must inspect current project state before planning a change. It
must not assume that a previously generated project remains unchanged.

For application creation, the generator starts from the parent directory,
inspects the proposed child target, and uses the same typed-intent, planning,
approval, execution, and conformance pipeline as existing-project changes. It
does not require the user to create or enter the child directory first.

## Observable Failure Behavior

The generator reports a bounded failure instead of applying speculative edits
when:

- the request is materially ambiguous;
- the intent is invalid or incompatible with the project;
- no Hatmax archetype admits the requested behavior;
- a required Hatmax primitive is unavailable;
- project state changed after planning and invalidated the plan;
- an edit cannot be applied without crossing the authorized scope;
- conformance or repository validation cannot pass.

A failure report identifies the rejected intent, applicable rule, missing
decision or capability, and the smallest action that can unblock the request.

## Acceptance Criteria

The umbrella direction is satisfied when:

- users can express supported changes in ordinary language;
- users can create a named canonical Hatmax application from its parent
  directory;
- equivalent requests converge to the same canonical archetype;
- materially ambiguous requests cause focused clarification;
- admitted intents expand all obligations defined by the Hatmax book;
- generated features follow one canonical Hatmax architecture;
- existing Hatmax capabilities are never replaced by competing
  implementations;
- unsupported capabilities are reported instead of silently improvised;
- application assembly remains explicit and uses Hatmax wiring and lifecycle
  patterns;
- generated projects remain ordinary Go projects without a required runtime
  generator or DSL;
- deterministic conformance checks can reject non-Hatmax structure even when
  it compiles;
- every edit is preceded by a visible, schema-valid, ephemeral execution plan;
- the model cannot add an undeclared dependency or affected surface during
  implementation;
- paraphrased equivalent requests pass regression evaluation with the same
  intent, archetype, and obligations;
- implementation requests do not modify documentation unless documentation is
  explicitly requested;
- requested documentation uses only the canonical Diataxis system;
- the generator reports exact validation evidence and unresolved boundaries.

## Deferred Specifications

This umbrella requires subordinate specifications for:

- canonical application scaffolding;
- the Hatmax book format and rule taxonomy;
- the typed intent schema and semantic validation;
- deterministic obligation expansion and plan representation;
- canonical feature architecture;
- models and Postgres persistence;
- services, dependencies, wiring, and lifecycle;
- HTTP, HTMX, forms, validation, and UI;
- events, subscribers, and scheduled work;
- testing and fixtures;
- conformance checking and diagnostics;
- boxed Diataxis documentation generation;
- existing-project adoption and migration;
- Hatmax capability-gap and exception handling;
- the interactive product surface and interpreter-backend boundary.

The initial implementation sequence resolved the core feature generator in
this order:

1. Hatmax book, rule taxonomy, version compatibility, and dependency policy.
2. Typed intent schema, semantic validation, and mandatory plan format.
3. One canonical `server_rendered_crud` vertical slice covering model,
   Postgres persistence, store, service, handler, HTMX form, runtime
   validation, templates, wiring, and tests.
4. Conformance checks and the regression evaluation corpus for that slice.
5. Interactive product surface and interpreter-backend boundary.
6. Boxed Diataxis documentation generation.

Application scaffolding and the conversational product surface extend that
kernel through their own approved delivery set. Existing-project adoption and
migration remain distinct from creating a new application in an absent, empty,
or explicitly admitted non-conflicting target.

The exact Go source layout, approval policy, existing-project admission
threshold, extension points, editing primitives, and initial product surface
remain subordinate decisions. They do not change the umbrella invariants and
must be resolved before their corresponding implementation begins.

No subordinate specification, plan, tracker, or runtime implementation is
authorized by this umbrella specification alone.
