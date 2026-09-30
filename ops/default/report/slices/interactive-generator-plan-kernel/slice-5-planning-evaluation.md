<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 5: Planning Evaluation

Status: delivered
Delivery set: interactive-generator-plan-kernel
Plan: `ops/default/plan/interactive-generator-plan-kernel.md`
Tracker: `ops/default/tracker/interactive-generator-plan-kernel.md`
Branch: `test/generator-planning-evaluation`
PR: `#26` (merged)

## Purpose

Prove deterministic convergence and failure behavior across the planning
kernel and define a provider-neutral interpreter boundary without claiming
that any production model has been evaluated.

## Delivered Behavior

The new versioned corpus under `generator/eval/testdata/corpus/v1/` covers
semantically equivalent admitted intents, ambiguity, invalid schema content,
unsupported capabilities, incompatible project state, non-Hatmax requests,
and stale plans. Expected statuses, diagnostic codes, clarification fields,
equivalence groups, and drift cases are explicit data rather than test prose.

The `generator/eval` package exposes a provider-neutral `Interpreter`
interface. An interpreter receives one natural-language prompt plus bounded
project and Book projections; it can return only a typed intent, focused
clarifications, or structured rejection diagnostics. It cannot return a plan,
file edit, command, or executable text. Typed output is independently passed
through intent validation and deterministic plan expansion.

Natural-language corpus cases run through a fixture interpreter only. The
tests prove request shaping and expected structured kernel behavior, including
that different phrasings mapped to equivalent intents produce the same plan
digest. They do not measure model quality or validate Codex, Pi, an API model,
or a subscription harness.

End-to-end tests inspect a copied Hatmax project fixture, compute its relevant
fingerprint, evaluate an intent, verify the sealed plan and stable
serialization, confirm evaluation makes no project changes, then modify one
relevant model file and require `HMGEN-PLAN-STALE` with classified drift.

## Implementation Notes

The corpus exposed that equivalent Hatmax version spellings (`0.4.0` and
`v0.4.0`) validated successfully but produced different plan digests. A
supporting fix now canonicalizes Hatmax versions at the intent boundary before
semantic validation and planning.

Evaluation context deliberately omits repository roots, arbitrary file
content, commands, and unselected Book implementation detail. The Book
projection contains supported archetypes, operations, capabilities,
prerequisites, and unsupported variants required for interpretation.

## Contracts Added or Changed

The public evaluation contract adds request, bounded project and Book context,
interpretation, interpreter, evaluation context, result, and error types.
`Evaluate` is the provider-neutral handoff into deterministic validation and
planning. No production interpreter adapter is included in this delivery set.

## Files of Interest

- `generator/eval/types.go`
- `generator/eval/request.go`
- `generator/eval/evaluate.go`
- `generator/eval/testdata/corpus/v1/`
- `generator/eval/kernel_test.go`
- `generator/intent/decode.go`
- `generator/intent/validate.go`

## Validation

- `go test ./generator/...` passed.
- `go test -race ./generator/eval/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.
- Evaluation package tests report 88.5% statement coverage.
- After merge, the exact integrated `dev` candidate `91d3a66` passed the full
  `make check` gate, including Docker-backed PostgreSQL integration tests, 85.2%
  statement coverage against the 80% threshold, vet, formatting, and strict
  lint.

## Risks and Follow-ups

No production model, provider prompt, response parser, API credential flow, or
subscription harness has been evaluated. Those belong to a later interactive
product delivery.
