package list

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"hatmax.adrianpk.com/examples/ticked/internal/dal"
)

var ErrNotFound = errors.New("todo list not found")

// Store defines the interface for todo list persistence.
type Store interface {
	FindByUserID(ctx context.Context, userID string) (*TodoList, error)
	Save(ctx context.Context, list *TodoList) error
	Delete(ctx context.Context, listID string) error
}

// DBProvider provides access to a database connection.
type DBProvider interface {
	GetDB() *sql.DB
}

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	dbProvider DBProvider
	db         *sql.DB
	q          *dal.Queries
}

// NewPostgresStore creates a new PostgreSQL-backed store.
func NewPostgresStore(dbProvider DBProvider) *PostgresStore {
	return &PostgresStore{dbProvider: dbProvider}
}

// Start obtains the database connection.
func (s *PostgresStore) Start(ctx context.Context) error {
	s.db = s.dbProvider.GetDB()
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	s.q = dal.New(s.db)

	return nil
}

// Stop is a no-op for PostgresStore.
func (s *PostgresStore) Stop(ctx context.Context) error {
	return nil
}

// FindByUserID retrieves a todo list by user ID.
func (s *PostgresStore) FindByUserID(ctx context.Context, userID string) (*TodoList, error) {
	dbList, err := s.q.GetTodoListByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get list: %w", err)
	}

	dbItems, err := s.q.GetTodoItemsByListID(ctx, dbList.ID)
	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}

	list := &TodoList{
		ListID:    dbList.ID,
		UserID:    dbList.UserID,
		Items:     make([]TodoItem, 0, len(dbItems)),
		CreatedAt: dbList.CreatedAt,
		UpdatedAt: dbList.UpdatedAt,
	}

	for _, dbItem := range dbItems {
		var completedAt *time.Time
		if dbItem.CompletedAt.Valid {
			completedAt = &dbItem.CompletedAt.Time
		}

		list.Items = append(list.Items, TodoItem{
			ItemID:      dbItem.ID,
			Text:        dbItem.Text,
			Completed:   dbItem.Completed,
			CreatedAt:   dbItem.CreatedAt,
			CompletedAt: completedAt,
		})
	}

	return list, nil
}

// Save persists a todo list and all its items.
func (s *PostgresStore) Save(ctx context.Context, list *TodoList) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	// Upsert the list
	err = qtx.UpsertTodoList(ctx, dal.UpsertTodoListParams{
		ID:        list.ListID,
		UserID:    list.UserID,
		CreatedAt: list.CreatedAt,
		UpdatedAt: list.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("upsert list: %w", err)
	}

	// Sync items
	err = s.syncItems(ctx, qtx, list)
	if err != nil {
		return fmt.Errorf("sync items: %w", err)
	}

	return tx.Commit()
}

func (s *PostgresStore) syncItems(ctx context.Context, qtx *dal.Queries, list *TodoList) error {
	// Get existing items
	existing, err := qtx.GetTodoItemsByListID(ctx, list.ListID)
	if err != nil {
		return err
	}

	existingByID := make(map[string]dal.TodoItem)
	for _, item := range existing {
		existingByID[item.ID] = item
	}

	seen := make(map[string]bool)

	// Insert or update items
	for _, item := range list.Items {
		seen[item.ItemID] = true

		var completedAt sql.NullTime
		if item.CompletedAt != nil {
			completedAt = sql.NullTime{Time: *item.CompletedAt, Valid: true}
		}

		if _, exists := existingByID[item.ItemID]; exists {
			err := qtx.UpdateTodoItem(ctx, dal.UpdateTodoItemParams{
				ID:          item.ItemID,
				Text:        item.Text,
				Completed:   item.Completed,
				CompletedAt: completedAt,
			})
			if err != nil {
				return fmt.Errorf("update item %s: %w", item.ItemID, err)
			}
		} else {
			err := qtx.InsertTodoItem(ctx, dal.InsertTodoItemParams{
				ID:          item.ItemID,
				ListID:      list.ListID,
				Text:        item.Text,
				Completed:   item.Completed,
				CreatedAt:   item.CreatedAt,
				CompletedAt: completedAt,
			})
			if err != nil {
				return fmt.Errorf("insert item %s: %w", item.ItemID, err)
			}
		}
	}

	// Delete removed items
	for id := range existingByID {
		if !seen[id] {
			err := qtx.DeleteTodoItem(ctx, id)
			if err != nil {
				return fmt.Errorf("delete item %s: %w", id, err)
			}
		}
	}

	return nil
}

// Delete removes a todo list and all its items.
func (s *PostgresStore) Delete(ctx context.Context, listID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	err = qtx.DeleteTodoItemsByListID(ctx, listID)
	if err != nil {
		return fmt.Errorf("delete items: %w", err)
	}

	err = qtx.DeleteTodoList(ctx, listID)
	if err != nil {
		return fmt.Errorf("delete list: %w", err)
	}

	return tx.Commit()
}
