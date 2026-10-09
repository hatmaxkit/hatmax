<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# CI Release Toolchain

Status: reviewing
Ticket: TKT-20261009085014
Branch: `ci/ticket-20261009085014-pin-release-toolchain`
PR: pending

## Delivered Behavior

Lint, test and build use exactly Go 1.27.1, matching the module and locally
validated release toolchain. Existing lint rules and test thresholds remain.

CI run 22 selected Go 1.27.2 through the floating version selector and failed
lint while reading `internal/goarch` export data. Its failed result does not
authorize tagging; canonical CI must pass on the corrected release candidate.

## Validation

- `make lint-strict`: passed locally with Go 1.27.1.
- Workflow checks verify exactly three `go-version: "1.27.1"` entries and no
  floating Go selector or `check-latest: true`.
- `git diff --check`: passed.
