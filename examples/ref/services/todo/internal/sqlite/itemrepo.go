package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/adrianpk/hatmax-ref/services/todo/internal/config"
	todo "github.com/adrianpk/hatmax-ref/services/todo/internal/todo"
	_ "github.com/adrianpk/hatmax/pkg/lib/hm"
)

// ItemRepo implements the ItemRepo interface for SQLite.
type ItemRepo struct {
	db      *sql.DB
	xparams config.XParams
}

// NewItemRepo creates a new, uninitialized ItemRepo.
func NewItemRepo(xparams config.XParams) *ItemRepo {
	return &ItemRepo{
		xparams: xparams,
	}
}

// Start opens the database connection and pings it.
func (r *ItemRepo) Start(ctx context.Context) error {
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
func (r *ItemRepo) Stop(ctx context.Context) error {
	if r.db != nil {
		if err := r.db.Close(); err != nil {
			return fmt.Errorf("cannot close database: %w", err)
		}
	}
	return nil
}

// Create inserts a new Item into the database.
func (r *ItemRepo) Create(ctx context.Context, item *todo.Item) error {
	// TODO: Handle item.BeforeCreate() if applicable
	_, err := r.db.ExecContext(ctx, QueryCreateItem, item.ID(), item.Text, item.Done, item.CreatedAt, item.UpdatedAt, item.CreatedBy, item.UpdatedBy)
	if err != nil {
		return fmt.Errorf("cannot create Item: %w", err)
	}
	return nil
}

// Get retrieves a Item by its ID.
func (r *ItemRepo) Get(ctx context.Context, id uuid.UUID) (*todo.Item, error) {
	var item todo.Item
	var scannedID uuid.UUID // Local variable to scan into
	row := r.db.QueryRowContext(ctx, QueryGetItem, id)
	err := row.Scan(&scannedID, &item.Text, &item.Done, &item.CreatedAt, &item.UpdatedAt, &item.CreatedBy, &item.UpdatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil, nil for not found
		}
		return nil, fmt.Errorf("cannot get Item: %w", err)
	}
	item.SetID(scannedID) // Set the ID using the exported method
	return &item, nil
}

// Update updates an existing Item in the database.
func (r *ItemRepo) Update(ctx context.Context, item *todo.Item) error {
	// TODO: Handle item.BeforeUpdate() if applicable
	_, err := r.db.ExecContext(ctx, QueryUpdateItem, item.Text, item.Done, item.UpdatedAt, item.CreatedBy, item.UpdatedBy, item.ID())
	if err != nil {
		return fmt.Errorf("cannot update Item: %w", err)
	}
	return nil
}

// Delete deletes a Item from the database by its ID.
func (r *ItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, QueryDeleteItem, id)
	if err != nil {
		return fmt.Errorf("cannot delete Item: %w", err)
	}
	return nil
}

// List retrieves all Item records from the database.
func (r *ItemRepo) List(ctx context.Context) ([]*todo.Item, error) {
	rows, err := r.db.QueryContext(ctx, QueryListItem)
	if err != nil {
		return nil, fmt.Errorf("cannot list Items: %w", err)
	}
	defer rows.Close()

	var items []*todo.Item
	for rows.Next() {
		var item todo.Item
		var scannedID uuid.UUID // Local variable to scan into
		err := rows.Scan(&scannedID, &item.Text, &item.Done, &item.CreatedAt, &item.UpdatedAt, &item.CreatedBy, &item.UpdatedBy)
		if err != nil {
			return nil, fmt.Errorf("cannot scan Item: %w", err)
		}
		item.SetID(scannedID) // Set the ID using the exported method
		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating Item rows: %w", err)
	}

	return items, nil
}
