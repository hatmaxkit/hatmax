---
id: TKT-20260930211817
title: Honor configured bcrypt cost during signup
status: reviewing
kind: bug
severity: medium
priority: normal
scope: domain
tags: architecture-review, domain, correctness
source: review
reported_at: 2026-09-30T21:18:17Z
ready_at: 2026-10-01T13:50:24Z
started_at: 2026-10-01T13:50:24Z
reviewed_at: 2026-10-01T13:57:31Z
branch: fix/ticket-20260930211817-bcrypt-cost
pr: pending
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Signup calls model.HashPassword, which always uses bcrypt.DefaultCost. Auth.BCryptCost is validated and defaults to 12, but the resulting password hash has cost 10.

Sources: `auth/service.go:79, model/password.go:17-18, config/config.go:198`.

Evidence: TestObservedIgnoredPasswordCost configured cost 12, created a user through Signup, and inspected the resulting bcrypt hash at cost 10.

Impact: Operators cannot apply the documented password hashing work factor; configuration gives a false impression of the policy being used.

## Expected Outcome

Pass the configured cost through the signup hashing boundary while preserving password comparison and the existing standalone helper's compatibility.

## Validation

Inspect hashes at two supported configured costs, verify password comparisons, and preserve invalid-cost validation.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f18).

## Delivery

Signup now passes Auth.BCryptCost through model.HashPasswordWithCost. The explicit-cost primitive rejects values outside 4 through 31 without fallback, preserves wrapped bcrypt errors, and returns no hash on failure. The original model.HashPassword signature and bcrypt.DefaultCost behavior remain unchanged.

`make check`, `make docs-check`, the auth/model race-enabled suite, and 20 repetitions of the focused cost, failure, and compatibility tests passed. Configured signup costs 4, 6, and Hatmax's default 12 are checked through bcrypt.Cost; sign-in remains valid after changing the configured cost.

Report: [Configured Signup Bcrypt Cost](../../report/20261001135731-bcrypt-cost.md).
