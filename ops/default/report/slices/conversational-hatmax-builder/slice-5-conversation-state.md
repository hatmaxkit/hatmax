# Slice 5: Conversation State

Status: delivered
Delivery set: conversational-hatmax-builder
Plan: [Conversational Hatmax Builder Delivery Plan](../../../plan/conversational-hatmax-builder.md)
Tracker: [Conversational Hatmax Builder Tracker](../../../tracker/conversational-hatmax-builder.md)
Branch: `feat/conversational-state`
PR: #60

## Purpose

Define and persist bounded Hatmax-owned conversation state without granting
stored text, backend history, typed intent, or sealed plans project authority.

## Delivered Behavior

The interpreter contract now admits `conversation_response` beside intent,
clarification, and unsupported results. A conversational response contains only
bounded user-visible text. It cannot carry paths, commands, dependencies,
edits, tool calls, approvals, or any executable mutation authority. Recent
dialogue supplied to interpretation is limited to 32 turns and 128 KiB, with a
32 KiB limit per turn.

Hatmax models opaque project and pre-project scopes, compatible Book and
interpreter contracts, backend identity and replaceable thread references,
monotonic visible turns, and proposed operations. Proposed operations retain
bounded request, typed intent, plan digest, source fingerprint, and outcome
summaries, but the durable schema deliberately has no approval field.

Conversation lifecycle operations support candidate, clarification, planning,
terminal transitions, drift invalidation, reset, pre-project target rebinding,
project rebinding, backend replay windows, non-terminal operation queries, and
bounded pruning. Reset creates a new identity without touching project source.
Rebinding clears the backend thread and makes any retained plan stale.

The local JSON store resolves the operating system's per-user state location,
uses opaque SHA-256 scope directories, and stores version 1 indexes and
conversation snapshots outside projects. It provides one non-blocking advisory
lock per scope, user-only permissions where supported, synchronized temporary
files, atomic rename replacement, strict decoding, explicit incompatible-state
handling, restart continuity, explicit conversation selection, scope rebinding,
and archive pruning.

## Implementation Notes

The interpretation contract and output schema are version 3. Previous
single-request evaluation entry points remain wrappers around the dialogue-aware
evaluation boundary.

The first durable state schema has no implicit migration. Unsupported or
invalid snapshots remain on disk, are marked incompatible in the scope index
when possible, and require an explicit fresh conversation. Credential-like
content is rejected before persistence, and the persisted types contain no raw
provider events, environment snapshots, repository files, or hidden reasoning.

The store keeps the prior valid snapshot when replacement fails before rename.
An unsuccessful write does not replace the session's durable in-memory view.
Scope locking uses native advisory locks on Unix and Windows; the Windows
implementation is cross-compiled in the slice gate.

## Files of Interest

- `generator/eval/types.go`
- `generator/eval/evaluate.go`
- `generator/eval/schema.go`
- `generator/conversation/types.go`
- `generator/conversation/lifecycle.go`
- `generator/conversation/validate.go`
- `internal/hatmaxstate/root.go`
- `internal/hatmaxstate/store.go`
- `internal/hatmaxstate/codec.go`
- `internal/hatmaxstate/lock_unix.go`
- `internal/hatmaxstate/lock_windows.go`

## Validation

- `go test ./generator/eval/... ./generator/conversation/... ./internal/hatmaxstate/...` passed.
- `go test -race ./generator/conversation/... ./internal/hatmaxstate/...` passed.
- `go test ./generator/...` passed.
- Windows cross-compilation of `internal/hatmaxstate` passed.
- `make vet` passed.
- `make lint-strict` passed.
- `git diff --check` passed.

## Risks and Follow-ups

- Slice 6 must bind this state model and store to the conversational coordinator,
  source reinspection, and resident backend lifecycle.
- The store intentionally does not invoke interpretation, planning, approval,
  or execution; those transitions remain coordinator responsibilities.
- Slice 7 must expose new, list, and resume controls without adding a second
  state authority in the TUI.
