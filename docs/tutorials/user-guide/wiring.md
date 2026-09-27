# Wiring

`app.Setup` collects `Start`, `Stop`, and `RegisterRoutes` in the order you
pass the components. `app.Start` runs the start functions in that order, and
registers routes only after every start function succeeds.

The lifecycle contract is in
[Application Lifecycle](../../reference/application-lifecycle/index.md).

## Order used in this guide

1. `db.Database`. Later components call `GetDB` during `Start`.
2. Components that create tables or schemas on that connection: the auth
   queries, the notes table, and the pubsub broker.
3. The pubsub listener. `Subscribe` runs after the broker's `Start`.
4. The seed runner. It inserts into a table created by an earlier `Start`.
5. The template manager. It parses templates before any page registers routes.
6. Route registrars. Their `RegisterRoutes` methods run last.

A component that needs another component's `Start` comes after it.

## Boundaries

These interfaces are the swappable edges. Pass another implementation to
`app.Setup` or to the constructor that stores it. The guide does not restate
their methods.

| Interface | Package | Reference |
| --- | --- | --- |
| `Mailer` | `mailer` | [Mailer](../../reference/mailer/index.md) |
| `Publisher` | `pubsub` | [Pubsub](../../reference/pubsub/index.md) |
| `Subscriber` | `pubsub` | [Pubsub](../../reference/pubsub/index.md) |
| `JobStore` | `scheduler` | [Scheduler](../../reference/scheduler/index.md) |

The database connection those services use is in
[Database](../../reference/database/index.md).
