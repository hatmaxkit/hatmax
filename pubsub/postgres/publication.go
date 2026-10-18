// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"hatmax.adrianpk.com/pubsub"
)

var (
	ErrEnvelope         = errors.New("pubsub: invalid bounded envelope")
	ErrCapacity         = errors.New("pubsub: retained message capacity exhausted")
	ErrIdentityConflict = errors.New("pubsub: message identity conflict")
	ErrTopicNotAdmitted = errors.New("pubsub: topic is not admitted for bounded delivery")
	ErrAdmissionUpgrade = errors.New("pubsub: explicit admission audit required")
	ErrPolicyConflict   = errors.New("pubsub: conflicting retained policy")
	ErrAdmissionLimit   = errors.New("pubsub: topic or subscriber limit exceeded")
)

// BoundedConfig admits up to 128 topics and fixes a shared capacity of 100–100000
// retained bounded messages (10000 by default). All replicas use the same capacity.
type BoundedConfig struct {
	Topics   []string
	Capacity int
}

func validIdentity(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}

	for i, c := range []byte(value) {
		alpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !alpha && (i == 0 || (c != '_' && c != '.' && c != ':' && c != '/' && c != '-')) {
			return false
		}
	}

	return true
}

func canonicalJSON(data []byte) ([]byte, error) {
	if !utf8.Valid(data) {
		return nil, ErrEnvelope
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var value any

	err := decoder.Decode(&value)
	if err != nil {
		return nil, ErrEnvelope
	}

	err = decoder.Decode(new(any))
	if err != io.EOF {
		return nil, ErrEnvelope
	}

	return json.Marshal(value)
}

func encodeBounded(topic string, env pubsub.Envelope) ([]byte, []byte, [32]byte, error) {
	var empty [32]byte
	if !validIdentity(topic) || !validIdentity(env.ID) || env.Topic != topic {
		return nil, nil, empty, ErrEnvelope
	}

	payload, err := json.Marshal(env.Payload)
	if err != nil || len(payload) > 65536 {
		return nil, nil, empty, ErrEnvelope
	}

	payload, err = canonicalJSON(payload)
	if err != nil || len(payload) > 65536 {
		return nil, nil, empty, ErrEnvelope
	}

	metadata := env.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}

	encoded, err := json.Marshal(metadata)
	if err != nil || len(encoded) > 8192 {
		return nil, nil, empty, ErrEnvelope
	}

	data, err := json.Marshal(struct {
		Topic    string
		Payload  json.RawMessage
		Metadata json.RawMessage
	}{topic, payload, encoded})
	if err != nil {
		return nil, nil, empty, ErrEnvelope
	}

	return payload, encoded, sha256.Sum256(data), nil
}

func (b *Broker) boundedTopic(ctx context.Context, topic string) (bool, error) {
	if b.db == nil {
		return false, ErrTopicNotAdmitted
	}

	var admitted bool

	err := b.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pubsub_topics WHERE topic=$1)`, topic).Scan(&admitted)

	return admitted, err
}

// PublishTx admits bounded publication in a caller-owned transaction. It never
// commits or rolls back tx. Exact duplicates are idempotent, including at capacity.
// The caller must roll back a transaction after any publication error.
func (b *Broker) PublishTx(ctx context.Context, tx *sql.Tx, topic string, env pubsub.Envelope) error {
	if tx == nil {
		return ErrEnvelope
	}

	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()

	if closed {
		return errors.New("pubsub: broker is closed")
	}

	payload, metadata, digest, err := encodeBounded(topic, env)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `LOCK TABLE pubsub_messages IN ROW EXCLUSIVE MODE`)
	if err != nil {
		return err
	}

	var admitted bool

	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pubsub_topics WHERE topic=$1)`, topic).Scan(&admitted)
	if err != nil {
		return err
	}

	if !admitted {
		return ErrTopicNotAdmitted
	}
	// Serialize only this message identity. Different publications can reserve
	// different slots and commit out of ID order, even in caller-owned transactions.
	_, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(current_schema() || ':pubsub:message:' || $1,0))`, env.ID)
	if err != nil {
		return err
	}

	var (
		oldDigest []byte
		bounded   bool
	)

	err = tx.QueryRowContext(ctx, `SELECT bounded,envelope_digest FROM pubsub_messages WHERE message_id=$1`, env.ID).Scan(&bounded, &oldDigest)
	if err == nil {
		if bounded && bytes.Equal(oldDigest, digest[:]) {
			return nil
		}

		return ErrIdentityConflict
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	var metadataSize int

	err = tx.QueryRowContext(ctx, `SELECT octet_length($1::jsonb::text)`, metadata).Scan(&metadataSize)
	if err != nil {
		return err
	}

	if metadataSize > 8192 {
		return ErrEnvelope
	}

	var slot int

	err = tx.QueryRowContext(ctx, `SELECT slot FROM pubsub_capacity_slots WHERE message_id IS NULL ORDER BY slot FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&slot)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrCapacity
	}

	if err != nil {
		return err
	}

	var id int64

	err = tx.QueryRowContext(ctx, `INSERT INTO pubsub_messages(message_id,topic,payload,metadata,created_at,bounded,envelope_digest,capacity_slot)
 VALUES($1,$2,$3,$4,clock_timestamp(),true,$5,$6) RETURNING id`, env.ID, topic, payload, metadata, digest[:], slot).Scan(&id)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE pubsub_capacity_slots SET message_id=$2 WHERE slot=$1`, slot, id)

	return err
}

// AdmitBoundedTopics explicitly audits and converts retained legacy messages.
// Quiesce old workers first. Existing acknowledgements and subscription modes
// remain intact; subscription policy upgrade is a separate explicit operation.
func (b *Broker) AdmitBoundedTopics(ctx context.Context, cfg BoundedConfig) error {
	return b.configureBounded(ctx, cfg, true)
}

func (b *Broker) configureBounded(ctx context.Context, cfg BoundedConfig, upgrade bool) error {
	if cfg.Capacity == 0 {
		cfg.Capacity = 10000
	}

	if cfg.Capacity < 100 || cfg.Capacity > 100000 || len(cfg.Topics) < 1 || len(cfg.Topics) > 128 {
		return ErrAdmissionLimit
	}

	topics := make(map[string]bool, len(cfg.Topics))
	for _, topic := range cfg.Topics {
		if !validIdentity(topic) || topics[topic] {
			return ErrEnvelope
		}

		topics[topic] = true
	}

	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO pubsub_bounded_config(singleton,capacity) VALUES(true,$1) ON CONFLICT DO NOTHING`, cfg.Capacity)
	if err != nil {
		return err
	}

	var capacity int

	err = tx.QueryRowContext(ctx, `SELECT capacity FROM pubsub_bounded_config WHERE singleton FOR UPDATE`).Scan(&capacity)
	if err != nil {
		return err
	}

	if capacity != cfg.Capacity {
		return ErrPolicyConflict
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO pubsub_capacity_slots(slot) SELECT generate_series(1,$1) ON CONFLICT DO NOTHING`, capacity)
	if err != nil {
		return err
	}

	var count int

	err = tx.QueryRowContext(ctx, `SELECT count(*) FROM pubsub_topics`).Scan(&count)
	if err != nil {
		return err
	}

	for _, topic := range cfg.Topics {
		var exists bool

		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pubsub_topics WHERE topic=$1)`, topic).Scan(&exists)
		if err != nil {
			return err
		}

		if exists {
			continue
		}

		if count >= 128 {
			return ErrAdmissionLimit
		}
		// Stop old publishers before this operator-owned lock/audit. New API writers
		// wait here, then observe the admitted topic in their own transaction.
		_, err = tx.ExecContext(ctx, `LOCK TABLE pubsub_messages IN SHARE ROW EXCLUSIVE MODE`)
		if err != nil {
			return err
		}

		var legacy int

		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM pubsub_messages WHERE topic=$1 AND NOT bounded`, topic).Scan(&legacy)
		if err != nil {
			return err
		}

		if legacy > 0 && !upgrade {
			return ErrAdmissionUpgrade
		}

		if legacy > capacity {
			return ErrCapacity
		}

		err = b.auditTopic(ctx, tx, topic, upgrade)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `INSERT INTO pubsub_topics(topic) VALUES($1)`, topic)
		if err != nil {
			return err
		}

		count++
	}

	return tx.Commit()
}

func (b *Broker) auditTopic(ctx context.Context, tx *sql.Tx, topic string, upgrade bool) error {
	var subscriptions int

	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM pubsub_subscriptions WHERE topic IN (SELECT topic FROM pubsub_topics) OR topic=$1`, topic).Scan(&subscriptions)
	if err != nil {
		return err
	}

	if subscriptions > 64 {
		return ErrAdmissionLimit
	}

	var (
		existing int
		invalid  int
	)

	err = tx.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER(WHERE octet_length(id) NOT BETWEEN 1 AND 128 OR id !~ '^[A-Za-z0-9][A-Za-z0-9_.:/-]*$') FROM pubsub_subscriptions WHERE topic=$1`, topic).Scan(&existing, &invalid)
	if err != nil {
		return err
	}

	if existing > 0 && !upgrade {
		return ErrAdmissionUpgrade
	}

	if invalid > 0 {
		return ErrEnvelope
	}
	// Audit a bounded page at a time; the upgrade transaction owns the table lock.
	var after int64
	for {
		messages, err := auditMessages(ctx, tx, topic, after)
		if err != nil {
			return err
		}

		if len(messages) == 0 {
			return nil
		}

		for _, msg := range messages {
			payload, metadata, digest, err := encodeBounded(topic, msg.env)
			if err != nil {
				return err
			}

			var metadataSize int

			err = tx.QueryRowContext(ctx, `SELECT octet_length($1::jsonb::text)`, metadata).Scan(&metadataSize)
			if err != nil {
				return err
			}

			if metadataSize > 8192 {
				return ErrEnvelope
			}

			var slot int

			err = tx.QueryRowContext(ctx, `SELECT slot FROM pubsub_capacity_slots WHERE message_id IS NULL ORDER BY slot FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&slot)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrCapacity
			}

			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx, `UPDATE pubsub_messages SET payload=$2,metadata=$3,bounded=true,envelope_digest=$4,capacity_slot=$5 WHERE id=$1`, msg.id, payload, metadata, digest[:], slot)
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx, `UPDATE pubsub_capacity_slots SET message_id=$2 WHERE slot=$1`, slot, msg.id)
			if err != nil {
				return err
			}

			after = msg.id
		}
	}
}

type auditMessage struct {
	id  int64
	env pubsub.Envelope
}

func auditMessages(ctx context.Context, tx *sql.Tx, topic string, after int64) ([]auditMessage, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,message_id,payload,metadata FROM pubsub_messages WHERE topic=$1 AND NOT bounded AND id>$2 ORDER BY id LIMIT 100`, topic, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []auditMessage

	for rows.Next() {
		var (
			msg               auditMessage
			payload, metadata []byte
		)

		err = rows.Scan(&msg.id, &msg.env.ID, &payload, &metadata)
		if err != nil {
			return nil, err
		}

		msg.env.Topic = topic
		msg.env.Payload = json.RawMessage(payload)

		err = json.Unmarshal(metadata, &msg.env.Metadata)
		if err != nil {
			return nil, ErrEnvelope
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}
