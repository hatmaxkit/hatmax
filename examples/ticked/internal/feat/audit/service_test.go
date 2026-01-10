package audit

import (
	"context"
	"testing"
	"time"

	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/pubsub"
)

// mockStore implements Store for testing.
type mockStore struct {
	saved []Record
}

func (m *mockStore) Save(ctx context.Context, record *Record) error {
	m.saved = append(m.saved, *record)
	return nil
}

func (m *mockStore) List(ctx context.Context, limit int) ([]Record, error) {
	if limit > len(m.saved) {
		limit = len(m.saved)
	}
	return m.saved[:limit], nil
}

func TestServiceStart(t *testing.T) {
	broker := pubsub.NewMemoryBroker()
	store := &mockStore{}
	logger := log.NewNoopLogger()

	svc := NewService(broker, store, logger)

	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
}

func TestServiceHandleEvent(t *testing.T) {
	broker := pubsub.NewMemoryBroker()
	store := &mockStore{}
	logger := log.NewNoopLogger()

	svc := NewService(broker, store, logger)

	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Publish an event (use map[string]interface{} to match JSON unmarshal behavior)
	env := pubsub.NewEnvelope(Topic, map[string]interface{}{
		"event_type": "todo.item.added",
		"item_id":    "item-123",
		"title":      "Buy milk",
	}).
		WithMetadata("user_id", "user-456").
		WithMetadata("source", "hatmax")

	if err := broker.Publish(ctx, Topic, env); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Give time for synchronous delivery in MemoryBroker
	time.Sleep(10 * time.Millisecond)

	// Verify record was saved
	if len(store.saved) != 1 {
		t.Fatalf("expected 1 saved record, got %d", len(store.saved))
	}

	record := store.saved[0]
	if record.ID != env.ID {
		t.Errorf("expected ID %s, got %s", env.ID, record.ID)
	}
	if record.EventType != "todo.item.added" {
		t.Errorf("expected EventType 'todo.item.added', got %s", record.EventType)
	}
	if record.ItemID != "item-123" {
		t.Errorf("expected ItemID 'item-123', got %s", record.ItemID)
	}
	if record.UserID != "user-456" {
		t.Errorf("expected UserID 'user-456', got %s", record.UserID)
	}
	if record.Source != "hatmax" {
		t.Errorf("expected Source 'hatmax', got %s", record.Source)
	}
}

func TestServiceHandleMultipleEvents(t *testing.T) {
	broker := pubsub.NewMemoryBroker()
	store := &mockStore{}
	logger := log.NewNoopLogger()

	svc := NewService(broker, store, logger)

	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Publish multiple events
	events := []string{"todo.item.added", "todo.item.completed", "todo.item.removed"}
	for i, eventType := range events {
		env := pubsub.NewEnvelope(Topic, map[string]interface{}{
			"event_type": eventType,
			"item_id":    "item-" + string(rune('a'+i)),
		}).
			WithMetadata("user_id", "user-1").
			WithMetadata("source", "hatmax")

		if err := broker.Publish(ctx, Topic, env); err != nil {
			t.Fatalf("Publish failed: %v", err)
		}
	}

	time.Sleep(10 * time.Millisecond)

	if len(store.saved) != 3 {
		t.Fatalf("expected 3 saved records, got %d", len(store.saved))
	}
}

func TestServiceHandleEventInvalidPayload(t *testing.T) {
	broker := pubsub.NewMemoryBroker()
	store := &mockStore{}
	logger := log.NewNoopLogger()

	svc := NewService(broker, store, logger)

	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Publish event with invalid payload type (int instead of map[string]string)
	env := pubsub.NewEnvelope(Topic, 12345). // Invalid payload
							WithMetadata("user_id", "user-1").
							WithMetadata("source", "hatmax")

	if err := broker.Publish(ctx, Topic, env); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Should not save anything due to invalid payload
	if len(store.saved) != 0 {
		t.Errorf("expected 0 saved records for invalid payload, got %d", len(store.saved))
	}
}

// errorStore returns an error on Save.
type errorStore struct{}

func (e *errorStore) Save(ctx context.Context, record *Record) error {
	return context.DeadlineExceeded
}

func (e *errorStore) List(ctx context.Context, limit int) ([]Record, error) {
	return nil, nil
}

func TestServiceHandleEventStoreError(t *testing.T) {
	broker := pubsub.NewMemoryBroker()
	store := &errorStore{}
	logger := log.NewNoopLogger()

	svc := NewService(broker, store, logger)

	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	env := pubsub.NewEnvelope(Topic, map[string]interface{}{
		"event_type": "todo.item.added",
		"item_id":    "item-123",
	}).
		WithMetadata("user_id", "user-1").
		WithMetadata("source", "hatmax")

	if err := broker.Publish(ctx, Topic, env); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
}

type failingBroker struct{}

func (f *failingBroker) Subscribe(ctx context.Context, topic string, handler pubsub.Handler, opts pubsub.SubscribeOptions) error {
	return context.DeadlineExceeded
}

func (f *failingBroker) Publish(ctx context.Context, topic string, env pubsub.Envelope) error {
	return nil
}

func TestServiceStartFailure(t *testing.T) {
	broker := &failingBroker{}
	store := &mockStore{}
	logger := log.NewNoopLogger()

	svc := NewService(broker, store, logger)

	ctx := context.Background()
	err := svc.Start(ctx)
	if err == nil {
		t.Error("expected Start to fail")
	}
}
