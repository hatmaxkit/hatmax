---
id: TKT-20260930211805
title: Do not acknowledge failed pubsub handlers
status: open
kind: bug
severity: high
priority: high
scope: persistence
tags: architecture-review, persistence, hardening
source: review
reported_at: 2026-09-30T21:18:05Z
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
