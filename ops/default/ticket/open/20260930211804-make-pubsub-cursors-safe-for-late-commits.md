---
id: TKT-20260930211804
title: Prevent pubsub loss across out-of-order commits
status: open
kind: bug
severity: high
priority: high
scope: persistence
tags: architecture-review, persistence, hardening
source: review
reported_at: 2026-09-30T21:18:04Z
commits:
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
