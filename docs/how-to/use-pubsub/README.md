<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Publish and Subscribe to Events

Use the Postgres broker for durable fan-out between components in one or more
Hatmax processes.

## Create and order the broker

This composition-root fragment assumes a loaded root config, logger, started
database provider, router and subscriber component:

```go
broker := postgres.New(database, cfg, logger)

starts, stops, registrars := app.Setup(
	ctx,
	router,
	database,
	broker,
	subscriber,
)
```

The database starts first. The broker creates its schema next. A subscriber
registers only after both are available. Execute startup steps and defer reverse
shutdown with a fresh context as described in the
[lifecycle reference](../../reference/application-lifecycle/README.md). `app.Setup`
builds the steps; it does not execute them.

## Subscribe with a stable identity

In the subscriber's `Start` method, handle registration errors:

```go
err := broker.Subscribe(ctx, "orders", handleOrder, pubsub.SubscribeOptions{
	SubscriberID: "billing-orders",
})
if err != nil {
	return err
}
```

A stable subscriber ID resumes its pending deliveries. An empty ID creates a
new ephemeral subscriber and does not resume previous state.

## Publish

In a publishing service method returning an error:

```go
event := pubsub.NewEnvelope("orders", order)
err := broker.Publish(ctx, "orders", event)
if err != nil {
	return err
}
```

Publishing stores the envelope. Handlers run later in polling goroutines.
Design handlers for at-least-once delivery and make repeated processing safe.
Return an error when processing fails: the message remains pending for a later
poll, while successful messages in the same batch can be acknowledged. Monitor
persistent failures; they can fill a batch and delay newer messages.
The stored topic is the `Publish` topic argument; keep `Envelope.Topic` aligned
for in-process consumers too. PostgreSQL payloads are decoded JSON values rather
than instances of the original Go type. Validate their shape before use.

## Verify the result

Publish an envelope after the subscriber starts. Confirm that the handler
receives it and that restarting with the same subscriber ID does not replay
already acknowledged messages. Make a handler fail once, then succeed: confirm
that only the failed message is retried, including after a named restart.
Close the broker before its database. Its polling context is independent of the
`Subscribe` context; canceling the latter does not stop delivery. `Close`
cancels handlers and waits for them, without honoring a `Stop` wait deadline.
Handlers must cooperate with cancellation.

See [Pubsub Reference](../../reference/pubsub/README.md).
