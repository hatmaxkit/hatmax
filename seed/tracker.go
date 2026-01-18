package seed

import (
	"context"
	"database/sql"
)

// Tracker tracks which seeds have been applied to the database.
// Similar to migration tracking, prevents seeds from running twice.
type Tracker struct {
	db *sql.DB
}

// NewTracker creates a new Tracker instance.
func NewTracker(db *sql.DB) *Tracker {
	return &Tracker{db: db}
}

// EnsureTable creates the _seeds tracking table if it doesn't exist.
func (t *Tracker) EnsureTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS _seeds (
			id TEXT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := t.db.ExecContext(ctx, query)
	return err
}

// IsApplied checks if a seed has already been applied.
func (t *Tracker) IsApplied(ctx context.Context, name string) (bool, error) {
	var count int
	err := t.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM _seeds WHERE id = $1", name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// MarkApplied records that a seed has been applied.
func (t *Tracker) MarkApplied(ctx context.Context, name string) error {
	_, err := t.db.ExecContext(ctx, "INSERT INTO _seeds (id) VALUES ($1)", name)
	return err
}
