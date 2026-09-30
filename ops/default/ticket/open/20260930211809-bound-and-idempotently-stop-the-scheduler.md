---
id: TKT-20260930211809
title: Make scheduler shutdown bounded and idempotent
status: open
kind: bug
severity: medium
priority: normal
scope: domain
tags: architecture-review, domain, correctness
source: review
reported_at: 2026-09-30T21:18:09Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Stop closes the same channel on every call, so a second call panics. It ignores its context while waiting. The run loop does not select ctx.Done, and repeated Start calls launch additional loops.

Sources: `scheduler/runner.go:82-121`.

Evidence: TestObservedRepeatedStopPanic reproduced the closed-channel panic. Cancellation and repeated-start behavior are source-confirmed.

Impact: Lifecycle callers cannot safely retry shutdown or enforce a shutdown deadline. Cancellation alone leaves polling active.

## Expected Outcome

Define runner lifecycle states, make repeated shutdown safe, and honor cancellation and shutdown deadlines. Prevent duplicate active loops.

## Validation

Cover Stop before Start, repeated Stop, repeated Start, canceled startup context, and a blocked handler with a short shutdown deadline.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f10).
