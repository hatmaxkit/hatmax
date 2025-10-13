package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/adrianpk/hatmax-ref/services/todo/internal/config"
	"github.com/adrianpk/hatmax-ref/services/todo/internal/todo"
)

// ListSQLiteRepo implements the ListRepo interface using SQLite.
// SQLite requires more complex logic to handle aggregates across multiple related tables.
type ListSQLiteRepo struct {
	db      *sql.DB
	xparams config.XParams
}

// NewListSQLiteRepo creates a new SQLite repository for List aggregates.
func NewListSQLiteRepo(xparams config.XParams) *ListSQLiteRepo {
	return &ListSQLiteRepo{
		xparams: xparams,
	}
}

// Start opens the database connection and pings it.
func (r *ListSQLiteRepo) Start(ctx context.Context) error {
	appCfg := r.xparams.Cfg

	dbPath := appCfg.Database.Path

	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", dbPath))
	if err != nil {
		return fmt.Errorf("cannot open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("cannot connect to database: %w", err)
	}
	r.db = db
	// TODO: Run migrations here
	return nil
}

// Stop closes the database connection.
func (r *ListSQLiteRepo) Stop(ctx context.Context) error {
	if r.db != nil {
		if err := r.db.Close(); err != nil {
			return fmt.Errorf("cannot close database: %w", err)
		}
	}
	return nil
}

// Create creates a new List aggregate in SQLite.
// This involves inserting the root and all child entities in a single transaction.
func (r *ListSQLiteRepo) Create(ctx context.Context, aggregate *todo.List) error {
	if aggregate == nil {
		return fmt.Errorf("aggregate cannot be nil")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	aggregate.EnsureID()
	aggregate.BeforeCreate()

	if err := r.insertRoot(ctx, tx, aggregate); err != nil {
		return fmt.Errorf("failed to insert aggregate root: %w", err)
	}

	if err := r.insertItems(ctx, tx, aggregate.GetID(), aggregate.Items); err != nil {
		return fmt.Errorf("failed to insert Items: %w", err)
	}

	if err := r.insertTags(ctx, tx, aggregate.GetID(), aggregate.Tags); err != nil {
		return fmt.Errorf("failed to insert Tags: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Get retrieves a complete List aggregate by ID from SQLite.
// This involves loading the root and all child entities from multiple tables.
func (r *ListSQLiteRepo) Get(ctx context.Context, id uuid.UUID) (*todo.List, error) {
	aggregate, err := r.getRoot(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregate root: %w", err)
	}

	Items, err := r.getItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get Items: %w", err)
	}
	aggregate.Items = Items

	Tags, err := r.getTags(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get Tags: %w", err)
	}
	aggregate.Tags = Tags

	return aggregate, nil
}

// Save performs a unit-of-work save operation on the List aggregate.
// This computes diffs and updates/inserts/deletes child entities as needed.
func (r *ListSQLiteRepo) Save(ctx context.Context, aggregate *todo.List) error {
	if aggregate == nil {
		return fmt.Errorf("aggregate cannot be nil")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	aggregate.BeforeUpdate()

	if err := r.updateRoot(ctx, tx, aggregate); err != nil {
		return fmt.Errorf("failed to update aggregate root: %w", err)
	}

	if err := r.saveItems(ctx, tx, aggregate.GetID(), aggregate.Items); err != nil {
		return fmt.Errorf("failed to save Items: %w", err)
	}

	if err := r.saveTags(ctx, tx, aggregate.GetID(), aggregate.Tags); err != nil {
		return fmt.Errorf("failed to save Tags: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete removes the entire List aggregate from SQLite.
// This cascades to all child entities.
func (r *ListSQLiteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := r.deleteItems(ctx, tx, id); err != nil {
		return fmt.Errorf("failed to delete Items: %w", err)
	}

	if err := r.deleteTags(ctx, tx, id); err != nil {
		return fmt.Errorf("failed to delete Tags: %w", err)
	}

	result, err := tx.ExecContext(ctx, QueryDeleteListRoot, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete aggregate root: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("List aggregate with ID %s not found for deletion", id.String())
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// List retrieves all List aggregates from SQLite.
// This loads each aggregate with all its child entities.
func (r *ListSQLiteRepo) List(ctx context.Context) ([]*todo.List, error) {
	rows, err := r.db.QueryContext(ctx, QueryListListRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregate IDs: %w", err)
	}
	defer rows.Close()

	var aggregates []*todo.List

	for rows.Next() {
		var idStr string
		if err := rows.Scan(&idStr); err != nil {
			return nil, fmt.Errorf("failed to scan aggregate ID: %w", err)
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UUID %s: %w", idStr, err)
		}

		aggregate, err := r.Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get aggregate %s: %w", idStr, err)
		}

		aggregates = append(aggregates, aggregate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return aggregates, nil
}

// Helper methods for aggregate root operations

func (r *ListSQLiteRepo) insertRoot(ctx context.Context, tx *sql.Tx, aggregate *todo.List) error {
	_, err := tx.ExecContext(ctx, QueryCreateListRoot, aggregate.ID.String(), aggregate.Name, aggregate.Description, aggregate.CreatedAt, aggregate.UpdatedAt)
	return err
}

func (r *ListSQLiteRepo) getRoot(ctx context.Context, id uuid.UUID) (*todo.List, error) {
	var aggregate todo.List
	var idStr string

	err := r.db.QueryRowContext(ctx, QueryGetListRoot, id.String()).Scan(
		&idStr, &aggregate.Name, &aggregate.Description, &aggregate.CreatedAt, &aggregate.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("List aggregate with ID %s not found", id.String())
		}
		return nil, fmt.Errorf("failed to scan aggregate root: %w", err)
	}

	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse UUID %s: %w", idStr, err)
	}
	aggregate.SetID(parsedID)

	return &aggregate, nil
}

func (r *ListSQLiteRepo) updateRoot(ctx context.Context, tx *sql.Tx, aggregate *todo.List) error {
	result, err := tx.ExecContext(ctx, QueryUpdateListRoot, aggregate.Name, aggregate.Description, aggregate.UpdatedAt, aggregate.ID.String())
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("List aggregate with ID %s not found for update", aggregate.ID.String())
	}

	return nil
}

// Helper methods for Items child entities

func (r *ListSQLiteRepo) insertItems(ctx context.Context, tx *sql.Tx, rootID uuid.UUID, items []todo.Item) error {
	if len(items) == 0 {
		return nil
	}

	query := `INSERT INTO items (id, List_id, text, done, created_at, updated_at) VALUES {{.Placeholders}}`

	var args []interface{}
	var placeholders []string

	for _, item := range items {
		item.EnsureID()
		item.BeforeCreate()

		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?)")
		args = append(args, item.ID.String(), rootID.String(), item.Text, item.Done, item.CreatedAt, item.UpdatedAt)
	}

	finalQuery := strings.Replace(query, "{{.Placeholders}}", strings.Join(placeholders, ", "), 1)
	_, err := tx.ExecContext(ctx, finalQuery, args...)
	return err
}

func (r *ListSQLiteRepo) getItems(ctx context.Context, rootID uuid.UUID) ([]todo.Item, error) {
	return r.getItemsWithTx(ctx, nil, rootID)
}

func (r *ListSQLiteRepo) getItemsWithTx(ctx context.Context, tx *sql.Tx, rootID uuid.UUID) ([]todo.Item, error) {
	query := `SELECT id, text, done, created_at, updated_at FROM items WHERE List_id = ? ORDER BY created_at`

	var rows *sql.Rows
	var err error

	if tx != nil {
		rows, err = tx.QueryContext(ctx, query, rootID.String())
	} else {
		rows, err = r.db.QueryContext(ctx, query, rootID.String())
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []todo.Item

	for rows.Next() {
		var item todo.Item
		var idStr string

		err := rows.Scan(&idStr, &item.Text, &item.Done, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UUID %s: %w", idStr, err)
		}
		item.SetID(id)

		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *ListSQLiteRepo) saveItems(ctx context.Context, tx *sql.Tx, rootID uuid.UUID, newItems []todo.Item) error {
	// Get current items from database using the transaction
	currentItems, err := r.getItemsWithTx(ctx, tx, rootID)
	if err != nil {
		return fmt.Errorf("failed to get current items: %w", err)
	}

	// Compute diff
	toInsert, toUpdate, toDelete := r.computeItemDiff(currentItems, newItems)

	// Apply changes
	if len(toDelete) > 0 {
		if err := r.deleteItemsByIDs(ctx, tx, toDelete); err != nil {
			return fmt.Errorf("failed to delete items: %w", err)
		}
	}

	if len(toInsert) > 0 {
		if err := r.insertItems(ctx, tx, rootID, toInsert); err != nil {
			return fmt.Errorf("failed to insert items: %w", err)
		}
	}

	if len(toUpdate) > 0 {
		if err := r.updateItems(ctx, tx, toUpdate); err != nil {
			return fmt.Errorf("failed to update items: %w", err)
		}
	}

	return nil
}

func (r *ListSQLiteRepo) deleteItems(ctx context.Context, tx *sql.Tx, rootID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM items WHERE List_id = ?`, rootID.String())
	return err
}

func (r *ListSQLiteRepo) deleteItemsByIDs(ctx context.Context, tx *sql.Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))

	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id.String()
	}

	query := fmt.Sprintf(`DELETE FROM items WHERE id IN (%s)`, strings.Join(placeholders, ", "))
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (r *ListSQLiteRepo) updateItems(ctx context.Context, tx *sql.Tx, items []todo.Item) error {
	for _, item := range items {
		item.BeforeUpdate()

		query := `UPDATE items SET text = ?, done = ?, updated_at = ? WHERE id = ?`
		_, err := tx.ExecContext(ctx, query, item.Text, item.Done, item.UpdatedAt, item.ID.String())
		if err != nil {
			return fmt.Errorf("failed to update item %s: %w", item.ID.String(), err)
		}
	}
	return nil
}

// computeItemDiff computes the difference between current and new items
func (r *ListSQLiteRepo) computeItemDiff(current, new []todo.Item) (toInsert, toUpdate []todo.Item, toDelete []uuid.UUID) {
	// Create maps for efficient lookup
	currentMap := make(map[string]todo.Item)
	newMap := make(map[string]todo.Item)

	for _, item := range current {
		currentMap[item.ID.String()] = item
	}

	for _, item := range new {
		if item.ID == uuid.Nil {
			// New item without ID - needs insert
			toInsert = append(toInsert, item)
		} else {
			newMap[item.ID.String()] = item
			if _, exists := currentMap[item.ID.String()]; exists {
				// Item exists - needs update
				toUpdate = append(toUpdate, item)
			} else {
				// Item with ID but not in current - needs insert
				toInsert = append(toInsert, item)
			}
		}
	}

	// Find items to delete (in current but not in new)
	for id := range currentMap {
		if _, exists := newMap[id]; !exists {
			uid, _ := uuid.Parse(id)
			toDelete = append(toDelete, uid)
		}
	}

	return toInsert, toUpdate, toDelete
}

// Helper methods for Tags child entities

func (r *ListSQLiteRepo) insertTags(ctx context.Context, tx *sql.Tx, rootID uuid.UUID, items []todo.Tag) error {
	if len(items) == 0 {
		return nil
	}

	query := `INSERT INTO tags (id, List_id, color, name, created_at, updated_at) VALUES {{.Placeholders}}`

	var args []interface{}
	var placeholders []string

	for _, item := range items {
		item.EnsureID()
		item.BeforeCreate()

		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?)")
		args = append(args, item.ID.String(), rootID.String(), item.Color, item.Name, item.CreatedAt, item.UpdatedAt)
	}

	finalQuery := strings.Replace(query, "{{.Placeholders}}", strings.Join(placeholders, ", "), 1)
	_, err := tx.ExecContext(ctx, finalQuery, args...)
	return err
}

func (r *ListSQLiteRepo) getTags(ctx context.Context, rootID uuid.UUID) ([]todo.Tag, error) {
	return r.getTagsWithTx(ctx, nil, rootID)
}

func (r *ListSQLiteRepo) getTagsWithTx(ctx context.Context, tx *sql.Tx, rootID uuid.UUID) ([]todo.Tag, error) {
	query := `SELECT id, color, name, created_at, updated_at FROM tags WHERE List_id = ? ORDER BY created_at`

	var rows *sql.Rows
	var err error

	if tx != nil {
		rows, err = tx.QueryContext(ctx, query, rootID.String())
	} else {
		rows, err = r.db.QueryContext(ctx, query, rootID.String())
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []todo.Tag

	for rows.Next() {
		var item todo.Tag
		var idStr string

		err := rows.Scan(&idStr, &item.Color, &item.Name, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UUID %s: %w", idStr, err)
		}
		item.SetID(id)

		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *ListSQLiteRepo) saveTags(ctx context.Context, tx *sql.Tx, rootID uuid.UUID, newItems []todo.Tag) error {
	// Get current items from database using the transaction
	currentItems, err := r.getTagsWithTx(ctx, tx, rootID)
	if err != nil {
		return fmt.Errorf("failed to get current items: %w", err)
	}

	// Compute diff
	toInsert, toUpdate, toDelete := r.computeTagDiff(currentItems, newItems)

	// Apply changes
	if len(toDelete) > 0 {
		if err := r.deleteTagsByIDs(ctx, tx, toDelete); err != nil {
			return fmt.Errorf("failed to delete items: %w", err)
		}
	}

	if len(toInsert) > 0 {
		if err := r.insertTags(ctx, tx, rootID, toInsert); err != nil {
			return fmt.Errorf("failed to insert items: %w", err)
		}
	}

	if len(toUpdate) > 0 {
		if err := r.updateTags(ctx, tx, toUpdate); err != nil {
			return fmt.Errorf("failed to update items: %w", err)
		}
	}

	return nil
}

func (r *ListSQLiteRepo) deleteTags(ctx context.Context, tx *sql.Tx, rootID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE List_id = ?`, rootID.String())
	return err
}

func (r *ListSQLiteRepo) deleteTagsByIDs(ctx context.Context, tx *sql.Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))

	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id.String()
	}

	query := fmt.Sprintf(`DELETE FROM tags WHERE id IN (%s)`, strings.Join(placeholders, ", "))
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (r *ListSQLiteRepo) updateTags(ctx context.Context, tx *sql.Tx, items []todo.Tag) error {
	for _, item := range items {
		item.BeforeUpdate()

		query := `UPDATE tags SET color = ?, name = ?, updated_at = ? WHERE id = ?`
		_, err := tx.ExecContext(ctx, query, item.Color, item.Name, item.UpdatedAt, item.ID.String())
		if err != nil {
			return fmt.Errorf("failed to update item %s: %w", item.ID.String(), err)
		}
	}
	return nil
}

// computeTagDiff computes the difference between current and new items
func (r *ListSQLiteRepo) computeTagDiff(current, new []todo.Tag) (toInsert, toUpdate []todo.Tag, toDelete []uuid.UUID) {
	// Create maps for efficient lookup
	currentMap := make(map[string]todo.Tag)
	newMap := make(map[string]todo.Tag)

	for _, item := range current {
		currentMap[item.ID.String()] = item
	}

	for _, item := range new {
		if item.ID == uuid.Nil {
			// New item without ID - needs insert
			toInsert = append(toInsert, item)
		} else {
			newMap[item.ID.String()] = item
			if _, exists := currentMap[item.ID.String()]; exists {
				// Item exists - needs update
				toUpdate = append(toUpdate, item)
			} else {
				// Item with ID but not in current - needs insert
				toInsert = append(toInsert, item)
			}
		}
	}

	// Find items to delete (in current but not in new)
	for id := range currentMap {
		if _, exists := newMap[id]; !exists {
			uid, _ := uuid.Parse(id)
			toDelete = append(toDelete, uid)
		}
	}

	return toInsert, toUpdate, toDelete
}
