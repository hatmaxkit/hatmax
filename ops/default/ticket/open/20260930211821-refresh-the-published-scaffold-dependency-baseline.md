---
id: TKT-20260930211821
title: Refresh the published scaffold dependency baseline
status: open
kind: follow_up
severity: high
priority: high
scope: ops
tags: architecture-review, dependencies, security
source: review
reported_at: 2026-09-30T21:18:21Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

The updated scaffold uses chi v5.3.2 but still requires the Book-selected Hatmax v0.5.0. A standalone consumer with the same application lifecycle imports resolves older pgx and x/text versions from that published module. govulncheck reports reachable GO-2026-5004 and GO-2026-5970; upgrading the checkout does not upgrade the published module.

Sources: `generator/execute/render_application.go:46-63, generator/book`.

Evidence: A separate consumer requiring Hatmax v0.5.0 and chi v5.3.2 was tidied and scanned. The scan found four advisory IDs, including two with reachable symbols. It had no local replacement.

Impact: A clean vulnerability scan of this checkout does not prove newly generated applications use the refreshed dependency baseline.

The affected-symbol advisories are [GO-2026-5004](https://pkg.go.dev/vuln/GO-2026-5004) and [GO-2026-5970](https://pkg.go.dev/vuln/GO-2026-5970). Reachability is scanner evidence, not proof of a concrete application exploit.

## Expected Outcome

After an explicitly approved publication, update the Book-supported Hatmax release, scaffold module requirement, and checksums together. Verify the generated consumer without a local replace. If publication is deferred, choose and test explicit dependency overrides rather than assuming the checkout's go.mod affects consumers.

Publication still requires maintainer approval. This ticket does not authorize a tag or release.

## Validation

Generate and scan a standalone application using only published dependencies. Confirm pgx/x/text affected symbols are absent, checksums resolve, and both scaffold and feature acceptance pass.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f22).
