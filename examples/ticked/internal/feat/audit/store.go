package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Record represents an audit log entry.
type Record struct {
	ID        string
	EventType string
	ItemID    string
	UserID    string
	Payload   map[string]string
	Source    string
	CreatedAt time.Time
}

// Store defines the interface for audit log persistence.
type Store interface {
	Save(ctx context.Context, record *Record) error
	List(ctx context.Context, limit int) ([]Record, error)
}

// DBProvider provides access to a database connection.
type DBProvider interface {
	GetDB() *sql.DB
}

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	dbProvider DBProvider
	db         *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed audit store.
func NewPostgresStore(dbProvider DBProvider) *PostgresStore {
	return &PostgresStore{dbProvider: dbProvider}
}

// Start obtains the database connection.
func (s *PostgresStore) Start(ctx context.Context) error {
	s.db = s.dbProvider.GetDB()
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	return nil
}

// Stop is a no-op for PostgresStore.
func (s *PostgresStore) Stop(ctx context.Context) error {
	return nil
}

// Save persists an audit record.
func (s *PostgresStore) Save(ctx context.Context, record *Record) error {
	payloadJSON, err := json.Marshal(record.Payload)
	if err != nil {
		return fmt.Errorf("cannot marshal payload: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO audit_log (id, event_type, item_id, user_id, payload, source, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, record.ID, record.EventType, record.ItemID, record.UserID, payloadJSON, record.Source, record.CreatedAt)
	if err != nil {
		return fmt.Errorf("cannot insert audit record: %w", err)
	}

	return nil
}

// List retrieves the most recent audit records.
func (s *PostgresStore) List(ctx context.Context, limit int) ([]Record, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, event_type, item_id, user_id, payload, source, created_at
		FROM audit_log
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("cannot query audit log: %w", err)
	}
	defer rows.Close()

	var records []Record

	for rows.Next() {
		var (
			r           Record
			itemID      sql.NullString
			payloadJSON []byte
		)

		err := rows.Scan(&r.ID, &r.EventType, &itemID, &r.UserID, &payloadJSON, &r.Source, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("cannot scan audit record: %w", err)
		}

		if itemID.Valid {
			r.ItemID = itemID.String
		}

		if len(payloadJSON) > 0 {
			err := json.Unmarshal(payloadJSON, &r.Payload)
			if err != nil {
				return nil, fmt.Errorf("cannot unmarshal payload: %w", err)
			}
		}

		records = append(records, r)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return records, nil
}
