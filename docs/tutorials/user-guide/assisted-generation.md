<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Assisted Generation

Hatmax provides two ways to use its application builder:

- `hm` opens the conversational terminal UI. This is the recommended way to
  create and evolve an application.
- `hm generate "<request>"` runs one focused request through the headless CLI.
  It is a supported alternative for terminal workflows and automation.

Both interfaces use the same Hatmax Book, planning, approval, execution, and
validation contracts. The TUI adds a persistent conversation around those
contracts; it does not replace them with a general coding agent.

This chapter follows one TUI conversation from an empty parent directory to a
new application, then shows how to continue working and when to use the
headless CLI.

## Prepare Hatmax

Install `hm` and authenticate the Codex CLI as described in the
[official Codex CLI documentation](https://learn.chatgpt.com/docs/codex/cli):

```sh
go install hatmax.adrianpk.com/cmd/hm@latest
```

The Codex CLI and its resident App Server must have compatible versions.
Hatmax reuses that resident process across conversation turns. It does not
start a separate Codex process for every message.

## Open the TUI

To create an application, start Hatmax from the directory that will contain
the new project:

```sh
cd ~/Projects
hm
```

The TUI opens a conversation for that directory. Hatmax creates no project
until a request becomes an admitted Hatmax operation, you review its plan, and
you approve it.

Starting `hm` from an existing compatible Hatmax project opens that project's
conversation instead. Starting it from an incompatible project still permits
ordinary conversation, but Hatmax rejects project mutations there.

## Talk to Hatmax

Write in the composer and press `Enter` to send. Use `Ctrl+J` when the message
needs a newline.

The conversation does not need to begin with a generation command. You can
ask a question, discuss the application, or provide product context. Ordinary
dialogue produces no files and grants no later operation implicit approval.

When a message describes supported Hatmax work, the conversation moves into a
visible change proposal. Hatmax can create an application, create a canonical
feature, add a field or validation rule, or document admitted Hatmax behavior.
Requests for another application stack or substitutes for Hatmax primitives
are outside the builder's mutation boundary.

## Create an Application

A complete first request can provide the application identity and an initial
feature together:

```text
Create a Hatmax application named Ledger with module path example.com/alex/ledger and an invoice feature with a required number string field and an optional notes text field.
```

A shorter request is also valid:

```text
Create an invoicing application with Hatmax.
```

Hatmax derives what it can from the request and current directory. If a
required product decision is still missing, it asks one focused question in
the conversation. Answer that question in the same composer. Optional details
do not block generation.

PostgreSQL, server-rendered HTML, HTMX interaction, explicit wiring, and the
canonical Hatmax feature shape are part of the Hatmax application model. You
do not need to request them on every turn.

## Review and Approve the Plan

Before changing the target, Hatmax displays a sealed plan. Review the concrete
result: application identity, requested features and fields, affected layers,
validation, tests, and documentation scope. No project file has changed at
this point.

The default view summarizes that result in product terms. Press `Ctrl+D` when
you need the complete typed plan, Book evidence, or machine diagnostic
identifiers; press it again to return to the summary.

Press `Ctrl+A` to approve the plan currently displayed in the TUI. Approval is
bound to that exact plan and the inspected project state; it is not a reusable
permission for later changes.

If the proposal is wrong, press `Esc` to cancel it without changing the
project. Then send a corrected request. If source or target state changes
before approval, Hatmax marks the plan stale and requires a fresh proposal.

## Follow Execution

After approval, Hatmax renders only the authorized Hatmax surfaces and runs
the applicable conformance and project checks. For a new application, it
creates a normalized child directory below the directory where the
conversation started.

The final state distinguishes these outcomes:

- `Completed` means mutation and required checks succeeded.
- `Completed; validation incomplete` means the generated application compiled
  and a declared external test prerequisite prevented a remaining check.
- `Execution failed` means rendering, conformance, compilation, or a project
  check failed.
- `Plan stale` means the approved proposal no longer matches current source or
  target state.
- `Cancelled` means execution or the pending proposal was stopped.

A failed project check is never reported as a successful generation. When an
operation fails, read its diagnostic and retained-change information before
submitting a revised request.

## Continue in the Same Conversation

After successful application creation, Hatmax associates the conversation
with the created project. Continue from the project directory:

```sh
cd ~/Projects/ledger
hm
```

Hatmax resumes the active local conversation and reinspects the project before
planning another change. For example:

```text
Add a required issued-at timestamp to invoice.
```

The new request follows the same cycle: conversation, clarification when
required, visible plan, explicit approval, execution, and validation. Prior
dialogue can preserve product context, but source inspection remains
authoritative.

Generated files are ordinary Go, SQL, templates, and assets. The application
does not require `hm` or Codex at runtime.

## Use the TUI Controls

| Key | Action |
| --- | --- |
| `Enter` | Send the composer contents. |
| `Ctrl+J` | Insert a newline in the composer. |
| `Ctrl+A` | Approve the currently displayed plan. |
| `Ctrl+D` | Toggle technical details when the current result provides them. |
| `Esc` | Cancel current work or the pending proposal; otherwise clear the composer. |
| `Ctrl+N` | Start a new conversation for the current directory or project. |
| `F1` or `Ctrl+H` | Show or hide complete key help. |
| `Ctrl+C` | Cancel active work and exit safely. |

## Resume or Reset Conversation State

Conversation state is local user state outside the application repository.
Opening `hm` in the same scope resumes its active compatible conversation.

Use the conversation commands when you need explicit selection:

```sh
hm conversation list
hm conversation resume <conversation-id>
hm
```

Start over without changing application source with either `Ctrl+N` in the
TUI or:

```sh
hm conversation new
hm
```

Losing or resetting conversation state does not damage the project. Hatmax
reconstructs project facts from source and the compatible Book.

## Use the Headless CLI

Use the headless CLI when one bounded request is more useful than a persistent
conversation:

```sh
hm generate "Add a required issued-at timestamp to invoice."
```

The command interprets one request, prints its plan, and accepts only `y` or
`yes` as approval. It uses the same Hatmax mutation boundary as the TUI. The
CLI remains suitable for focused terminal work, automation, and acceptance
checks; the TUI is the recommended interface for iterative application work.

The previous `hatmax` and `hatmax generate` forms are temporary compatibility
aliases. New usage and scripts should use `hm`.

The [Generator Reference](../../reference/generator/README.md) defines the
complete command surface, supported operations, state locations, runtime
contract, exit statuses, and compatibility policy.

---

[Previous: Testing and Evolution](testing-and-evolution.md) ·
[User Guide](README.md)
