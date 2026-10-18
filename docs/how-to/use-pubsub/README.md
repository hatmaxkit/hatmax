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
For legacy unadmitted topics, the stored topic is the `Publish` argument; keep `Envelope.Topic` aligned
for in-process consumers too. PostgreSQL payloads are decoded JSON values rather
than instances of the original Go type. Validate their shape before use.

## Verify the result

Publish an envelope after the subscriber starts. Confirm that the handler
receives it and that restarting with the same subscriber ID does not replay
already acknowledged messages. Make a handler fail once, then succeed: confirm
that only the failed message is retried, including after a named restart.
Close the broker before its database. Its polling context is independent of the
`Subscribe` context in legacy mode. Bounded subscribers inherit that context.
`Close` cancels handlers and waits for them; `Stop(ctx)` bounds the wait with the
caller's context deadline and reports outstanding handlers honestly.
Handlers must cooperate with cancellation.

See [Pubsub Reference](../../reference/pubsub/README.md).

## Enable bounded publication and delivery

Use explicit PostgreSQL configuration after quiescing older workers:

```go
broker := postgres.NewBroker(database, postgres.Config{
    PollInterval: 100*time.Millisecond,
    BatchSize: 50,
    Bounded: &postgres.BoundedConfig{Topics: []string{"orders"}, Capacity: 10000},
}, logger)
```

Start the broker before registering handlers. Use stable message and subscriber
identities and select history explicitly:

```go
options := pubsub.DurableOptions{
    Policy: pubsub.DefaultDeliveryPolicy(),
    History: pubsub.RetainedHistory,
}
err := broker.Subscribe(ctx, "orders", handleOrder, pubsub.SubscribeOptions{
    SubscriberID: "billing-orders",
    Durable: &options,
})
if err != nil {
    return err
}
```

Use the subscriber lifecycle context, not an HTTP request context. New attempts
freeze a cooperative handler deadline, lease, total budget and retry delay. A
failed delivery eventually becomes terminal without acknowledging it; later
messages can progress. Protect repeated effects transactionally or with external
idempotency keys, including after a worker dies or misses its lease.

## Publish within a domain transaction

In a function returning an error, use the same database schema for domain writes
and publication. Construct a valid `event` and perform the domain writes through
`tx` before publication:

```go
tx, err := database.GetDB().BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// Perform the application's domain writes through tx here.
if err = broker.PublishTx(ctx, tx, "orders", event); err != nil {
    return err
}
return tx.Commit()
```

The broker leaves commit and rollback to the caller. Reuse the message identity
for an exact retry; changed topic, payload or metadata is a conflict. Handle
`ErrCapacity` as explicit backpressure. Do not treat failed admission as a sent
event or delete pending messages to make room.

## Upgrade an existing topic

1. Stop old workers and wait for their handlers to return; keep their database open.
2. Start the upgraded broker without bounded topic configuration to apply the
   additive schema while preserving existing message and acknowledgement rows.
3. Audit and opt the topic in with
   `broker.AdmitBoundedTopics(ctx, postgres.BoundedConfig{Topics: []string{"orders"}, Capacity: 10000})`.
   Invalid identities, JSON, sizes, counts or capacity abort the transaction.
4. Upgrade each named subscriber with
   `broker.ConfigureDurableSubscription(ctx, "billing-orders", "orders", options)`.
   Select retained history to keep its pending rows while preserving successful
   acknowledgements. Reopening without this explicit upgrade is rejected.
5. Restart durable subscribers with matching options. Verify late commits, retained
   attempt counts, terminal failures and progress on later messages using a test
   database before operating the upgraded workers.

Do not run mixed unfenced and durable workers. Configuration changes for future
deliveries do not reset an existing delivery budget. The
[bounded reference](../../reference/pubsub/README.md#bounded-durable-events) defines
input limits, classified outcomes and retained storage.

## Inspect and replay retained failures

Use a separate authorized operator workflow with a finite context:

```go
inspectCtx, cancelInspect := context.WithTimeout(ctx, 5*time.Second)
defer cancelInspect()

rows, err := broker.ListDeliveries(inspectCtx, postgres.DeliveryQuery{
    SubscriberID: "billing-orders", State: "failed", Limit: 50,
})
if err != nil {
    return err
}
```

Use the last row's `SubscriberID` and `MessageID` in `DeliveryQuery.After` for the
next page. Inspect codes and timestamps; payloads and ownership tokens are excluded.
Check the current domain state and reconcile any previous external effect before
selecting one failed row for replay. A replay never asserts that an old effect failed.

With `observed` set to that inspected row, grant a finite retry budget:

```go
err = broker.ReplayDelivery(ctx, postgres.ReplayRequest{
    SubscriberID: observed.SubscriberID,
    MessageID: observed.MessageID,
    Attempt: observed.Attempt,
    UpdatedAt: observed.UpdatedAt,
    AdditionalAttempts: 2,
    Reason: postgres.ReplayDependency,
})
if err != nil {
    return err
}
```

A stale revision is refused. Inspect again before deciding on another replay;
never reset counters or retry acknowledged work. A grant allows at most eight
additional attempts and the lifetime total stays at or below 32. Continue using
idempotent domain handlers to reject obsolete effects.

## Retire and retain subscribers

Stop that subscriber's workers and wait for their handlers. Call
`broker.RetireSubscriber(ctx, "billing-orders", postgres.RetireOptions{})` to retire
only when every entitled message is terminal. If unfinished work must be discarded,
make that decision explicitly in the authorized operator path and use
`postgres.RetireOptions{DiscardPending: true}`. This records failed retained attempts
and fences old owners; it cannot terminate an old handler or roll back its effects.
The name remains reserved and cannot be reopened for another topic.

Choose a retention cutoff after considering every entitled subscriber and any
replay investigation. Run bounded message cleanup with a finite context:

```go
pruneCtx, cancelPrune := context.WithTimeout(ctx, 5*time.Second)
defer cancelPrune()

removed, err := broker.PruneMessages(pruneCtx, time.Now().Add(-30*24*time.Hour), 1000)
if err != nil {
    return err
}
logger.Infof("pruned %d terminal messages", removed)
```

Repeat batches when needed. Pending, active and retry-waiting work remains;
recent terminal outcomes remain until their cutoff passes. A failed delivery
is terminal without a success acknowledgement. Acknowledgement and delivery rows
are deleted with each eligible message, and its capacity reservation is released
atomically. Current-only history exclusions remain valid without synthetic attempt
rows. Legacy subscriptions awaiting upgrade keep their pending entitlement.

For scheduled-run inspection, adoption and terminal cleanup, use the
[scheduler operating procedure](../run-background-jobs/README.md#inspect-and-retain-history).
