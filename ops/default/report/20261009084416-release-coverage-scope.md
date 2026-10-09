<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Release Coverage Scope

Status: delivered
Ticket: TKT-20261009084416
Branch: `fix/ticket-20261009084416-release-coverage-scope`
PR: [#120](https://forge.adrianpk.com/hatmax/hatmax/pulls/120)
Integrated dev: `a15200f935ea894720539edcf654b7ab3186c00d`

## Delivered Behavior

Product coverage excludes repository workflow tools under `scripts/`, alongside
the existing example-application exclusion. The threshold remains 80%.
`make test` still runs every default test, including the documentation tool's
controls. Its executable walkthroughs retain their separate documentation gate.
The repository playbook records the scope and the badge reflects 80.7%.

## Validation

- Release candidate `a2dd0a0bb679` passed all default tests but failed the aggregate
  coverage check at 73.2%. The documentation command contributes 3,293 statements
  with 18.7% unit coverage; the product measures 80.7% in that same profile.
- `make test-coverage-check`: passed at 80.7% with Go 1.27.1 and invocation-owned
  PostgreSQL. The cluster stopped after the check.
- `GOWORK=off go test ./scripts/documentation-conformance`: passed.
- Package-selection checks retain `generator/execute` and reject `scripts/` and
  `examples/` in the coverage package list.
- `make lint-strict`: passed.
- `git diff --check`: passed.

The complete release gate runs against the final integrated candidate.
