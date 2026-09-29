# Boxed Diataxis Documentation Generation

Status: Approved
Kind: Subordinate capability specification
Umbrella: `ops/default/spec/interactive-hatmax-generator.md`
Book: `ops/default/spec/hatmax-book.md`
Intent: `ops/default/spec/intent-and-planning.md`
Execution: `ops/default/spec/execution-and-conformance.md`
Product surface: `ops/default/spec/interactive-product-surface.md`

## Purpose

The boxed documentation capability creates and updates the smallest warranted
Diataxis documentation surface for a canonical Hatmax feature. Documentation
is generated only from explicit user intent and remains constrained by Hatmax
structure, terminology, project evidence, and validation.

Codex interprets the requested reader need. Hatmax selects the canonical
documentation targets, derives their paths, renders their managed content,
updates navigation, and checks conformance. The model does not choose a
different documentation architecture or edit arbitrary Markdown.

## Activation

Documentation remains `not_requested` unless the user explicitly asks to
document behavior. Feature creation, field changes, validation changes,
novelty, size, or user visibility never imply documentation authorization.

The initial capability supports:

- a documentation-only request for an existing `server_rendered_crud`
  feature;
- documentation combined with `create_feature`, `add_field`, or
  `add_validation`;
- one or more explicitly requested Diataxis needs for the same feature.

A documentation-only request uses the `document_feature` operation with
`document_existing_behavior`. A combined implementation request keeps its
implementation operation and uses `document_planned_change`.

If a request says only "document this" and the intended reader outcome cannot
be determined, the interpreter asks one focused question. It must not create
one document in every quadrant as a substitute for clarification.

## Typed Documentation Intent

The existing `documentation` field remains the authorization mode:

- `not_requested`;
- `document_existing_behavior`;
- `document_planned_change`.

An active mode requires a bounded `documentation_targets` list. Each target
contains:

```yaml
quadrant: reference
subject: invoice
reader_goal: Find the exact invoice fields, routes, and validation behavior.
```

`quadrant` is one of `tutorial`, `how_to`, `reference`, or `explanation`.
`subject` identifies an admitted feature or one of its documented behaviors.
`reader_goal` records the user-facing need without prescribing paths,
headings, prose, or implementation details.

Hatmax derives slugs, paths, titles, index placement, and required sections.
The interpreter cannot return a file path, Markdown body, navigation edit, or
arbitrary documentation kind. `documentation_targets` must be empty when the
mode is `not_requested`.

The initial contract permits at most one target per quadrant and subject and
at most four targets in one interaction. Duplicate or contradictory targets
are rejected before planning.

## Evidence and Source of Truth

Documentation is derived from the selected Hatmax Book, the admitted plan,
the execution manifest, and inspected canonical project structure. For an
existing feature, Hatmax inspects its model, service boundary, handler routes,
templates, persistence mapping, validation, wiring, and tests through the
same bounded structural parsers used by execution and conformance.

The model does not receive arbitrary project files and does not author the
Markdown body. It only classifies the reader need into typed documentation
intent. Generated prose must not claim behavior that cannot be established
from project evidence or the approved planned change.

Project source remains authoritative. Generated documentation is ordinary
Markdown and never becomes an input required to build or run the project.

## Canonical Layout

Generated artifacts use only these roots:

```text
docs/
├── README.md
├── tutorials/
│   ├── README.md
│   └── <subject>/README.md
├── how-to/
│   ├── README.md
│   └── <subject>/README.md
├── reference/
│   ├── README.md
│   └── <subject>/README.md
└── explanation/
    ├── README.md
    └── <subject>/README.md
```

Only requested quadrants and the README entrypoint chain needed to reach them
are created or changed. Hatmax does not create empty quadrant directories or
placeholder documents.

Existing project conventions may supply a compatible title or navigation
wrapper, but they cannot relocate a document outside its Diataxis quadrant or
replace the canonical README entrypoint chain.

## Diataxis Semantics

Each target has one primary reader intent:

- `tutorial` provides a guided learning path with a concrete outcome and
  ordered steps suitable for a newcomer;
- `how_to` provides a focused procedure for a reader who already knows the
  project and needs to complete one task;
- `reference` describes exact observed contracts, fields, routes, validation,
  lifecycle participation, and supported behavior without becoming a lesson;
- `explanation` describes rationale, ownership, tradeoffs, and relationships
  without becoming a procedure or contract inventory.

A document must not mix quadrants to avoid creating the warranted companion
artifact. Cross-links are allowed when they send the reader to a distinct
need.

## Boxed Ownership

Hatmax owns only explicitly marked generated sections. A generated document
contains one balanced managed section:

```markdown
<!-- hatmax:generated:start -->
...
<!-- hatmax:generated:end -->
```

Users may add or edit content outside that section. Regeneration replaces only
the managed section and preserves all other content byte-for-byte. If the
target exists without a valid managed section, Hatmax reports a conflict and
does not overwrite it.

Canonical README entrypoints use the same bounded-section rule for generated
navigation entries. Hatmax may create a missing root or quadrant entrypoint,
but it does not reorder or rewrite user-authored navigation outside the
managed section.

Markers identify mutation ownership only. They do not store intent, project
state, or authority and are not required by the application at runtime.

## Deterministic Planning

Documentation targets are expanded into a sealed plan before any edit. The
plan records:

- authorization mode and typed reader goals;
- selected quadrant and derived target path;
- required root and quadrant entrypoint links;
- inspected evidence and fingerprints;
- the `documentation` affected surface;
- documentation conformance rules and repository validation commands.

A documentation-only plan admits no Go, SQL, template, configuration, or
runtime dependency changes. A combined plan includes documentation edits in
the same manifest as the implementation edits so approval covers one exact
project transition.

Relevant source or documentation drift invalidates approval. A planned-change
document is rendered from the admitted plan and then reconciled against the
committed execution result before repository validation.

## Rendering Requirements

Hatmax-owned renderers produce stable Markdown from typed project evidence.
Every generated document must:

- use a specific, reader-oriented title;
- state its scope without claiming unsupported behavior;
- use Hatmax and project terminology consistently;
- link only to known local targets;
- contain no TODO, placeholder, speculative contract, or model commentary;
- avoid credentials, local paths, internal prompts, and generator state;
- remain useful when read outside the generator.

Reference output enumerates exact contracts. Tutorial and how-to commands must
come from inspected repository policy or canonical Hatmax commands.
Explanation output may state only rationale established by the Book or the
selected archetype.

## Conformance and Failure Behavior

Documentation conformance checks:

- explicit authorization for every documentation edit;
- canonical quadrant and path placement;
- one primary Diataxis intent per document;
- balanced managed-section markers;
- preservation of content outside managed sections;
- complete root and quadrant entrypoint reachability;
- valid local links and no links to missing generated targets;
- agreement between documented contracts and inspected or executed behavior;
- absence of undeclared non-documentation effects.

Stable diagnostics distinguish invalid intent, ambiguous quadrant, missing
feature evidence, unmanaged target conflict, entrypoint conflict, stale
evidence, rendering failure, conformance failure, and repository
documentation-gate failure.

Any conflict discovered before commit leaves the project unchanged. A
repository documentation-gate failure after an atomic documentation commit
retains the declared edits and reports exact changed paths, matching the
existing execution policy.

## Product Surface

`hatmax generate "<request>"` remains the only initial command. The displayed
plan identifies each documentation target, quadrant, path, index effect, and
evidence fingerprint before approval.

The final report lists created, updated, and unchanged documentation paths,
managed-section preservation, conformance, and exact repository-command
evidence. It does not print generated bodies, model output, or target-project
contents as backend provenance.

## Boundaries

The initial capability does not provide a wiki, free-form handbook, arbitrary
Markdown editing, documentation-site theme generation, screenshots, API docs
from reflection, translation, or publication. Those require separate explicit
capabilities.

## Acceptance Criteria

- Implementation requests with `not_requested` produce no documentation
  changes.
- A pure existing-feature request produces only the requested Diataxis target
  and its required index links.
- A combined request includes implementation and documentation in one visible
  plan and bounded manifest.
- Ambiguous reader intent produces focused clarification rather than four
  generic documents.
- All four quadrants render their distinct canonical structure from typed
  evidence.
- Regeneration replaces only managed sections and preserves user additions.
- Existing unmanaged target files fail closed without changes.
- Documentation conformance rejects wrong placement, mixed intent, stale
  contracts, broken links, and undeclared documentation edits.
- The repository documentation gate passes for the exact generated project.
