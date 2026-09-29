---
id: TKT-20260929200201
title: Prepare generated project dependencies before execution
status: open
kind: bug
severity: medium
priority: normal
scope: infra
tags: generator, dependencies, go-modules
source: manual_test
reported_at: 2026-09-29T20:02:01Z
commits:
---

## Observed Behavior

A manual conversational bootstrap from an empty parent requested a Hatmax
application with one initial Postgres-backed feature. The plan was admitted
and approved, but execution failed because the generated workflow invoked a
tool that existed in the Book dependency model but was not materialized by the
generated project. The target application was rolled back.

SQLC exposed the issue, but the defect is generic: the harness attempted to
execute a selected project dependency before preparing the generated Go
module. The manifest currently orders `sqlc generate` before `go mod tidy`.

## Expected Outcome

Ensure the generated project declares the dependencies selected by its Hatmax
plan. After rendering those declarations, the harness must run `go mod tidy`
before invoking project dependencies. SQLC is the first observed case, but the
ordering and ownership rule applies generically rather than as an SQLC-only
exception.

## Validation

- Generate an application with a selected dependency and no matching global
  executable installed.
- Verify the generated module declares the selected dependency.
- Verify `go mod tidy` runs before commands that require generated-project
  dependencies.
- Verify the dependency command succeeds from the prepared module.
- Verify the generated application compiles and its tests can start.
