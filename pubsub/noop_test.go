package pubsub

import (
	"context"
	"sync"
	"testing"
)

func TestNoopBrokerPublish(t *testing.T) {
	broker := NewNoopBroker()
	ctx := context.Background()

	env := NewEnvelope("test-topic", "test-payload")
	if err := broker.Publish(ctx, "test-topic", env); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	published := broker.Published()
	if len(published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(published))
	}

	if published[0].ID != env.ID {
		t.Errorf("expected ID %s, got %s", env.ID, published[0].ID)
	}
}

func TestNoopBrokerSubscribe(t *testing.T) {
	broker := NewNoopBroker()
	ctx := context.Background()

	called := false
	handler := func(ctx context.Context, env Envelope) error {
		called = true
		return nil
	}

	if err := broker.Subscribe(ctx, "test-topic", handler, SubscribeOptions{}); err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// NoopBroker doesn't deliver messages
	if called {
		t.Error("handler should not be called for NoopBroker")
	}
}

func TestNoopBrokerReset(t *testing.T) {
	broker := NewNoopBroker()
	ctx := context.Background()

	broker.Publish(ctx, "test", NewEnvelope("test", "payload1"))
	broker.Publish(ctx, "test", NewEnvelope("test", "payload2"))

	if len(broker.Published()) != 2 {
		t.Fatal("expected 2 published messages before reset")
	}

	broker.Reset()

	if len(broker.Published()) != 0 {
		t.Error("expected 0 published messages after reset")
	}
}

func TestNoopBrokerClose(t *testing.T) {
	broker := NewNoopBroker()
	if err := broker.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestMemoryBrokerPublishSubscribe(t *testing.T) {
	broker := NewMemoryBroker()
	ctx := context.Background()

	var received []Envelope
	var mu sync.Mutex

	handler := func(ctx context.Context, env Envelope) error {
		mu.Lock()
		received = append(received, env)
		mu.Unlock()
		return nil
	}

	if err := broker.Subscribe(ctx, "test-topic", handler, SubscribeOptions{}); err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	env := NewEnvelope("test-topic", "test-payload")
	if err := broker.Publish(ctx, "test-topic", env); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

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

func TestMemoryBrokerFanOut(t *testing.T) {
	broker := NewMemoryBroker()
	ctx := context.Background()

	var count1, count2 int
	var mu sync.Mutex

	handler1 := func(ctx context.Context, env Envelope) error {
		mu.Lock()
		count1++
		mu.Unlock()
		return nil
	}

	handler2 := func(ctx context.Context, env Envelope) error {
		mu.Lock()
		count2++
		mu.Unlock()
		return nil
	}

	broker.Subscribe(ctx, "test-topic", handler1, SubscribeOptions{SubscriberID: "sub1"})
	broker.Subscribe(ctx, "test-topic", handler2, SubscribeOptions{SubscriberID: "sub2"})

	env := NewEnvelope("test-topic", "test-payload")
	broker.Publish(ctx, "test-topic", env)

	mu.Lock()
	defer mu.Unlock()

	if count1 != 1 {
		t.Errorf("handler1 expected 1 call, got %d", count1)
	}
	if count2 != 1 {
		t.Errorf("handler2 expected 1 call, got %d", count2)
	}
}

func TestMemoryBrokerDifferentTopics(t *testing.T) {
	broker := NewMemoryBroker()
	ctx := context.Background()

	var received1, received2 int

	broker.Subscribe(ctx, "topic1", func(ctx context.Context, env Envelope) error {
		received1++
		return nil
	}, SubscribeOptions{})

	broker.Subscribe(ctx, "topic2", func(ctx context.Context, env Envelope) error {
		received2++
		return nil
	}, SubscribeOptions{})

	broker.Publish(ctx, "topic1", NewEnvelope("topic1", "payload"))

	if received1 != 1 {
		t.Errorf("topic1 handler expected 1 call, got %d", received1)
	}
	if received2 != 0 {
		t.Errorf("topic2 handler expected 0 calls, got %d", received2)
	}
}

func TestMemoryBrokerClose(t *testing.T) {
	broker := NewMemoryBroker()
	ctx := context.Background()

	called := false
	broker.Subscribe(ctx, "test", func(ctx context.Context, env Envelope) error {
		called = true
		return nil
	}, SubscribeOptions{})

	broker.Close()

	// After close, publish should not reach handlers (subscriptions cleared)
	broker.Publish(ctx, "test", NewEnvelope("test", "payload"))

	if called {
		t.Error("handler should not be called after Close")
	}
}

func TestEnvelopeNewEnvelope(t *testing.T) {
	env := NewEnvelope("my-topic", "my-payload")

	if env.Topic != "my-topic" {
		t.Errorf("expected topic 'my-topic', got '%s'", env.Topic)
	}

	if env.Payload != "my-payload" {
		t.Errorf("expected payload 'my-payload', got '%v'", env.Payload)
	}

	if env.ID == "" {
		t.Error("expected non-empty ID")
	}

	if env.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	if env.Metadata == nil {
		t.Error("expected non-nil metadata map")
	}
}

func TestEnvelopeWithMetadata(t *testing.T) {
	env := NewEnvelope("topic", "payload")
	env = env.WithMetadata("key1", "value1")
	env = env.WithMetadata("key2", "value2")

	if env.Metadata["key1"] != "value1" {
		t.Errorf("expected key1='value1', got '%s'", env.Metadata["key1"])
	}

	if env.Metadata["key2"] != "value2" {
		t.Errorf("expected key2='value2', got '%s'", env.Metadata["key2"])
	}
}

func TestEnvelopeNewEnvelopeWithMetadata(t *testing.T) {
	metadata := map[string]string{
		"source":  "test",
		"version": "1.0",
	}

	env := NewEnvelopeWithMetadata("topic", "payload", metadata)

	if env.Metadata["source"] != "test" {
		t.Errorf("expected source='test', got '%s'", env.Metadata["source"])
	}

	if env.Metadata["version"] != "1.0" {
		t.Errorf("expected version='1.0', got '%s'", env.Metadata["version"])
	}
}
