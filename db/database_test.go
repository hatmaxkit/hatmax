package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

//go:embed testdata
var testAssetsFS embed.FS

func setupTestDBWithConfig(t *testing.T) (*config.Config, func()) {
	t.Helper()

	ctx := context.Background()

	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		return setupCIDatabaseWithConfig(t, ctx)
	}

	return setupTestContainerWithConfig(t, ctx)
}

func setupCIDatabaseWithConfig(t *testing.T, ctx context.Context) (*config.Config, func()) {
	t.Helper()

	dbHost := os.Getenv("DB_HOST")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "postgres")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "postgres")
	dbName := getEnvOrDefault("DB_NAME", "postgres")

	schema := fmt.Sprintf("test_%d_%s", time.Now().UnixNano(), randomString(8))

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("cannot open database: %v", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema))
	if err != nil {
		t.Fatalf("cannot create schema %s: %v", schema, err)
	}

	port := 5432
	fmt.Sscanf(dbPort, "%d", &port)

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     dbHost,
			Port:     port,
			User:     dbUser,
			Password: dbPassword,
			Database: dbName,
			Schema:   schema,
			SSLMode:  "disable",
		},
	}

	cleanup := func() {
		db, err := sql.Open("pgx", connStr)
		if err != nil {
			t.Logf("cannot open database for cleanup: %v", err)

			return
		}
		defer db.Close()

		_, _ = db.ExecContext(context.Background(),
			fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
	}

	return cfg, cleanup
}

func setupTestContainerWithConfig(t *testing.T, ctx context.Context) (*config.Config, func()) {
	t.Helper()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2)),
	)
	if err != nil {
		t.Fatalf("cannot start postgres container: %v", err)
	}

	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("cannot get container host: %v", err)
	}

	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("cannot get container port: %v", err)
	}

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     host,
			Port:     port.Int(),
			User:     "postgres",
			Password: "postgres",
			Database: "testdb",
			Schema:   "public",
			SSLMode:  "disable",
		},
	}

	cleanup := func() {
		terminateErr := pgContainer.Terminate(context.Background())
		if terminateErr != nil {
			t.Logf("cannot terminate container: %v", terminateErr)
		}
	}

	return cfg, cleanup
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, length)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}

	return string(b)
}

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

func TestStopWithoutStart(t *testing.T) {
	logger := log.NewTestLogger("error")
	cfg := &config.Config{}

	db := New(testAssetsFS, "postgres", cfg, logger)
	ctx := context.Background()

	err := db.Stop(ctx)
	if err != nil {
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

func TestStartAndStopIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg, cleanup := setupTestDBWithConfig(t)
	defer cleanup()

	logger := log.NewTestLogger("error")
	db := New(testAssetsFS, "postgres", cfg, logger)
	ctx := context.Background()

	err := db.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	migrator := NewMigrator(db, testAssetsFS, "postgres", logger)
	migrator.SetPath("testdata/migration/postgres")

	err = migrator.Start(ctx)
	if err != nil {
		t.Fatalf("Migrator.Start() error = %v", err)
	}

	if db.GetDB() == nil {
		t.Error("expected GetDB to return non-nil after Start")
	}

	err = db.DB.PingContext(ctx)
	if err != nil {
		t.Errorf("cannot ping database after Start: %v", err)
	}

	err = db.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestStartWithSchemaCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg, cleanup := setupTestDBWithConfig(t)
	defer cleanup()

	cfg.Database.Schema = "test_schema_" + randomString(8)

	logger := log.NewTestLogger("error")
	db := New(testAssetsFS, "postgres", cfg, logger)
	ctx := context.Background()

	err := db.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer db.Stop(ctx)

	migrator := NewMigrator(db, testAssetsFS, "postgres", logger)
	migrator.SetPath("testdata/migration/postgres")

	err = migrator.Start(ctx)
	if err != nil {
		t.Fatalf("Migrator.Start() error = %v", err)
	}

	var schemaExists bool

	err = db.DB.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)",
		cfg.Database.Schema).Scan(&schemaExists)
	if err != nil {
		t.Errorf("cannot check schema existence: %v", err)
	}

	if !schemaExists {
		t.Error("expected schema to be created")
	}
}
