package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/hatmaxkit/hatmax/log"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Database wraps a sql.DB connection with lifecycle management and migrations.
type Database struct {
	DB               *sql.DB
	assetsFS         embed.FS
	engine           string
	connectionString string
	schema           string
	migrationPath    string
	log              logger.Logger
}

// Config holds database configuration.
type Config struct {
	ConnectionString string
	Schema           string
}

// New creates a new Database instance.
// The engine parameter specifies the database type (e.g., "postgres").
// Migrations will be loaded from assets/migration/{engine}/ by default.
func New(assetsFS embed.FS, engine string, cfg Config, log logger.Logger) *Database {
	return &Database{
		assetsFS:         assetsFS,
		engine:           engine,
		connectionString: cfg.ConnectionString,
		schema:           cfg.Schema,
		log:              log,
	}
}

// SetMigrationPath sets a custom migration path.
// If not set, defaults to assets/migration/{engine}/
func (d *Database) SetMigrationPath(path string) {
	d.migrationPath = path
}

// Start opens the database connection, verifies connectivity, and runs migrations.
// Implements app.Startable interface.
func (d *Database) Start(ctx context.Context) error {
	db, err := sql.Open("pgx", d.connectionString)
	if err != nil {
		return fmt.Errorf("cannot open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("cannot ping database: %w", err)
	}

	d.DB = db
	d.log.Info("Database connection established")

	if err := d.ensureSchema(ctx); err != nil {
		return fmt.Errorf("cannot ensure schema: %w", err)
	}

	migrator := newMigrator(d.assetsFS, d.engine, d.log)
	migrator.setDB(d.DB)
	if d.migrationPath != "" {
		migrator.setPath(d.migrationPath)
	}
	if err := migrator.run(ctx); err != nil {
		return fmt.Errorf("cannot run migrations: %w", err)
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
	if _, err := d.DB.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("cannot create schema %s: %w", d.schema, err)
	}

	d.log.Infof("Schema %s ensured", d.schema)
	return nil
}
