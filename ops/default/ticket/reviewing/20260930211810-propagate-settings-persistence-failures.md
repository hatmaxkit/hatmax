---
id: TKT-20260930211810
title: Distinguish missing settings from persistence failures
status: reviewing
kind: bug
severity: medium
priority: normal
scope: persistence
tags: architecture-review, persistence, correctness
source: review
reported_at: 2026-09-30T21:18:10Z
ready_at: 2026-10-01T09:39:48Z
started_at: 2026-10-01T09:39:48Z
reviewed_at: 2026-10-01T09:49:27Z
branch: fix/ticket-20260930211810-settings-errors
pr: pending
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

GetString, GetInt, and GetBool replace every store error with a default value and a nil error. A storage outage is indistinguishable from an absent setting.

Sources: `settings/service.go:27-58`.

Evidence: TestObservedSettingsFailure returned a valid true default and no error after the fake store reported a storage failure.

Impact: Runtime switches can silently revert during failures. Callers cannot choose whether to preserve prior state, fail closed, or report degraded operation.

## Expected Outcome

Define a not-found contract for Store. Apply defaults only for the intended missing-value case and propagate other errors, including context cancellation.

## Validation

Cover missing values, explicit empty values, malformed values, cancellation, and storage failures for all three typed getters.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f11).

## Implementation

Store.Get now has an explicit ErrNotFound contract for absent keys. Service getters recognize wrapped absence through errors.Is and apply defaults only in that case. All other store errors are returned unchanged with zero values, including cancellation and expired deadlines. Partial values returned alongside a failed lookup do not become successful results.

GetString preserves present empty strings. GetInt and GetBool parse present values, including empty strings, instead of replacing them with defaults; malformed values return parse errors. Missing unregistered keys and empty defaults retain zero-value behavior. Invalid defaults are parsed only for absence. Getters do not persist fallback values.

The guide's in-memory adapter now reports ErrNotFound and propagates context errors. Tests cover the three typed getters, wrapped absence, stored empty and zero values, malformed values, invalid defaults, partial read failures, cancellation, and the example adapter. Documentation states the custom-adapter migration and deletion-based default reset.

Delivery: [Settings read failures](../../report/20261001094927-settings-read-failures.md).
