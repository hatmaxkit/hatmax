package db

import (
	"context"
	"embed"
	"testing"

	"github.com/hatmaxkit/hatmax/config"
	"github.com/hatmaxkit/hatmax/log"
)

//go:embed testdata
var testAssetsFS embed.FS

func TestNew(t *testing.T) {
	logger := log.NewTestLogger("error")
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "dev",
			Password: "dev",
			Database: "dev",
			SSLMode:  "disable",
			Schema:   "test",
		},
	}

	db := New(testAssetsFS, "postgres", cfg, logger)

	if db == nil {
		t.Error("expected database to be created")
	}

	if db.DB != nil {
		t.Error("expected DB to be nil before Start")
	}

	if db.engine != "postgres" {
		t.Errorf("expected engine to be postgres, got %s", db.engine)
	}

	expectedConnStr := cfg.Database.ConnectionString()
	if db.connectionString != expectedConnStr {
		t.Errorf("expected connectionString to be %s, got %s", expectedConnStr, db.connectionString)
	}

	if db.schema != cfg.Database.Schema {
		t.Errorf("expected schema to be %s, got %s", cfg.Database.Schema, db.schema)
	}
}

func TestSetMigrationPath(t *testing.T) {
	logger := log.NewTestLogger("error")
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host: "localhost",
		},
	}

	db := New(testAssetsFS, "postgres", cfg, logger)

	customPath := "custom/migration/path"
	db.SetMigrationPath(customPath)

	if db.migrationPath != customPath {
		t.Errorf("expected migrationPath to be %s, got %s", customPath, db.migrationPath)
	}
}

func TestStopWithoutStart(t *testing.T) {
	logger := log.NewTestLogger("error")
	cfg := &config.Config{}

	db := New(testAssetsFS, "postgres", cfg, logger)
	ctx := context.Background()

	if err := db.Stop(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetDBBeforeStart(t *testing.T) {
	logger := log.NewTestLogger("error")
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host: "localhost",
		},
	}

	db := New(testAssetsFS, "postgres", cfg, logger)

	if db.GetDB() != nil {
		t.Error("expected GetDB to return nil before Start")
	}
}
