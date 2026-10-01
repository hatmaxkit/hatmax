---
id: TKT-20260930211815
title: Bound generator command output during capture
status: reviewing
kind: bug
severity: medium
priority: normal
scope: ops
tags: architecture-review, ops, correctness
source: review
reported_at: 2026-09-30T21:18:15Z
ready_at: 2026-10-01T12:42:32Z
started_at: 2026-10-01T12:42:32Z
reviewed_at: 2026-10-01T12:56:38Z
branch: fix/ticket-20260930211815-command-capture
pr: pending
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Repository validation captures stdout and stderr in an unbounded bytes.Buffer or CombinedOutput. The evidence limit is applied only after the command exits; application staging retains full output.

Sources: `generator/execute/report.go:146-183, generator/execute/application_workspace.go:409-430`.

Evidence: Source-confirmed buffer ownership and post-execution truncation; no unbounded-output process was launched.

Impact: A noisy or failing tool can consume unbounded process memory and produce oversized diagnostic state even though the final evidence helper looks bounded.

## Expected Outcome

Use bounded output capture while continuously draining the child process. Preserve a useful failure summary and truncation marker across both execution paths.

## Validation

Run a bounded high-output child process and verify the capture limit, completion without pipe deadlock, nonzero exit evidence, truncation marker, and cancellation.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f16).

## Delivery

Both execution paths now drain stdout and stderr into the same bounded capture. Output retains at most 8 KiB per command, including a truncation marker between the beginning and end. Staging infrastructure classification inspects bounded stream windows, including discarded bytes; its existing marker policy remains unchanged.

`make check`, `make docs-check`, and 20 focused race-enabled repetitions passed. The subprocess fixture emits a finite 2 MiB across stdout and stderr, covering success, nonzero exit, unavailable test infrastructure, and cancellation after overflow.

Report: [Bounded Command Capture](../../report/20261001125505-command-capture.md).
