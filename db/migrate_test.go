package db

import (
	"embed"
	"testing"

	"github.com/hatmaxkit/hatmax/log"
)

//go:embed testdata
var migrateTestAssetsFS embed.FS

func TestNewMigrator(t *testing.T) {
	logger := log.NewTestLogger("error")

	tests := []struct {
		name   string
		engine string
	}{
		{
			name:   "postgres engine",
			engine: "postgres",
		},
		{
			name:   "sqlite engine",
			engine: "sqlite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrator := newMigrator(migrateTestAssetsFS, tt.engine, logger)

			if migrator == nil {
				t.Error("expected migrator to be created")
			}

			if migrator.engine != tt.engine {
				t.Errorf("expected engine %q, got %q", tt.engine, migrator.engine)
			}
		})
	}
}

func TestMigratorSetPath(t *testing.T) {
	logger := log.NewTestLogger("error")
	migrator := newMigrator(migrateTestAssetsFS, "postgres", logger)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "custom path",
			path: "custom/migrations",
		},
		{
			name: "empty path",
			path: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrator.setPath(tt.path)

			if migrator.path != tt.path {
				t.Errorf("expected path %q, got %q", tt.path, migrator.path)
			}
		})
	}
}

func TestMigratorLoadFileMigrations(t *testing.T) {
	logger := log.NewTestLogger("error")
	migrator := newMigrator(migrateTestAssetsFS, "postgres", logger)
	migrator.setPath("testdata/migration/postgres")

	migrations, err := migrator.loadFileMigrations()
	if err != nil {
		t.Fatalf("loadFileMigrations failed: %v", err)
	}

	if len(migrations) == 0 {
		t.Error("expected at least one migration file")
	}

	// Verify migration structure
	for _, m := range migrations {
		if m.Datetime == "" {
			t.Error("expected migration to have datetime")
		}
		if m.Name == "" {
			t.Error("expected migration to have name")
		}
		if m.Up == "" {
			t.Error("expected migration to have Up section")
		}
	}
}

func TestMigratorLoadFileMigrationsOrdering(t *testing.T) {
	logger := log.NewTestLogger("error")
	migrator := newMigrator(migrateTestAssetsFS, "postgres", logger)
	migrator.setPath("testdata/migration/postgres")

	migrations, err := migrator.loadFileMigrations()
	if err != nil {
		t.Fatalf("loadFileMigrations failed: %v", err)
	}

	// Verify migrations are sorted by datetime
	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].Datetime > migrations[i].Datetime {
			t.Errorf("migrations not sorted: %s > %s", migrations[i-1].Datetime, migrations[i].Datetime)
		}
	}
}

func TestMigratorFindPendingMigrations(t *testing.T) {
	logger := log.NewTestLogger("error")
	migrator := newMigrator(migrateTestAssetsFS, "postgres", logger)

	fileMigrations := []Migration{
		{Datetime: "20250101", Name: "create-users"},
		{Datetime: "20250102", Name: "create-posts"},
		{Datetime: "20250103", Name: "create-comments"},
	}

	dbMigrations := []Migration{
		{Datetime: "20250101", Name: "create-users"},
	}

	pending := migrator.findPendingMigrations(fileMigrations, dbMigrations)

	if len(pending) != 2 {
		t.Errorf("expected 2 pending migrations, got %d", len(pending))
	}

	if pending[0].Name != "create-posts" {
		t.Errorf("expected first pending to be create-posts, got %s", pending[0].Name)
	}

	if pending[1].Name != "create-comments" {
		t.Errorf("expected second pending to be create-comments, got %s", pending[1].Name)
	}
}

func TestMigratorFindPendingMigrationsNone(t *testing.T) {
	logger := log.NewTestLogger("error")
	migrator := newMigrator(migrateTestAssetsFS, "postgres", logger)

	fileMigrations := []Migration{
		{Datetime: "20250101", Name: "create-users"},
		{Datetime: "20250102", Name: "create-posts"},
	}

	dbMigrations := []Migration{
		{Datetime: "20250101", Name: "create-users"},
		{Datetime: "20250102", Name: "create-posts"},
	}

	pending := migrator.findPendingMigrations(fileMigrations, dbMigrations)

	if len(pending) != 0 {
		t.Errorf("expected 0 pending migrations, got %d", len(pending))
	}
}
