---
id: TKT-20261009085014
title: Pin CI to the project Go toolchain
status: reviewing
kind: bug
severity: medium
priority: high
scope: ops
tags: release, ci, go
source: implementation
reported_at: 2026-10-09T08:50:14Z
ready_at: 2026-10-09T08:50:45Z
started_at: 2026-10-09T08:50:45Z
reviewed_at: 2026-10-09T08:50:45Z
branch: ci/ticket-20261009085014-pin-release-toolchain
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

PR #121 CI run 22 selects Go 1.27.2 through `1.27.x` and `check-latest: true`.
Lint fails reading `internal/goarch` export data: version 5 exceeds supported
version 4. The exact candidate passes strict lint and the complete local gate
with the project's Go 1.27.1 toolchain.

## Expected Outcome

Pin the lint, test and build jobs to Go 1.27.1, matching `go.mod` and the locally
validated release toolchain. Preserve existing lint rules and test thresholds.

## Validation

- Verify all three CI setup steps select exactly Go 1.27.1.
- Run local strict lint and whitespace checks.
- Require green canonical CI on the resulting release candidate before tagging.
