---
id: TKT-20261009084416
title: Preserve product coverage scope after documentation tool migration
status: solved
kind: bug
severity: medium
priority: high
scope: ops
tags: release, coverage, documentation
source: implementation
reported_at: 2026-10-09T08:44:16Z
ready_at: 2026-10-09T08:45:34Z
started_at: 2026-10-09T08:45:34Z
reviewed_at: 2026-10-09T08:45:34Z
branch: fix/ticket-20261009084416-release-coverage-scope
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/120
closed_at: 2026-10-09T08:46:13Z
resolution: fixed
commits: a15200f935ea894720539edcf654b7ab3186c00d
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

Release candidate a2dd0a0bb679 passes the complete default test suite but fails
the 80% coverage threshold at 73.2%. The newly migrated Go documentation
workflow command contributes 3,293 statements with 18.7% unit-test coverage.
The same profile measures the product packages at 80.7%. The command's actual
documentation walkthroughs are validated through a separate execution gate.

## Expected Outcome

Keep the existing 80% product threshold and all default tests. Exclude
repository workflow tools from the product coverage denominator, consistently
with the existing exclusion of example applications. Keep this scope explicit
in the repository validation contract and refresh the measured badge.

## Validation

- Verify package selection retains product packages and excludes workflow tools.
- Run the unchanged default test suite and product coverage gate with PostgreSQL.
- Run strict lint and whitespace checks.

## Resolution

PR #120 integrated the scope correction into dev with Rebase + Fast-forward
at `a15200f935ea894720539edcf654b7ab3186c00d`. Product coverage passes at 80.7%.
All default tests remain selected and the documentation command's tests pass.
