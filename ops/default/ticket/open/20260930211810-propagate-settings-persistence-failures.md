---
id: TKT-20260930211810
title: Distinguish missing settings from persistence failures
status: open
kind: bug
severity: medium
priority: normal
scope: persistence
tags: architecture-review, persistence, correctness
source: review
reported_at: 2026-09-30T21:18:10Z
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
