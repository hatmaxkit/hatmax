# Slice 6: Conversational Coordinator

Status: delivered
Delivery set: conversational-hatmax-builder
Plan: [Conversational Hatmax Builder Delivery Plan](../../../plan/conversational-hatmax-builder.md)
Tracker: [Conversational Hatmax Builder Tracker](../../../tracker/conversational-hatmax-builder.md)
Branch: `feat/conversational-coordinator`
PR: #61

## Purpose

Coordinate open conversation and closed Hatmax mutations as one resumable,
terminal-independent state machine while preserving live source, the Hatmax
Book, sealed plans, and explicit approval as the only mutation authority.

## Delivered Behavior

The interaction kernel now previews conversational turns without invoking an
approver or changing project source. It distinguishes bounded dialogue,
candidate Hatmax work, focused clarification, unsupported requests, rejected
intent, and an inspectable `plan_ready` outcome. Visible dialogue and collected
clarification decisions are supplied to interpretation, while a later approval
must name the exact plan digest recomputed from current source.

The conversation coordinator opens project or pre-project scope sessions,
persists visible turns and non-authoritative proposal state, resumes focused
clarification, and records validated typed intent and plan identity. Project
resume reinspects the complete Book fingerprint and makes drifted plans stale.
Application planning rebinds the conversation from the session parent to the
derived target, and successful publication rebinds it to the created project
with the project Book contract.

Codex provenance now exposes the managed thread identifier as bounded,
non-secret adapter metadata. The coordinator records a replacement identifier
after any turn, so a missing backend thread does not discard Hatmax-owned
dialogue or structured decisions. New-conversation reset remains independent
of project source.

Cancellation retains the proposal, decisions, plan identity, and visible
history without retaining approval. Invalid approval digests do not invoke
execution or consume the valid proposal, so the current displayed digest can
be retried. Execution failure records diagnostics and every retained target.
Environmental application-test unavailability is exposed as
`validation_incomplete`, distinct from passed validation and failed execution.

If local state replacement fails, the coordinator emits
`HMGEN-STATE-PERSISTENCE-FAILED`, disables further persistence for that process,
and continues with an isolated in-memory snapshot. It does not claim that this
state will resume after process exit.

## Implementation Notes

The existing `Coordinator.Run` headless path remains behaviorally compatible.
Conversational adapters use `PreviewTurn` and `RunApprovedTurn`, which share the
same inspection, validation, planning, freshness, rendering, execution, and
conformance kernels.

Durable proposed operations retain pending clarification questions and explicit
answers, but no approval field. Rebinding may change the compatible Book
contract only while moving from a pre-project scope to its proposed target or
created project; it always clears the replaceable backend thread reference.

The conversation coordinator contains no terminal presentation types. Slice 7
can render its states and actions without acquiring planning, approval, or
execution authority.

## Files of Interest

- `generator/interaction/types.go`
- `generator/interaction/coordinator.go`
- `generator/interaction/execute.go`
- `generator/conversation/coordinator.go`
- `generator/conversation/types.go`
- `generator/conversation/lifecycle.go`
- `generator/conversation/validate.go`
- `generator/eval/types.go`
- `generator/backend/codex/interpreter.go`
- `internal/hatmaxstate/store.go`

## Validation

- `go test ./generator/conversation/... ./generator/interaction/... ./generator/backend/codex/...` passed.
- `go test -race ./generator/conversation/... ./generator/interaction/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.

## Risks and Follow-ups

- Slice 7 must present conversation, clarification, plan inspection, approval,
  progress, cancellation, incomplete validation, retained changes, and local
  persistence failure without duplicating coordinator semantics.
- Slice 7 must wire new, list, resume, and reset controls through this state
  machine and retain headless kernel parity.
- Slice 8 must exercise complete deterministic conversations and authenticated
  resident-backend thread reuse across process restart and scope rebinding.
