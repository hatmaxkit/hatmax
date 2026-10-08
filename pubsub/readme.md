<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# pubsub

Publish/subscribe messaging with fan-out semantics. This complete in-process
example runs without a database:

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"hatmax.adrianpk.com/pubsub"
)

func main() {
	ctx := context.Background()
	broker := pubsub.NewMemoryBroker()
	defer broker.Close()
	err := broker.Subscribe(ctx, "user.created", func(ctx context.Context, env pubsub.Envelope) error {
		fmt.Println(env.Topic, env.Payload)
		return nil
	}, pubsub.SubscribeOptions{SubscriberID: "email-sender"})
	if err != nil {
		log.Fatal(err)
	}
	env := pubsub.NewEnvelope("user.created", "user-1")
	if err := broker.Publish(ctx, "user.created", env); err != nil {
		log.Fatal(err)
	}
}
```

Output: `user.created user-1`. Memory delivery is synchronous; handler errors
are ignored and other handlers still run. Subscriber IDs have no durable
meaning there. `NewNoopBroker` captures envelopes and never delivers them.

For PostgreSQL, `postgres.New(database, rootConfig, logger)` accepts a
`DBProvider` and `*config.Config`; `postgres.NewBroker(database, brokerConfig,
logger)` accepts `postgres.Config`. Start the database and broker before
subscribing or publishing, and close the broker before its database. PostgreSQL
decodes payloads from JSON: a Go struct arrives as a map rather than its original
type. Validate and decode it in the consumer. Follow the
[Postgres workflow](../docs/how-to/use-pubsub/README.md) for durable delivery.

## API

```go
type Publisher interface {
    Publish(ctx context.Context, topic string, env Envelope) error
}

type Subscriber interface {
    Subscribe(ctx context.Context, topic string, handler Handler, opts SubscribeOptions) error
}
```

Named PostgreSQL subscribers resume from per-message acknowledgement records.
Handler errors leave deliveries pending for interval-spaced retries; successful
messages in the same batch can complete independently.
See the [delivery contract](../docs/reference/pubsub/README.md#postgres) for
commit visibility, migration, and failure behavior.
