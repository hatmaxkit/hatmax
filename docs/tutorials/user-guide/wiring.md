# Wiring

`app.Setup` collects `Start`, `Stop`, and `RegisterRoutes` in the order you
pass the components. `app.Start` runs the start functions in that order, and
registers routes only after every start function succeeds.

The lifecycle contract is in
[Application Lifecycle](../../reference/application-lifecycle/index.md).

## Order used by the guide companion

1. `db.Database`. Later components call `GetDB` during `Start`.
2. `noteStore`. It creates its table on the open connection.
3. The Postgres broker. It creates its schema on the same connection.
4. `eventListener`. It subscribes after the broker's `Start`.
5. The template manager. It parses templates before any page registers routes.
6. `pages`. It contributes routes but no startup function.

A component that needs another component's `Start` comes after it.

The first five components implement both `Startable` and `Stoppable`. This
keeps the independently collected start and stop slices aligned for startup
rollback. A route-only component does not enter either slice.

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

For the design consequences of this API, see
[Component Order and Startup Rollback](../../explanation/component-order-and-startup/index.md).
