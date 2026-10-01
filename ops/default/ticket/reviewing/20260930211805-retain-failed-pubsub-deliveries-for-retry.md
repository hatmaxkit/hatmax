---
id: TKT-20260930211805
title: Do not acknowledge failed pubsub handlers
status: reviewing
kind: bug
severity: high
priority: high
scope: persistence
tags: architecture-review, persistence, hardening
source: review
reported_at: 2026-09-30T21:18:05Z
ready_at: 2026-10-01T05:52:50Z
started_at: 2026-10-01T05:52:50Z
reviewed_at: 2026-10-01T06:01:01Z
branch: fix/ticket-20260930211805-pubsub-retry
pr: pending
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

A handler error is logged, but lastProcessedID still advances and the durable offset is saved. The failed message is skipped on future polls and restarts. Existing handler-error tests assert one attempt rather than redelivery.

Sources: `pubsub/postgres/broker.go:354-375, docs/reference/pubsub/README.md:10-12`.

Evidence: TestObservedFailedPubsubAck verified a nonzero durable offset after a handler returned an error. TestBrokerHandlerError currently codifies the non-retry behavior.

Impact: The documented at-least-once delivery guarantee does not hold when processing fails.

## Expected Outcome

Define failed-delivery acknowledgement and retry behavior, including what happens to later messages in a batch. Persist acknowledgement only when the chosen delivery policy permits it.

## Validation

Verify transient failure followed by success, restart after failure, bounded retry behavior, batch ordering, and durable acknowledgement.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f6).

## Implementation

Only rows whose handlers return nil enter the atomic acknowledgement batch.
Handler failures and invalid JSON payloads remain pending; later selected rows
can succeed independently. Named restarts retain failures without replaying
acknowledged successes, even when their diagnostic progress is higher.

Each message gets at most one attempt per poll within the configured batch size.
Polling waits the configured interval after each completed cycle, so slow failures
cannot queue immediate retries. There is no total attempt cap or automatic discard;
persistent failures may fill the batch and require application remediation.

Delivery: [Pubsub failed-delivery retries](../../report/20261001060101-pubsub-failed-delivery-retries.md).
