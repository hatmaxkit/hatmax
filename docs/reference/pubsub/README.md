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

The copy is shallow: an existing metadata map remains shared.
`NewEnvelopeWithMetadata` also retains the supplied map. Copy application-owned
maps before concurrent mutation when a snapshot is needed. IDs are strings.

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

Captured envelopes are shallow copies; their payload and metadata are not deep
snapshots. No-op and memory brokers do not inspect cancellation themselves.
Memory subscriber IDs are ignored. There are no acknowledgements, persisted
retries or polling goroutines for this backend.

## Postgres

`postgres.New` reads `cfg.PubSub.PollIntervalDuration` and `BatchSize`.
`NewBroker` uses `WithDefaults`: a non-positive interval becomes 100
milliseconds, and a non-positive batch size becomes 100.

`Start` requires `GetDB` and executes the package schema in a transaction. A
nil database returns `database connection not available`. Bounded topic configuration
is applied before startup succeeds.

`New` accepts root `*config.Config`; `NewBroker` accepts `postgres.Config`.
Both receive a provider exposing `GetDB() *sql.DB`, not a bare connection. Call
`Start` before publication or subscription and keep the database alive until
shutdown completes. `Stop(ctx)` cancels subscriptions and waits within the caller's
context deadline; a timed-out wait leaves outstanding handlers running. Repeated
stops wait on the same loops. `Close()` waits without a deadline. Neither closes
the caller-owned database.

`Publish` rejects a closed broker, JSON-encodes the payload and metadata, and
inserts `pubsub_messages`. `Subscribe` rejects a duplicate subscriber id. An
empty id becomes a UUID. A named subscriber resumes from its per-message
acknowledgements. `Close` is idempotent and cancels the poll loops.

The `Publish` topic argument is stored, rather than `Envelope.Topic`. Payload
JSON is decoded to ordinary JSON values: objects become `map[string]any` and
numbers become `float64`. Consumers validate and decode that shape instead of
asserting the original Go type.

The following legacy delivery sections describe unadmitted topics and subscriptions
without `SubscribeOptions.Durable`. Admitted topics require explicit durable options.

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

## Bounded durable events

Use `postgres.Config.Bounded` with `BoundedConfig{Topics: ..., Capacity: ...}` in
`NewBroker` and supply `SubscribeOptions.Durable` for named subscribers. This is
an explicit PostgreSQL capability; no-op and memory brokers provide no leases,
retained budgets or transactional publication guarantees. Root-config `New` keeps
legacy defaults. Upgraded API writers route already admitted topics through bounded
publication even when constructed with legacy config. A broker constructed with
`Bounded` rejects publication to unconfigured topics.

### Admission and identity

| Input | Bound |
| --- | --- |
| Topics, message IDs and named subscriber IDs | 1–128 ASCII bytes |
| Identity characters | Alphanumeric first; then letters, digits and `_.:/-` |
| Configured topics | At most 128 |
| Durable subscribers | At most 64 across topics |
| Payload JSON | At most 64 KiB |
| Metadata JSON | At most 8 KiB, including PostgreSQL JSONB text form |
| Retained bounded messages | Default 10000; configured 100–100000 |

`Envelope.Topic` must match the publication argument. Empty or display-text
identities are invalid. Metadata nil and an empty map share the same canonical
identity. JSON objects are canonicalized by key order without converting exact
numbers to floating point. Numeric spellings remain part of the encoded identity.
The digest includes topic, payload and metadata, and excludes `Timestamp`.
Bounded publication retains the first supplied `Envelope.Timestamp` separately
from the database publication time in `created_at`, which continues to govern
retention. Retries, restarts and idempotent duplicate publication preserve that
first occurrence time at PostgreSQL microsecond precision. Zero timestamps and
rows created before this additive upgrade fall back to their stored publication
time; source history cannot be recovered for older rows. Durable subscribers
receive the retained occurrence time. Delivery decodes JSON numbers as
`json.Number`, rather than legacy `float64`.

Repeating an existing identity with the same canonical envelope succeeds without
another row or capacity reservation. Different topic, payload or metadata returns
`ErrIdentityConflict`. Capacity is shared across all bounded topics and persists
across replicas/restarts. Capacity changes conflict with the retained configuration.
At the default capacity and maximum payload, payload data alone can approach 625 MiB,
plus metadata, indexes, acknowledgements and delivery rows.

`PublishTx(ctx, tx, topic, envelope)` accepts a caller-owned `*sql.Tx` against the
same schema. It never commits or rolls back that transaction. Domain state and the
message commit or roll back together. Roll back the caller transaction after any
publication error, and retain normal database transaction retry handling.
An open transaction reserves capacity until commit/rollback. Independent messages
reserve separate slots and can commit out of ID order. A duplicate at full capacity
still succeeds. `ErrCapacity` reports exhausted or currently reserved capacity;
it never drops older pending messages. `Publish` uses the same rules in its own
transaction. Publication does not wait for handlers.

Admission errors include `ErrEnvelope`, `ErrTopicNotAdmitted`, `ErrCapacity`,
`ErrIdentityConflict`, `ErrAdmissionLimit`, `ErrPolicyConflict` and
`ErrAdmissionUpgrade`. Use `errors.Is` for these classified outcomes. Existing
legacy rows are not adopted by repeated identity publication.

### Subscriber policy and claims

`DurableOptions` explicitly supplies `Policy` and `History`:

| DeliveryPolicy field | Default | Admitted values |
| --- | --- | --- |
| `HandlerTimeout` | 10 seconds | 1 second through 10 minutes |
| `LeaseDuration` | 30 seconds | Handler timeout plus 5 seconds through 15 minutes |
| `MaxAttempts` | 8 | 1–32 total attempts |
| `RetryDelay` | 30 seconds | 1 second through 1 hour |

Durations must be whole milliseconds. `DefaultDeliveryPolicy()` returns these
values. `History` must be `CurrentPublications` or `RetainedHistory`; zero is
invalid. A bounded subscriber requires a stable valid ID and non-nil handler.
`CurrentPublications` excludes only the topic history visible at registration;
`RetainedHistory` admits retained messages without replaying existing successful
acknowledgements. A lower ID committing later remains eligible in either mode.
No global commit order or strict processing order is promised.

A name remains bound to its original topic, policy and history admission. Reopening
with conflicting options returns `ErrSubscriptionConflict`. Explicit
`ConfigureDurableSubscription(ctx, id, topic, options)` upgrades a legacy named
subscription or changes the policy of an existing durable subscription for future
deliveries. It cannot change the topic or an established durable history mode.
Existing deliveries retain their original policy snapshot and cumulative attempts.

Each subscriber claims independently. First claim inserts `running` and a policy
snapshot atomically. Due retries and expired leases replace the token and increment
the same delivery's attempt. Database time controls deadlines. The last allowed
expired attempt becomes `failed` without invoking another handler. Only the live
generation can complete; an expired, changed or terminal generation returns
`ErrStaleDelivery`. Successful completion atomically inserts its acknowledgement,
updates diagnostic progress and records `succeeded`. A failed delivery has no success
acknowledgement and is excluded from polling, allowing later messages to progress.

Each poll fetches at most `min(BatchSize, 100)` candidate IDs and closes the result
set before claiming/invoking handlers. It holds no fetch transaction or connection
across handler work. Errors, recovered synchronous panics, elapsed cooperative
handler deadlines and malformed retained data consume the frozen attempt budget.
Retained codes are `handler_error`, `handler_panic`, `handler_deadline`,
`invalid_payload`, `invalid_metadata` and `lease_expired`; explicit retirement
can record `subscriber_retired` for unfinished attempts. No arbitrary error or
panic text is stored as diagnostic data.

Bounded polling inherits the `Subscribe` context; cancellation stops new claims
and propagates into the active handler. `Stop(ctx)` also cancels polling and bounds
the wait for the original handler. No heartbeat or replacement handler goroutine
is created. An unreturned handler can outlive its lease while another process
owns a later attempt. Applications must protect effects with transactional
deduplication or external idempotency. Transport success is not exactly-once
business-effect evidence. `Close()` remains an unbounded wait for compatible
lifecycle callers.

### Upgrade and retained storage

Stop old workers before applying the additive schema or new ownership semantics.
Legacy messages/subscriptions remain unbounded and existing acknowledgements stay
successful. Starting a bounded topic over legacy rows returns `ErrAdmissionUpgrade`.
After legacy startup applies the schema, explicitly call
`AdmitBoundedTopics(ctx, BoundedConfig)` to audit and convert retained messages.
The audit validates identities, JSON, metadata sizes, subscriber names/counts and
capacity in one transaction using bounded pages. Invalid data or insufficient
capacity aborts the whole audit; nothing is silently truncated or deleted.

Configure each legacy named subscriber explicitly before restarting durable workers.
Its successful acknowledgements remain excluded without synthetic attempt history.
Choose retained history to admit its remaining unacknowledged rows. Mixed old and
new workers are unsupported. Legacy polling and acknowledgement methods cannot
write durable delivery state.

The additive tables are `pubsub_bounded_config`, `pubsub_topics`,
`pubsub_capacity_slots` and `pubsub_deliveries`; messages gain bounded mode, digest
and a capacity reservation. Conditional constraints leave unrelated legacy rows
unchanged. Slot reservations roll back with publication and release through the
message foreign key when an eligible message is deleted. Failed delivery state
refers to the retained original message; there is no copied dead-letter payload.
This mode has no automatic cleanup. Reaching capacity returns `ErrCapacity`;
pending work is retained for explicit operator-owned retention.

## Retained-work operations

The PostgreSQL broker exposes bounded operator operations. Applications own their
authorization, caller contexts, retention cutoff and business-effect reconciliation.
No inspection result contains payload, metadata, a lease token or raw handler text.

### Inspect deliveries

`ListDeliveries(ctx, DeliveryQuery)` returns 50 rows by default, at most 100.
Filter by `SubscriberID`, `Topic` and `State`: `pending`, `running`, `expired`,
`retry_wait`, `succeeded`, `failed`, or empty for all. Invalid bounds or cursors
return `ErrWorkQuery`. A `DeliveryCursor` contains the last row's `SubscriberID`
and numeric `MessageID`; ordering is the stable `(subscriber_id, message_id)` key.
Concurrent changes do not provide a snapshot across multiple pages.

`DeliveryInfo` includes subscriber/message/envelope/topic identity, state, attempt,
attempt limit, creation/update/terminal/eligibility times, classified failure,
replay time and reason. `pending` means entitled but unclaimed, with zero attempt
and budget rather than fabricated history. An expired running lease is reported as
`expired`. Historical acknowledgements without a delivery row are excluded.
Retired subscribers expose their retained attempt rows without admitting new work.

### Replay a failed delivery

`ReplayDelivery(ctx, ReplayRequest)` requires the inspected subscriber/message key,
`Attempt` and `UpdatedAt`. It accepts only failed, unacknowledged, non-retired work.
`AdditionalAttempts` is 1–8 and the resulting lifetime attempt limit cannot exceed
32. The new limit is the observed attempt plus that grant; the cumulative attempt
counter, handler deadline, lease duration and retry delay are preserved.

`Reason` must be `ReplayOperator` (`operator_retry`), `ReplayDependency`
(`dependency_restored`) or `ReplayDataRepair` (`data_repaired`). A successful replay
records its reason/time, clears the terminal time and enters `retry_wait` with
immediate eligibility. It changes no business receipt. Missing, stale, acknowledged,
active or concurrently changed work returns `ErrReplayConflict`; invalid input or
a grant beyond the lifetime bound returns `ErrWorkQuery`. Only one concurrent
request can consume an observed revision. Old claim tokens cannot acknowledge it.

### Retire a subscriber

`RetireSubscriber(ctx, id, RetireOptions{})` refuses unfinished entitled work with
`ErrPendingDelivery`. After handlers are quiesced, the caller may explicitly set
`RetireOptions{DiscardPending: true}` to discard pending entitlement. Existing
active/retry attempts become terminal `failed` with `subscriber_retired`, preserving
their consumed counters and never inserting a successful acknowledgement.
Unclaimed work is not assigned invented attempt history.

Retirement atomically stops new entitlement and fences old claims. It cannot stop
a handler that ignores cancellation. Repeated retirement is idempotent. Retired
identities remain reserved with their original topic and count toward the 64-name
registration bound; they cannot be reopened or reconfigured (`ErrSubscriberRetired`).
New publications are not entitled to a retired subscriber. Retirement does not
automatically delete messages or tracking history.

### Prune terminal messages

`PruneMessages(ctx, before, limit)` removes at most 1–1000 eligible bounded messages
and their acknowledgement/delivery rows in one transaction, releasing capacity in
the same transaction. The cutoff must be nonzero. Use a context deadline. Repeat
bounded batches when needed; cutoff eligibility does not guarantee removal in one
call or start an automatic cleanup service.

A message must precede the cutoff, have no active/retry delivery, and have no terminal
delivery outcome newer than the cutoff. Every still-entitled subscriber must have
an acknowledgement or terminal delivery. This includes legacy subscribers awaiting
explicit upgrade. Current-only historical acknowledgements require no fabricated
attempt history. Explicitly retired subscribers waive entitlement as recorded by
the retirement operation. Unbounded legacy messages remain untouched.

Pruning serializes entitlement changes and locks candidate messages. It rechecks
eligibility in a fresh snapshot after acquiring those locks so concurrent replay
cannot cause active work to be deleted. Retention deletes tracking rows and frees
the same capacity reservation atomically; failures roll back both. A subscriber
registered after pruning can admit only the history still retained. Publication
idempotency/conflict detection applies while a message identity is retained;
applications own longer-lived domain identity and deduplication.

The retirement upgrade adds nullable `retired_at`; existing subscriptions stay
active. A delivery message index supports bounded retention lookups. Existing
schema upgrades, successful acknowledgements and attempt counters remain intact.
