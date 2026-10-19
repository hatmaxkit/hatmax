<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Generated Project Dependencies

Status: delivered
Ticket: [TKT-20260929200201](../ticket/solved/20260929200201-own-generated-project-dependencies.md)
Branch: `fix/ticket-20260929200201-project-dependencies`
PR: #134
Integrated into dev: `931220c74a8338142c1409d40eb2e2d823c2ef6b`
Implementation: `28e5160f5f6175cb84d9adf99f665953b3eedf9f`
Validated candidate: `931220c74a8338142c1409d40eb2e2d823c2ef6b`

## Changes

Application rendering declares every selected Book tool dependency through a
Go module tool directive. The execution manifest runs `go mod tidy` before
invoking generation tools, then reconciles generated imports before build and
test validation. SQLC runs through `go tool` for applications with initial
PostgreSQL features, including their generated Makefile and reopened project
commands. Existing native-tool commands remain supported.

The application Book declares `internal/dal/doc.go`, which documents the
query-adapter package before SQLC creates its generated files. This lets the
first module preparation resolve local imports without ignoring errors or
rendering undeclared files. Bare applications do not acquire an unselected tool.

## Validation

The acceptance regression first failed because the global SQLC executable was
absent. After the fix, it publishes an application with an initial PostgreSQL
feature, runs its module-owned SQLC and readonly tests, and verifies reopened
project command discovery. Database checks use an isolated native PostgreSQL
instance through the existing `DB_*` configuration. No external database is
used.

- `GOWORK=off go test ./generator/... ./internal/hatmaxcli`: passed.
- `GOWORK=off go test -tags=acceptance ./generator/execute -run '^Test(ApplicationScaffoldAcceptance|ScaffoldOwnedTools|ScaffoldBaseline|PrepareApplicationCoversExactPlannedScaffold)$' -count=1 -timeout 10m`: passed.
- `GOWORK=off go vet ./generator/... ./internal/hatmaxcli`: passed.
- `GOWORK=off make lint-strict`: passed.
- `GOWORK=off make docs-check`: passed.
- `git diff --check`: passed.

## Integrated Validation

The validated candidate includes PRs #132, #133 and #134. The affected-package
gate passed against that exact integrated commit with an isolated native
PostgreSQL instance. The scaffold acceptance excludes global SQLC and verifies
the application module's own tool, build, readonly tests and reopened commands.

- `GOWORK=off go test ./mailer ./generator/... ./internal/hatmaxcli`: passed.
- `GOWORK=off go test -tags=acceptance ./generator/execute -run '^Test(ApplicationScaffoldAcceptance|ScaffoldOwnedTools|ScaffoldBaseline|PrepareApplicationCoversExactPlannedScaffold)$' -count=1 -timeout 10m`: passed.
- `GOWORK=off go vet ./mailer ./generator/... ./internal/hatmaxcli`: passed.
- `GOWORK=off make lint-strict`: passed.

Post-merge closure: `GOWORK=off make docs-check` and `git diff --check` passed.
