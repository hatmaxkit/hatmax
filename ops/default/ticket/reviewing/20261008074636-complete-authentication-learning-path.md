---
id: TKT-20261008074636
title: Complete the authentication learning path across Diataxis
status: reviewing
kind: task
severity: unclassified
priority: normal
scope: docs
tags: authentication, diataxis, user-guide
source: chat
reported_at: 2026-10-08T07:46:36Z
ready_at: 2026-10-08T07:47:45Z
started_at: 2026-10-08T07:47:45Z
reviewed_at: 2026-10-08T08:00:05Z
branch: dev
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

The authentication reference covers delivered passkey, fallback, factor-change,
recovery and session contracts. The User Guide still describes assertion
completion, established-factor management and browser acceptance as unavailable.
Focused authenticator/recovery procedures and an explanation of proof, enrollment and
recovery are missing from their respective quadrants.

## Expected Outcome

Complete the authentication learning path in all four existing Diataxis
quadrants. Correct outdated tutorial claims, add focused authenticator and
recovery procedures, explain proof and recovery boundaries, and connect the current
reference to these entrypoints. Preserve the repository's README entrypoint
convention and document only implemented public behavior.

## Validation

- `make docs-check`: structure, local links, example compilation and whitespace.
- `make source-license-check`: required notices.
- `make lint-strict`: repository pre-commit requirement.
- Check changed commands, routes and symbols against current source.
- Check that the new pages are linked from their quadrant indexes.

## Validation Results

- `make docs-check`: passed, including example compilation.
- `make source-license-check`: passed.
- `make lint-strict`: passed with zero issues.
- Local links, heading anchors, all four quadrant entrypoints, page reachability,
  fenced blocks and whitespace: passed.
- Changed public symbols, example routes and proof semantics checked against
  current source; no runtime files changed.
