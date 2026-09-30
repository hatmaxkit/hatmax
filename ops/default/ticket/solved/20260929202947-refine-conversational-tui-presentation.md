---
id: TKT-20260929202947
title: Refine conversational TUI presentation
status: solved
kind: task
severity: low
priority: normal
scope: ui
tags: generator, tui, conversation, presentation
source: manual_test
reported_at: 2026-09-29T20:29:47Z
ready_at: 2026-09-30T06:27:37Z
started_at: 2026-09-30T06:27:37Z
reviewed_at: 2026-09-30T17:37:41Z
closed_at: 2026-09-30T17:37:41Z
resolution: fixed
branch: dev
commits: 41c3c91c723b, 8428afbccb51
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->


## Observed Behavior

A full-terminal manual test of the conversational generator showed that the
TUI leaves much of the available viewport unused. Conversation input is
rendered as several lines prefixed with `>`, which makes the composer look
like a marked text area instead of a chat input. Separate permanent lines for
command reminders and execution status add another visual band below it.

The progress indicator uses a saturated fuchsia accent that is disconnected
from Hatmax's visual identity and dominates an otherwise restrained terminal
palette. After adopting the Hatmax color family, manual testing also showed
that the bar remains fixed at an arbitrary partial value, so it does not make
ongoing work perceptible or represent meaningful completion.

When interpretation needs multiple missing inputs, each clarification is
currently rendered as a separate consecutive Hatmax turn. The result looks
like two unanswered conversational exchanges instead of one grouped request
that the user can answer in a single response.

The idle footer permanently explains conventional send and quit controls and
adds a redundant state such as `Ready` or `Needs input`. The user and composer
backgrounds also reserve horizontal space only on their right edge, leaving
the two blue surfaces visibly asymmetric in a full-width terminal.

Validation-incomplete outcomes currently place complete command output and Go
stack traces directly in the conversation. This obscures the actual outcome:
the project was generated, but external test infrastructure was unavailable.

Before approval, the TUI presents a large YAML-like representation of Book
rules, required levels, diagnostic identifiers, and governed surfaces. That
representation is useful for inspection and debugging, but it does not tell a
regular Hatmax user clearly and concretely what the generator is about to
build.

## Expected Outcome

Keep the interface sober and recognizably chat-oriented while making effective
use of the terminal viewport:

- Present history as an alternating conversation between the user and Hatmax.
  Differentiate user messages with restrained background, color, or spacing
  rather than prefixing every line with `>`.
- Keep a single `>` as the composer prompt. Let the composer remain visually
  distinct from conversation history and grow with its content.
- Replace separate command-reminder and status rows with one compact contextual
  footer. Show only shortcuts relevant to the current state on one side and a
  brief state such as `Ready`, `Planning`, `Executing`, or `Failed` on the
  other.
- Use a subdued blue from the Hatmax logo's color family for progress and
  activity accents. Render active work as a visibly animated indeterminate
  signal rather than an invented completion percentage. The exact saturated
  logo color is a reference, not a mandatory literal value.
- Group consecutive clarification questions into one numbered Hatmax message
  so their shared response boundary is explicit. Bind one combined user reply
  to every pending clarification for backend interpretation.
- Keep conventional send and quit bindings available without spending
  permanent footer space on them. Put `F1` help on the right, retain `Ctrl+H`
  as an alias, and show states only while work is actually active.
- Give user-message and composer backgrounds equal left and right margins.
- Keep result messages outcome-oriented. State what changed and the shortest
  actionable validation limitation; never render stack traces or raw command
  output into the conversation or expanded technical view.
- Provide one development command that compiles the current checkout, prepares
  a fresh isolated playground with required local tools, and opens the TUI.
- Make the default approval view explain what Hatmax will build: the
  application or feature, fields and behavior, persistence, user-facing
  surfaces, generated layers, tests, and documentation scope when applicable.
- Keep the typed plan, complete Book rules, and machine diagnostic identifiers
  available as details on demand. When a diagnostic must appear in the
  conversation, lead with a human explanation and present its identifier as
  supporting evidence.

Do not turn the first refinement into a panel-heavy dashboard. Visual polish
should preserve the simple terminal-chat metaphor.

## Validation

- Run the TUI in a full-terminal viewport and verify that conversation history
  uses the available width and height without unnecessary reserved space.
- Exchange multiple user and Hatmax messages and verify that their roles remain
  clear without repeated `>` prefixes.
- Enter multiline text and verify that the composer uses one prompt and remains
  distinct from the footer.
- Exercise idle, planning, approval, execution, and failure states and verify
  that shortcuts and status share one contextual footer without ambiguity.
- Verify that progress uses a restrained blue accent consistent with the
  Hatmax identity, moves while work is active, and does not imply a fabricated
  percentage.
- Trigger a request that needs multiple clarifications and verify that Hatmax
  presents one numbered message that can be answered in a single user turn;
  every pending field must reach the interpreter with that explicit reply.
- Verify that the idle footer contains no send, quit, ready, or needs-input
  reminder; `F1` and `Ctrl+H` must both toggle help.
- Verify that user-message and composer backgrounds have equal horizontal
  margins in a full-width terminal.
- Trigger an infrastructure-bound validation result and verify that the chat
  reports applied changes and the unavailable dependency without raw output,
  stack frames, or internal reasoning.
- Run `make generator-playground` twice and verify that each invocation uses a
  newly created scope and the latest locally compiled `hm` executable.
- Request a generated application and verify that the approval view first
  communicates the concrete result, while the complete typed plan and Book
  rules remain inspectable on demand.
- Trigger a diagnostic and verify that its human explanation precedes the
  machine identifier.
