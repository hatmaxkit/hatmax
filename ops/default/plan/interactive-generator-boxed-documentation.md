<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Interactive Generator Boxed Documentation Delivery Plan

Status: Approved
Delivery set: interactive-generator-boxed-documentation
Slice strategy: layered
Reason: documentation generation crosses the typed interpreter contract,
project evidence and fingerprints, deterministic planning, bounded Markdown
mutation, conformance, and the interactive product surface. Delivering those
layers in dependency order keeps model interpretation separate from Hatmax
ownership of documentation structure and content.
Umbrella spec: `ops/default/spec/interactive-hatmax-generator.md`
Specs:

- `ops/default/spec/hatmax-book.md`
- `ops/default/spec/intent-and-planning.md`
- `ops/default/spec/execution-and-conformance.md`
- `ops/default/spec/interactive-product-surface.md`
- `ops/default/spec/boxed-diataxis-documentation.md`

Tracker: `ops/default/tracker/interactive-generator-boxed-documentation.md`
Base branch: `dev`
Planning base: `1549e663d0d5648e00cdcc7e5fae1c50c5dd29c3`
Active slice: Slice 1
Execution gate: satisfied when this approved plan and tracker are merged into
`dev`

## Objective

Deliver explicit boxed Diataxis generation for existing and planned canonical
Hatmax features. Codex identifies the reader need through a bounded typed
intent. Hatmax derives the quadrant, path, managed Markdown section, index
effects, evidence, plan, edits, and conformance deterministically.

The delivery completes step 6 of the umbrella specification's initial
implementation sequence. It does not resolve the separately captured
generator hardening tickets.

## Delivered Prerequisites

- The plan kernel supplies the Book, project inventory, typed intent, sealed
  plans, stable diagnostics, and deterministic digests.
- Execution and conformance supply bounded manifests, atomic mutation,
  protected-path handling, drift checks, and repository-command evidence.
- The product surface supplies Codex interpretation, clarification, plan
  approval, terminal reporting, and authenticated live validation.
- The boxed documentation specification defines activation, Diataxis
  semantics, managed-section ownership, deterministic rendering, and failure
  behavior.

## Planning Decisions

- Pure documentation uses `document_feature` with
  `document_existing_behavior`; combined changes keep their implementation
  operation and use `document_planned_change`.
- Active documentation requires one to four typed targets. Hatmax derives
  slugs, paths, titles, sections, and index placement.
- A broad request with no determinable reader need asks one clarification. It
  never expands mechanically to all four quadrants.
- Documentation bodies are rendered from typed Book, project, plan, and
  execution evidence. Codex does not author Markdown.
- Hatmax mutates only balanced managed sections and managed navigation blocks.
  Content outside those sections is preserved byte-for-byte.
- An existing target without valid managed markers is an unmanaged conflict
  and fails before mutation.
- Pure documentation admits only the `documentation` surface. Combined
  implementation and documentation are one approved plan and one bounded
  manifest.
- The initial capability targets one canonical `server_rendered_crud` feature
  per interaction and uses the existing `hatmax generate` command.

## Scope

The delivery includes:

- `document_feature` and bounded typed documentation targets;
- Book obligations and rules for explicit documentation, the four Diataxis
  quadrants, managed ownership, and index reachability;
- documentation inventory, feature evidence, fingerprints, and drift;
- deterministic target and index planning;
- managed Markdown and index mutation with user-content preservation;
- canonical tutorial, how-to, reference, and explanation renderers;
- documentation conformance and stable diagnostics;
- pure and combined interaction coordination;
- Codex interpretation, clarification, terminal plan presentation, and result
  reporting for documentation intent;
- deterministic acceptance and authenticated documentation smoke coverage;
- one `Unreleased` changelog entry when the capability becomes usable.

It excludes resolution of the open hardening tickets, free-form Markdown
editing, alternative documentation systems, themes, screenshots,
translation, publication, reflection-generated API documentation, new
application archetypes, Git operations, and release work.

## Architecture Boundaries

Existing package ownership remains authoritative:

```text
generator/book/             Documentation obligations and conformance rules
generator/intent/           Typed authorization mode and reader targets
generator/project/          Documentation inventory, evidence, and fingerprints
generator/plan/             Deterministic target and index expansion
generator/eval/             Backend-neutral structured interpretation contract
generator/backend/codex/    Bounded documentation-intent interpretation
generator/execute/          Markdown rendering, mutation, and conformance
generator/interaction/      Pure and combined documentation lifecycle
internal/hatmaxcli/         Plan and result presentation
```

No backend package may receive target-project files or return Markdown, paths,
edits, or navigation. Documentation rendering remains inside deterministic
Hatmax execution.

## Ordered Slices

| Slice | Short Name | Branch | Pull Request Title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | Documentation contracts | `feat/generator-documentation-contracts` | `feat(slice-1): define boxed documentation contracts` | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-1-documentation-contracts.md` |
| Slice 2 | Documentation inventory and planning | `feat/generator-documentation-planning` | `feat(slice-2): plan boxed documentation changes` | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-2-documentation-planning.md` |
| Slice 3 | Diataxis rendering and conformance | `feat/generator-documentation-rendering` | `feat(slice-3): render boxed Diataxis documentation` | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-3-documentation-rendering.md` |
| Slice 4 | Interactive documentation product | `feat/generator-documentation-product` | `feat(slice-4): deliver interactive documentation generation` | `ops/default/report/slices/interactive-generator-boxed-documentation/slice-4-documentation-product.md` |

## Slice 1: Documentation Contracts

### Purpose

Define documentation as a strict Hatmax intent and Book capability without
adding project inspection, rendering, or mutation.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Add `document_feature`, typed bounded documentation targets, schema validation, cloning, YAML/JSON decoding, stable diagnostics, and the invariant that inactive documentation has no targets. | `feat(generator): define documentation intent` |
| T1.2 | Extend the Book with documentation obligations and required rules for quadrant semantics, managed ownership, index reachability, and explicit authorization; expand them deterministically into plans and digests. | `feat(generator): add boxed documentation rules` |
| T1.3 | Extend the evaluation corpus with pure, combined, ambiguous, equivalent, invalid, and implicit-documentation cases without calling a live model. | `test(generator): cover documentation contracts` |

### Validation

```sh
go test ./generator/intent/... ./generator/book/... ./generator/plan/... ./generator/eval/...
go test -race ./generator/intent/... ./generator/plan/... ./generator/eval/...
go test ./generator/...
make vet
make lint-strict
git diff --check
```

## Slice 2: Documentation Inventory and Planning

### Purpose

Turn admitted documentation intent into exact evidence, paths, fingerprints,
and bounded manifest effects without rendering or committing Markdown.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T2.1 | Inspect canonical documentation roots, indexes, links, managed markers, protected paths, and documentation validation commands without treating arbitrary prose as structured authority. | `feat(generator): inspect documentation surfaces` |
| T2.2 | Extract bounded canonical feature evidence for existing behavior and reconcile planned-change evidence from sealed plans and manifests; include every relevant source and documentation input in freshness checks. | `feat(generator): collect documentation evidence` |
| T2.3 | Derive canonical target paths, titles, required index effects, snapshots, postconditions, and documentation-only or combined execution manifests; reject unmanaged and overlapping targets before mutation. | `feat(generator): plan documentation effects` |

### Validation

```sh
go test ./generator/project/... ./generator/plan/... ./generator/execute/...
go test -race ./generator/project/... ./generator/execute/...
go test ./generator/...
make vet
make lint-strict
git diff --check
```

## Slice 3: Diataxis Rendering and Conformance

### Purpose

Render and apply canonical Markdown while preserving user-owned content and
independently proving the resulting documentation surface.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T3.1 | Add balanced managed-section and navigation editors that create missing canonical indexes, replace only owned content, preserve all surrounding bytes, and remain deterministic and idempotent. | `feat(generator): manage boxed documentation sections` |
| T3.2 | Add distinct tutorial, how-to, reference, and explanation renderers from typed evidence, including canonical headings, commands, local links, and terminology without placeholders or speculative behavior. | `feat(generator): render Diataxis documentation` |
| T3.3 | Add conformance for authorization, placement, quadrant intent, markers, preservation, index reachability, links, evidence agreement, and undeclared effects; cover conflicts, drift, rollback, and repository documentation-gate failures. | `test(generator): enforce documentation conformance` |

### Validation

```sh
go test ./generator/execute/... ./generator/project/...
go test -race ./generator/execute/...
go test ./generator/...
make docs-check
make vet
make lint-strict
git diff --check
```

## Slice 4: Interactive Documentation Product

### Purpose

Expose pure and combined boxed documentation through the existing interactive
command and prove the complete authenticated product boundary.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T4.1 | Coordinate documentation-only execution and combined implementation-plus-documentation execution, including clarification, one approval, freshness, post-implementation evidence reconciliation, atomic mutation, and exact result ownership. | `feat(generator): coordinate documentation generation` |
| T4.2 | Extend the Codex schema and prompt plus terminal plan and result presentation for documentation targets without exposing paths or prose to backend control. | `feat(generator): interpret documentation requests` |
| T4.3 | Add terminal acceptance for no-documentation, pure existing behavior, combined planned change, all quadrants, regeneration preservation, unmanaged conflicts, and failures; add an authenticated documentation smoke and the completed behavior to `CHANGELOG.md`. | `test(generator): validate boxed documentation generation` |

### Validation

```sh
go test ./cmd/hatmax/... ./internal/hatmaxcli/... ./generator/interaction/... ./generator/backend/codex/...
go test -race ./generator/interaction/... ./generator/backend/codex/... ./generator/execute/...
go test ./generator/...
make check
make generator-live-smoke
make docs-check
git diff --check
```

## Delivery-Set Gate

After all four slices are merged into `dev`:

1. capture the exact `origin/dev` commit;
2. run `make check` with Docker access against that commit;
3. run the complete deterministic generator corpus with the race detector;
4. run `make generator-live-smoke` with an explicit documentation request;
5. verify ordinary implementation requests retain `not_requested` and make no
   documentation changes;
6. verify pure documentation changes only documentation and required indexes;
7. verify a combined request produces one approved plan and exact manifest;
8. verify all quadrants, regeneration preservation, unmanaged conflicts,
   stale evidence, and project isolation;
9. run `make docs-check` and `git diff --check`;
10. close the tracker only when the exact candidate is green.

No hardening-ticket implementation, `main` alignment, mirror update, tag, or
release is implied by delivery-set closure.

## Completion Criteria

- Documentation remains absent unless explicitly requested.
- Pure and combined requests produce strict typed documentation targets.
- Hatmax derives every path, section, index effect, and Markdown body.
- All four quadrants have distinct deterministic output.
- User-authored content outside managed sections is preserved exactly.
- Unmanaged conflicts and stale evidence fail closed.
- Documentation plans, manifests, conformance, and terminal reports expose the
  complete bounded effect without exposing project content to Codex.
- The exact integrated `dev` candidate passes deterministic, repository,
  documentation, and authenticated live gates.
