<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 7: Hatmax TUI

Status: delivered
Delivery set: conversational-hatmax-builder
Plan: [Conversational Hatmax Builder Delivery Plan](../../../plan/conversational-hatmax-builder.md)
Tracker: [Conversational Hatmax Builder Tracker](../../../tracker/conversational-hatmax-builder.md)
Branch: `feat/hatmax-tui`
PR: #62

## Purpose

Deliver `hm` as the conversational Hatmax terminal product while preserving
the deterministic generator and conversation kernels as the only mutation
authority shared by TUI and headless operations.

## Delivered Behavior

Running `hm` opens a Bubble Tea v2 interface in the current directory. The
interface owns one scrollable conversation viewport, a multiline composer,
contextual status, progress, keyboard help, resize handling, cancellation, and
alternate-screen restoration. The TUI resumes the active local conversation
and presents prior visible turns without reading model-owned hidden state.

User turns execute through cancellable commands and return typed conversation
messages to the UI. The adapter presents ordinary dialogue, focused
clarification, unsupported and rejected requests, sealed plan YAML and digest,
explicit approval, execution results, stale source, diagnostics, validation
evidence, retained changes, and local persistence failure. It has no alternate
planning, approval, execution, or persistence semantics.

`hm generate` uses the existing line-oriented generator path. `hm conversation
new`, `list`, and `resume` control only user-local conversation state. The
existing `hatmax` executable remains a behaviorally equivalent compatibility
alias for interactive and headless use. Both executable names route headless
generation through the same `hatmaxcli.App`, coordinator factory, result
projection, and stable exit-status mapping.

## Implementation Notes

Bubble Tea v2, Bubbles v2, and Lip Gloss v2 are confined to
`internal/hatmaxtui`. The selected v2.0.0 releases keep Hatmax on the Go 1.24
toolchain line; the module patch requirement advances from Go 1.24.0 to
1.24.2.

The TUI receives a narrow session interface and never imports the local state
implementation. Production assembly creates the resident Codex interpreter,
shared interaction coordinator, versioned state store, and persistent
conversation coordinator in `internal/hatmaxcli`. Preview and approved turns
therefore reuse the same Book, inspection, planning, execution, conformance,
and reporting kernels as headless generation.

The slice adds `make generator-conversation-acceptance` because the approved
plan required that gate but the target was not yet present. It exercises the
conversation coordinator, local state, TUI adapter, and headless adapter as one
stable acceptance boundary.

## Files of Interest

- `cmd/hm/main.go`
- `cmd/hatmax/main.go`
- `internal/hatmaxtui/app.go`
- `internal/hatmaxtui/model.go`
- `internal/hatmaxcli/app.go`
- `internal/hatmaxcli/local.go`
- `generator/conversation/coordinator.go`
- `Makefile`

## Validation

- `go test ./internal/hatmaxtui/... ./internal/hatmaxcli/... ./cmd/hm/... ./cmd/hatmax/...` passed.
- `go test -race ./internal/hatmaxtui/... ./generator/interaction/... ./generator/conversation/...` passed.
- `go test ./generator/... ./internal/hatmaxtui/... ./internal/hatmaxcli/...` passed.
- `make generator-conversation-acceptance` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.

## Risks and Follow-ups

- Slice 8 must exercise complete application creation and evolution through the
  actual terminal command surface, including restart, reset, drift,
  cancellation, failure recovery, and off-domain rejection.
- Slice 8 must validate authenticated resident Codex dialogue, thread reuse,
  scope rebinding, and isolation with exact live evidence.
- Slice 8 must document the command migration, current TUI controls, known
  product boundaries, and the initial status of the conversational builder.
