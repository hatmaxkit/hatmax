package audit

import (
	"context"
	"testing"
	"time"

	"github.com/hatmaxkit/hatmax/testhelper"
)

func setupTestStore(t *testing.T) (*PostgresStore, func()) {
	t.Helper()

	db, _, cleanup := testhelper.SetupTestDB(t)

	// Create audit_log table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_log (
			id TEXT PRIMARY KEY,
			event_type TEXT NOT NULL,
			item_id TEXT,
			user_id TEXT NOT NULL,
			payload JSONB,
			source TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("cannot create audit_log table: %v", err)
	}

	store := NewPostgresStore(db)
	return store, cleanup
}

func TestPostgresStoreSave(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	record := &Record{
		ID:        "test-id-1",
		EventType: "todo.item.added",
		ItemID:    "item-123",
		UserID:    "user-456",
		Payload:   map[string]string{"title": "Buy milk"},
		Source:    "hatmax",
		CreatedAt: time.Now(),
	}

	err := store.Save(ctx, record)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify by listing
	records, err := store.List(ctx, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if records[0].ID != record.ID {
		t.Errorf("expected ID %s, got %s", record.ID, records[0].ID)
	}
	if records[0].EventType != record.EventType {
		t.Errorf("expected EventType %s, got %s", record.EventType, records[0].EventType)
	}
	if records[0].ItemID != record.ItemID {
		t.Errorf("expected ItemID %s, got %s", record.ItemID, records[0].ItemID)
	}
	if records[0].UserID != record.UserID {
		t.Errorf("expected UserID %s, got %s", record.UserID, records[0].UserID)
	}
	if records[0].Source != record.Source {
		t.Errorf("expected Source %s, got %s", record.Source, records[0].Source)
	}
	if records[0].Payload["title"] != "Buy milk" {
		t.Errorf("expected payload title 'Buy milk', got %s", records[0].Payload["title"])
	}
}

func TestPostgresStoreList(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Insert multiple records
	for i := 0; i < 5; i++ {
		record := &Record{
			ID:        "test-id-" + string(rune('a'+i)),
			EventType: "todo.item.added",
			ItemID:    "item-" + string(rune('a'+i)),
			UserID:    "user-1",
			Payload:   map[string]string{},
			Source:    "hatmax",
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := store.Save(ctx, record); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// List with limit
	records, err := store.List(ctx, 3)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	// Should be ordered by created_at DESC (most recent first)
	if records[0].ID != "test-id-e" {
		t.Errorf("expected most recent record first, got %s", records[0].ID)
	}
}

func TestPostgresStoreListEmpty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	records, err := store.List(ctx, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestPostgresStoreSaveNilItemID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	record := &Record{
		ID:        "test-id-nil",
		EventType: "system.event",
		ItemID:    "", // Empty item ID
		UserID:    "user-1",
		Payload:   map[string]string{},
		Source:    "hatmax",
		CreatedAt: time.Now(),
	}

	err := store.Save(ctx, record)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	records, err := store.List(ctx, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if records[0].ItemID != "" {
		t.Errorf("expected empty ItemID, got %s", records[0].ItemID)
	}
}
