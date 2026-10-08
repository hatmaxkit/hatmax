---
id: TKT-20261008135600
title: Keep SQLC query rows compatible after required timestamp generation
status: in_progress
kind: bug
severity: unclassified
priority: unclassified
scope: persistence
tags: generator, sqlc, timestamp
source: implementation
reported_at: 2026-10-08T13:56:00Z
ready_at: 2026-10-08T15:36:37Z
started_at: 2026-10-08T15:36:37Z
branch: dev
commits: 2d8f46019c7fed059c5909be6ff5f14a875c403e
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

Slice 6 real commands created an invoice feature in the canonical existing
project fixture. The published required `issued_at` timestamp request then
passed rendering, conformance, SQLC generation and formatting, but `make check`
failed compilation. `dal.ListInvoicesRow` and `dal.GetInvoiceRow` cannot be
passed to the store mapper accepting `dal.Invoice`. This occurred with SQLC
1.30.0 and 1.31.1. The command returns execution failure and retains changes.

The fixture uses the current toolkit through an explicit module replacement;
the separate bare scaffold check uses its published dependency unchanged.
The timestamp examples and full assisted-generation journey remain blocked.
Documentation work does not authorize this runtime correction.

## Expected Outcome

Required timestamp generation produces compatible query and store row types
and completes real generated-project checks, including persistence tests.

## Validation

Use a canonical existing project, real SQLC, Go and owned PostgreSQL. Create
invoice, add the published required timestamp, compile and test the resulting
project, and verify stored timestamp values. Preserve failure/retained-change
reporting when a project command fails.
