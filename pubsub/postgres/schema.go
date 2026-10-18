// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

// Schema contains the SQL migrations for the PostgreSQL pubsub backend.
const Schema = `
-- Messages table (append-only log)
CREATE TABLE IF NOT EXISTS pubsub_messages (
    id BIGSERIAL PRIMARY KEY,
    message_id TEXT UNIQUE NOT NULL,
    topic TEXT NOT NULL,
    payload BYTEA NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pubsub_messages_topic ON pubsub_messages(topic);
CREATE INDEX IF NOT EXISTS idx_pubsub_messages_topic_id ON pubsub_messages(topic, id);
CREATE INDEX IF NOT EXISTS idx_pubsub_messages_created_at ON pubsub_messages(created_at);

-- Legacy offsets remain diagnostic; acknowledgements determine pending delivery.
CREATE TABLE IF NOT EXISTS pubsub_subscriptions (
    id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    last_message_id BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pubsub_subscriptions_topic ON pubsub_subscriptions(topic);

ALTER TABLE pubsub_subscriptions
    ADD COLUMN IF NOT EXISTS acknowledgements_initialized BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS pubsub_acknowledgements (
    subscriber_id TEXT NOT NULL REFERENCES pubsub_subscriptions(id) ON DELETE CASCADE,
    message_id BIGINT NOT NULL REFERENCES pubsub_messages(id) ON DELETE CASCADE,
    PRIMARY KEY (subscriber_id, message_id)
);

-- Seed visible legacy history once. Later low-ID commits must not be reseeded.
WITH initialized AS (
    UPDATE pubsub_subscriptions
    SET acknowledgements_initialized = TRUE
    WHERE NOT acknowledgements_initialized
    RETURNING id, topic, last_message_id
)
INSERT INTO pubsub_acknowledgements (subscriber_id, message_id)
SELECT s.id, m.id
FROM initialized s
JOIN pubsub_messages m ON m.topic = s.topic AND m.id <= s.last_message_id
ON CONFLICT DO NOTHING;
` + BoundedSchema

// BoundedSchema adds opt-in admission without rewriting legacy messages.
const BoundedSchema = `
CREATE TABLE IF NOT EXISTS pubsub_bounded_config (
 singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK(singleton),
 capacity INT NOT NULL CHECK(capacity BETWEEN 100 AND 100000)
);
CREATE TABLE IF NOT EXISTS pubsub_topics (
 topic TEXT PRIMARY KEY CHECK(octet_length(topic) BETWEEN 1 AND 128 AND topic ~ '^[A-Za-z0-9][A-Za-z0-9_.:/-]*$')
);
CREATE TABLE IF NOT EXISTS pubsub_capacity_slots (
 slot INT PRIMARY KEY,
 message_id BIGINT UNIQUE REFERENCES pubsub_messages(id) ON DELETE SET NULL
);
ALTER TABLE pubsub_messages ADD COLUMN IF NOT EXISTS bounded BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE pubsub_messages ADD COLUMN IF NOT EXISTS envelope_digest BYTEA;
ALTER TABLE pubsub_messages ADD COLUMN IF NOT EXISTS capacity_slot INT UNIQUE REFERENCES pubsub_capacity_slots(slot);
DO $$ BEGIN
 ALTER TABLE pubsub_messages ADD CONSTRAINT pubsub_bounded_message CHECK(NOT bounded OR (
 envelope_digest IS NOT NULL AND octet_length(envelope_digest)=32 AND capacity_slot IS NOT NULL
 AND octet_length(payload)<=65536 AND metadata IS NOT NULL AND octet_length(metadata::text)<=8192
 AND octet_length(message_id) BETWEEN 1 AND 128 AND message_id ~ '^[A-Za-z0-9][A-Za-z0-9_.:/-]*$'
 AND octet_length(topic) BETWEEN 1 AND 128 AND topic ~ '^[A-Za-z0-9][A-Za-z0-9_.:/-]*$'
 ));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
`
