---
id: TKT-20260930211818
title: Resolve inert scheduler retry configuration
status: open
kind: follow_up
severity: medium
priority: normal
scope: domain
tags: architecture-review, domain, correctness
source: review
reported_at: 2026-09-30T21:18:18Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

RetryAttempts and RetryBackoff are accepted and defaulted, but never used by process. CreateRun always records attempt 1. scheduler/README.md presents these as retry settings while the reference correctly states that retries are not implemented.

Sources: `scheduler/config.go:14-20, scheduler/runner.go:194-240, scheduler/postgres/store.go:64-72`.

Evidence: Source review found no retry execution path; docs/reference/scheduler/README.md:53 explicitly acknowledges the limitation.

Impact: The API advertises operational controls that have no effect. This is a documented capability gap, not a newly discovered hidden implementation promise.

## Expected Outcome

Choose one explicit retry contract: implement bounded retry/backoff and attempt persistence, or remove/deprecate unsupported controls and align package guidance. Keep this independent of recurring slot advancement.

## Validation

Verify the selected contract through fake-clock and PostgreSQL tests. If retries are implemented, cover maximum attempts, backoff, restart, and terminal failure.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f19).
