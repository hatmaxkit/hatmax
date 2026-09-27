# Save a Record

This chapter adds a todo item in Ticked and verifies that the item survives a
process restart.

## Before You Begin

Complete [Sign In](sign-in.md). Keep Ticked running and keep
`/tmp/ticked-guide.cookies`.

## Add an item

```sh
curl -fsS -b /tmp/ticked-guide.cookies \
  -d 'text=Review the Hatmax guide' \
  http://localhost:8080/add-item | grep -F 'Review the Hatmax guide'
```

The handler parses the form, reads the authenticated user from the context,
and calls the list service. The service gets or creates that user's list,
updates it, and saves the aggregate through its Postgres store. The response
is the rendered item partial.

## Restart the process

Stop `make run-fg` with Ctrl+C, then start it again:

```sh
make run-fg
```

The session and todo data remain in Postgres. Request the list with the same
cookie jar:

```sh
curl -fsS -b /tmp/ticked-guide.cookies \
  http://localhost:8080/list-items | grep -F 'Review the Hatmax guide'
```

## Inspect the persistence boundary

The Ticked list service depends on its own small store interface. The Postgres
adapter owns SQL and maps rows to the list model. Hatmax supplies the shared
database lifecycle, identifiers, time helpers, migrations, and validation
building blocks without owning the application's todo schema.

## Recover from persistence problems

- `Unauthorized` means the session cookie was not sent or no longer validates.
- `Failed to load list` indicates a store failure; inspect the process log and
  Postgres connection.
- `Text is required` means the form value was empty.

## Verify the result

This chapter is complete when the exact item text appears after restarting the
process.

See [Database](../../reference/database/index.md),
[Model](../../reference/model/index.md), and
[Validation](../../reference/validation/index.md).

Continue with [Work Outside the Request](background-work.md).

---

[Previous: Sign In](sign-in.md) · [User Guide](index.md) ·
[Next: Work Outside the Request](background-work.md)
