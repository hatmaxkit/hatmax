package seed

import (
	"context"
	"database/sql"
	"fmt"

	"hatmax.adrianpk.com/log"
)

// DBProvider is the interface for database access.
type DBProvider interface {
	GetDB() *sql.DB
}

// Runner orchestrates the execution of registered seeders.
// It checks the tracker to skip already-applied seeds and marks new ones as applied.
// Implements app.Startable for lifecycle integration.
type Runner struct {
	dbProvider DBProvider
	tracker    *Tracker
	seeders    []Seeder
	log        log.Logger
}

// NewRunner creates a new Runner with the given database, seeders, and logger.
// Seeders are executed in the order provided.
func NewRunner(db DBProvider, seeders []Seeder, log log.Logger) *Runner {
	return &Runner{
		dbProvider: db,
		seeders:    seeders,
		log:        log,
	}
}

// Register adds a seeder to be executed when Start/Run is called.
// Seeders are executed in registration order.
// Prefer passing seeders to NewRunner instead.
func (r *Runner) Register(s Seeder) {
	r.seeders = append(r.seeders, s)
}

// Start implements app.Startable and executes all registered seeders that haven't been applied yet.
// Creates the _seeds tracking table if it doesn't exist.
// Each seeder is run in order, and failures stop execution.
func (r *Runner) Start(ctx context.Context) error {
	// Initialize tracker lazily - DB connection is only available after database.Start()
	if r.tracker == nil {
		r.tracker = NewTracker(r.dbProvider.GetDB())
	}

	err := r.tracker.EnsureTable(ctx)
	if err != nil {
		return fmt.Errorf("ensure seeds table: %w", err)
	}

	for _, seeder := range r.seeders {
		name := seeder.Name()

		applied, err := r.tracker.IsApplied(ctx, name)
		if err != nil {
			return fmt.Errorf("check seed %s: %w", name, err)
		}

		if applied {
			r.log.Debugf("Seed %s already applied, skipping", name)

			continue
		}

		r.log.Infof("Applying seed: %s", name)

		err = seeder.Seed(ctx)
		if err != nil {
			return fmt.Errorf("seed %s: %w", name, err)
		}

		err = r.tracker.MarkApplied(ctx, name)
		if err != nil {
			return fmt.Errorf("mark seed %s applied: %w", name, err)
		}

		r.log.Infof("Seed %s applied successfully", name)
	}

	return nil
}

// Run is an alias for Start for backward compatibility.
func (r *Runner) Run(ctx context.Context) error {
	return r.Start(ctx)
}
