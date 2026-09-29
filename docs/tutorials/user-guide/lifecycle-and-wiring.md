# Lifecycle and Wiring

Wiring turns constructed values into a running Hatmax application. The goal of
this chapter is to reason about component order, startup failure, route
visibility, and shutdown from the composition root.

## Wiring Is Application Design

The order passed to `app.Setup` states runtime dependencies. Hatmax does not
infer a graph or start services from constructors.

```go
components := []any{
	database,
	migrator,
	templates,
	broker,
	featureStore,
	listener,
	featureHandler,
}

starts, stops, registrars := app.Setup(ctx, router, components...)
```

Each component appears after anything it needs during startup. In this
example, the migrator needs the database, the store needs an open connection,
the listener needs the broker, and the handler needs templates and its
already-constructed service dependencies.

This list is not a service locator. Feature code receives dependencies through
constructors and fields. The list exists only to collect application lifecycle
and route capabilities.

## Capabilities Define Participation

`app.Setup` recognizes three small interfaces:

| Capability | Application responsibility |
| --- | --- |
| `app.Startable` | Perform fallible initialization in application order. |
| `app.Stoppable` | Release owned resources during rollback or shutdown. |
| `app.RouteRegistrar` | Register routes after startup succeeds. |

A component may implement one or more capabilities. A database normally starts
and stops. A template manager may parse templates during startup. A handler
normally registers routes without owning a background resource.

Plain models and services do not enter the component list merely because other
objects depend on them. The composition root constructs them and passes them
directly to their consumers.

## Construction Before Startup

Constructors establish valid in-memory objects. They may store dependencies,
normalize lightweight options, and reject invalid local arguments. They do not
perform work that depends on the environment.

Put these actions in `Start`, not in constructors:

- opening and checking a database connection;
- applying migrations or creating owned schema;
- parsing embedded templates;
- subscribing to an event broker;
- starting polling or background processing.

This separation keeps assembly deterministic. Before startup, the application
has a complete inspectable object graph but no partially initialized external
state.

## Startup and Route Visibility

`app.Setup` inspects components and collects start functions, stop functions,
and route registrars in the order supplied. It does not execute them.

`app.Start` then:

1. executes every collected start function in order;
2. stops and returns an error when a start function fails;
3. registers routes only after all startup functions succeed.

Handlers therefore do not receive traffic while a declared startup dependency
is unavailable. The application calls `app.Serve` only after `app.Start`
returns successfully.

## Rollback Alignment

Startup and stop capabilities are collected into independent slices. On a
startup failure, Hatmax uses the failing start position to stop earlier
components in reverse order.

An application that relies on this rollback must keep startup and stop entries
positionally aligned. Ordered resource components should normally implement
both `Startable` and `Stoppable`. Inserting a start-only or stop-only component
among them shifts one slice and breaks that correspondence.

Route-only components do not affect this alignment because they enter neither
slice. Plain services also remain outside both slices.

This is a current application-lifecycle constraint. The
[Application Lifecycle Reference](../../reference/application-lifecycle/README.md)
defines its exact failure behavior.

## Shutdown Reverses Ownership

Normal shutdown stops resource-owning components from last to first. This is
the inverse of startup: subscribers stop before brokers, stores stop before the
database, and downstream work stops before the resources it uses.

The process should stop accepting new work, propagate cancellation, and then
walk the collected stop functions in reverse order. A stop error is logged,
but remaining components still receive their stop call.

The result is a visible lifecycle:

```text
construct -> start in dependency order -> register routes -> serve
                                                      |
                              stop in reverse order <-+
```

## Review the Composition Root

When wiring a capability, check four questions:

1. Which dependencies must already be started?
2. Does this value own a resource or only provide in-memory behavior?
3. Which lifecycle interfaces does it genuinely implement?
4. Where must it stop relative to its dependencies?

If those answers are visible in `main.go`, a reader can understand how the
application becomes operational without tracing hidden initialization.

For the rationale behind this contract, read
[Component Order and Startup Rollback](../../explanation/component-order-and-startup/README.md).

---

[Previous: Application Anatomy](application-anatomy.md) ·
[User Guide](README.md) ·
[Next: The Request Boundary](request-boundary.md)
