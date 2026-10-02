<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 4: Interactive Documentation Product

Status: delivered
Delivery set: interactive-generator-boxed-documentation
Plan: `ops/default/plan/interactive-generator-boxed-documentation.md`
Tracker: `ops/default/tracker/interactive-generator-boxed-documentation.md`
Branch: `feat/generator-documentation-product`
PR: #44 (merged)

## Purpose

Expose pure and combined boxed documentation through `hatmax generate`, show
the complete bounded effect before approval, and prove the product boundary
through deterministic terminal acceptance and authenticated Codex inference.

## Delivered Behavior

The interaction coordinator now dispatches `document_feature` directly to the
documentation renderer and combines documentation mutations with
`create_feature`, `add_field`, or `add_validation` mutations under one sealed
manifest, one approval, and one atomic workspace commit. Post-commit
conformance reconciles planned documentation evidence with the structurally
inspected feature that was actually produced.

The Codex prompt defines the documentation-only and combined authorization
modes, the four bounded reader needs, and focused clarification for ambiguous
requests. Codex still cannot return paths, Markdown, edits, commands, plans,
or approval decisions. The existing structured-output schema admits the
typed documentation operation, mode, and targets defined by the delivered
intent contract.

Terminal output now presents documentation mode, quadrant, subject, reader
goal, derived path, index effects, evidence basis, created or updated paths,
and managed-section ownership. It never prints generated bodies or model
output as provenance.

## Acceptance Coverage

Terminal acceptance proves that ordinary implementation requests create no
documentation, pure existing-feature requests affect only documentation, and
combined requests produce implementation and documentation under one plan.
It covers all four quadrants, exact user-byte preservation during
regeneration, unmanaged target conflicts without mutation, declared-effect
isolation, and retained changes plus exact command evidence after a
documentation-gate failure.

The authenticated smoke now includes an explicit existing-feature reference
request and requires Codex to return a typed `document_feature` intent with
`document_existing_behavior` and one reference target.

## Files of Interest

- `generator/interaction/execute.go`
- `generator/execute/conformance_documentation.go`
- `generator/backend/codex/prompt.go`
- `generator/backend/codex/live_smoke_test.go`
- `internal/hatmaxcli/report.go`
- `internal/hatmaxcli/acceptance_test.go`
- `CHANGELOG.md`

## Validation

- `go test ./cmd/hatmax/... ./internal/hatmaxcli/... ./generator/interaction/... ./generator/backend/codex/...`
  passed.
- `go test -race ./generator/interaction/... ./generator/backend/codex/... ./generator/execute/...`
  passed.
- `go test ./generator/...` passed.
- `newgrp docker -c 'make check'` passed with 82.3% aggregate coverage.
- `make generator-live-smoke` passed with the installed Codex 0.158.0
  executable selected to match the resident App Server 0.158.0.
- `make docs-check` passed.
- `git diff --check` passed.

The exact integrated `dev` candidate
`3d0dffb10016574d1c3f300a3fb8162eef1f2d41` passed `make check` with
82.3% aggregate coverage, the complete generator corpus with the race
detector, focused terminal documentation acceptance, the authenticated
documentation smoke, `make docs-check`, and `git diff --check`.

## Risks and Follow-ups

The default shell path currently resolves Codex 0.153.0 while the resident
managed App Server is 0.158.0. The authenticated validation selected the
installed 0.158.0 executable explicitly. Runtime version mismatch continues
to fail closed as specified.

The three generator-quality hardening tickets remain outside this delivery
set. The delivery set is closed. No `main` alignment, mirror update, tag, or
release is implied.
