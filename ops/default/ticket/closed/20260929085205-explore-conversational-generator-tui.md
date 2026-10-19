---
id: TKT-20260929085205
title: Explore a constrained conversational TUI for Hatmax generation
status: closed
kind: research
severity: unclassified
priority: unclassified
scope: ui
tags: generator, tui, product-boundary
source: chat
reported_at: 2026-09-29T08:52:05Z
commits: 341857afdefc00ee174487b963b7097d2739903a, 99e683a93889515fea6a38c1c34cb1b0357f6f31
resolution: answered
closed_at: 2026-10-19T16:56:37Z
branch: chore/ticket-20260929085205-close-tui-exploration
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/133
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->


## Observed Behavior

Hatmax currently exposes interactive generation through a line-oriented CLI.
That surface validates the generator engine and supports headless operation,
but it is not the intended primary product experience.

The desired direction is a simple conversational TUI centered on Hatmax
generation. It may support informal dialogue and harmless general responses,
but it must not become a general coding harness. Conversation must never grant
implicit authority to generate code, dependencies, or project structures
outside Hatmax's canonical domain.

## Expected Outcome

Run a bounded product-discovery quiz and establish an agreed interaction model
for the TUI. Define how open conversation coexists with closed-world Hatmax
mutations, how a conversational request becomes a typed Hatmax intent, and how
the user inspects and controls planning and execution.

The exploration must produce enough alignment to decide whether the result
should be promoted to an umbrella specification and later implementation
work. It must not start TUI implementation.

## Exploration Questions

- What is the smallest useful conversational generator loop?
- How does the interface distinguish dialogue from a proposed project change?
- When does a request become a typed Hatmax intent that may produce a plan?
- How are clarifications, plans, diffs, approvals, execution, and validation
  presented without turning the product into a general agent console?
- What project and conversation context persists, and how is it isolated?
- How can the product remain informal and personable without allowing
  off-domain conversation to influence generated code?
- How should unsupported, ambiguous, cancelled, stale, partially applied, and
  failed operations appear to the user?
- Which headless CLI capabilities remain as automation and test surfaces after
  the TUI becomes primary?

## Non-Goals

- Implement the TUI or select its UI framework.
- Define the final screen layout or component hierarchy.
- Generalize Hatmax into an unrestricted coding harness.
- Treat conversational context as implicit approval for project mutation.

## Validation

- Complete a bounded quiz covering the exploration questions.
- Record an agreed product boundary: conversation may be open, but mutations
  are exclusively Hatmax-owned and Book-constrained.
- Describe the primary interaction states and transitions without prescribing
  implementation details.
- Identify unresolved decisions and the evidence needed to resolve them.
- Promote the result to a specification or implementation work only after
  explicit maintainer approval.

## Resolution

The exploration was answered by the curated
[application-generation quiz](../../quiz/00001-hatmax-conversational-application-generation/artifact.md)
and promoted to the approved
[conversational surface specification](../../spec/conversational-hatmax-surface.md).
The [builder tracker](../../tracker/conversational-hatmax-builder.md) records
all eight slices delivered, including the TUI in PR #62 and integrated
acceptance in PR #63. This closes the discovery ticket administratively;
there is no additional runtime change.
