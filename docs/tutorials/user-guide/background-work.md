# Work Outside the Request

This chapter follows a todo event from the request handler through Postgres
pubsub to Ticked's durable audit store.

## Before You Begin

Complete [Save a Record](records.md). Keep Ticked running with the first
account and its cookie jar.

## Publish an event

Add another item:

```sh
curl -fsS -b /tmp/ticked-guide.cookies \
  -d 'text=Observe the audit event' \
  http://localhost:8080/add-item >/dev/null
```

The list service saves the item, then publishes an `audit.todo` envelope. The
HTTP response does not wait for the audit subscriber to persist that envelope.

## Observe the subscriber

Wait for at least one broker poll, then request the admin event list:

```sh
sleep 1
curl -fsS -b /tmp/ticked-guide.cookies \
  http://localhost:8080/admin/list-events | grep -F 'todo.item.added'
```

The audit service uses the stable subscriber ID `audit-persistence`. It maps
the copied envelope into an audit record and saves it through its store.

## Restart and verify durability

Restart Ticked, then request the event list again. The event remains in the
audit table. The named subscriber resumes from its stored offset and does not
intentionally replay already acknowledged messages.

Handlers must still tolerate at-least-once delivery. A failure after an
external effect but before offset advancement can repeat work.

## Verify the result

This chapter is complete when `todo.item.added` remains visible after restart.

See [Pubsub](../../reference/pubsub/README.md), the focused
[pubsub how-to](../../how-to/use-pubsub/README.md), and
[Postgres-first Infrastructure](../../explanation/postgres-first/README.md).

Continue with [Change Settings at Runtime](settings.md).

---

[Previous: Save a Record](records.md) · [User Guide](README.md) ·
[Next: Change Settings at Runtime](settings.md)
