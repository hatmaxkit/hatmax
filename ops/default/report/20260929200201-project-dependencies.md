<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Generated Project Dependencies

Status: reviewing
Ticket: [TKT-20260929200201](../ticket/reviewing/20260929200201-own-generated-project-dependencies.md)
Branch: `fix/ticket-20260929200201-project-dependencies`
PR: #134

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
