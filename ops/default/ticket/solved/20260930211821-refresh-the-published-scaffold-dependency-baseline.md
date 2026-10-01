---
id: TKT-20260930211821
title: Refresh the published scaffold dependency baseline
status: solved
resolution: fixed
closed_at: 2026-10-01T17:08:47Z
kind: follow_up
severity: high
priority: high
scope: ops
tags: architecture-review, dependencies, security
source: review
reported_at: 2026-09-30T21:18:21Z
ready_at: 2026-10-01T15:43:00Z
started_at: 2026-10-01T15:43:00Z
reviewed_at: 2026-10-01T16:28:02Z
branch: fix/ticket-20260930211821-scaffold-baseline
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/90
commits: ac9355a08a966aa129f84cbcc481560b685f08bf, 352bd5a7f97541f1129bd172e057a03cf8feb998
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

## Selected Contract

Keep the Book's supported published Hatmax v0.5.0 and add explicit scaffold requirements for pgx v5.11.0 and x/text v0.42.0, aligned with the reviewed checkout and verified published modules. Seed their module and archive checksums together. This selects corrected transitive dependencies through Go module version selection without a local replace or release. Add consumer acceptance that verifies selected versions and rejects replacements, plus a standalone scaffold vulnerability gate. Existing applications are not rewritten automatically; no tag or publication is authorized.

## Validation Checkpoint

- The regression test reproduced missing scaffold pgx/x/text requirements and checksums before the fix. A generated published-dependency consumer scan reproduced reachable GO-2026-5004 and GO-2026-5970.
- With explicit overrides, `make generator-scaffold-acceptance generator-scaffold-security` passed. The generated application resolves published Hatmax v0.5.0, pgx v5.11.0, and x/text v0.42.0 with no replacements. govulncheck found zero reachable vulnerabilities and zero affected imported packages, while reporting 17 advisories in required modules whose vulnerable symbols were not called.
- `go test ./generator/execute ./generator/book ./generator/plan -count=1` and acceptance-tagged strict lint passed.
- Real composite generation initially failed at template startup because hxAttrs was not registered. The separate [template-function ticket](../solved/20261001155116-register-scaffold-template-functions.md) was integrated through PR #89 before repeating F22 acceptance. Both real terse and detailed creation now pass against published dependencies with PostgreSQL available. The template fix is not part of the F22 diff.
- `make generator-project-acceptance` passed real feature generation and its project gate against the local Hatmax checkout. This is additional regression evidence, not the published-module security proof.
- `make check` and `make docs-check` passed. Total coverage is 81.7%; acceptance-tagged strict lint reports zero issues.

Report: [Published Scaffold Baseline](../../report/20261001162718-published-scaffold-baseline.md).

## Integration

PR #90 is verified merged into dev at 352bd5a7f97541f1129bd172e057a03cf8feb998. Final validation on that exact commit passed make check, make docs-check, all race-enabled tests, standalone scaffold acceptance and security, real terse and detailed application creation, existing-project feature acceptance, and checkout govulncheck. Total coverage is 81.7%. All F1-F22 tickets are solved; publication remains outside this delivery.
