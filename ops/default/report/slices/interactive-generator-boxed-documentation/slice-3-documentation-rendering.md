# Slice 3: Diataxis Rendering and Conformance

Status: reviewing
Delivery set: interactive-generator-boxed-documentation
Plan: `ops/default/plan/interactive-generator-boxed-documentation.md`
Tracker: `ops/default/tracker/interactive-generator-boxed-documentation.md`
Branch: `feat/generator-documentation-rendering`
PR: #43

## Purpose

Render and apply canonical Diataxis Markdown from sealed Hatmax evidence while
preserving user-owned content and independently checking the resulting
documentation surface.

## Delivered Behavior

Execution now owns one balanced Markdown section delimited by exact Hatmax
markers. It creates missing documentation files, replaces only the managed
section in existing files, and preserves every byte before and after that
section. Preparation seals the digest of those outside bytes so conformance
can detect later replacement of user-owned content.

Distinct deterministic renderers produce tutorials, how-to guides, reference
pages, explanations, and canonical navigation indexes. Their headings,
commands, terminology, links, fields, routes, tables, and validation details
come from the sealed documentation plan and feature evidence rather than
model-authored prose. Documentation edits coexist with implementation edits
in one combined manifest while the implementation renderers retain ownership
of their own surfaces.

Documentation conformance independently checks explicit authorization, exact
manifest scope, managed markers, outside-byte preservation, quadrant-specific
structure, canonical backlinks and index links, local-link resolution,
agreement with feature evidence, and the absence of placeholders. Stable
diagnostics identify documentation scope, Diataxis, ownership, navigation,
and evidence failures.

## Implementation Notes

The execution-manifest schema is now version 3. Existing managed targets gain
a `managed_outside_digest` postcondition; newly created targets require one
balanced managed section after application. Rendering remains deterministic
and contains no backend or model dependency.

Documentation uses the existing atomic workspace. A failed multi-file commit
rolls back every target, while a repository documentation-gate failure is
reported after committed changes remain available for inspection under the
existing execution contract.

## Contracts Added or Changed

- `execute.RenderDocumentation` renders only planned documentation effects.
- Managed Markdown markers define Hatmax ownership without claiming the rest
  of the file.
- Pure documentation manifests remain documentation-only; combined manifests
  can be rendered by both the implementation and documentation renderers.
- Conformance validates the rendered files against sealed plan and project
  evidence instead of trusting renderer output.
- Documentation validation commands remain repository-declared manifest
  commands.

## Files of Interest

- `generator/execute/managed_markdown.go`
- `generator/execute/render_documentation.go`
- `generator/execute/conformance_documentation.go`
- `generator/execute/managed_markdown_test.go`
- `generator/execute/render_documentation_test.go`
- `generator/execute/documentation_conformance_test.go`

## Validation

- `go test ./generator/execute/... ./generator/project/...` passed.
- `go test -race ./generator/execute/...` passed.
- `go test ./generator/...` passed.
- `make docs-check` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.

## Risks and Follow-ups

This slice exposes deterministic rendering and conformance APIs but does not
yet route documentation requests through the interactive product. Slice 4
must coordinate pure and combined execution, extend Codex interpretation and
terminal presentation, add end-to-end regeneration and authenticated smoke
coverage, and record the usable capability in the changelog.
