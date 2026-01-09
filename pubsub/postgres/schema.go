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
CREATE INDEX IF NOT EXISTS idx_pubsub_messages_created_at ON pubsub_messages(created_at);

-- Subscriber offsets (per-subscriber cursor for fan-out)
CREATE TABLE IF NOT EXISTS pubsub_subscriptions (
    id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    last_message_id BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pubsub_subscriptions_topic ON pubsub_subscriptions(topic);
`
