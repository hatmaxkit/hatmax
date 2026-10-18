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
` + BoundedSchema + DeliverySchema

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
ALTER TABLE pubsub_messages ADD COLUMN IF NOT EXISTS occurred_at TIMESTAMPTZ;
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

// DeliverySchema keeps successful legacy acknowledgements and adds fenced attempts.
const DeliverySchema = `
ALTER TABLE pubsub_subscriptions ADD COLUMN IF NOT EXISTS retired_at TIMESTAMPTZ;
ALTER TABLE pubsub_subscriptions ADD COLUMN IF NOT EXISTS bounded BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE pubsub_subscriptions ADD COLUMN IF NOT EXISTS handler_timeout_ms INT;
ALTER TABLE pubsub_subscriptions ADD COLUMN IF NOT EXISTS lease_duration_ms INT;
ALTER TABLE pubsub_subscriptions ADD COLUMN IF NOT EXISTS attempt_limit INT;
ALTER TABLE pubsub_subscriptions ADD COLUMN IF NOT EXISTS retry_delay_ms INT;
ALTER TABLE pubsub_subscriptions ADD COLUMN IF NOT EXISTS history_mode TEXT;
DO $$ BEGIN
 ALTER TABLE pubsub_subscriptions ADD CONSTRAINT pubsub_bounded_subscription CHECK(NOT bounded OR (
 octet_length(id) BETWEEN 1 AND 128 AND id ~ '^[A-Za-z0-9][A-Za-z0-9_.:/-]*$'
 AND handler_timeout_ms IS NOT NULL AND handler_timeout_ms BETWEEN 1000 AND 600000
 AND lease_duration_ms IS NOT NULL AND lease_duration_ms BETWEEN handler_timeout_ms+5000 AND 900000
 AND attempt_limit IS NOT NULL AND attempt_limit BETWEEN 1 AND 32
 AND retry_delay_ms IS NOT NULL AND retry_delay_ms BETWEEN 1000 AND 3600000
 AND history_mode IS NOT NULL AND history_mode IN ('current','retained')
 ));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
CREATE TABLE IF NOT EXISTS pubsub_deliveries (
 subscriber_id TEXT NOT NULL REFERENCES pubsub_subscriptions(id) ON DELETE CASCADE,
 message_id BIGINT NOT NULL REFERENCES pubsub_messages(id) ON DELETE CASCADE,
 state TEXT NOT NULL CHECK(state IN ('running','retry_wait','succeeded','failed')),
 attempt INT NOT NULL CHECK(attempt BETWEEN 1 AND 32),
 attempt_limit INT NOT NULL CHECK(attempt_limit BETWEEN 1 AND 32 AND attempt_limit>=attempt),
 handler_timeout_ms INT NOT NULL CHECK(handler_timeout_ms BETWEEN 1000 AND 600000),
 lease_duration_ms INT NOT NULL CHECK(lease_duration_ms BETWEEN handler_timeout_ms+5000 AND 900000),
 retry_delay_ms INT NOT NULL CHECK(retry_delay_ms BETWEEN 1000 AND 3600000),
 lease_token TEXT,
 lease_expires_at TIMESTAMPTZ,
 retry_at TIMESTAMPTZ,
 failure_code TEXT CHECK(failure_code IS NULL OR failure_code ~ '^[a-z_]{1,64}$'),
 replayed_at TIMESTAMPTZ,
 replay_reason TEXT CHECK(replay_reason IS NULL OR replay_reason ~ '^[a-z_]{1,64}$'),
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 finished_at TIMESTAMPTZ,
 PRIMARY KEY(subscriber_id,message_id),
 CHECK((state='running' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL AND retry_at IS NULL AND finished_at IS NULL)
 OR (state='retry_wait' AND lease_token IS NULL AND lease_expires_at IS NULL AND retry_at IS NOT NULL AND finished_at IS NULL)
 OR (state IN ('succeeded','failed') AND lease_token IS NULL AND lease_expires_at IS NULL AND retry_at IS NULL AND finished_at IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS idx_pubsub_delivery_message ON pubsub_deliveries(message_id);
CREATE INDEX IF NOT EXISTS idx_pubsub_delivery_retry ON pubsub_deliveries(subscriber_id,retry_at) WHERE state='retry_wait';
CREATE INDEX IF NOT EXISTS idx_pubsub_delivery_lease ON pubsub_deliveries(subscriber_id,lease_expires_at) WHERE state='running';
CREATE INDEX IF NOT EXISTS idx_pubsub_delivery_terminal ON pubsub_deliveries(finished_at,subscriber_id,message_id) WHERE state IN ('succeeded','failed');
`
