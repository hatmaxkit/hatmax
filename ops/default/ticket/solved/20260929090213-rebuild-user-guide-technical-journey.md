---
id: TKT-20260929090213
title: Rebuild the User Guide as a Hatmax application journey
status: solved
kind: task
severity: unclassified
priority: unclassified
scope: docs
tags: user-guide, diataxis, application-journey
source: chat
reported_at: 2026-09-29T09:02:13Z
ready_at: 2026-09-29T09:41:45Z
started_at: 2026-09-29T09:43:43Z
reviewed_at: 2026-09-29T09:48:26Z
closed_at: 2026-09-29T10:44:35Z
resolution: fixed
branch: dev
commits: 9605605e8f0a, b8fecef2c8dd, 92e7ee648f89, 4dcd0c653e14, c13e44e43c4d, 24bae79, 0500f9d, 949bef8, ea89fa9, ebcfa57, f7317d4, 9d142e7, efe9828, a7a8a4b, 214cf19, e2702dd, ff3072a, b40e6e7, d313be8, 3859877, 588547e, 3460cca, 49e7dec, 6cf4ce8
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->


## Observed Behavior

The current User Guide presents accurate Hatmax material, but its progression
follows verification exercises against repository examples. Readers start and
inspect existing applications, enable components, and run isolated checks.
They do not receive an organic technical journey through the decisions,
layers, and primitives used to compose a Hatmax web application.

The result resembles a fragmented tutorial and package tour. It does not
establish the complete application shape early, develop that shape from
foundation to supporting services, or make wiring the continuous thread that
connects Hatmax primitives.

## Expected Outcome

Replace the current sequence with a front-to-back User Guide for people who
want to understand how a Hatmax application is composed. The guide must follow
the order in which an application is understood and designed without becoming
a cumulative build tutorial.

Reuse verified technical material from the existing guide, reference,
how-to, explanation, and examples. Do not retain the current guide sequence as
the editorial foundation. Cover every public Hatmax primitive either in the
main journey or through a deliberate supporting link.

Keep the future practical Todo application tutorial separate. End the guide
with a short, factual orientation to the experimental generator: it operates
on an existing Hatmax application, currently exposes a CLI, and is expected to
gain a TUI. Do not make generation the guide's organizing concept.

Execution is owned by the proposed delivery plan and tracker:

- `ops/default/plan/user-guide-technical-journey.md`
- `ops/default/tracker/user-guide-technical-journey.md`

## Validation

- The User Guide index presents one clear progressive technical journey.
- Early chapters establish application anatomy and keep wiring visible across
  later capabilities.
- The guide distinguishes progressive guidance from the future Todo tutorial,
  focused how-to procedures, exact reference contracts, and explanations.
- Every public Hatmax package is intentionally covered or linked from the
  journey.
- The final generator chapter stays brief, factual, and explicitly
  experimental.
- Every delivery slice passes `make docs-check` and `git diff --check`.
