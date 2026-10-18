<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Why Hatmax Uses Postgres-First Infrastructure

Hatmax applications already need durable relational state. Reusing Postgres
for migrations, sessions, scheduled jobs, and pubsub reduces the number of
services required by a small application and keeps local operation simple.

The choice is a default, not a hidden dependency. Authentication accepts a
`Queries` implementation. Scheduler accepts a `JobStore`. Pubsub exposes
publisher and subscriber interfaces. An application can replace those
adapters without changing the handlers and services that use them.

## Benefits

- one connection and operational model for common durable capabilities;
- transactional SQL where application state and supporting records meet;
- fewer development services and credentials;
- explicit schemas that can be inspected and migrated with ordinary tools.

## Tradeoffs

Database-backed polling is not a substitute for every queue or streaming
system. High fan-out, strict latency, independent scaling, or specialized
delivery guarantees can justify another backend. Hatmax keeps those decisions
at interfaces so applications can introduce them when their requirements are
concrete.

The [Database](../../reference/database/README.md),
[Pubsub](../../reference/pubsub/README.md), and
[Scheduler](../../reference/scheduler/README.md) references state the current
backend behavior.

## Ownership of scheduled attempts

Durable scheduler execution keeps claim, lease expiry, retained budget and terminal
schedule advancement in PostgreSQL transactions. A fresh token fences the previous
worker when another process reclaims an expired attempt. The database clock avoids
using independent process clocks to decide ownership.

This bounds transport attempts and supports recovery after process interruption.
It cannot stop a handler that ignores cancellation, or prove whether an external
effect happened before the interruption. Applications still own transactional
deduplication and external idempotency. Retention removes only terminal history in
caller-owned batches; it never silently discards unfinished work. See the
[leased scheduler contract](../../reference/scheduler/README.md#leased-execution).

## Transactions and independent event delivery

Bounded publication uses the application's PostgreSQL transaction so domain state
and its message share one visibility boundary. Capacity reservations belong to that
transaction and roll back with it. Independent reservations allow a lower message
ID to commit later; acknowledgement-based selection keeps that late commit eligible.
Neither an ID nor a registration timestamp is a global commit-order cursor.

Each named durable subscriber owns a separate retained attempt and frozen policy.
A terminal failed delivery quarantines its transport failure without claiming a
successful business effect or blocking every future poll. Handler deadlines remain
cooperative. Applications own deduplication and operator authority; retained capacity
creates explicit backpressure rather than silently dropping pending work. See the
[bounded event contract](../../reference/pubsub/README.md#bounded-durable-events).
