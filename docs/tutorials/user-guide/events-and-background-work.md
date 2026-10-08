<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Events and Background Work

Requests should complete without absorbing every secondary effect. Hatmax
provides two explicit boundaries for work outside the request:

```text
completed workflow -> pubsub event -> interested subscribers
scheduled record   -> scheduler    -> registered task handler
```

Neither mechanism changes feature ownership. A service decides when an event
is meaningful; a background handler calls application workflows through
narrow interfaces.

## Publish Facts After Durable Work

Depend on `pubsub.Publisher`, construct an envelope, and publish only after the
state change it describes has succeeded:

```go
envelope := pubsub.NewEnvelope("invoices", invoice.ID)
if err := service.publisher.Publish(ctx, "invoices", envelope); err != nil {
	return fmt.Errorf("publish invoice creation: %w", err)
}
```

The fragment belongs in an invoice service method returning an error, with a
`pubsub.Publisher` dependency. The subscriber below supplies its lifecycle
context; neither fragment is a standalone program.

An envelope carries an ID, topic, timestamp, payload, and metadata. Keep the
payload small and versionable; identifiers let the subscriber load current
state through its own service rather than coupling features through shared
models.

The memory broker invokes subscribers synchronously and is useful for tests.
The Postgres broker persists JSON envelopes and polls them. Named subscriber
IDs retain pending deliveries across restarts. Returning an error leaves a
message pending for another poll; successful messages in the same batch can
complete. Handlers must tolerate retries and repeated external effects. Monitor
persistent failures, which can fill a batch and delay newer messages; see the
[retry contract](../../reference/pubsub/README.md#failures-and-retries).
Postgres decodes payloads as JSON values, so consumers validate and decode their
shape rather than asserting the original Go struct type. Memory broker handler
errors are ignored and it has no persisted retry or acknowledgement state.

Publishing after a database commit still leaves a gap between the state
change and event insertion. When losing that event is unacceptable, the
application needs an explicit transactional outbox workflow; ordinary broker
publication does not claim that guarantee.

## Give Subscribers Lifecycle Ownership

A subscriber component registers in `Start`, after its broker and stores are
ready:

```go
func (service *AuditService) Start(ctx context.Context) error {
	return service.subscriber.Subscribe(
		ctx,
		"invoices",
		service.handle,
		pubsub.SubscribeOptions{SubscriberID: "invoice-audit"},
	)
}
```

The handler validates the envelope, performs one idempotent application
workflow, and returns an error for observable failure. It does not reuse an
HTTP handler or fabricate an `http.Request`.

## Schedule Durable Jobs

The scheduler polls a `JobStore` for due jobs and dispatches each `TaskType` to
one registered handler. Jobs carry identity, payload, scheduled time, attempt,
and metadata. Schedules calculate future UTC times from daily, weekly, or
interval rules.

Construct the runner with its Postgres store, configuration, logger, and
optional settings provider. Register every handler before startup. Disabled
runners start without a goroutine; enabled runners tick immediately and then
at the configured interval.

The first startup context owns the polling loop. Repeated starts do not launch
extra loops. Canceling that context or calling `Stop` cancels active execution;
handlers must cooperate with cancellation. Give `Stop` a fresh timeout context
and handle its error: a timed-out wait does not mean the handler has exited.
Repeated stops are safe; restarting after shutdown requires a new runner.

Handlers return `scheduler.Result`. The runner records running, successful,
failed, retry-waiting, and unknown-task outcomes. A handler panic follows the
retry policy without stopping other jobs; failures to persist that state are logged. Stores complete
the run and its schedule together: a one-shot job is retired, while a recurring
job advances from terminal completion time, including after exhausted failure. Missed occurrences
are skipped. Configure the schedule on the job rather than updating its next
run from a handler. The Postgres reference defines its stored schedule format.

Set `scheduler.retry_attempts` to the maximum total attempts per slot, including
the first (`3` by default, `1` for no retries). `scheduler.retry_backoff` controls
the fixed wait after a failed attempt. Retry waits survive restart and keep the
slot's original budget; they do not hold workers or advance its schedule. Make
handlers safe to repeat and use the stable `Job.RunID` when tracking effects.
The next scheduled occurrence is separate from a retry. Interrupted pending or
running claims still require application repair; see the
[retry contract](../../reference/scheduler/README.md#retries).

Use deterministic clocks and fake stores in unit tests. Use Postgres tests for
locking, due-job selection, and persisted run state.

## Wire Background Components After Their Dependencies

The visible order is:

```text
database -> migrator -> broker and job store -> subscribers and scheduler
```

Publishers that only retain a broker interface need no lifecycle entry.
Subscribers and runners do because they start loops or register durable
consumers. Reverse shutdown stops consumers before their broker and database.

For exact contracts, see [Pubsub](../../reference/pubsub/README.md),
[Scheduler](../../reference/scheduler/README.md), and the focused
[pubsub how-to](../../how-to/use-pubsub/README.md).

---

[Previous: Configuration and Runtime Settings](configuration-and-runtime-settings.md) ·
[User Guide](README.md) ·
[Next: Application Services](application-services.md)
