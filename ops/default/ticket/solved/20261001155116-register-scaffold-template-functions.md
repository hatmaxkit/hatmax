---
id: TKT-20261001155116
title: Register scaffold template functions
status: solved
resolution: fixed
closed_at: 2026-10-01T16:24:11Z
kind: bug
severity: high
priority: high
scope: ui
tags: generator, templates, correctness
source: implementation
reported_at: 2026-10-01T15:51:16Z
ready_at: 2026-10-01T15:57:00Z
started_at: 2026-10-01T15:57:00Z
reviewed_at: 2026-10-01T16:02:07Z
branch: fix/ticket-20261001155116-scaffold-template-functions
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/89
commits: a7f40fa3ea0ef7cf9bebc9e55625db0012c183e8, ce2ce2b89d38393a6e12e4e7a2693d511300a9f6
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

A real application creation with an initial invoice feature builds, but TestNeutralWebSurface fails when TemplateManager.Start parses the generated feature form: function "hxAttrs" not defined. The neutral web renderer and composite application renderer both construct template managers without WithFuncMap, while feature templates require HTMX helpers. Their template function registry is an application wiring responsibility, independent of the pgx/x/text dependency overrides.

Sources: `generator/execute/render_application_web.go`, `generator/execute/render_application_features.go`, and `htmx/funcmap.go`.

Evidence: With real Go/sqlc commands and available PostgreSQL, TestTerminalSurfaceCreatesTerseAndDetailedApplications passed its terse case and failed its detailed composite case during generated application tests. The application used published Hatmax v0.5.0 with no module replacement. Existing default CLI tests substitute execution commands and do not exercise this template startup boundary.

## Expected Outcome

Register the appropriate Hatmax-owned template functions on every scaffold-created template manager. Verify real application creation with initial features and neutral template startup against the supported published library. Keep the fix separate from dependency version selection.

## Validation

Run the real terse and detailed application acceptance with PostgreSQL available. Both generated applications must build and pass tests. Cover required template-function registration in renderer regression tests and confirm no alternate helper implementation or dependency is introduced.

Related: [F22 scaffold dependency baseline](../reviewing/20260930211821-refresh-the-published-scaffold-dependency-baseline.md).

## Delivery

Both generated template managers now use web.WithFuncMap(ui.FuncMap()), matching Hatmax's example and registering the existing render, HTMX, and UI helpers before startup. Regression cases check the neutral and composite assembly paths, and generated template startup parses a real HTMX-helper probe.

Real terse and detailed application creation passed against published Hatmax v0.5.0 with no replacement and PostgreSQL available. The detailed case includes initial invoice generation, sqlc, compilation, and generated tests. `make check` and `make docs-check` passed; total coverage is 81.7%. Dependency baseline changes remain separate in F22.

Report: [Scaffold Template Functions](../../report/20261001160207-scaffold-template-functions.md).

Integration verified on Forgejo: PR #89 merged into dev at ce2ce2b89d38393a6e12e4e7a2693d511300a9f6. F22 remains a separate pending delivery.
