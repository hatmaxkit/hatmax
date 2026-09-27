# Run the First Process

This chapter starts the versioned companion application without Postgres. By
the end, both the health endpoint and the HTML page respond.

## Before You Begin

Complete the repository checkout described in the
[User Guide](index.md#before-you-begin). Port `8080` must be available.

## Start the application

```sh
cd examples/guide
go run .
```

The process loads `config.yaml`, validates it, creates a logger and router,
starts the template manager, registers routes, and listens on `:8080`.

`GUIDE_DATABASE_ENABLED` is unset, so the database, note store, broker, and
subscriber are not assembled.

## Check the result

In another terminal:

```sh
curl -fsS http://localhost:8080/ping
curl -fsS http://localhost:8080/ | grep -F 'Hatmax Guide'
```

The first command prints `{"status":"ok"}`. The second finds the page title.
Open `http://localhost:8080/` to see the form and the notice that database
features are disabled.

Stop the process with Ctrl+C.

## Understand the assembly

Open `examples/guide/main.go` and locate `components`. In this mode it contains
the template manager and page registrar. `app.Setup` collects lifecycle and
route capabilities. `app.Start` starts the template manager before it registers
the page routes.

For the exact contract, see
[Application Lifecycle](../../reference/application-lifecycle/index.md).

## Recover from startup problems

- A missing `config.yaml` returns a load error. Run from `examples/guide`.
- An occupied port returns a listen error. Stop the other process or change
  `server.port`.
- A template parse error stops startup before routes are registered. Restore
  valid Go template syntax and run again.

## Verify the result

This chapter is complete when `/ping` and `/` both respond from the same
process.

Continue with [Add Postgres](postgres.md).

---

[User Guide](index.md) · [Next: Add Postgres](postgres.md)
