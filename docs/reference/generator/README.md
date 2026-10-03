<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Generator

`hm` is the conversational Hatmax application builder. With no arguments it
opens the recommended terminal UI, which keeps planning, approval, execution,
and follow-up work in one conversation. The supported `hm generate` headless
CLI exposes the same Hatmax-owned planning and execution kernel for automation
and focused terminal work.

Codex interprets natural language into a bounded schema. It cannot inspect the
project, call tools, select files, approve a plan, or perform a mutation.
Hatmax owns project inspection, Book selection, planning, approval, rendering,
conformance, and validation.

## Installation Requirements

Install the canonical command:

```sh
go install hatmax.adrianpk.com/cmd/hm@latest
```

The command also requires:

- a compatible, authenticated `codex` executable on `PATH`; installation and
  authentication are covered by the
  [official Codex CLI documentation](https://learn.chatgpt.com/docs/codex/cli);
- project-owned generation and validation tools required by the discovered
  commands; a Postgres feature normally requires `sqlc`, and project linting
  may require `golangci-lint`.

Hatmax checks that the Codex CLI and resident App Server have compatible
versions before inference. It reports a backend compatibility diagnostic
instead of restarting an incompatible resident process.

## Command Surface

| Command | Behavior |
| --- | --- |
| `hm` | Resume the active conversation for the current scope in the TUI. |
| `hm generate "<request>"` | Run one headless request with terminal clarification and approval. |
| `hm conversation new` | Reset to a new conversation for the current scope. |
| `hm conversation list` | List retained conversations for the current scope. |
| `hm conversation resume <conversation-id>` | Select a retained conversation as active. |

Run `hm` from a parent directory to create an application in a normalized
child directory. Run it from a compatible Hatmax application root to evolve
that application. A pre-project conversation follows the created application
after successful bootstrap.

The headless generator accepts exactly one non-empty request argument:

```sh
hm generate "Create an invoice feature with a required number."
```

Any other `generate` shape prints usage and exits with status `2`.

## TUI Controls

| Key | Action |
| --- | --- |
| `Enter` | Send the composer contents. |
| `Ctrl+J` | Insert a newline. |
| `Ctrl+A` | Approve the currently displayed plan. |
| `Ctrl+D` | Toggle the typed plan and technical details when available. |
| `Esc` | Cancel in-flight work, cancel the pending proposal, or clear the composer. |
| `Ctrl+N` | Start a new conversation for the same scope. |
| `F1` or `Ctrl+H` | Toggle the complete key help. |
| `Ctrl+C` | Cancel in-flight work and exit. |

Approval is available only for the exact pending plan digest. Ordinary
conversation, questions, and informal replies never create an implicit
operation.

The approval view presents a human summary of the concrete application or
feature change by default. Use `Ctrl+D` to inspect the complete typed plan,
Book evidence, and machine diagnostic identifiers. Conversation results keep
validation failures concise; raw command output and stack traces are not
rendered into the chat.

Validation commands continuously drain stdout and stderr while retaining at
most 8 KiB of output per command. Longer output keeps its beginning and end,
with an `[output truncated]` marker included in that budget. The middle is
discarded during capture, not after command completion. Exit status and
cancellation remain unchanged; staging test infrastructure classification
also inspects discarded output. Successful staging commands retain no output.

The compact footer omits conventional send and quit reminders. It shows only
contextual actions, active work phases, and the `F1` help entrypoint. Multiple
clarification questions appear as one numbered Hatmax message and accept one
combined response.

## Development Playground

From a Hatmax source checkout, create a fresh isolated scope, compile the
current `hm`, prepare its local generation tools, and open the TUI with:

```sh
make generator-playground
```

Each invocation creates a new timestamped directory under
`~/Projects/playground/hatmax`; it never resumes an earlier test conversation.

## Supported Operations

| Operation | Effect |
| --- | --- |
| `create_application` | Creates a canonical compiling Hatmax application in a child directory, optionally with initial features. |
| `create_feature` | Creates a canonical server-rendered CRUD feature. |
| `add_field` | Adds one field across an existing feature's required surfaces. |
| `add_validation` | Adds a Hatmax-owned validation rule and its required coverage. |
| `document_feature` | Documents inspected existing behavior or an admitted planned change. |

Application creation requires a display name and Go module path. Hatmax
normalizes the project slug and directory, can use compatible remote evidence
for the module path, and asks only for required information it cannot derive.
Description, niche, and initial features are optional. Initial features appear
as dependent units in the same visible application plan.

Feature operations use the server-rendered CRUD archetype and Book-owned
capabilities for Postgres persistence, Hatmax validation, and HTMX forms.
Requests for alternate databases, ORMs, routers, validation frameworks,
client-side application state, or non-Hatmax application generation are
unsupported.

Documentation changes require explicit documentation intent. A
documentation-only request may affect only documentation. A combined request
may document the admitted implementation. Generated pages use the selected
Diataxis quadrant and preserve text outside Hatmax-managed sections.

### Scaffold Dependency Baseline

New applications require Go 1.27.1 and the Book-selected published Hatmax
v0.5.0. The scaffold explicitly selects chi v5.3.2, pgx v5.11.0, and
x/text v0.42.0 and seeds their archive and module checksums. Module validation
runs `go mod tidy` to resolve the complete dependency graph and checksums.
The pgx and x/text requirements override older versions required transitively
by Hatmax v0.5.0; they do not replace Hatmax or select unpublished code.
Existing applications are not rewritten automatically.

`make generator-scaffold-acceptance` validates a standalone application using
published modules and checks the selected versions without local replacements.
`make generator-scaffold-security` additionally requires `govulncheck` on
`PATH` and scans that generated application, not the Hatmax checkout. The scan
reports reachable vulnerable symbols separately from advisories in unused
modules; a passing scan is not a claim that every dependency is advisory-free.

## Interaction Contract

For a project-changing request, Hatmax:

1. Resolves the current pre-project or project scope and inspects source truth.
2. Gives Codex only bounded Hatmax context and asks for a schema-constrained
   interpretation.
3. Collects focused clarifications when an irreducible product decision is
   missing.
4. Expands the typed intent through the compatible Hatmax Book.
5. Displays the sealed, digest-bound plan without changing the project.
6. Recomputes the plan after explicit approval and rejects drift.
7. Applies only the effects authorized by the plan.
8. Checks Hatmax conformance and runs applicable project validation commands.
9. Retains the result, diagnostics, and safe next action in the conversation.

Rejecting a headless approval or cancelling a TUI proposal leaves the project
unchanged. Stored plans and conversation history are never reusable approval.

## Conversation State and Privacy

Conversation snapshots are user-local state, not project files:

- macOS: `~/Library/Application Support/Hatmax/State`;
- Windows: `%LOCALAPPDATA%\Hatmax\State`;
- other systems: `$XDG_STATE_HOME/hatmax` when the variable is absolute,
  otherwise `~/.local/state/hatmax`.

Hatmax stores bounded turns, proposal summaries, diagnostics, Book identity,
and an optional backend thread reference. It does not store credentials,
environment variables, arbitrary repository contents, hidden reasoning, raw
App Server streams, or approval authority. If persistence fails, the current
TUI session may continue in memory and reports that recovery boundary.

## Codex Runtime

Hatmax locates `codex` on `PATH`, checks the CLI and managed App Server
versions, and reuses a compatible resident daemon. It starts the daemon only
when none is available and never stops or restarts it. Each Hatmax process
opens a short-lived proxy. Backend threads are isolated by Hatmax scope and
cleared when a pre-project conversation is rebound to a created project.

## Outcomes and Validation

The TUI distinguishes conversation, clarification, unsupported requests,
plan-ready work, cancellation, stale plans, execution failure, incomplete
validation, and completion. A compiling scaffold may complete with validation
incomplete only when a declared external test prerequisite is unavailable.
An observed generated-code test failure remains an execution failure. Hatmax
never reports a failed check as passed.

Headless exit statuses are stable:

| Status | Meaning |
| ---: | --- |
| `0` | Generation completed. |
| `1` | Setup failed, an unclassified failure occurred, or mutation completed with incomplete validation. |
| `2` | Invalid command usage. |
| `3` | Clarification or approval was cancelled. |
| `4` | Required product decisions remain unresolved. |
| `5` | The request is outside the Hatmax Book. |
| `6` | The typed intent failed deterministic validation. |
| `7` | Project drift invalidated the approved plan. |
| `8` | Rendering, mutation, conformance, or project validation failed. |

## Command Migration

`hm` is the canonical command. `hatmax` and `hatmax generate` remain
behaviorally equivalent compatibility aliases for the first two tagged minor
releases that contain `hm`. The alias becomes eligible for removal only in a
later minor release after the immediately preceding release notes announce
the removal.

For a guided first session, see
[Assisted Generation](../../tutorials/user-guide/assisted-generation.md).
