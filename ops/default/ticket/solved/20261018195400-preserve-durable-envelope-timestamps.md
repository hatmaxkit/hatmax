---
id: TKT-20261018195400
title: Preserve durable envelope occurrence timestamps
status: solved
kind: bug
severity: medium
priority: high
scope: persistence
tags: pubsub, postgres, timestamps
source: implementation
reported_at: 2026-10-18T19:54:00Z
ready_at: 2026-10-18T19:54:30Z
started_at: 2026-10-18T19:54:30Z
commits: 4752f2f41ff7, 29a022775875
reviewed_at: 2026-10-18T19:58:37Z
closed_at: 2026-10-18T19:58:37Z
resolution: fixed
validated_commit: 29a022775875cb6bc309c15e6ce1aa6a145b5f32
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

## Resolution

Both real PostgreSQL regressions failed before correction and passed afterward.
The affected-package gate passed without changing its selected checks:

- `GOWORK=off scripts/check-durable-background-work.sh slice 2`

This ran pubsub vet, both pubsub packages with race detection, strict lint and
`make docs-check`. Owned PostgreSQL startup and shutdown completed. No full
repository gate, release, tag or historical delivery receipt was changed.
