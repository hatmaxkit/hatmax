<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 5: Execution Conformance

Status: delivered
Delivery set: interactive-generator-execution-conformance
Plan: `ops/default/plan/interactive-generator-execution-conformance.md`
Tracker: `ops/default/tracker/interactive-generator-execution-conformance.md`
Branch: `test/generator-execution-conformance`
PR: `#31` (merged)

## Purpose

Independently prove that generated project state follows the selected Hatmax
Book rules, then produce bounded and honest validation evidence.

## Delivered Behavior

`CheckConformance` evaluates the executed project independently of rendering.
It checks cohesive feature ownership, Hatmax-owned model and validation
primitives, handler-to-service boundaries, explicit composition order,
Postgres and SQLC consistency, reversible migrations, Hatmax HTMX helpers,
layered validation, required boundary tests, and documentation scope. Failures
use stable rule-attributed `HMGEN-*` diagnostics.

`ValidateExecution` runs conformance before any repository command. It accepts
only commands that exactly match the current inspected repository policy,
orders generation before formatting and validation, invokes executables
directly without a shell, bounds captured output to 8 KiB, stops at the first
failure, and records the exact result in an `ExecutionReport`.

The execution report includes plan and manifest identities, Book and Hatmax
versions, allowed surfaces, dependency effects, conformance results, command
evidence, repairs, and warnings. It is evidence for one run and does not become
project source of truth.

The end-to-end conformance corpus admits generated `create_feature`,
`add_field`, durable `add_validation`, and client-only `add_validation`
results. Negative cases prove rejection of missing feature ownership,
substituted validation dependencies, hidden wiring, incomplete SQLC queries,
missing boundary tests, raw HTMX attributes, and direct handler persistence.

Existing atomic execution tests remain the evidence for stale-plan rejection,
rollback safety, reapplication conflicts, and unchanged project state after a
failed commit. The complete generator suite runs those tests together with the
new conformance corpus.

## Implementation Notes

Conformance reads only bounded files from the fresh project inventory and does
not modify the project. It verifies plan and manifest digests before evaluating
selected Book rules.

Repository commands are rechecked against the live inventory instead of being
trusted only because they were sealed earlier. Executable paths must be
unqualified repository-owned tool names; no shell expression is evaluated.

## Contracts Added or Changed

The public execution contract adds `ConformanceResult`, `CheckConformance`,
`CommandEvidence`, `ExecutionReport`, and `ValidateExecution`.

## Files of Interest

- `generator/execute/conformance.go`
- `generator/execute/report.go`
- `generator/execute/conformance_test.go`
- `generator/execute/report_test.go`
- `generator/execute/conformance_corpus_test.go`

## Validation

- `go test ./generator/execute/...` passed.
- `go test -race ./generator/execute/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.
- Execution package tests report 81.2% statement coverage.
- `newgrp docker -c 'make check'` passed, including Docker-backed PostgreSQL
  integration tests, 84.4% total statement coverage against the 80% threshold,
  vet, formatting, and strict lint.
- After merge, exact integrated `dev` candidate `b5f5135` passed the execution
  corpus, reapplication, drift, semantic-conflict, and rollback checks with the
  race detector, followed by the complete Docker-backed `make check` gate.

## Risks and Follow-ups

This delivery closes deterministic execution and structural conformance. It
does not add a production model provider, subscription harness, interactive
session, prompt adapter, or API credential flow. Those remain a separate
interactive product delivery.
