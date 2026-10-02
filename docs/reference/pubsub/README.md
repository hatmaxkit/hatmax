<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Pubsub

`pubsub` fans a message out to every subscriber of a topic. Acknowledgement
and ordering semantics depend on the backend. The implementation note is
[pubsub/readme.md](../../../pubsub/readme.md).

## Envelope

`Envelope` has `ID`, `Topic`, `Timestamp`, `Payload`, and `Metadata`.
`NewEnvelope` sets a new UUID, the current local time, and an empty metadata
map. `WithMetadata` returns a copy with one key set. A nil metadata map is
created first.

`Publish` reports transport or persistence errors. It does not report handler
errors. `SubscribeOptions.SubscriberID` names the subscriber. An empty id is
ephemeral.

`Broker` is `Publish`, `Subscribe`, and `Close`.

## In process

`NewNoopBroker` records published envelopes and does not call handlers.
`Published` returns the recorded copies. `Reset` clears them. `Close` returns
nil.

`NewMemoryBroker` calls every handler registered for the topic before
`Publish` returns. A handler error is ignored and delivery continues.
`Publish` returns nil. `Close` clears the subscriber lists and returns nil.

## Postgres

`postgres.New` reads `cfg.PubSub.PollIntervalDuration` and `BatchSize`.
`NewBroker` uses `WithDefaults`: a non-positive interval becomes 100
milliseconds, and a non-positive batch size becomes 100.

`Start` requires `GetDB` and executes the package schema in a transaction. A
nil database returns `database connection not available`. `Stop` calls `Close`.

`Publish` rejects a closed broker, JSON-encodes the payload and metadata, and
inserts `pubsub_messages`. `Subscribe` rejects a duplicate subscriber id. An
empty id becomes a UUID. A named subscriber resumes from its per-message
acknowledgements. `Close` is idempotent and cancels the poll loops.

### Delivery tracking

Pending messages have no `pubsub_acknowledgements` row for that subscriber.
Polling selects committed pending rows of the requested topic in ascending ID
order, with at most `BatchSize` rows per poll. A lower ID that commits after a
higher ID was processed remains pending and is delivered in a later poll.
IDs are not a commit-order or global delivery-order guarantee.

New subscriptions acknowledge only the topic history visible in their
registration statement's snapshot. They do not replay that history, but rows
that were still uncommitted remain eligible after commit, even below the
initial diagnostic offset. Each subscriber has independent acknowledgements
for fan-out. Named subscriber IDs must remain bound to their original topic.

The batch's exact acknowledgement rows and `last_message_id` are committed
atomically after processing. The scalar field is retained as diagnostic
progress, not used to exclude pending messages. A crash or persistence failure
between handler execution and durable acknowledgement can repeat delivery;
handlers must tolerate duplicates. The poll loop uses a background context,
not the subscribe context.

### Failures and retries

A message is acknowledged only after its handler returns nil. Handler errors
and invalid JSON payloads are logged and remain pending. Failure does not stop
the batch: later selected messages are attempted and successful ones are
acknowledged independently. A failed message can therefore succeed after a
later message; strict processing order is not guaranteed. Restarting with the
same subscriber ID retains the failed delivery without replaying acknowledged
successes. Invalid metadata retains the existing empty-map fallback.

Each pending message gets at most one handler attempt per poll, within
`BatchSize`. The first poll waits `PollInterval`; subsequent polls wait that
interval after the previous poll completes, including processing and persistence.
There is no immediate retry loop or accumulated ticker backlog. `Close` cancels
the poll context and waits for its loop; handlers must honor cancellation.

There is no total attempt limit, exponential backoff, or automatic dead-letter
queue. A message remains pending until processing succeeds or the application
explicitly removes it. Persistent failures can occupy every batch slot and
delay later messages. Applications own monitoring, remediation, handler
timeouts, and external-effect idempotency. Returning nil means the application
accepts successful processing; the broker cannot verify external side effects.

### Existing databases and storage

`Start` adds the acknowledgement table and an initialization marker. For each
legacy subscription it seeds acknowledgements once for committed rows visible
at migration whose topic matches and whose ID is at or below the old offset.
Repeating startup does not seed new low-ID rows. The whole schema migration
rolls back on failure. Stop older broker versions before upgrading; mixed old
cursor-based and new acknowledgement-based writers are not supported.

Previously skipped rows already below a legacy offset cannot be distinguished
from successfully processed history and are not recovered automatically.
Pending rows above that offset remain pending. Applications that need to
recover older skipped work require an explicit replay policy.

Messages already acknowledged by older brokers, including failed deliveries,
are not automatically replayed by this retry policy.

Acknowledgement storage grows by one row per subscriber/message pair, including
history excluded at registration. The message log and acknowledgement ledger
have no automatic retention cleanup. Batch limits bound returned rows, handler
calls, and in-memory ID tracking; database scan work can grow with retained
history. Indexes cover `(topic, id)` and `(subscriber_id, message_id)`.
Retention and database capacity remain application-owned. Deleting a message
or subscription also deletes its acknowledgement rows through foreign keys.
