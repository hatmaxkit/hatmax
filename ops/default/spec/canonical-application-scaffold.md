# Canonical Hatmax Application Scaffold

Status: Approved
Kind: Subordinate generator specification
Umbrella: `ops/default/spec/interactive-hatmax-generator.md`
Book: `ops/default/spec/hatmax-book.md`
Intent: `ops/default/spec/intent-and-planning.md`
Execution: `ops/default/spec/execution-and-conformance.md`
Discovery: `ops/default/quiz/00001-hatmax-conversational-application-generation/artifact.md`

Delivery boundary: this contract becomes operational with intent schema
version 3 and Book release 2. Book release 1 and intent schema version 2 remain
the delivered runtime authority until the corresponding implementation slices
land.

## Purpose

This specification defines how Hatmax creates the first runnable version of a
new Hatmax application. It adds `create_application` as a first-class operation
instead of requiring a compatible application to exist before Hatmax can act.

The scaffold is opinionated. Natural language supplies product identity and
optional initial scope. The Hatmax Book supplies architecture, dependencies,
layout, lifecycle, validation, and conformance.

## Product Outcome

A successful operation creates an ordinary Go application below the directory
where the Hatmax session was started. The application compiles, uses the
compatible Hatmax release selected by the Book, and provides a neutral runnable
web surface without inventing a business domain.

The user may request a business niche, description, or initial features. These
values enrich the result but do not become mandatory fields for a general
scaffold.

## Adaptive Input Collection

Hatmax accepts requests ranging from a terse application name to a detailed
near-specification. It does not implement a fixed wizard or fixed question
sequence.

The interpreter accumulates supplied decisions. Deterministic intent validation
classifies each missing value as:

- required before a valid plan can exist;
- inferable from unambiguous session or repository context;
- optional enrichment that must not delay planning.

Hatmax asks only for required values that cannot be inferred. Once the intent
is plannable, it presents the plan instead of continuing to solicit optional
detail. Later requests can grow the application.

## Application Identity

The application name is required. Hatmax derives distinct technical identities
rather than applying one spelling everywhere.

For an application named `Real Estate`, the normalized identities include:

```yaml
display_name: Real Estate
project_slug: real-estate
root_package: main
```

The project slug uses lower kebab case and names the default child directory.
Go package names use lowercase semantic words without hyphens or underscores.
Application-owned packages are named for their responsibility or domain, not
mechanically after the application display name.

The Go module path uses this precedence:

1. derive it from an existing target-repository remote when unambiguous;
2. request it as a clarification.

Hatmax must not invent a hosting provider or account owner. A user-level module
base preference is deferred; it does not participate in first-delivery
planning or inference.

An application description is optional. Its absence does not block planning.

## Target Selection and Admission

The session directory is the default parent directory. Hatmax appends the
normalized project slug:

```text
session directory: /workspace/projects
project slug:      real-estate
target:            /workspace/projects/real-estate
```

The user does not need to create or enter the target directory first. An
explicitly supplied target may override the default only after the same safety
and admission checks.

The target is admitted when:

- it does not exist;
- it is empty; or
- it contains Git metadata and non-conflicting repository files that the plan
  will preserve.

An existing compatible Hatmax application selects evolution mode instead of
`create_application`. Another Go module, an incompatible application, a path
outside the authorized base, or an undeclared scaffold collision blocks the
operation. Existing content can be integrated only when a Book rule defines
the exact merge behavior.

The plan lists every preserved file and every planned create or update effect.
Hatmax never silently overwrites target content.

Hatmax preserves existing Git metadata but does not initialize a repository,
create branches, or change remotes. Source-control initialization remains an
explicit user action outside `create_application`.

## Typed Intent

The intent adds the `create_application` operation and an application identity
object through intent schema version 3 and Book release 2. A minimal normalized
projection is:

```yaml
schema_version: 3
operation: create_application
application:
  display_name: Real Estate
  project_slug: real-estate
  module_path: code.example/alex/real-estate
target:
  base: session_directory
archetype: server_rendered_hatmax_application
book_version: 2
documentation: not_requested
initial_features: []
```

The intent records product decisions. It does not contain file paths below the
target, imports, dependency versions, wiring order, or scaffold templates owned
by the Book.

## Composite Initial Requests

A request may combine application creation with one or more initial features.
Hatmax represents that request as one sealed composite plan containing distinct
ordered units:

```text
create_application
create_feature: catalog
create_feature: inventory
```

Feature units use their canonical archetypes and depend on the application
unit. Every unit remains visible and can be revised or removed before approval.
The approved composite plan executes as one coordinated operation; Hatmax does
not request a new confirmation between its units.

A later standalone feature request remains a normal single operation. It does
not inherit the expanded presentation ceremony of an initial composite plan.

## Canonical Scaffold Obligations

The Book must define one canonical `server_rendered_hatmax_application`
archetype. Its minimum scaffold includes:

- a Go module with a compatible, explicit Hatmax dependency;
- `main.go` containing only the `main` function;
- application construction, dependency builders, and lifecycle assembly in
  `internal/application`;
- startup configuration and validation;
- canonical Hatmax logging;
- router, middleware, server, and graceful shutdown wiring;
- Postgres database lifecycle wiring;
- template and static-asset embedding;
- a neutral server-rendered landing page;
- migration support that becomes active with the first Book-owned persistent
  feature;
- basic meaningful tests and a composition compile gate;
- canonical local build, test, formatting, and lint commands;
- ignore rules for generated binaries, runtime state, and local secrets.

`main.go` delegates process execution to `internal/application` and contains no
builders, component declarations, route registration, or helper functions.
The application package exposes the single process entrypoint used by `main`
and owns configuration, construction, lifecycle order, serving, shutdown, and
exit classification.

The neutral scaffold does not create a no-op migration, an application table,
or another persistence artifact without a domain owner. It wires the Postgres
database because Postgres is part of the canonical Hatmax application, but it
does not register `db.Migrator` until a later admitted operation adds the first
real migration. That operation creates the migration directory and migration,
adds the migrator in canonical lifecycle order, and validates both effects in
one plan.

The mandatory root metadata is:

- `go.mod` and `go.sum`;
- `config.yaml`;
- `Makefile`;
- `.gitignore`.

A README, license, CI configuration, Git repository, and Diataxis tree are not
implied scaffold metadata and are not generated without separate intent.

The exact package and file layout belongs to the application archetype. It must
preserve explicit dependency assembly and must not hide application wiring in
reflection, generated runtime registries, or a persistent project DSL.

The neutral page proves that request routing, templates, rendering, and assets
are assembled. It is application shell behavior, not an invented domain
feature.

## Documentation Boundary

Application creation does not activate Diataxis documentation. Documentation
is generated only when the normalized request contains explicit documentation
intent. A required repository metadata file may be included only when the Book
classifies it as scaffold metadata rather than inferred product documentation.

## Planning and Approval

Every scaffold produces an inspectable plan before target creation. The plan
includes:

- normalized application identity and target;
- selected Hatmax and Book versions;
- scaffold archetype and capabilities;
- preserved, created, and updated paths;
- initial feature units, when any;
- dependencies and project commands;
- compilation, conformance, and validation obligations;
- a digest over the target observations and semantic plan.

Approval is bound to that digest. A newly created target, changed preserved
file, remote change, or other relevant drift invalidates approval.

## Execution and Atomicity

Hatmax prepares the new application in an isolated staging location under the
authorized base. It validates paths before writing and publishes the staged
tree to the target only after structural preparation succeeds.

If the target already contains admitted files, Hatmax captures rollback
material and applies only the effects listed by the plan. A conflict or unsafe
rollback produces exact diagnostics instead of overwriting concurrent work.

Cancellation before publication leaves the target unchanged. Failures retain
the conversational proposal and diagnostics but never retain approval.

## Validation

Successful compilation is the mandatory scaffold gate. Hatmax must repair a
compilation failure within the admitted plan or report generation failure.

The scaffold includes tests that express real contracts and are expected to be
green. Test execution is best effort when external infrastructure is not
available:

- an observed test failure caused by generated behavior is a generation
  failure;
- a test that cannot run because a declared external prerequisite is missing
  produces `validation_incomplete` and does not invalidate a compiling
  scaffold;
- Hatmax reports every command attempted, skipped, blocked, passed, or failed.

Conformance remains independent from compilation and tests. A compiling
scaffold that violates its Book rules is not successful.

## Observable Results

Application creation returns one of:

- `clarification_required`;
- `target_incompatible`;
- `capability_unsupported`;
- `cancelled`;
- `plan_stale`;
- `generation_failed`;
- `completed_with_incomplete_validation`;
- `completed`.

Every non-completed result identifies whether the target exists, which changes
were retained, and the smallest safe next action.

## Cross-Specification Reconciliation

The Book, intent, execution, umbrella, and CRUD contracts define pre-project
inventory, target fingerprints, selected Hatmax versions, and
`internal/application` as the canonical composition location. Implementations
must use those shared contracts rather than special-case application creation
outside the ordinary planning and conformance pipeline.

## Acceptance Criteria

- A user can create a named Hatmax application from its parent directory.
- A terse valid request asks only for irreducible missing identity.
- Detailed requests can include niche context and multiple visible initial
  feature units.
- Name, slug, module path, and Go package identities follow distinct rules.
- Existing non-conflicting repository files are preserved and listed.
- Incompatible targets and collisions fail before overwrite.
- `main.go` contains only `main`; canonical construction lives elsewhere.
- Git is preserved when present and is never initialized implicitly.
- The neutral scaffold does not invent a bootstrap migration.
- The neutral application compiles and satisfies scaffold conformance.
- Missing external test infrastructure is distinguished from a failing test.
- No domain feature or Diataxis documentation is invented.
- The generated application remains ordinary Go source without a required
  Hatmax generator runtime.

## Deferred Decisions

- The location and format of an optional user-level module-base preference.
- Source-control initialization as a separately authorized capability.
- Compatibility-alias duration, owned by the conversational product surface.
