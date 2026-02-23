package list

import (
	"context"
	"testing"

	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/pubsub"
)

// mockStore implements Store for testing.
type mockStore struct {
	lists map[string]*TodoList
}

func newMockStore() *mockStore {
	return &mockStore{lists: make(map[string]*TodoList)}
}

func (m *mockStore) FindByUserID(ctx context.Context, userID string) (*TodoList, error) {
	if list, ok := m.lists[userID]; ok {
		return list, nil
	}

	return nil, ErrNotFound
}

func (m *mockStore) Save(ctx context.Context, list *TodoList) error {
	m.lists[list.UserID] = list

	return nil
}

func (m *mockStore) Delete(ctx context.Context, listID string) error {
	for userID, list := range m.lists {
		if list.ListID == listID {
			delete(m.lists, userID)

			return nil
		}
	}

	return nil
}

func TestServiceAddItemPublishesEvent(t *testing.T) {
	store := newMockStore()
	broker := pubsub.NewNoopBroker()
	logger := log.NewNoopLogger()

	svc := NewService(store, broker, logger)

	ctx := context.Background()

	_, err := svc.AddItem(ctx, "user-1", "Buy milk")
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	published := broker.Published()
	if len(published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(published))
	}

	payload, ok := published[0].Payload.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string payload, got %T", published[0].Payload)
	}

	if payload["event_type"] != "todo.item.added" {
		t.Errorf("expected event_type 'todo.item.added', got %s", payload["event_type"])
	}

	if payload["title"] != "Buy milk" {
		t.Errorf("expected title 'Buy milk', got %s", payload["title"])
	}

	if published[0].Metadata["user_id"] != "user-1" {
		t.Errorf("expected user_id 'user-1', got %s", published[0].Metadata["user_id"])
	}
}

func TestServiceToggleItemPublishesEventWhenCompleted(t *testing.T) {
	store := newMockStore()
	broker := pubsub.NewNoopBroker()
	logger := log.NewNoopLogger()

	svc := NewService(store, broker, logger)

	ctx := context.Background()
	item, _ := svc.AddItem(ctx, "user-1", "Test item")

	broker.Reset() // Clear the add event

	_, err := svc.ToggleItem(ctx, "user-1", item.ItemID)
	if err != nil {
		t.Fatalf("ToggleItem failed: %v", err)
	}

	published := broker.Published()
	if len(published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(published))
	}

	payload, _ := published[0].Payload.(map[string]string)
	if payload["event_type"] != "todo.item.completed" {
		t.Errorf("expected event_type 'todo.item.completed', got %s", payload["event_type"])
	}
}

func TestServiceToggleItemDoesNotPublishWhenUncompleted(t *testing.T) {
	store := newMockStore()
	broker := pubsub.NewNoopBroker()
	logger := log.NewNoopLogger()

	svc := NewService(store, broker, logger)

	ctx := context.Background()
	item, _ := svc.AddItem(ctx, "user-1", "Test item")
	svc.ToggleItem(ctx, "user-1", item.ItemID) // Complete

	broker.Reset()

	// Toggle again to uncomplete
	_, err := svc.ToggleItem(ctx, "user-1", item.ItemID)
	if err != nil {
		t.Fatalf("ToggleItem failed: %v", err)
	}

	published := broker.Published()
	if len(published) != 0 {
		t.Errorf("expected 0 events when uncompleting, got %d", len(published))
	}
}

func TestServiceRemoveItemPublishesEvent(t *testing.T) {
	store := newMockStore()
	broker := pubsub.NewNoopBroker()
	logger := log.NewNoopLogger()

	svc := NewService(store, broker, logger)

	ctx := context.Background()
	item, _ := svc.AddItem(ctx, "user-1", "Test item")

	broker.Reset()

	err := svc.RemoveItem(ctx, "user-1", item.ItemID)
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}

	published := broker.Published()
	if len(published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(published))
	}

	payload, _ := published[0].Payload.(map[string]string)
	if payload["event_type"] != "todo.item.removed" {
		t.Errorf("expected event_type 'todo.item.removed', got %s", payload["event_type"])
	}
}

func TestServiceWorksWithNilPublisher(t *testing.T) {
	store := newMockStore()
	logger := log.NewNoopLogger()

	svc := NewService(store, nil, logger)

	ctx := context.Background()

	_, err := svc.AddItem(ctx, "user-1", "Test item")
	if err != nil {
		t.Fatalf("AddItem should work with nil publisher: %v", err)
	}
}

func TestServiceGetOrCreateListCreatesNew(t *testing.T) {
	store := newMockStore()
	logger := log.NewNoopLogger()

	svc := NewService(store, nil, logger)

	ctx := context.Background()

	list, err := svc.GetOrCreateList(ctx, "new-user")
	if err != nil {
		t.Fatalf("GetOrCreateList failed: %v", err)
	}

	if list.UserID != "new-user" {
		t.Errorf("expected UserID 'new-user', got %s", list.UserID)
	}

	if list.ListID == "" {
		t.Error("ListID should not be empty")
	}
}

func TestServiceGetOrCreateListReturnsExisting(t *testing.T) {
	store := newMockStore()
	logger := log.NewNoopLogger()

	svc := NewService(store, nil, logger)

	ctx := context.Background()
	list1, _ := svc.GetOrCreateList(ctx, "user-1")
	list2, _ := svc.GetOrCreateList(ctx, "user-1")

	if list1.ListID != list2.ListID {
		t.Error("should return same list for same user")
	}
}

func TestServiceUpdateItem(t *testing.T) {
	store := newMockStore()
	logger := log.NewNoopLogger()

	svc := NewService(store, nil, logger)

	ctx := context.Background()
	item, _ := svc.AddItem(ctx, "user-1", "Original")

	updated, err := svc.UpdateItem(ctx, "user-1", item.ItemID, "Modified")
	if err != nil {
		t.Fatalf("UpdateItem failed: %v", err)
	}

	if updated.Text != "Modified" {
		t.Errorf("expected text 'Modified', got %s", updated.Text)
	}
}

func TestServiceUpdateItemNotFound(t *testing.T) {
	store := newMockStore()
	logger := log.NewNoopLogger()

	svc := NewService(store, nil, logger)

	ctx := context.Background()
	svc.AddItem(ctx, "user-1", "Test")

	_, err := svc.UpdateItem(ctx, "user-1", "nonexistent", "Modified")
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

func TestServiceToggleItemNotFound(t *testing.T) {
	store := newMockStore()
	logger := log.NewNoopLogger()

	svc := NewService(store, nil, logger)

	ctx := context.Background()
	svc.AddItem(ctx, "user-1", "Test")

	_, err := svc.ToggleItem(ctx, "user-1", "nonexistent")
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

func TestServiceRemoveItemNotFound(t *testing.T) {
	store := newMockStore()
	logger := log.NewNoopLogger()

	svc := NewService(store, nil, logger)

	ctx := context.Background()
	svc.AddItem(ctx, "user-1", "Test")

	err := svc.RemoveItem(ctx, "user-1", "nonexistent")
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

func TestServiceToggleItemUserNotFound(t *testing.T) {
	store := newMockStore()
	logger := log.NewNoopLogger()

	svc := NewService(store, nil, logger)

	ctx := context.Background()

	_, err := svc.ToggleItem(ctx, "nonexistent-user", "item-1")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestServiceRemoveItemUserNotFound(t *testing.T) {
	store := newMockStore()
	logger := log.NewNoopLogger()

	svc := NewService(store, nil, logger)

	ctx := context.Background()

	err := svc.RemoveItem(ctx, "nonexistent-user", "item-1")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestServiceUpdateItemUserNotFound(t *testing.T) {
	store := newMockStore()
	logger := log.NewNoopLogger()

	svc := NewService(store, nil, logger)

	ctx := context.Background()

	_, err := svc.UpdateItem(ctx, "nonexistent-user", "item-1", "text")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
