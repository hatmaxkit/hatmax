<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 3: Codex Interpreter

Status: delivered
Delivery set: interactive-generator-product-surface
Plan: `ops/default/plan/interactive-generator-product-surface.md`
Tracker: `ops/default/tracker/interactive-generator-product-surface.md`
Branch: `feat/generator-codex-interpreter`
PR: `#34` (merged)

## Purpose

Implement `eval.Interpreter` through one schema-constrained App Server turn
while preserving deterministic Hatmax authority over intent validation,
planning, and execution.

## Delivered Behavior

The adapter compiles one bounded `eval.Request` into explicit Hatmax
interpretation instructions, the exact serialized request, and the versioned
output schema. It supplies no target-project path, repository file, command,
tool, implementation plan, or model override. Each turn has a two-minute
default deadline.

The interpreter verifies managed ChatGPT authentication without reading or
returning tokens, derives an opaque account-bound context identity, acquires
the isolated project thread, and starts one structured-output turn. It retains
only bounded notifications and the last completed agent message. The result is
strictly decoded through `eval.DecodeInterpretation` before it crosses the
provider-neutral boundary.

Malformed structured output receives exactly one correction turn on the same
thread. A second malformed result fails without a plan. Cancellation and
timeout interrupt the active turn. Failed, interrupted, oversized, tool-using,
event-flooding, or protocol-invalid turns return stable bounded backend errors.
Clarification answers remain explicit in the current request while the reused
thread remains a discardable context cache.

## Contracts Added or Changed

- `CompileTurn` produces bounded schema-constrained App Server input.
- `Interpreter` implements the existing `eval.Interpreter` interface.
- `InterpreterConfig` supplies the opaque project identity, Hatmax context
  root, optional explicit model, and turn timeout.
- `Client.Failure` exposes only the stable terminal connection category.
- `make generator-live-smoke` runs the opt-in authenticated observation.

## Files of Interest

- `generator/backend/codex/prompt.go`
- `generator/backend/codex/prompt_test.go`
- `generator/backend/codex/interpreter.go`
- `generator/backend/codex/interpreter_test.go`
- `generator/backend/codex/live_smoke_test.go`
- `generator/backend/codex/client.go`
- `Makefile`

## Validation

- `go test ./generator/backend/codex/... ./generator/eval/...` passed.
- `go test -race ./generator/backend/codex/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.
- `make generator-live-smoke` ran and stopped before inference with
  `backend_start_failed`: the installed `mise` Codex binary is not the
  standalone installation required by `codex app-server daemon start`.

## Live Smoke Observation

The current Codex CLI is `0.153.0` and exposes the App Server daemon, proxy,
v2 JSON-RPC, thread, turn, output-schema, and account methods used by the
adapter. The daemon start command nevertheless requires a separately managed
standalone executable at
`~/.codex/packages/standalone/current/codex`. That executable is absent on this
host, so no daemon, proxy, model turn, authentication payload, raw event, or
model output was produced by the live smoke.

Installing another Codex distribution is an external workstation change and
is not performed by this slice. The target remains available to rerun after
that prerequisite is deliberately satisfied.

## Risks and Follow-ups

The deterministic suite proves request compilation, schema correction,
clarification continuation, cancellation, timeout interruption, runtime and
thread reuse, project isolation, authentication failure, protocol
incompatibility, tool rejection, and event bounds with fake process and App
Server boundaries.

It does not prove an authenticated model response on this host. Slice 5 still
requires a passing live smoke against the exact integrated candidate. That
gate remains unresolved until the supported standalone Codex daemon is
available.
