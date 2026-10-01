<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Scaffold Template Functions

Status: delivered
Ticket: [TKT-20261001155116](../ticket/solved/20261001155116-register-scaffold-template-functions.md)
Related: [F22 scaffold baseline](../ticket/open/20260930211821-refresh-the-published-scaffold-dependency-baseline.md)
Branch: `fix/ticket-20261001155116-scaffold-template-functions`
PR: [#89](https://forge.adrianpk.com/hatmax/hatmax/pulls/89)
Implementation: `a7f40fa3ea0ef7cf9bebc9e55625db0012c183e8`
Integrated dev: `ce2ce2b89d38393a6e12e4e7a2693d511300a9f6`

## Delivered Behavior

- The neutral web renderer and composite application renderer pass web.WithFuncMap(ui.FuncMap()) to their template managers. Templates can use the existing Hatmax render, HTMX, and UI functions before parsing starts.
- Newly created applications with initial features no longer fail template startup because hxAttrs is missing. Real terse and detailed creation complete with published Hatmax v0.5.0 and available PostgreSQL.
- Unreleased describes the startup fix. No dependency versions, Book selections, public APIs, templates, or helper implementations change.

## Ownership and Regression

Template function registration belongs to application assembly. This change uses the same public API as the Ticked example and works with the supported published library. Each existing manager receives its own function map; manager ownership and component ordering are unchanged.

Named renderer cases check both assembly paths. The generated-application test additionally loads a template that calls hxAttrs, hxPost, hxTargetID, and hxSwapOuter through the generated manager, preventing a test-owned function map from masking registration failures.

## Validation

All checks used Go 1.26.7. Database tests used an isolated native PostgreSQL 18.6 cluster, stopped after validation.

- Before the fix, the focused regression failed for both assembly paths and generated template startup reported function "hxAttrs" not defined.
- `make check`: passed licensing, formatting, vet, all tests, coverage, and strict lint. Total coverage: 81.7%; generator/execute: 79.1%.
- `make docs-check`: passed.
- `go test ./generator/execute -run '^(TestScaffoldTemplateFunctions|TestRenderedApplicationCompilesAndPassesGeneratedTests)$' -count=1`: passed. Its compilation test uses the local Hatmax checkout.
- `HATMAX_REAL_APPLICATION_COMMANDS=1 GO_TEST_COMMAND='go test ./...' go test -v -count=1 -run '^TestTerminalSurfaceCreatesTerseAndDetailedApplications$' ./internal/hatmaxcli`: passed both terse and detailed cases. These generated consumers use published Hatmax v0.5.0 without a local replace; real Go/sqlc commands and PostgreSQL-backed generated tests ran.
- `git diff --check`: passed.

## Boundary

This independently approved ticket fixes the template-registration defect discovered during F22 validation. F22 dependency overrides, its security gate, existing application rewrites, publication, tags, and mirrors are outside this PR. PR #89 is integrated into dev; F22 remains pending until its full acceptance is repeated.
