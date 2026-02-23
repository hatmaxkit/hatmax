package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/hatmaxkit/hatmax/config"
	"github.com/hatmaxkit/hatmax/log"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Database wraps a sql.DB connection with lifecycle management.
type Database struct {
	DB               *sql.DB
	assetsFS         embed.FS
	engine           string
	connectionString string
	schema           string
	log              log.Logger
}

// New creates a new Database instance from configuration.
func New(assetsFS embed.FS, engine string, cfg *config.Config, log log.Logger) *Database {
	return &Database{
		assetsFS:         assetsFS,
		engine:           engine,
		connectionString: cfg.Database.ConnectionString(),
		schema:           cfg.Database.Schema,
		log:              log,
	}
}

// Start opens the database connection and verifies connectivity.
func (d *Database) Start(ctx context.Context) error {
	db, err := sql.Open("pgx", d.connectionString)
	if err != nil {
		return fmt.Errorf("cannot open database: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()

		return fmt.Errorf("cannot ping database: %w", err)
	}

	d.DB = db
	d.log.Info("Database connection established")

	err = d.ensureSchema(ctx)
	if err != nil {
		return fmt.Errorf("cannot ensure schema: %w", err)
	}

	return nil
}

// Stop closes the database connection.
// Implements app.Stoppable interface.
func (d *Database) Stop(ctx context.Context) error {
	if d.DB != nil {
		d.log.Info("Closing database connection")

		return d.DB.Close()
	}

	return nil
}

// GetDB returns the underlying *sql.DB connection.
func (d *Database) GetDB() *sql.DB {
	return d.DB
}

func (d *Database) ensureSchema(ctx context.Context) error {
	if d.schema == "" {
		return nil
	}

	query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", d.schema)

	_, err := d.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("cannot create schema %s: %w", d.schema, err)
	}

	d.log.Infof("Schema %s ensured", d.schema)

	return nil
}
