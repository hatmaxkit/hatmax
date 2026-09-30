<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Publish and Subscribe to Events

Use the Postgres broker for durable fan-out between components in one or more
Hatmax processes.

## Create and order the broker

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
registers only after both are available.

## Subscribe with a stable identity

```go
err := broker.Subscribe(ctx, "orders", handleOrder, pubsub.SubscribeOptions{
	SubscriberID: "billing-orders",
})
```

A stable subscriber ID resumes from its stored offset. An empty ID creates a
new ephemeral subscriber and does not resume previous state.

## Publish

```go
event := pubsub.NewEnvelope("orders", order)
err := broker.Publish(ctx, "orders", event)
```

Publishing stores the envelope. Handlers run later in polling goroutines.
Design handlers for at-least-once delivery and make repeated processing safe.

## Verify the result

Publish an envelope after the subscriber starts. Confirm that the handler
receives it and that restarting with the same subscriber ID does not replay
already acknowledged offsets.

See [Pubsub Reference](../../reference/pubsub/README.md).
