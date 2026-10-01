// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/pubsub"
)

// Config holds PostgreSQL pubsub configuration.
type Config struct {
	// PollInterval is the delay before the first poll and between completed polls.
	PollInterval time.Duration
	BatchSize    int
}

// WithDefaults applies defaults for zero values.
func (c Config) WithDefaults() Config {
	if c.PollInterval <= 0 {
		c.PollInterval = 100 * time.Millisecond
	}

	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}

	return c
}

// DefaultConfig returns sensible defaults for PostgreSQL pubsub.
func DefaultConfig() Config {
	return Config{}.WithDefaults()
}

// subscription holds a registered handler and its tracking state.
type subscription struct {
	id         string
	topic      string
	handler    pubsub.Handler
	lastOffset int64
	cancel     context.CancelFunc
	done       chan struct{}
}

// DBProvider provides access to the database connection.
// This allows the broker to be created before the database is started.
type DBProvider interface {
	GetDB() *sql.DB
}

// Broker implements pubsub.Broker using PostgreSQL.
// Messages are stored in an append-only table and delivered via polling.
// Each subscriber maintains its own message acknowledgements for fan-out.
type Broker struct {
	dbProvider    DBProvider
	db            *sql.DB
	cfg           Config
	log           log.Logger
	mu            sync.RWMutex
	subscriptions map[string]*subscription // keyed by subscriber ID
	closed        bool
}

// New creates a new PostgreSQL-backed pubsub broker using application config.
func New(dbProvider DBProvider, cfg *config.Config, log log.Logger) *Broker {
	return NewBroker(dbProvider, Config{
		PollInterval: cfg.PubSub.PollIntervalDuration(),
		BatchSize:    cfg.PubSub.BatchSize,
	}, log)
}

// NewBroker creates a new PostgreSQL-backed pubsub broker with explicit config.
func NewBroker(dbProvider DBProvider, cfg Config, log log.Logger) *Broker {
	cfg = cfg.WithDefaults()

	return &Broker{
		dbProvider:    dbProvider,
		cfg:           cfg,
		log:           log.With("component", "pubsub"),
		subscriptions: make(map[string]*subscription),
	}
}

// Start initializes the pubsub schema.
// Implements app.Startable interface.
func (b *Broker) Start(ctx context.Context) error {
	b.db = b.dbProvider.GetDB()
	if b.db == nil {
		return fmt.Errorf("database connection not available")
	}

	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot begin pubsub schema migration: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, Schema)
	if err != nil {
		return fmt.Errorf("cannot create pubsub schema: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("cannot commit pubsub schema migration: %w", err)
	}

	b.log.Info("PubSub schema initialized")

	return nil
}

// Stop gracefully shuts down all subscriptions.
// Implements app.Stoppable interface.
func (b *Broker) Stop(ctx context.Context) error {
	return b.Close()
}

// Publish stores a message and makes it available to all subscribers.
func (b *Broker) Publish(ctx context.Context, topic string, env pubsub.Envelope) error {
	b.mu.RLock()

	if b.closed {
		b.mu.RUnlock()

		return fmt.Errorf("broker is closed")
	}

	b.mu.RUnlock()

	payload, err := json.Marshal(env.Payload)
	if err != nil {
		return fmt.Errorf("cannot marshal payload: %w", err)
	}

	metadata, err := json.Marshal(env.Metadata)
	if err != nil {
		return fmt.Errorf("cannot marshal metadata: %w", err)
	}

	query := `
		INSERT INTO pubsub_messages (message_id, topic, payload, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = b.db.ExecContext(ctx, query, env.ID, topic, payload, metadata, env.Timestamp)
	if err != nil {
		return fmt.Errorf("cannot insert message: %w", err)
	}

	b.log.Debugf("Published message %s to topic %s", env.ID, topic)

	return nil
}

// Subscribe registers a handler for the given topic.
// Each call creates a new subscription with fan-out semantics.
// If opts.SubscriberID is empty, a UUID is generated (ephemeral subscription).
// Named subscribers resume their unacknowledged messages after restart.
func (b *Broker) Subscribe(ctx context.Context, topic string, handler pubsub.Handler, opts pubsub.SubscribeOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return fmt.Errorf("broker is closed")
	}

	subscriberID := opts.SubscriberID
	if subscriberID == "" {
		subscriberID = uuid.New().String()
	}

	// Check for duplicate subscriber ID
	if _, exists := b.subscriptions[subscriberID]; exists {
		return fmt.Errorf("subscriber %s already registered", subscriberID)
	}

	// Get or create subscription record
	lastOffset, err := b.getOrCreateSubscription(ctx, subscriberID, topic)
	if err != nil {
		return fmt.Errorf("cannot initialize subscription: %w", err)
	}

	subCtx, cancel := context.WithCancel(context.Background())
	sub := &subscription{
		id:         subscriberID,
		topic:      topic,
		handler:    handler,
		lastOffset: lastOffset,
		cancel:     cancel,
		done:       make(chan struct{}),
	}

	b.subscriptions[subscriberID] = sub

	// Start polling goroutine
	go b.pollLoop(subCtx, sub)

	b.log.Infof("Subscriber %s registered for topic %s (offset: %d)", subscriberID, topic, lastOffset)

	return nil
}

// Close stops all subscriptions and releases resources.
func (b *Broker) Close() error {
	b.mu.Lock()

	if b.closed {
		b.mu.Unlock()

		return nil
	}

	b.closed = true

	// Cancel all subscription contexts
	for _, sub := range b.subscriptions {
		sub.cancel()
	}

	subs := make([]*subscription, 0, len(b.subscriptions))
	for _, sub := range b.subscriptions {
		subs = append(subs, sub)
	}

	b.mu.Unlock()

	// Wait for all poll loops to finish
	for _, sub := range subs {
		<-sub.done
	}

	b.log.Info("PubSub broker closed")

	return nil
}

func (b *Broker) getOrCreateSubscription(ctx context.Context, subscriberID, topic string) (int64, error) {
	var lastOffset int64

	// Try to get existing subscription
	err := b.db.QueryRowContext(ctx,
		"SELECT last_message_id FROM pubsub_subscriptions WHERE id = $1",
		subscriberID,
	).Scan(&lastOffset)

	if err == sql.ErrNoRows {
		// One statement snapshot excludes only committed history visible now.
		// In-flight messages, including lower IDs, remain eligible after commit.
		err = b.db.QueryRowContext(ctx, `
			WITH registered AS (
				INSERT INTO pubsub_subscriptions
					(id, topic, last_message_id, acknowledgements_initialized)
				VALUES ($1, $2,
					(SELECT COALESCE(MAX(id), 0) FROM pubsub_messages WHERE topic = $2), TRUE)
				RETURNING id, last_message_id
			), history AS (
				INSERT INTO pubsub_acknowledgements (subscriber_id, message_id)
				SELECT s.id, m.id FROM registered s
				JOIN pubsub_messages m ON m.topic = $2
			)
			SELECT last_message_id FROM registered`, subscriberID, topic,
		).Scan(&lastOffset)
	}

	return lastOffset, err
}

func (b *Broker) pollLoop(ctx context.Context, sub *subscription) {
	defer close(sub.done)

	timer := time.NewTimer(b.cfg.PollInterval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			err := b.pollMessages(ctx, sub)
			if err != nil {
				b.log.Errorf("Poll error for subscriber %s: %v", sub.id, err)
			}

			// Wait after completion; slow failed handlers cannot queue an immediate retry.
			timer.Reset(b.cfg.PollInterval)
		}
	}
}

func (b *Broker) pollMessages(ctx context.Context, sub *subscription) error {
	query := `
		SELECT m.id, m.message_id, m.topic, m.payload, m.metadata, m.created_at
		FROM pubsub_messages m
		WHERE m.topic = $1 AND NOT EXISTS (
			SELECT 1 FROM pubsub_acknowledgements a
			WHERE a.subscriber_id = $2 AND a.message_id = m.id
		)
		ORDER BY m.id
		LIMIT $3
	`

	rows, err := b.db.QueryContext(ctx, query, sub.topic, sub.id, b.cfg.BatchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	var successfulIDs []int64

	lastProcessedID := sub.lastOffset

	for rows.Next() {
		var (
			id        int64
			messageID string
			topic     string
			payload   []byte
			metadata  []byte
			createdAt time.Time
		)

		err := rows.Scan(&id, &messageID, &topic, &payload, &metadata, &createdAt)
		if err != nil {
			return err
		}

		var payloadData any

		err = json.Unmarshal(payload, &payloadData)
		if err != nil {
			b.log.Errorf("Cannot unmarshal payload for message %s: %v", messageID, err)

			continue
		}

		var metadataMap map[string]string

		err = json.Unmarshal(metadata, &metadataMap)
		if err != nil {
			metadataMap = make(map[string]string)
		}

		env := pubsub.Envelope{
			ID:        messageID,
			Topic:     topic,
			Timestamp: createdAt,
			Payload:   payloadData,
			Metadata:  metadataMap,
		}

		err = sub.handler(ctx, env)
		if err != nil {
			b.log.Errorf("Handler error for message %s: %v", messageID, err)

			continue
		}

		successfulIDs = append(successfulIDs, id)
		lastProcessedID = max(lastProcessedID, id)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	// Persist exact rows, even when their IDs are below the diagnostic offset.
	if len(successfulIDs) > 0 {
		err := b.acknowledgeBatch(ctx, sub.id, successfulIDs, lastProcessedID)
		if err != nil {
			return err
		}

		sub.lastOffset = lastProcessedID
	}

	return nil
}

func (b *Broker) acknowledgeBatch(ctx context.Context, subscriberID string, ids []int64, offset int64) error {
	// A single statement commits acknowledgements and diagnostic progress together.
	_, err := b.db.ExecContext(ctx, `
		WITH acknowledged AS (
			INSERT INTO pubsub_acknowledgements (subscriber_id, message_id)
			SELECT $1, UNNEST($2::bigint[])
			ON CONFLICT DO NOTHING
		)
		UPDATE pubsub_subscriptions
		SET last_message_id = GREATEST(last_message_id, $3), updated_at = NOW()
		WHERE id = $1`, subscriberID, pq.Array(ids), offset)

	return err
}
