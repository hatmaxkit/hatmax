package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/google/uuid"

	_ "github.com/adrianpk/hatmax/pkg/lib/hm"
	"github.com/adrianpk/hatmax-ref/services/todo/internal/config"
	todo "github.com/adrianpk/hatmax-ref/services/todo/internal/todo"
)

// TagRepo implements the TagRepo interface for SQLite.
type TagRepo struct {
	db      *sql.DB
	xparams config.XParams
}

// NewTagRepo creates a new, uninitialized TagRepo.
func NewTagRepo(xparams config.XParams) *TagRepo {
	return &TagRepo{
		xparams: xparams,
	}
}

// Start opens the database connection and pings it.
func (r *TagRepo) Start(ctx context.Context) error {
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
func (r *TagRepo) Stop(ctx context.Context) error {
	if r.db != nil {
		if err := r.db.Close(); err != nil {
			return fmt.Errorf("cannot close database: %w", err)
		}
	}
	return nil
}

// Create inserts a new Tag into the database.
func (r *TagRepo) Create(ctx context.Context, item *todo.Tag) error {
	// TODO: Handle item.BeforeCreate() if applicable
	_, err := r.db.ExecContext(ctx, QueryCreateTag, item.ID(), item.Name, item.Color, item.CreatedAt, item.UpdatedAt, item.CreatedBy, item.UpdatedBy)
	if err != nil {
		return fmt.Errorf("cannot create Tag: %w", err)
	}
	return nil
}

// Get retrieves a Tag by its ID.
func (r *TagRepo) Get(ctx context.Context, id uuid.UUID) (*todo.Tag, error) {
	var item todo.Tag
	var scannedID uuid.UUID // Local variable to scan into
	row := r.db.QueryRowContext(ctx, QueryGetTag, id)
	err := row.Scan(&scannedID, &item.Name, &item.Color, &item.CreatedAt, &item.UpdatedAt, &item.CreatedBy, &item.UpdatedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil, nil for not found
		}
		return nil, fmt.Errorf("cannot get Tag: %w", err)
	}
	item.SetID(scannedID) // Set the ID using the exported method
	return &item, nil
}

// Update updates an existing Tag in the database.
func (r *TagRepo) Update(ctx context.Context, item *todo.Tag) error {
	// TODO: Handle item.BeforeUpdate() if applicable
	_, err := r.db.ExecContext(ctx, QueryUpdateTag, item.Name, item.Color, item.UpdatedAt, item.CreatedBy, item.UpdatedBy, item.ID())
	if err != nil {
		return fmt.Errorf("cannot update Tag: %w", err)
	}
	return nil
}

// Delete deletes a Tag from the database by its ID.
func (r *TagRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, QueryDeleteTag, id)
	if err != nil {
		return fmt.Errorf("cannot delete Tag: %w", err)
	}
	return nil
}

// List retrieves all Tag records from the database.
func (r *TagRepo) List(ctx context.Context) ([]*todo.Tag, error) {
	rows, err := r.db.QueryContext(ctx, QueryListTag)
	if err != nil {
		return nil, fmt.Errorf("cannot list Tags: %w", err)
	}
	defer rows.Close()

	var items []*todo.Tag
	for rows.Next() {
		var item todo.Tag
		var scannedID uuid.UUID // Local variable to scan into
		err := rows.Scan(&scannedID, &item.Name, &item.Color, &item.CreatedAt, &item.UpdatedAt, &item.CreatedBy, &item.UpdatedBy)
		if err != nil {
			return nil, fmt.Errorf("cannot scan Tag: %w", err)
		}
		item.SetID(scannedID) // Set the ID using the exported method
		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating Tag rows: %w", err)
	}

	return items, nil
}