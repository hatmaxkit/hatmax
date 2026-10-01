---
id: TKT-20260930211807
title: Pair startup rollback with component identity
status: reviewing
kind: bug
severity: high
priority: high
scope: domain
tags: architecture-review, domain, hardening
source: review
reported_at: 2026-09-30T21:18:07Z
ready_at: 2026-10-01T06:55:22Z
started_at: 2026-10-01T06:55:22Z
reviewed_at: 2026-10-01T07:04:45Z
branch: fix/ticket-20260930211807-startup-rollback
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/75
commits: da6fa8463a9ba64f4a90943596f4996724b99980
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Setup collects start and stop capabilities independently, while Start indexes stops using a start index. A start-only component before a failing component causes an index panic; other mixed capability layouts can stop the wrong component.

Sources: `app/lifecycle.go:39-84`.

Evidence: TestObservedRollbackPanic reproduced the index panic. docs/explanation/component-order-and-startup/README.md already documents the positional limitation.

Impact: Startup failure loses its original error and can perform incorrect cleanup. The positional restriction is documented, but the public independent capability model remains fragile.

## Expected Outcome

Preserve component identity when assembling rollback. Stop only successfully started components that own a stop capability, in reverse order.

## Validation

Cover start-only, stop-only, paired, and route-only components, failure at each index, rollback order, and preservation of the original startup error.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f8).

## Implementation

Setup now returns StartupStep values pairing each Start with the same component's optional Stop. Start rolls back completed steps in reverse order, skips start-only components, and returns the original startup error even when a stop fails. Stop-only components remain in the independent normal-shutdown list. Routes remain gated on complete startup success.

Existing Setup-based calls retain their source form, including generated applications pinned to the current published Hatmax version. Manually assembled startup lists now use []app.StartupStep rather than function slices. The lifecycle reference, User Guide, explanation, package note, and Unreleased document that contract.

Delivery: [Startup rollback identity](../../report/20261001070445-startup-rollback-identity.md).
