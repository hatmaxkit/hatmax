package db

import (
	"context"
	"embed"
	"testing"

	"github.com/hatmaxkit/hatmax/log"
)

//go:embed testdata
var testAssetsFS embed.FS

func TestNew(t *testing.T) {
	log := logger.NewLogger("error")
	cfg := Config{
		ConnectionString: "host=localhost port=5432 user=dev password=dev dbname=dev sslmode=disable",
		Schema:           "test",
	}

	db := New(testAssetsFS, "postgres", cfg, log)

	if db == nil {
		t.Error("expected database to be created")
	}

	if db.DB != nil {
		t.Error("expected DB to be nil before Start")
	}

	if db.engine != "postgres" {
		t.Errorf("expected engine to be postgres, got %s", db.engine)
	}

	if db.connectionString != cfg.ConnectionString {
		t.Errorf("expected connectionString to be %s, got %s", cfg.ConnectionString, db.connectionString)
	}

	if db.schema != cfg.Schema {
		t.Errorf("expected schema to be %s, got %s", cfg.Schema, db.schema)
	}
}

func TestSetMigrationPath(t *testing.T) {
	log := logger.NewLogger("error")
	cfg := Config{
		ConnectionString: "host=localhost",
	}

	db := New(testAssetsFS, "postgres", cfg, log)

	customPath := "custom/migration/path"
	db.SetMigrationPath(customPath)

	if db.migrationPath != customPath {
		t.Errorf("expected migrationPath to be %s, got %s", customPath, db.migrationPath)
	}
}

func TestStopWithoutStart(t *testing.T) {
	log := logger.NewLogger("error")
	cfg := Config{}

	db := New(testAssetsFS, "postgres", cfg, log)
	ctx := context.Background()

	if err := db.Stop(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetDBBeforeStart(t *testing.T) {
	log := logger.NewLogger("error")
	cfg := Config{
		ConnectionString: "host=localhost",
	}

	db := New(testAssetsFS, "postgres", cfg, log)

	if db.GetDB() != nil {
		t.Error("expected GetDB to return nil before Start")
	}
}
