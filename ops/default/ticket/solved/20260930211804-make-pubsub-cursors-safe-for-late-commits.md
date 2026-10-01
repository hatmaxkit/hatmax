---
id: TKT-20260930211804
title: Prevent pubsub loss across out-of-order commits
status: solved
resolution: fixed
kind: bug
severity: high
priority: high
scope: persistence
tags: architecture-review, persistence, hardening
source: review
reported_at: 2026-09-30T21:18:04Z
ready_at: 2026-09-30T23:54:34Z
started_at: 2026-09-30T23:54:34Z
reviewed_at: 2026-10-01T00:09:27Z
closed_at: 2026-10-01T05:49:34Z
branch: fix/ticket-20260930211804-pubsub-commits
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/72
commits: a061bda59b92403791d1f189bd8479bed0797fcc, 4015e895b583de703201ef73cd02d77a32ea1b4e
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

The subscriber selects id > lastOffset and advances to the highest delivered ID. PostgreSQL assigns sequence IDs before commit. A lower-ID transaction committed after a higher ID has been acknowledged becomes permanently invisible to the subscriber.

Sources: `pubsub/postgres/broker.go:296-375, pubsub/postgres/schema.go`.

Evidence: TestObservedPubsubCommitOrder held one insert transaction open, delivered a later insert, advanced the cursor, committed the earlier insert, and observed that it remained undispatched.

Impact: Durable subscribers lose committed messages even when every invoked handler succeeds. Sequence allocation order is not commit order.

## Expected Outcome

Use a durable delivery or acknowledgement strategy that cannot skip late-committed rows. Preserve named-subscriber restart semantics and bounded polling.

## Validation

Add a deterministic two-transaction PostgreSQL regression, then verify restart, batch boundaries, and ordinary fan-out delivery.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f5).

## Implementation

PostgreSQL polling now selects rows without an exact acknowledgement for the
subscriber instead of excluding every ID below a scalar cursor. Late commits
remain eligible across polls and named-subscriber restarts. Registration excludes
only visible topic history in one statement snapshot; legacy offsets are migrated
once, transactionally, without replaying their visible processed history.

Batch acknowledgements and diagnostic progress commit atomically. Batch bounds,
fan-out, startup rollback, and persistence rollback have deterministic database
coverage. Handler-error retry policy remains assigned to the separate F6 ticket.

Delivery: [Pubsub commit-safe delivery](../../report/20261001000927-pubsub-commit-safe-delivery.md).

Merged into `dev` through PR #72 at
`4015e895b583de703201ef73cd02d77a32ea1b4e`.
