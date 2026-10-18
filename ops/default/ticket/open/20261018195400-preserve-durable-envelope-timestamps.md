---
id: TKT-20261018195400
title: Preserve durable envelope occurrence timestamps
status: open
kind: bug
severity: medium
priority: high
scope: persistence
tags: pubsub, postgres, timestamps
source: implementation
reported_at: 2026-10-18T19:54:00Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

At b1ce5387f553, bounded PublishTx discards Envelope.Timestamp. Durable delivery
returns the database publication time instead. Delayed publication consequently
changes the occurrence time visible to handlers, unlike legacy publication.

## Expected Outcome

Preserve the first publication's supplied occurrence time through retries and
restart without using it for retention or identity comparison. Keep database
publication time independent, define the zero-time fallback, and migrate existing
rows additively without inventing source history.

## Validation

Extend existing real PostgreSQL delivery coverage for source time, redelivery,
duplicate publication, zero time and pre-upgrade rows. Run affected pubsub checks
and documentation validation; retain historical delivery evidence unchanged.
