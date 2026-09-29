# Conversational Hatmax Surface State Model

Status: Draft
Kind: Model companion
Specification: `ops/default/spec/drafts/conversational-hatmax-surface.md`
Discovery: `ops/default/quiz/00001-hatmax-conversational-application-generation/artifact.md`

## Purpose

This model defines the local durable state needed to resume a Hatmax
conversation. It is an operational continuity model, not application source,
an authorization ledger, or a project-definition DSL.

## Ownership and Storage

Hatmax owns the store in the operating system's per-user state location. The
store is outside generated repositories and must not be committed with an
application.

The storage technology and exact path remain implementation decisions. The
logical model and lifecycle below are stable regardless of whether the first
implementation uses files or a local database.

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

The scope key may derive from canonical local identity but must not expose a
repository path in user-visible diagnostics. A pre-project conversation is
rebound to the created project's identity after successful application
creation.

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

Hatmax bounds retained turns, diagnostics, and result summaries. Eviction must
preserve enough terminal metadata to explain the last visible operation.
Retention limits and user-facing cleanup controls remain open product details.

## Privacy and Integrity Rules

- Persist no credentials, tokens, environment variables, or authentication
  payloads.
- Persist no hidden model reasoning or unbounded provider events.
- Persist no arbitrary repository files or diffs as conversation content.
- Treat stored text as untrusted input on reload.
- Version every structured intent and operation record.
- Reject or archive incompatible stored contracts rather than guessing an
  upgrade.
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

## Open Questions

- What storage format best supports bounded local state and migrations?
- Which operating-system state-directory convention supplies the root path?
- What default retention count or duration applies?
- Should reset archive or immediately delete prior user-visible turns?
