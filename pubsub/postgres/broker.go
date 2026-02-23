package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hatmaxkit/hatmax/config"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/pubsub"
)

// Config holds PostgreSQL pubsub configuration.
type Config struct {
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
// Each subscriber maintains its own offset for fan-out semantics.
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

	_, err := b.db.ExecContext(ctx, Schema)
	if err != nil {
		return fmt.Errorf("cannot create pubsub schema: %w", err)
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
// Named subscribers resume from their last offset after restart.
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
		// Create new subscription starting from current position
		// This means new subscribers only see messages published after they subscribe
		var maxID sql.NullInt64

		err = b.db.QueryRowContext(ctx,
			"SELECT COALESCE(MAX(id), 0) FROM pubsub_messages WHERE topic = $1",
			topic,
		).Scan(&maxID)
		if err != nil {
			return 0, err
		}

		lastOffset = maxID.Int64

		_, err = b.db.ExecContext(ctx,
			`INSERT INTO pubsub_subscriptions (id, topic, last_message_id, created_at, updated_at)
			 VALUES ($1, $2, $3, NOW(), NOW())`,
			subscriberID, topic, lastOffset,
		)
		if err != nil {
			return 0, err
		}

		return lastOffset, nil
	}

	return lastOffset, err
}

func (b *Broker) pollLoop(ctx context.Context, sub *subscription) {
	defer close(sub.done)

	ticker := time.NewTicker(b.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := b.pollMessages(ctx, sub)
			if err != nil {
				b.log.Errorf("Poll error for subscriber %s: %v", sub.id, err)
			}
		}
	}
}

func (b *Broker) pollMessages(ctx context.Context, sub *subscription) error {
	query := `
		SELECT id, message_id, topic, payload, metadata, created_at
		FROM pubsub_messages
		WHERE topic = $1 AND id > $2
		ORDER BY id
		LIMIT $3
	`

	rows, err := b.db.QueryContext(ctx, query, sub.topic, sub.lastOffset, b.cfg.BatchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	var lastProcessedID int64

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

			lastProcessedID = id

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
			// Continue processing other messages
		}

		lastProcessedID = id
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	// Update offset if we processed any messages
	if lastProcessedID > sub.lastOffset {
		err := b.updateOffset(ctx, sub.id, lastProcessedID)
		if err != nil {
			return err
		}

		sub.lastOffset = lastProcessedID
	}

	return nil
}

func (b *Broker) updateOffset(ctx context.Context, subscriberID string, offset int64) error {
	_, err := b.db.ExecContext(ctx,
		"UPDATE pubsub_subscriptions SET last_message_id = $1, updated_at = NOW() WHERE id = $2",
		offset, subscriberID,
	)

	return err
}
