package mongo

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/adrianpk/hatmax-ref/services/todo/internal/todo"
)

func setupTestMongoDB(t *testing.T) (*mongo.Database, func()) {
	t.Helper()

	// Use MongoDB container or local instance for testing
	mongoURI := os.Getenv("MONGO_TEST_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	// Create unique database name for test isolation
	dbName := "List_test_" + uuid.New().String()[:8]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skipf("MongoDB not available for testing: %v", err)
		return nil, func() {}
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("Cannot ping MongoDB: %v", err)
		return nil, func() {}
	}

	db := client.Database(dbName)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Drop test database
		if err := db.Drop(ctx); err != nil {
			t.Errorf("Failed to drop test database: %v", err)
		}

		// Close client
		if err := client.Disconnect(ctx); err != nil {
			t.Errorf("Failed to disconnect from MongoDB: %v", err)
		}
	}

	return db, cleanup
}

func setupRepo(t *testing.T, db *mongo.Database) *ListMongoRepo {
	t.Helper()

	// Create repository directly with test database
	repo := NewListMongoRepo(db)

	return repo
}

// TODO: Expand tests to achieve 100% coverage
// Current tests cover happy path and basic corner cases for each repository method.
// Future expansion should include:
// - More edge cases and error conditions
// - Concurrent access scenarios
// - Database constraint violations
// - Large dataset performance tests
// - Complex document query scenarios

// TestListMongoRepoCreate tests the Create method with various scenarios
func TestListMongoRepoCreate(t *testing.T) {
	db, cleanup := setupTestMongoDB(t)
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

// TestListMongoRepoGet tests the Get method with various scenarios
func TestListMongoRepoGet(t *testing.T) {
	db, cleanup := setupTestMongoDB(t)
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

// TestListMongoRepoSave tests the Save method with various scenarios
func TestListMongoRepoSave(t *testing.T) {
	db, cleanup := setupTestMongoDB(t)
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
		// - Large document updates
		// - Concurrent update scenarios
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

// TestListMongoRepoDelete tests the Delete method with various scenarios
func TestListMongoRepoDelete(t *testing.T) {
	db, cleanup := setupTestMongoDB(t)
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
		// - Delete with complex document structure
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

			// Verify document doesn't exist in collection
			collection := db.Collection("lists")
			count, err := collection.CountDocuments(ctx, bson.M{"_id": id.String()})
			if err != nil {
				t.Errorf("Failed to check document count: %v", err)
			}
			if count != 0 {
				t.Errorf("Expected 0 documents, found %d", count)
			}
		})
	}
}

// TestListMongoRepoList tests the List method with various scenarios
func TestListMongoRepoList(t *testing.T) {
	db, cleanup := setupTestMongoDB(t)
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
		// - Performance with complex documents
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
func cleanDatabase(t *testing.T, db *mongo.Database) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Drop all collections in the test database
	collections, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		t.Fatalf("Failed to list collections: %v", err)
	}

	for _, name := range collections {
		if err := db.Collection(name).Drop(ctx); err != nil {
			t.Fatalf("Failed to drop collection %s: %v", name, err)
		}
	}
}

func TestListMongoRepoErrorCases(t *testing.T) {
	db, cleanup := setupTestMongoDB(t)
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
