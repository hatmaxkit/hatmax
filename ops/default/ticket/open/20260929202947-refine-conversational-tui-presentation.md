---
id: TKT-20260929202947
title: Refine conversational TUI presentation
status: open
kind: enhancement
severity: low
priority: normal
scope: ui
tags: generator, tui, conversation, presentation
source: manual_test
reported_at: 2026-09-29T20:29:47Z
commits:
---

## Observed Behavior

A full-terminal manual test of the conversational generator showed that the
TUI leaves much of the available viewport unused. Conversation input is
rendered as several lines prefixed with `>`, which makes the composer look
like a marked text area instead of a chat input. Separate permanent lines for
command reminders and execution status add another visual band below it.

The progress indicator uses a saturated fuchsia accent that is disconnected
from Hatmax's visual identity and dominates an otherwise restrained terminal
palette.

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
  activity accents. The exact saturated logo color is a reference, not a
  mandatory literal value.
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
  Hatmax identity.
- Request a generated application and verify that the approval view first
  communicates the concrete result, while the complete typed plan and Book
  rules remain inspectable on demand.
- Trigger a diagnostic and verify that its human explanation precedes the
  machine identifier.
