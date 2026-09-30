<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Pubsub

`pubsub` fans a message out to every subscriber of a topic. Delivery is
at-least-once. Order depends on the backend. The implementation note is
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

`Start` requires `GetDB` and executes the package schema. A nil database
returns `database connection not available`. `Stop` calls `Close`.

`Publish` rejects a closed broker, JSON-encodes the payload and metadata, and
inserts `pubsub_messages`. `Subscribe` rejects a duplicate subscriber id. An
empty id becomes a UUID. A named subscriber resumes from its stored offset.
The poll loop uses a background context, not the subscribe context. A handler
error is backend-specific and does not remove the message by itself.
`Close` is idempotent and cancels the poll loops.
