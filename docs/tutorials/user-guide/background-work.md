# Work Outside the Request

Saving a note publishes one message. A subscriber receives that message on a
later poll, after the HTTP handler has returned.

The scheduler and the mailer are the other two ways to leave the request.
Their contracts are in
[Scheduler](../../reference/scheduler/index.md) and
[Mailer](../../reference/mailer/index.md).
This chapter uses
[Pubsub](../../reference/pubsub/index.md).

## Start the broker before the subscriber

`postgres.New` uses the database component and `cfg.PubSub`. `Start` creates
the pubsub tables. Put the broker in `app.Setup` after `database`.

A listener subscribes in its own `Start`, after the broker. An empty
`SubscriberID` would be a new subscriber on every start. Use `guide-notes`.
A new named subscriber begins at the current last message, so it receives
only messages published after it subscribes.

```go
type listener struct {
	broker *postgres.Broker
	mu     sync.Mutex
	seen   []string
}

func (l *listener) Start(ctx context.Context) error {
	return l.broker.Subscribe(ctx, "notes", l.handle, pubsub.SubscribeOptions{
		SubscriberID: "guide-notes",
	})
}

func (l *listener) Stop(context.Context) error { return nil }

func (l *listener) handle(ctx context.Context, env pubsub.Envelope) error {
	title, _ := env.Payload.(string)

	l.mu.Lock()
	l.seen = append(l.seen, title)
	l.mu.Unlock()

	return nil
}
```

`app.Setup` order is `database`, the broker, the listener, then the notes
component. `Publish` inserts a row and returns. The poll loop waits for
`cfg.PubSub.PollInterval` before the first read. The default interval is
`100ms`.

## Publish when the note is saved

After the insert from Save a Record succeeds, publish the title:

```go
err = notes.Insert(r.Context(), title)
if err != nil {
	http.Error(w, "cannot save", http.StatusInternalServerError)
	return
}

err = broker.Publish(r.Context(), "notes", pubsub.NewEnvelope("notes", title))
if err != nil {
	http.Error(w, "cannot publish", http.StatusInternalServerError)
	return
}
```

`Publish` JSON-encodes the string payload. The subscriber decodes it back to
a string.

Add `GET /effects`. It writes the received titles, one per line. That list
is in memory. A restart clears it. The subscriber does not receive the old
messages again.

## Check the result

Run the process against the Postgres from Add Postgres.

```sh
curl -sS -H 'Origin: http://localhost:8080' \
  -d 'title=Hello' -o /dev/null -w '%{http_code}\n' \
  http://localhost:8080/notes
sleep 1
curl -sS http://localhost:8080/effects
```

The POST status is `200`. That response returns before the subscriber runs.
After the pause, `/effects` prints `Hello`.

The scheduler would run a job on its own interval. The mailer would send from
`Mailer.Send`. Neither runs in this chapter.
