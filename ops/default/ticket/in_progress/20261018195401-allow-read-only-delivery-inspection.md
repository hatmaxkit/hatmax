---
id: TKT-20261018195401
title: Allow delivery inspection without schema writes
status: in_progress
kind: bug
severity: medium
priority: high
scope: api
tags: pubsub, postgres, inspection
source: implementation
reported_at: 2026-10-18T19:54:01Z
ready_at: 2026-10-18T19:54:30Z
started_at: 2026-10-18T19:54:30Z
commits:
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
