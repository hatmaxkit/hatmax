<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# PostgreSQL Connection and Schema Encoding

Status: reviewing
Ticket: [TKT-20260930211816](../ticket/reviewing/20260930211816-encode-postgres-values-and-schema-identifiers.md)
Branch: `fix/ticket-20260930211816-postgres-encoding`
PR: pending
Implementation: pending

## Delivered Behavior

- DatabaseConfig.ConnectionString keeps its PostgreSQL keyword/value format and keyword order. String values are single-quoted; backslashes and apostrophes are escaped. Spaces, quotes, backslashes, explicit empty passwords, and connection-option-looking text remain values rather than extra options. Port remains numeric; driver validation and defaults are unchanged.
- A non-empty Schema becomes one double-quoted SQL identifier inside the escaped search_path connection value. Identifier quoting doubles embedded double quotes. NUL bytes remain present so pgx rejects malformed connection strings rather than silently changing a schema name. Empty Schema still omits search_path.
- Database.ensureSchema creates the configured name through pgx.Identifier.Sanitize. Creation and selection agree for mixed-case, hyphenated, reserved-word, quote-containing, space-containing, comma-containing, and SQL-punctuation names. Creation remains idempotent.
- Configuration and database reference documentation, Godoc, and Unreleased describe the corrected encoding. Existing deployments that relied on unquoted case folding must configure their actual schema name or migrate explicitly; no existing schema is renamed or migrated automatically.

## Contracts and Ownership

Connection serialization remains in config with one private keyword-value helper. SQL statement identifier quoting remains in db using the existing pgx dependency. No public API, dependency, configuration key, startup ordering, pool setting, migration behavior, or connection lifecycle change was introduced.

The keyword encoder follows [PostgreSQL's connection-string rules](https://www.postgresql.org/docs/16/libpq-connect.html#LIBPQ-CONNSTRING-KEYWORD-VALUE). Schema DDL uses [pgx Identifier.Sanitize](https://pkg.go.dev/github.com/jackc/pgx/v5@v5.11.0#Identifier.Sanitize); the local v5.11.0 parser and identifier source were checked. Search-path quoting retains NUL because the identifier sanitizer removes it, whereas the connection parser rejects it.

PostgreSQL identifier length limits, schema privileges, and special search-path handling remain server-owned. Configuration remains a trusted input; the original finding established no remote injection path. Invalid SSL modes still fail driver parsing instead of being interpreted as a valid mode followed by extra connection parameters.

## Validation

All Go checks used Go 1.26.7. Database integration ran against an isolated native PostgreSQL 18.6 cluster on loopback, with task-local data and compilation scratch under ignored `.tmp`. The server was stopped after validation. This is local evidence, not CI against PostgreSQL 16 or acceptance of external database credentials.

- Before the fix, `go test ./config -run '^(TestConnectionValues|TestConnectionHosts|TestConnectionOptions)$' -count=1` failed on unescaped values, explicit empty values, and connection-option reinterpretation.
- Before the fix, `go test ./db -run '^TestSchemaIdentifiers$' -count=1` failed on literal schema names: mixed case was folded, punctuation caused SQL errors, and selection disagreed with the requested name.
- `make check`: passed licensing, formatting, vet, all tests, coverage, and strict lint. Total coverage: 81.5%.
- `make docs-check`: passed.
- `go test -race ./config ./db -count=20`: passed all 20 repetitions.
- `go test ./config -run '^$' -fuzz '^FuzzConnectionValues$' -fuzztime=10s -parallel=2`: passed, with 152,407 executions and no failing input.
- `git diff --check`: passed.

Unit cases cover ordinary values, spaces, tabs/newlines, apostrophes, backslashes, double quotes, combined escaping, Unicode, explicit empty fields, Unix socket paths, IPv6, invalid SSL mode text, and NUL rejection. Fuzzing checks round-trip user, password, database, and nested search_path through the real pgx parser. Database cases ensure names twice, inspect current_schema on two simultaneously leased connections, and verify unqualified table creation in the intended schema. Tests skip a pre-existing reserved-word schema rather than deleting caller-owned objects.

## Boundary

Only F17 is corrected. Authentication policy, TLS defaults, schema migration, startup failure cleanup, test-helper connection serialization, generated dependency versions, and other review findings remain unchanged.
