package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hatmaxkit/hatmax/pubsub"
	"github.com/hatmaxkit/hatmax/testhelper"
)

// testDBProvider wraps *sql.DB to implement DBProvider for tests.
type testDBProvider struct {
	db *sql.DB
}

func (t *testDBProvider) GetDB() *sql.DB {
	return t.db
}

func wrapDB(db *sql.DB) DBProvider {
	return &testDBProvider{db: db}
}

func TestBrokerStartCreatesSchema(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := NewBroker(wrapDB(db), DefaultConfig(), testhelper.TestLogger())

	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}
	defer broker.Close()

	// Verify tables were created
	var count int

	err := db.QueryRow("SELECT COUNT(*) FROM pubsub_messages").Scan(&count)
	if err != nil {
		t.Fatalf("pubsub_messages table not created: %v", err)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM pubsub_subscriptions").Scan(&count)
	if err != nil {
		t.Fatalf("pubsub_subscriptions table not created: %v", err)
	}
}

func TestBrokerPublishStoresMessage(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := NewBroker(wrapDB(db), DefaultConfig(), testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}
	defer broker.Close()

	env := pubsub.NewEnvelope("test-topic", "test-payload")

	publishErr := broker.Publish(ctx, "test-topic", env)
	if publishErr != nil {
		t.Fatalf("Publish failed: %v", publishErr)
	}

	// Verify message was stored
	var storedID string

	err := db.QueryRow("SELECT message_id FROM pubsub_messages WHERE topic = $1", "test-topic").Scan(&storedID)
	if err != nil {
		t.Fatalf("Message not stored: %v", err)
	}

	if storedID != env.ID {
		t.Errorf("expected message_id %s, got %s", env.ID, storedID)
	}
}

func TestBrokerSubscribeDeliversMessages(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	cfg := DefaultConfig()
	cfg.PollInterval = 10 * time.Millisecond // Speed up for tests

	broker := NewBroker(wrapDB(db), cfg, testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}
	defer broker.Close()

	var (
		received []pubsub.Envelope
		mu       sync.Mutex
	)

	handler := func(ctx context.Context, env pubsub.Envelope) error {
		mu.Lock()

		received = append(received, env)

		mu.Unlock()

		return nil
	}

	// Subscribe first
	err := broker.Subscribe(ctx, "test-topic", handler, pubsub.SubscribeOptions{
		SubscriberID: "test-sub",
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Then publish
	env := pubsub.NewEnvelope("test-topic", "test-payload")

	publishErr := broker.Publish(ctx, "test-topic", env)
	if publishErr != nil {
		t.Fatalf("Publish failed: %v", publishErr)
	}

	// Wait for delivery
	time.Sleep(50 * time.Millisecond)

	mu.Lock()

	count := len(received)

	mu.Unlock()

	if count != 1 {
		t.Fatalf("expected 1 received message, got %d", count)
	}

	if received[0].ID != env.ID {
		t.Errorf("expected ID %s, got %s", env.ID, received[0].ID)
	}
}

func TestBrokerFanOut(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	cfg := DefaultConfig()
	cfg.PollInterval = 10 * time.Millisecond

	broker := NewBroker(wrapDB(db), cfg, testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}
	defer broker.Close()

	var (
		count1, count2 int
		mu             sync.Mutex
	)

	handler1 := func(ctx context.Context, env pubsub.Envelope) error {
		mu.Lock()

		count1++

		mu.Unlock()

		return nil
	}

	handler2 := func(ctx context.Context, env pubsub.Envelope) error {
		mu.Lock()

		count2++

		mu.Unlock()

		return nil
	}

	// Two subscribers to same topic
	broker.Subscribe(ctx, "test-topic", handler1, pubsub.SubscribeOptions{SubscriberID: "sub1"})
	broker.Subscribe(ctx, "test-topic", handler2, pubsub.SubscribeOptions{SubscriberID: "sub2"})

	// Publish one message
	broker.Publish(ctx, "test-topic", pubsub.NewEnvelope("test-topic", "payload"))

	// Wait for delivery
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if count1 != 1 {
		t.Errorf("handler1 expected 1 call, got %d", count1)
	}

	if count2 != 1 {
		t.Errorf("handler2 expected 1 call, got %d", count2)
	}
}

func TestBrokerDifferentTopics(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	cfg := DefaultConfig()
	cfg.PollInterval = 10 * time.Millisecond

	broker := NewBroker(wrapDB(db), cfg, testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}
	defer broker.Close()

	var (
		received1, received2 int
		mu                   sync.Mutex
	)

	broker.Subscribe(ctx, "topic1", func(ctx context.Context, env pubsub.Envelope) error {
		mu.Lock()

		received1++

		mu.Unlock()

		return nil
	}, pubsub.SubscribeOptions{SubscriberID: "sub1"})

	broker.Subscribe(ctx, "topic2", func(ctx context.Context, env pubsub.Envelope) error {
		mu.Lock()

		received2++

		mu.Unlock()

		return nil
	}, pubsub.SubscribeOptions{SubscriberID: "sub2"})

	broker.Publish(ctx, "topic1", pubsub.NewEnvelope("topic1", "payload"))

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if received1 != 1 {
		t.Errorf("topic1 handler expected 1 call, got %d", received1)
	}

	if received2 != 0 {
		t.Errorf("topic2 handler expected 0 calls, got %d", received2)
	}
}

func TestBrokerNamedSubscriberResumes(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	cfg := DefaultConfig()
	cfg.PollInterval = 10 * time.Millisecond

	ctx := context.Background()

	// First broker instance
	broker1 := NewBroker(wrapDB(db), cfg, testhelper.TestLogger())

	startErr := broker1.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}

	var (
		count1 int
		mu     sync.Mutex
	)

	broker1.Subscribe(ctx, "test-topic", func(ctx context.Context, env pubsub.Envelope) error {
		mu.Lock()

		count1++

		mu.Unlock()

		return nil
	}, pubsub.SubscribeOptions{SubscriberID: "persistent-sub"})

	broker1.Publish(ctx, "test-topic", pubsub.NewEnvelope("test-topic", "msg1"))
	time.Sleep(50 * time.Millisecond)

	broker1.Close()

	// Second broker instance (simulating restart)
	broker2 := NewBroker(wrapDB(db), cfg, testhelper.TestLogger())

	startErr = broker2.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}
	defer broker2.Close()

	var count2 int

	broker2.Subscribe(ctx, "test-topic", func(ctx context.Context, env pubsub.Envelope) error {
		mu.Lock()

		count2++

		mu.Unlock()

		return nil
	}, pubsub.SubscribeOptions{SubscriberID: "persistent-sub"})

	// Publish another message
	broker2.Publish(ctx, "test-topic", pubsub.NewEnvelope("test-topic", "msg2"))
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if count1 != 1 {
		t.Errorf("first broker expected 1 message, got %d", count1)
	}
	// Second broker should only receive msg2, not msg1 (resumed from offset)
	if count2 != 1 {
		t.Errorf("second broker expected 1 message (resumed), got %d", count2)
	}
}

func TestBrokerEphemeralSubscriberGetsNewID(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	cfg := DefaultConfig()
	cfg.PollInterval = 10 * time.Millisecond

	broker := NewBroker(wrapDB(db), cfg, testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}
	defer broker.Close()

	// Subscribe without SubscriberID (ephemeral)
	err := broker.Subscribe(ctx, "test-topic", func(ctx context.Context, env pubsub.Envelope) error {
		return nil
	}, pubsub.SubscribeOptions{})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Verify a subscription was created
	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM pubsub_subscriptions WHERE topic = $1", "test-topic").Scan(&count)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 subscription, got %d", count)
	}
}

func TestBrokerDuplicateSubscriberIDFails(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := NewBroker(wrapDB(db), DefaultConfig(), testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}
	defer broker.Close()

	handler := func(ctx context.Context, env pubsub.Envelope) error { return nil }

	err := broker.Subscribe(ctx, "test-topic", handler, pubsub.SubscribeOptions{
		SubscriberID: "same-id",
	})
	if err != nil {
		t.Fatalf("First subscribe failed: %v", err)
	}

	// Second subscribe with same ID should fail
	err = broker.Subscribe(ctx, "test-topic", handler, pubsub.SubscribeOptions{
		SubscriberID: "same-id",
	})
	if err == nil {
		t.Error("expected error for duplicate subscriber ID")
	}
}

func TestBrokerPublishAfterCloseFails(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := NewBroker(wrapDB(db), DefaultConfig(), testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}

	broker.Close()

	err := broker.Publish(ctx, "test-topic", pubsub.NewEnvelope("test-topic", "payload"))
	if err == nil {
		t.Error("expected error publishing to closed broker")
	}
}

func TestBrokerSubscribeAfterCloseFails(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := NewBroker(wrapDB(db), DefaultConfig(), testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}

	broker.Close()

	err := broker.Subscribe(ctx, "test-topic", func(ctx context.Context, env pubsub.Envelope) error {
		return nil
	}, pubsub.SubscribeOptions{})
	if err == nil {
		t.Error("expected error subscribing to closed broker")
	}
}

func TestBrokerStopMethod(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := NewBroker(wrapDB(db), DefaultConfig(), testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}

	stopErr := broker.Stop(ctx)
	if stopErr != nil {
		t.Fatalf("Stop failed: %v", stopErr)
	}
}

func TestBrokerCloseIdempotent(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	broker := NewBroker(wrapDB(db), DefaultConfig(), testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}

	closeErr := broker.Close()
	if closeErr != nil {
		t.Fatalf("First close failed: %v", closeErr)
	}

	closeErr = broker.Close()
	if closeErr != nil {
		t.Fatalf("Second close failed: %v", closeErr)
	}
}

func TestBrokerHandlerError(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	cfg := DefaultConfig()
	cfg.PollInterval = 10 * time.Millisecond

	broker := NewBroker(wrapDB(db), cfg, testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}
	defer broker.Close()

	var (
		callCount int
		mu        sync.Mutex
	)

	broker.Subscribe(ctx, "test-topic", func(ctx context.Context, env pubsub.Envelope) error {
		mu.Lock()

		callCount++

		mu.Unlock()

		return fmt.Errorf("handler error")
	}, pubsub.SubscribeOptions{SubscriberID: "test-sub"})

	broker.Publish(ctx, "test-topic", pubsub.NewEnvelope("test-topic", "msg1"))
	broker.Publish(ctx, "test-topic", pubsub.NewEnvelope("test-topic", "msg2"))

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if callCount != 2 {
		t.Errorf("expected 2 calls despite errors, got %d", callCount)
	}
}

func TestBrokerWithMetadata(t *testing.T) {
	db, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	cfg := DefaultConfig()
	cfg.PollInterval = 10 * time.Millisecond

	broker := NewBroker(wrapDB(db), cfg, testhelper.TestLogger())
	ctx := context.Background()

	startErr := broker.Start(ctx)
	if startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}
	defer broker.Close()

	var (
		received pubsub.Envelope
		mu       sync.Mutex
	)

	broker.Subscribe(ctx, "test-topic", func(ctx context.Context, env pubsub.Envelope) error {
		mu.Lock()

		received = env

		mu.Unlock()

		return nil
	}, pubsub.SubscribeOptions{SubscriberID: "test-sub"})

	env := pubsub.NewEnvelope("test-topic", "payload").
		WithMetadata("key1", "value1").
		WithMetadata("key2", "value2")

	broker.Publish(ctx, "test-topic", env)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if received.Metadata["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %s", received.Metadata["key1"])
	}

	if received.Metadata["key2"] != "value2" {
		t.Errorf("expected key2=value2, got %s", received.Metadata["key2"])
	}
}
