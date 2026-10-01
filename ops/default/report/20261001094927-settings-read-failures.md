<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Settings Read Failures

Status: reviewing
Ticket: [TKT-20260930211810](../ticket/reviewing/20260930211810-propagate-settings-persistence-failures.md)
Branch: `fix/ticket-20260930211810-settings-errors`
PR: pending

## Delivered Behavior

- Service getters select registered defaults only when Store.Get reports ErrNotFound, including wrapped forms recognized by errors.Is. An arbitrary error with similar text is not absence.
- All other persistence and context errors propagate unchanged with the getter's zero value. A partial raw value returned with a read error does not become a successful result.
- Stored empty strings remain empty through GetString. GetInt and GetBool parse present values and reject malformed or empty input rather than substituting a default. Stored zero and false values remain explicit overrides.
- Missing unregistered keys and empty defaults retain zero-value behavior. Invalid defaults return parse errors only when the key is absent. Reads do not write fallback values.
- The guide's in-memory adapter implements the absence and context-error contract, with tests that exercise it through the real settings service.

## Contracts and Ownership

Absence belongs to the application-owned Store adapter. It must translate backend-specific not-found errors into settings.ErrNotFound, may wrap that sentinel, and must not misclassify a read failure or a stored empty value. Service owns default selection and typed parsing. Public method signatures remain unchanged; there is no new dependency or PostgreSQL settings adapter.

Existing custom adapters that returned an empty string with nil error, an arbitrary error, or an untranslated backend error for absence must adopt ErrNotFound. Empty strings no longer reset defaults; delete the key to restore its default on the next read. Applications must check getter errors before using their values and choose their own failure policy.

Schema validation, Set, Delete, All, registry behavior, and standalone ParseInt/ParseBool helpers are unchanged. Optional empty values can still pass schema validation, but the typed getters reject stored empty numeric or boolean input. Standalone parsing helpers retain empty-to-zero behavior and do not select service defaults.

The configuration/settings reference, User Guide, package usage, and Unreleased document the read contract and adapter migration. The guide example uses the same contract rather than a string-matched missing-value error.

## Validation

All Go checks used `GOTOOLCHAIN=go1.26.7`. Database-related repository tests used an isolated native PostgreSQL 18.6 cluster on loopback, stopped after verification. Settings failures are exercised through an application-owned in-memory adapter with injected failures, and through the guide's real adapter; this is not a claim of a database settings implementation or PostgreSQL 16 CI validation.

- Before the fix, `go test ./settings -run '^TestServiceStoreError$' -count=1` failed: a store failure returned the registered fallback and nil error.
- `go test -race ./settings ./examples/guide -count=20`: passed all 20 repetitions.
- `make check`: passed licensing, formatting, vet, all tests, coverage, and strict lint. Total coverage: 80.4%; settings: 100.0%.
- `make docs-check`: passed.
- `git diff --check`: passed.

An initial aggregate run failed at lint because temporary compilation storage exhausted its quota. The successful rerun used GOTMPDIR in an ignored task-local build directory. No repository tooling or system configuration change was required.

Named table cases cover absence, wrapped absence, present values, explicit empty and zero values, malformed stored values, valid and invalid defaults, unregistered keys, untranslated absence, wrapped failures, partial read failures, cancellation, deadlines, and default reads without persistence writes. The example adapter is checked for absent, empty, stored, and canceled reads.

## Boundary

Only finding F11 is addressed. Existing scheduler and mailer consumer failure policies remain unchanged. No cache, retries, new persistence adapter, schema-validation redesign, or unrelated architecture correction is added.
