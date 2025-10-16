package sqlite

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/adrianpk/hatmax-ref/services/todo/internal/config"
	"github.com/adrianpk/hatmax-ref/services/todo/internal/todo"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "List_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp database file: %v", err)
	}
	tmpFile.Close()

	db, err := sql.Open("sqlite3", tmpFile.Name()+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create tables
	if err := createTestTables(db); err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
		// Remove temporary database file
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Errorf("Failed to remove temp database file: %v", err)
		}
	}

	return db, cleanup
}

func createTestTables(db *sql.DB) error {
	// Create lists table
	_, err := db.Exec(`
		CREATE TABLE lists (
			id TEXT PRIMARY KEY,
			-- TODO: Add proper column definitions for name, description
			name TEXT,
			description TEXT,
			created_at DATETIME,
			created_by TEXT,
			updated_at DATETIME,
			updated_by TEXT
		)
	`)
	if err != nil {
		return err
	}

	// Create items table
	_, err = db.Exec(`
		CREATE TABLE items (
			id TEXT PRIMARY KEY,
			List_id TEXT NOT NULL,
			-- TODO: Add proper column definitions for text, done
			name TEXT,
			color TEXT,
			text TEXT,
			done BOOLEAN,
			created_at DATETIME,
			created_by TEXT,
			updated_at DATETIME,
			updated_by TEXT,
			FOREIGN KEY (List_id) REFERENCES lists(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Create tags table
	_, err = db.Exec(`
		CREATE TABLE tags (
			id TEXT PRIMARY KEY,
			List_id TEXT NOT NULL,
			-- TODO: Add proper column definitions for color, name
			name TEXT,
			color TEXT,
			text TEXT,
			done BOOLEAN,
			created_at DATETIME,
			created_by TEXT,
			updated_at DATETIME,
			updated_by TEXT,
			FOREIGN KEY (List_id) REFERENCES lists(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

func setupRepo(t *testing.T, db *sql.DB) *ListSQLiteRepo {
	t.Helper()

	// Mock xparams with test database configuration
	xparams := config.XParams{
		Cfg: &config.Config{
			Database: config.DatabaseConfig{
				Path: ":memory:", // Not used since we're injecting the db directly
			},
		},
	}

	repo := NewListSQLiteRepo(xparams)
	// Inject the test database directly (bypassing Start() method)
	repo.db = db

	return repo
}

// TODO: Expand tests to achieve 100% coverage
// Current tests cover happy path and basic corner cases for each repository method.
// Future expansion should include:
// - More edge cases and error conditions
// - Concurrent access scenarios
// - Database constraint violations
// - Large dataset performance tests
// - Transactional rollback scenarios

// TestListSQLiteRepoCreate tests the Create method with various scenarios
func TestListSQLiteRepoCreate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := setupRepo(t, db)
	ctx := context.Background()

	tests := []struct {
		name        string
		aggregate   *todo.List
		expectError bool
		errorMsg    string
	}{
		{
			name: "HappyPath_EmptyAggregate",
			aggregate: &todo.List{
				// TODO: Set appropriate test values for root fields

				Items: []todo.Item{},

				Tags: []todo.Tag{},
			},
			expectError: false,
		},
		{
			name: "HappyPath_WithChildren",
			aggregate: &todo.List{
				// TODO: Set appropriate test values for root fields

				Items: []todo.Item{
					{
						// TODO: Set appropriate test values for child fields
					},
					{
						// TODO: Set appropriate test values for second child
					},
				},

				Tags: []todo.Tag{
					{
						// TODO: Set appropriate test values for child fields
					},
					{
						// TODO: Set appropriate test values for second child
					},
				},
			},
			expectError: false,
		},
		{
			name:        "EdgeCase_NilAggregate",
			aggregate:   nil,
			expectError: true,
			errorMsg:    "aggregate cannot be nil",
		},
		// TODO: Add more edge cases:
		// - Aggregate with invalid field values
		// - Aggregate with duplicate child IDs
		// - Aggregate exceeding size limits
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanDatabase(t, db)

			err := repo.Create(ctx, tt.aggregate)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify ID was set
			if tt.aggregate.GetID() == uuid.Nil {
				t.Error("Aggregate ID should be set after creation")
			}

			// Verify aggregate can be retrieved
			retrieved, err := repo.Get(ctx, tt.aggregate.GetID())
			if err != nil {
				t.Errorf("Failed to retrieve created aggregate: %v", err)
				return
			}

			// Verify children count matches

			if len(retrieved.Items) != len(tt.aggregate.Items) {
				t.Errorf("Expected %d Items, got %d", len(tt.aggregate.Items), len(retrieved.Items))
			}

			if len(retrieved.Tags) != len(tt.aggregate.Tags) {
				t.Errorf("Expected %d Tags, got %d", len(tt.aggregate.Tags), len(retrieved.Tags))
			}

		})
	}
}

// TestListSQLiteRepoGet tests the Get method with various scenarios
func TestListSQLiteRepoGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := setupRepo(t, db)
	ctx := context.Background()

	// Setup: Create a test aggregate
	testAggregate := &todo.List{
		// TODO: Set appropriate test values

		Items: []todo.Item{
			{
				// TODO: Set test values for child fields
			},
		},

		Tags: []todo.Tag{
			{
				// TODO: Set test values for child fields
			},
		},
	}
	err := repo.Create(ctx, testAggregate)
	if err != nil {
		t.Fatalf("Failed to create test aggregate: %v", err)
	}

	tests := []struct {
		name        string
		id          uuid.UUID
		expectError bool
		errorMsg    string
	}{
		{
			name:        "HappyPath_ExistingAggregate",
			id:          testAggregate.GetID(),
			expectError: false,
		},
		{
			name:        "EdgeCase_NonExistentID",
			id:          uuid.New(),
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "EdgeCase_NilUUID",
			id:          uuid.Nil,
			expectError: true,
		},
		// TODO: Add more edge cases:
		// - Malformed UUID handling
		// - Database connection errors
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.Get(ctx, tt.id)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Expected aggregate but got nil")
				return
			}

			// Verify aggregate has correct ID
			if result.GetID() != tt.id {
				t.Errorf("Expected ID %v, got %v", tt.id, result.GetID())
			}

			// Verify children are loaded

			if len(result.Items) != len(testAggregate.Items) {
				t.Errorf("Expected %d Items, got %d", len(testAggregate.Items), len(result.Items))
			}

			if len(result.Tags) != len(testAggregate.Tags) {
				t.Errorf("Expected %d Tags, got %d", len(testAggregate.Tags), len(result.Tags))
			}

		})
	}
}

// TestListSQLiteRepoSave tests the Save method with various scenarios
func TestListSQLiteRepoSave(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := setupRepo(t, db)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() *todo.List
		modify      func(agg *todo.List)
		expectError bool
		errorMsg    string
	}{
		{
			name: "HappyPath_AddChildren",
			setup: func() *todo.List {
				agg := &todo.List{
					// TODO: Set test values

					Items: []todo.Item{},

					Tags: []todo.Tag{},
				}
				repo.Create(ctx, agg)
				return agg
			},
			modify: func(agg *todo.List) {

				agg.Items = append(agg.Items, todo.Item{
					// TODO: Set test values for new child
				})

				agg.Tags = append(agg.Tags, todo.Tag{
					// TODO: Set test values for new child
				})

			},
		},
		{
			name: "HappyPath_UpdateChildren",
			setup: func() *todo.List {
				agg := &todo.List{

					Items: []todo.Item{
						{
							// TODO: Set initial test values
						},
					},

					Tags: []todo.Tag{
						{
							// TODO: Set initial test values
						},
					},
				}
				repo.Create(ctx, agg)
				return agg
			},
			modify: func(agg *todo.List) {

				if len(agg.Items) > 0 {
					// TODO: Modify child values for update test
				}

				if len(agg.Tags) > 0 {
					// TODO: Modify child values for update test
				}

			},
		},
		{
			name: "HappyPath_RemoveChildren",
			setup: func() *todo.List {
				agg := &todo.List{

					Items: []todo.Item{
						{ /* TODO: child 1 */ },
						{ /* TODO: child 2 */ },
					},

					Tags: []todo.Tag{
						{ /* TODO: child 1 */ },
						{ /* TODO: child 2 */ },
					},
				}
				repo.Create(ctx, agg)
				return agg
			},
			modify: func(agg *todo.List) {

				if len(agg.Items) > 1 {
					agg.Items = agg.Items[:1] // Keep only first child
				}

				if len(agg.Tags) > 1 {
					agg.Tags = agg.Tags[:1] // Keep only first child
				}

			},
		},
		{
			name:        "EdgeCase_NilAggregate",
			setup:       func() *todo.List { return nil },
			modify:      func(agg *todo.List) {},
			expectError: true,
			errorMsg:    "aggregate cannot be nil",
		},
		// TODO: Add more scenarios:
		// - Complex child modifications
		// - Transaction rollback scenarios
		// - Large batch updates
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanDatabase(t, db)

			agg := tt.setup()
			if agg != nil {
				tt.modify(agg)
			}

			err := repo.Save(ctx, agg)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify changes were persisted
			retrieved, err := repo.Get(ctx, agg.GetID())
			if err != nil {
				t.Errorf("Failed to retrieve saved aggregate: %v", err)
				return
			}

			// Verify child counts match

			if len(retrieved.Items) != len(agg.Items) {
				t.Errorf("Expected %d Items, got %d", len(agg.Items), len(retrieved.Items))
			}

			if len(retrieved.Tags) != len(agg.Tags) {
				t.Errorf("Expected %d Tags, got %d", len(agg.Tags), len(retrieved.Tags))
			}

		})
	}
}

// TestListSQLiteRepoDelete tests the Delete method with various scenarios
func TestListSQLiteRepoDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := setupRepo(t, db)
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func() uuid.UUID
		expectError bool
		errorMsg    string
	}{
		{
			name: "HappyPath_ExistingAggregate",
			setup: func() uuid.UUID {
				agg := &todo.List{

					Items: []todo.Item{
						{ /* TODO: test child */ },
					},

					Tags: []todo.Tag{
						{ /* TODO: test child */ },
					},
				}
				repo.Create(ctx, agg)
				return agg.GetID()
			},
		},
		{
			name:        "EdgeCase_NonExistentID",
			setup:       func() uuid.UUID { return uuid.New() },
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "EdgeCase_NilUUID",
			setup:       func() uuid.UUID { return uuid.Nil },
			expectError: true,
		},
		// TODO: Add more edge cases:
		// - Delete with foreign key constraints
		// - Concurrent delete scenarios
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanDatabase(t, db)

			id := tt.setup()

			err := repo.Delete(ctx, id)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify aggregate is gone
			_, err = repo.Get(ctx, id)
			if err == nil {
				t.Error("Aggregate should not exist after deletion")
			}

			// Verify children are gone (cascade delete)

			var countItems int
			err = db.QueryRow("SELECT COUNT(*) FROM items WHERE List_id = ?", id.String()).Scan(&countItems)
			if err != nil {
				t.Errorf("Failed to check Items count: %v", err)
			}
			if countItems != 0 {
				t.Errorf("Expected 0 Items, found %d", countItems)
			}

			var countTags int
			err = db.QueryRow("SELECT COUNT(*) FROM tags WHERE List_id = ?", id.String()).Scan(&countTags)
			if err != nil {
				t.Errorf("Failed to check Tags count: %v", err)
			}
			if countTags != 0 {
				t.Errorf("Expected 0 Tags, found %d", countTags)
			}

		})
	}
}

// TestListSQLiteRepoList tests the List method with various scenarios
func TestListSQLiteRepoList(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := setupRepo(t, db)
	ctx := context.Background()

	tests := []struct {
		name          string
		setup         func() []*todo.List
		expectedCount int
		expectError   bool
	}{
		{
			name:          "HappyPath_EmptyList",
			setup:         func() []*todo.List { return nil },
			expectedCount: 0,
		},
		{
			name: "HappyPath_SingleAggregate",
			setup: func() []*todo.List {
				agg := &todo.List{

					Items: []todo.Item{
						{ /* TODO: test child */ },
					},

					Tags: []todo.Tag{
						{ /* TODO: test child */ },
					},
				}
				repo.Create(ctx, agg)
				return []*todo.List{agg}
			},
			expectedCount: 1,
		},
		{
			name: "HappyPath_MultipleAggregates",
			setup: func() []*todo.List {
				aggs := make([]*todo.List, 3)
				for i := 0; i < 3; i++ {
					aggs[i] = &todo.List{

						Items: []todo.Item{
							{ /* TODO: test child */ },
						},

						Tags: []todo.Tag{
							{ /* TODO: test child */ },
						},
					}
					repo.Create(ctx, aggs[i])
				}
				return aggs
			},
			expectedCount: 3,
		},
		// TODO: Add more scenarios:
		// - Large dataset pagination
		// - Filtering and sorting
		// - Performance with many children
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanDatabase(t, db)

			created := tt.setup()

			results, err := repo.List(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d aggregates, got %d", tt.expectedCount, len(results))
				return
			}

			// Verify each aggregate has children loaded
			for i, result := range results {
				if result == nil {
					t.Errorf("Aggregate %d is nil", i)
					continue
				}

				if created != nil && i < len(created) {

					if len(result.Items) != len(created[i].Items) {
						t.Errorf("Aggregate %d expected %d Items, got %d", i, len(created[i].Items), len(result.Items))
					}

					if len(result.Tags) != len(created[i].Tags) {
						t.Errorf("Aggregate %d expected %d Tags, got %d", i, len(created[i].Tags), len(result.Tags))
					}

				}
			}
		})
	}
}

// Helper function to clean database between tests
func cleanDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec("DELETE FROM lists")
	if err != nil {
		t.Fatalf("Failed to clean lists table: %v", err)
	}

	_, err = db.Exec("DELETE FROM items")
	if err != nil {
		t.Fatalf("Failed to clean items table: %v", err)
	}

	_, err = db.Exec("DELETE FROM tags")
	if err != nil {
		t.Fatalf("Failed to clean tags table: %v", err)
	}

}

func TestListSQLiteRepoErrorCases(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := setupRepo(t, db)
	ctx := context.Background()

	t.Run("GetNonExistentAggregate", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := repo.Get(ctx, nonExistentID)
		if err == nil {
			t.Error("Should return error for non-existent aggregate")
		}
	})

	t.Run("DeleteNonExistentAggregate", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(ctx, nonExistentID)
		if err == nil {
			t.Error("Should return error when deleting non-existent aggregate")
		}
	})

	t.Run("CreateNilAggregate", func(t *testing.T) {
		err := repo.Create(ctx, nil)
		if err == nil {
			t.Error("Should return error for nil aggregate")
		}
	})

	t.Run("SaveNilAggregate", func(t *testing.T) {
		err := repo.Save(ctx, nil)
		if err == nil {
			t.Error("Should return error for nil aggregate")
		}
	})
}
