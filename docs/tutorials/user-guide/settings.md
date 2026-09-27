# Change Settings at Runtime

This chapter changes a schema-checked greeting without changing static process
configuration. Restarting the process restores the schema default because the
guide uses an in-memory settings store.

## Before You Begin

Complete [Work Outside the Request](background-work.md), then stop Ticked. From
the repository root, start the smaller companion again:

```sh
cd examples/guide
go run .
```

The entrypoint registers `guide.greeting` as a string setting with default
`Hello`. `config.yaml` has no key for it.

## Read and change the setting

```sh
curl -fsS http://localhost:8080/greeting

curl -sS -D - -o /dev/null \
  -H 'Origin: http://localhost:8080' \
  -d 'greeting=Hi' \
  http://localhost:8080/greeting

curl -fsS http://localhost:8080/greeting
```

The first response is `Hello`. The POST returns a `303` redirect. The final
response is `Hi`.

`settings.Service.Set` validates registered values before it writes them. The
in-memory store changes immediately; `config.Config` remains unchanged.

## Restart and verify the lifetime

Stop the process with Ctrl+C and start it again:

```sh
go run .
curl -fsS http://localhost:8080/greeting
```

The response is `Hello` because this example store is not durable.

## Recover from setting problems

- A rejected value returns `400` with the schema validation error.
- A store error on `GetString` falls back to the registered default.
- A durable application must supply a `settings.Store`; Hatmax does not ship a
  Postgres implementation for settings.

## Verify the result

This chapter is complete when the value changes without a restart and returns
to its schema default after a restart.

See [Configuration](../../reference/configuration/index.md) and
[Static Configuration and Runtime Settings](../../explanation/configuration-boundaries/index.md).

---

[Previous: Work Outside the Request](background-work.md) ·
[User Guide](index.md)
