---
id: TKT-20260930211807
title: Pair startup rollback with component identity
status: open
kind: bug
severity: high
priority: high
scope: domain
tags: architecture-review, domain, hardening
source: review
reported_at: 2026-09-30T21:18:07Z
commits:
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
