<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Interpreter Contracts

Status: delivered
Delivery set: interactive-generator-product-surface
Plan: `ops/default/plan/interactive-generator-product-surface.md`
Tracker: `ops/default/tracker/interactive-generator-product-surface.md`
Branch: `feat/generator-interpreter-contracts`
PR: `#32` (merged)

## Purpose

Turn the planning kernel's evaluation-only interpreter boundary into a strict,
versioned interactive contract without introducing a production backend or
granting a model authority over planning and execution.

## Delivered Behavior

Interpreter requests now carry a contract version, bounded natural-language
prompt, explicit clarification exchanges, and the existing bounded project and
Book projection. Interpreter results separate the model-controlled typed
interpretation from adapter-controlled, non-secret provenance.

The output boundary exposes a closed JSON Schema for exactly one typed intent,
clarification result, or unsupported result. Strict decoding rejects unknown
fields, trailing JSON, oversized output, unsupported schema versions,
contradictory variants, and executable fields such as plans, paths, commands,
dependencies, edits, or approval decisions.

The evaluation corpus now covers clarified interactions, ambiguous requests,
equivalent paraphrases, attempts to escape the Hatmax Book, requests for tool
use, and malformed model results. All coverage uses a deterministic fixture
interpreter; this slice does not invoke Codex.

## Implementation Notes

`eval.Interpreter` returns an `InterpreterResult` containing an
`Interpretation` and `Provenance`. Backend failures use stable generic codes so
the later Codex adapter can distinguish availability, authentication,
compatibility, lifecycle, timeout, cancellation, output, turn, and protocol
failures without leaking raw backend events.

The output schema is generated from a closed Go-owned description and the
decoder independently repeats structural checks before semantic intent
validation and deterministic planning. JSON tags were added to intent-domain
types so the model boundary uses the same snake-case field names as the YAML
contract.

## Contracts Added or Changed

- `eval.Request` now includes `contract_version` and detached clarification
  exchanges.
- `eval.Interpreter` now returns `eval.InterpreterResult` with bounded
  provenance.
- `eval.Interpretation` now requires an explicit schema version.
- `eval.DecodeInterpretation` is the only strict JSON decoding path for model
  results.
- `eval.InterpretationOutputSchema` defines the versioned backend output
  constraint.
- `eval.BackendError` and `BackendFailureCode` define stable backend failure
  categories.

## Files of Interest

- `generator/eval/types.go`
- `generator/eval/request.go`
- `generator/eval/evaluate.go`
- `generator/eval/decode.go`
- `generator/eval/schema.go`
- `generator/eval/schema_test.go`
- `generator/eval/testdata/corpus/v1/`
- `generator/intent/types.go`

## Validation

- `go test ./generator/eval/... ./generator/intent/... ./generator/plan/...`
  passed.
- `go test -race ./generator/eval/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.

## Risks and Follow-ups

The App Server structured-output implementation has not yet exercised this
schema. Slice 2 supplies the transport and thread runtime; Slice 3 validates
the schema against a real authenticated Codex turn. The schema and Go intent
types are versioned together and must change together when the contract
evolves.
