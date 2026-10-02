<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Startup Rollback Identity

Status: delivered
Ticket: [TKT-20260930211807](../ticket/solved/20260930211807-pair-startup-rollback-with-component-identity.md)
Branch: `fix/ticket-20260930211807-startup-rollback`
PR: [#75](https://forge.adrianpk.com/hatmax/hatmax/pulls/75)
Implementation: `da6fa8463a9ba64f4a90943596f4996724b99980`
Integrated into `dev`: `7520c38c0b9e640ffa851cd0b6129085fd426b50`

## Delivered Behavior

- Setup pairs each component's start function with its own optional stop function in a StartupStep. Independent capability counts and component order are preserved.
- Startup failure stops only completed steps with a stop capability, in reverse order. It never stops the failed component, a later component, or a stop-only component.
- Rollback uses an independent background context, continues after stop errors, logs those errors, and returns the exact original startup error. Routes are registered only after all starts succeed.
- Normal shutdown still stops every Stoppable component in reverse component order, including stop-only components. No startup bookkeeping changes that list.

## Contracts and Ownership

The first Setup return value and the startup argument of Start now use `[]app.StartupStep`. Manually assembled function slices must be converted to steps with a non-nil Start and their own optional Stop. Setup-based calls keep their existing source form. Start retains the stop-list argument but no longer uses it for rollback; the separate stop list belongs to Shutdown.

Retaining that call form also keeps generated source compatible with the current published Hatmax dependency and with this working tree. No generator version pin, rendering recipe, template compatibility layer, or release metadata was changed. Both published-dependency scaffold acceptance and local-module generated-project acceptance passed.

Pairing belongs to Setup and rollback execution belongs to Start. A failing Start still owns cleanup of its own partial initialization; a start-only component has no rollback operation. This is not a transaction or an all-or-nothing guarantee.

Setup, startup, and rollback remain linear in the supplied component count. Storage remains linear; no hidden registry, shared state, goroutines, dependencies, retries, or new shutdown timeouts were introduced. Serve and normal Shutdown behavior are unchanged. The reference, explanation, User Guide, package note, and Unreleased describe the contract and manual-list migration.

## Validation

All Go commands used `GOTOOLCHAIN=go1.26.7`. Database-dependent checks used an isolated native PostgreSQL 18.6 cluster on loopback, stopped after validation. These are local results, not PostgreSQL 16 CI evidence.

- Before the fix, `go test ./app -run '^TestStartOnlyRollback$' -count=1` failed because the failed component was stopped instead of the start-only component.
- `go test -race ./app -count=50 -timeout=60s`: passed.
- `go test -race ./app -run '^(TestStartOnlyRollback|TestMixedRollback|TestRollbackWithoutStops|TestMixedShutdown)$' -count=50 -timeout=60s`: passed all 50 repetitions.
- `make check`: passed, including source licensing, formatting, vet, all tests, coverage, and strict lint. Total coverage: 80.2%; `app`: 86.7%.
- `make docs-check`: passed.
- `make generator-scaffold-acceptance generator-project-acceptance`: passed.
- `git diff --check`: passed.

Named mixed-capability cases fail at each startup index and assert exact start, stop, and route-registration order. Stop errors are logged without replacing the original error or interrupting remaining rollback. Cancellation at startup failure does not cancel cleanup. Separate regressions cover a missing stop list and normal reverse shutdown with stop-only components. Existing manual-step tests preserve the direct Start contract.

## Boundary

Only finding F8 is addressed. Startup or stop panics, partial cleanup inside a failing component, HTTP lifecycle coordination, shutdown deadlines, dependency graph inference, and other architecture-review findings are unchanged.
