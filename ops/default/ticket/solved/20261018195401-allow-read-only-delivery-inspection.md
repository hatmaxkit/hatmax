---
id: TKT-20261018195401
title: Allow delivery inspection without schema writes
status: solved
kind: bug
severity: medium
priority: high
scope: api
tags: pubsub, postgres, inspection
source: implementation
reported_at: 2026-10-18T19:54:01Z
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

NewBroker retains only its provider. ListDeliveries cannot use an already-open
database until Start binds it; Start executes schema DDL and admission writes.
Read-only operators therefore cannot use the public inspection method safely.

## Expected Outcome

Bind an available provider connection during lightweight construction without
queries, migrations, admission changes or subscriber startup. Keep deferred
provider initialization working through Start. Database permissions continue to
control inspection and explicit mutation operations.

## Validation

Extend the existing real PostgreSQL work-operations coverage with a broker used
without Start on a database-enforced read-only connection. Retain deferred startup
coverage and run affected pubsub checks and documentation validation.

## Resolution

Both real PostgreSQL regressions failed before correction and passed afterward.
The affected-package gate passed without changing its selected checks:

- `GOWORK=off scripts/check-durable-background-work.sh slice 2`

This ran pubsub vet, both pubsub packages with race detection, strict lint and
`make docs-check`. Owned PostgreSQL startup and shutdown completed. No full
repository gate, release, tag or historical delivery receipt was changed.
