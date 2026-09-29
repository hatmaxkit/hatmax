# Interfaces and Adapter Ownership

Hatmax places small interfaces at boundaries where an application may need a
different implementation. The package that consumes a capability owns the
interface whenever practical. Concrete infrastructure stays in an adapter.

Examples include `auth.Queries`, `scheduler.JobStore`, `pubsub.Publisher`,
`image.Store`, and `mailer.Mailer`. Handlers and domain services depend on the
behavior they use, while the application entrypoint selects the concrete
implementation.

This arrangement has three consequences:

- assembly remains visible in `main`;
- tests can use small fakes without replacing unrelated infrastructure;
- adding an adapter does not create a second application architecture.

Interfaces do not remove lifecycle responsibilities. A Postgres adapter may
still need `Start` and `Stop`, and its position in `app.Setup` must follow its
dependencies.
[Lifecycle and Wiring](../../tutorials/user-guide/lifecycle-and-wiring.md)
places those edges in the application composition flow.
