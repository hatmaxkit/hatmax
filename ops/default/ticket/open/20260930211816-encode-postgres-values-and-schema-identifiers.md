---
id: TKT-20260930211816
title: Encode PostgreSQL connection values and schema identifiers
status: open
kind: bug
severity: medium
priority: normal
scope: persistence
tags: architecture-review, persistence, correctness
source: review
reported_at: 2026-09-30T21:18:16Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

ConnectionString interpolates keyword values without quoting. A password containing a space does not round-trip through pgx.ParseConfig. ensureSchema interpolates a schema name into SQL without identifier quoting.

Sources: `config/config.go:375-384, db/database.go:82-95`.

Evidence: TestObservedDSNEncoding reproduced the space-containing password failure. Schema identifier interpolation is source-confirmed.

Impact: Valid credentials or schema names can fail or be interpreted as connection syntax or SQL. Configuration is a trusted input here; no remote injection path was established.

## Expected Outcome

Use a PostgreSQL-aware connection encoder and quote schema identifiers with the driver's identifier support. Preserve literal configured values rather than treating them as syntax.

## Validation

Round-trip spaces, quotes, backslashes, and empty connection values. Test mixed-case, hyphenated, reserved-word, and quote-containing schema identifiers in an isolated database.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f17).
