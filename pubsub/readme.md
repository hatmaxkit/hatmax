# pubsub

Publish/subscribe messaging with fan-out semantics.

## Usage

```go
// Create broker (Postgres or Noop)
broker := postgres.NewBroker(db, cfg, log)
broker := pubsub.NewNoopBroker()

// Publish
env := pubsub.Envelope{
    ID:      model.NewID(),
    Topic:   "user.created",
    Payload: user,
}
broker.Publish(ctx, "user.created", env)

// Subscribe (each subscriber gets all messages)
broker.Subscribe(ctx, "user.created", func(ctx context.Context, env pubsub.Envelope) error {
    user := env.Payload.(User)
    // handle...
    return nil
}, pubsub.SubscribeOptions{SubscriberID: "email-sender"})
```

## API

```go
type Publisher interface {
    Publish(ctx context.Context, topic string, env Envelope) error
}

type Subscriber interface {
    Subscribe(ctx context.Context, topic string, handler Handler, opts SubscribeOptions) error
}
```

At-least-once delivery. Named subscribers resume from last offset.
