---
id: TKT-20261008135846
title: Generate validation model tests that satisfy project lint
status: solved
kind: bug
severity: unclassified
priority: unclassified
scope: api
tags: generator, validation, lint
source: implementation
reported_at: 2026-10-08T13:58:46Z
ready_at: 2026-10-08T15:36:37Z
started_at: 2026-10-08T15:41:32Z
branch: dev
reviewed_at: 2026-10-08T15:42:18Z
closed_at: 2026-10-08T15:42:18Z
resolution: fixed
commits: 2d8f46019c7fed059c5909be6ff5f14a875c403e
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

Slice 6 created invoice in the canonical existing-project fixture with real
commands, then added a durable minimum-length rule of three characters to its
required number field. Rendering, conformance, SQLC, compilation and Go tests
passed. The generated `internal/feat/invoice/model_test.go` fails the project's
`wsl_v5` lint at `_, err := NewInvoice(input)`: missing whitespace after multiple
input assignments. `make check` fails and the operation retains its changes.

This affects the declared validation evolution capability. The documentation
records the failure; its delivery set does not authorize a renderer correction.

## Expected Outcome

Generated validation tests satisfy the project's formatting and lint contracts
and the approved operation completes real project checks.

## Validation

Use real Go, SQLC and project lint in a canonical existing project. Create
invoice, add the durable minimum-length number rule, then run full generated
project checks and validate rejection of invalid number values.

## Resolution

Separate the prepared input assignments from constructor validation in the
generated test. The domain rejection and generated assertion remain unchanged.

`TestValidationRule` reproduced the generated `wsl_v5` failure with real tools
before the correction. Afterward, native SQLC generation, formatting, build,
generated model rejection tests, PostgreSQL tests and project lint passed.
The invocation-owned PostgreSQL cluster stopped successfully.

Successful checks:

- `GOWORK=off go test -tags=acceptance -run '^TestValidationRule$' -count=1 -timeout=4m ./generator/execute` under the owned data fixture.
- `GOWORK=off go test -run '^TestRenderAddValidation' -count=1 ./generator/execute`
- `make lint-strict`
