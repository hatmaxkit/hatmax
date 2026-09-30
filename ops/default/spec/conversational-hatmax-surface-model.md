<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Conversational Hatmax Surface State Model

Status: Approved
Kind: Model companion
Specification: `ops/default/spec/conversational-hatmax-surface.md`
Discovery: `ops/default/quiz/00001-hatmax-conversational-application-generation/artifact.md`

## Purpose

This model defines the local durable state needed to resume a Hatmax
conversation. It is an operational continuity model, not application source,
an authorization ledger, or a project-definition DSL.

## Ownership and Storage

Hatmax owns the store in the operating system's per-user state location. The
store is outside generated repositories and must not be committed with an
application.

The first implementation stores versioned JSON snapshots under:

- `${HOME}/Library/Application Support/Hatmax/State` on macOS;
- `%LOCALAPPDATA%\\Hatmax\\State` on Windows;
- `${XDG_STATE_HOME}/hatmax` when `XDG_STATE_HOME` is an absolute path on other
  Unix systems;
- `${HOME}/.local/state/hatmax` on other Unix systems when `XDG_STATE_HOME` is
  unavailable.

Failure to resolve a per-user state location disables persistence with an
explicit diagnostic; it never redirects state into the project.

The store layout is:

```text
<state-root>/v1/
└── scopes/<scope-key>/
    ├── index.json
    ├── lock
    └── conversations/<conversation-id>.json
```

`index.json` records the active conversation and bounded archive order. Each
conversation file contains its metadata, turns, and proposed-operation
summaries. Writes use a temporary sibling, file synchronization, and atomic
rename. Files and directories use user-only permissions where the platform
supports them. A process holds one per-scope advisory lock while it has that
conversation open for mutation.

## Project Conversation

A project conversation records:

| Field | Meaning |
| --- | --- |
| `id` | Opaque local conversation identifier |
| `scope_key` | Stable opaque key for one project or pre-project target |
| `scope_kind` | `project` or `pre_project` |
| `book_contract` | Compatible Book and interpretation contract identity |
| `backend_identity` | Non-secret backend identity classification |
| `backend_thread_id` | Optional managed interpreter thread reference |
| `status` | `active`, `reset`, `incompatible`, or `archived` |
| `created_at` | Local creation time |
| `updated_at` | Last meaningful interaction time |

The scope key is the lowercase hexadecimal SHA-256 digest of the scope kind, a
zero byte, and the canonical absolute scope path after platform path
normalization. A project uses its root. A pre-project session initially uses
the parent directory and rebinds to the proposed target as soon as application
identity resolves it. The digest prevents repository paths from appearing in
store directory names or user-visible diagnostics. Moving a project creates a
new scope; the first delivery does not attempt path-independent repository
identity. After successful application creation, the target-scoped
pre-project conversation is rebound to the created project's identity.

At most one conversation is resumed by default for a compatible scope and
Book contract. Resetting a conversation creates a new identity; it does not
delete or modify project source.

## Conversation Turn

A retained turn records:

| Field | Meaning |
| --- | --- |
| `id` | Monotonic identifier within the conversation |
| `role` | `user` or `hatmax` |
| `kind` | Dialogue, goal, clarification, decision, result, or diagnostic |
| `content` | Bounded user-visible text |
| `created_at` | Turn creation time |
| `operation_id` | Optional related operation |

Hatmax retains only user-visible content needed for continuity. It does not
store hidden reasoning, raw provider events, repository file contents,
credentials, or environment snapshots as conversation turns.

Each retained turn contains at most 32 KiB of UTF-8 text. A turn that exceeds
the limit is rejected before it reaches the store or interpreter; retained
turns are never silently truncated.

## Proposed Operation

A proposed operation records:

| Field | Meaning |
| --- | --- |
| `id` | Opaque operation identifier |
| `conversation_id` | Owning conversation |
| `status` | Candidate, clarifying, planned, cancelled, stale, failed, or completed |
| `request_summary` | Bounded user-visible goal |
| `intent_contract` | Intent schema version |
| `intent` | Optional validated typed intent |
| `plan_digest` | Optional sealed plan digest |
| `source_fingerprint` | Optional relevant project or target fingerprint |
| `result_summary` | Optional bounded execution outcome |
| `created_at` | Proposal creation time |
| `updated_at` | Last state transition time |

The store may retain an intent and plan projection for continuity and audit,
but they are never current authority. Hatmax validates their contracts and
recomputes live inventory before offering execution.

## Approval

Approval is deliberately not durable authority. Hatmax may record that a plan
was previously approved as historical result text, but it must not restore an
executable approval after process exit, cancellation, drift, or failure.

Every execution requires approval bound to the currently displayed plan digest
and current source fingerprint.

## Lifecycle

### Resume

On resume, Hatmax loads the compatible conversation, inspects current source or
target state, and classifies every non-terminal proposed operation:

- `planned` becomes `stale` when its fingerprint or contract changed;
- `clarifying` can continue with retained answers;
- `candidate` can be reinterpreted from retained user-visible context;
- terminal results remain history and cannot be executed again directly.

### Reset

Reset marks the current conversation `reset` and starts a new conversation for
the same scope. Reset does not delete the application, modify Git, or erase
execution evidence required for a currently displayed failure.

### Backend replacement

An invalid, missing, or incompatible backend thread can be replaced without
losing Hatmax-owned conversation state. Backend thread history is a cache, not
the durable product record.

### Retention

One conversation retains at most:

- the most recent 200 turns;
- 8 MiB of turn content;
- the most recent 50 proposed-operation summaries;
- 16 KiB in one result summary or aggregate diagnostic.

One scope retains one active and at most nine archived conversations. Pruning
removes the oldest eligible turns, terminal operations, and archived
conversations in that order. It always preserves the active non-terminal
operation, its collected answers, and the latest terminal operation summary.
If those protected records alone exceed a limit, persistence stops and reports
`state_limit_exceeded` rather than discarding required recovery context.

When a backend thread must be replaced, Hatmax replays at most the newest 32
turns and 128 KiB of user-visible conversation plus the active operation's
structured decisions. Older retained history remains locally inspectable but
does not enter model context.

Reset archives the current conversation and creates a new active identity. It
does not immediately delete prior turns; ordinary pruning applies the archive
limit.

### Recovery

- Failure to acquire the per-scope lock returns `state_busy`; a second process
  does not open a concurrent mutable session for that scope.
- An interrupted snapshot write leaves the previous valid snapshot active.
- Invalid or unsupported state is preserved for diagnosis, marked
  incompatible in the index when possible, and never guessed into a newer
  schema. After the diagnostic, the user may explicitly start a new
  conversation without overwriting the incompatible snapshot.
- A failed state write keeps the current in-memory conversation usable for the
  process, shows `state_persistence_failed`, and promises no later resume.
- A missing or invalid backend thread is replaced and receives only the
  bounded replay context. Approval is not replayed.
- Process loss during a proposed operation causes live source inspection on
  restart. The operation becomes stale or failed and is never resumed or
  retried automatically.
- Cancellation interrupts the active backend or execution context, persists a
  user-visible terminal transition when possible, and always clears approval.

## Privacy and Integrity Rules

- Persist no credentials, tokens, environment variables, or authentication
  payloads.
- Persist no hidden model reasoning or unbounded provider events.
- Persist no arbitrary repository files or diffs as conversation content.
- Treat stored text as untrusted input on reload.
- Version every structured intent and operation record.
- Reject or archive incompatible stored contracts rather than guessing an
  upgrade.
- Require an explicit, tested state migrator before accepting a future schema;
  version 1 has no implicit migration path.
- Losing or deleting the store never changes application behavior.

## Required Queries

The persistence implementation must support:

- find the default compatible conversation for a scope;
- list recent conversations for explicit selection;
- load bounded recent turns in stable order;
- load non-terminal proposed operations;
- mark a plan stale;
- append a user-visible turn or operation transition atomically;
- reset the active conversation;
- prune state according to the retention policy.

## Acceptance Criteria

- Conversation resumes without making stored state authoritative over source.
- Project and pre-project conversations remain isolated.
- Successful scaffold creation can rebind pre-project context to the project.
- Plans invalidated by drift cannot regain approval from storage.
- Backend thread loss does not destroy Hatmax-owned continuity.
- Reset and pruning do not modify the target project.
- Stored state contains no credentials, hidden reasoning, or repository file
  snapshots.
- Concurrent processes cannot mutate one scope's state simultaneously.
- Interrupted writes retain the previous valid snapshot.
