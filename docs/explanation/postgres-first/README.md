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
