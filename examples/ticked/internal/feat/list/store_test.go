package list

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"hatmax.adrianpk.com/testhelper"
)

type fakeDBProvider struct {
	db *sql.DB
}

func (f *fakeDBProvider) GetDB() *sql.DB {
	return f.db
}

func setupTestStore(t *testing.T) (*PostgresStore, func()) {
	t.Helper()

	db, _, cleanup := testhelper.SetupTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS todo_lists (
			id TEXT PRIMARY KEY,
			user_id TEXT UNIQUE NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE IF NOT EXISTS todo_items (
			id TEXT PRIMARY KEY,
			list_id TEXT NOT NULL,
			text TEXT NOT NULL CHECK (length(text) <= 500),
			completed BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMPTZ NOT NULL,
			completed_at TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_todo_items_list_id ON todo_items(list_id);
	`)
	if err != nil {
		t.Fatalf("cannot create tables: %v", err)
	}

	dbProvider := &fakeDBProvider{db: db}

	store := NewPostgresStore(dbProvider)

	err = store.Start(context.Background())
	if err != nil {
		t.Fatalf("cannot start store: %v", err)
	}

	return store, cleanup
}

func TestPostgresStoreSaveAndFind(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond)

	list := &TodoList{
		ListID:    "list-1",
		UserID:    "user-1",
		Items:     []TodoItem{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := store.Save(ctx, list)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	found, err := store.FindByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("FindByUserID failed: %v", err)
	}

	if found.ListID != list.ListID {
		t.Errorf("expected ListID %s, got %s", list.ListID, found.ListID)
	}

	if found.UserID != list.UserID {
		t.Errorf("expected UserID %s, got %s", list.UserID, found.UserID)
	}
}

func TestPostgresStoreSaveWithItems(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond)

	list := &TodoList{
		ListID: "list-1",
		UserID: "user-1",
		Items: []TodoItem{
			{ItemID: "item-1", Text: "First", Completed: false, CreatedAt: now},
			{ItemID: "item-2", Text: "Second", Completed: true, CreatedAt: now, CompletedAt: &now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := store.Save(ctx, list)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	found, err := store.FindByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("FindByUserID failed: %v", err)
	}

	if len(found.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(found.Items))
	}
}

func TestPostgresStoreUpdateItems(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond)

	list := &TodoList{
		ListID: "list-1",
		UserID: "user-1",
		Items: []TodoItem{
			{ItemID: "item-1", Text: "Original", Completed: false, CreatedAt: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.Save(ctx, list)

	list.Items[0].Text = "Updated"
	list.Items[0].Completed = true
	list.UpdatedAt = now.Add(time.Second)
	store.Save(ctx, list)

	found, _ := store.FindByUserID(ctx, "user-1")
	if found.Items[0].Text != "Updated" {
		t.Errorf("expected text 'Updated', got %s", found.Items[0].Text)
	}

	if !found.Items[0].Completed {
		t.Error("expected item to be completed")
	}
}

func TestPostgresStoreDeleteItems(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond)

	list := &TodoList{
		ListID: "list-1",
		UserID: "user-1",
		Items: []TodoItem{
			{ItemID: "item-1", Text: "First", CreatedAt: now},
			{ItemID: "item-2", Text: "Second", CreatedAt: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.Save(ctx, list)

	list.Items = list.Items[:1]
	list.UpdatedAt = now.Add(time.Second)
	store.Save(ctx, list)

	found, _ := store.FindByUserID(ctx, "user-1")
	if len(found.Items) != 1 {
		t.Errorf("expected 1 item after delete, got %d", len(found.Items))
	}
}

func TestPostgresStoreFindNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	_, err := store.FindByUserID(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPostgresStoreDelete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond)

	list := &TodoList{
		ListID: "list-1",
		UserID: "user-1",
		Items: []TodoItem{
			{ItemID: "item-1", Text: "Test", CreatedAt: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.Save(ctx, list)

	err := store.Delete(ctx, list.ListID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.FindByUserID(ctx, "user-1")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
