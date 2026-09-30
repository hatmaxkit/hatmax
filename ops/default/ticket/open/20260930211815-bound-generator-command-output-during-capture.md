---
id: TKT-20260930211815
title: Bound generator command output during capture
status: open
kind: bug
severity: medium
priority: normal
scope: ops
tags: architecture-review, ops, correctness
source: review
reported_at: 2026-09-30T21:18:15Z
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
