package mongo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/adrianpk/hatmax-ref/services/todo/internal/todo"
)

// ListMongoRepo implements the ListRepo interface using MongoDB.
// MongoDB is ideal for aggregates since each aggregate can be stored as a single document.
type ListMongoRepo struct {
	collection *mongo.Collection
}

// NewListMongoRepo creates a new MongoDB repository for List aggregates.
func NewListMongoRepo(db *mongo.Database) *ListMongoRepo {
	return &ListMongoRepo{
		collection: db.Collection("lists"),
	}
}

// Create creates a new List aggregate in MongoDB.
// The entire aggregate (root + children) is stored as a single document.
func (r *ListMongoRepo) Create(ctx context.Context, aggregate *todo.List) error {
	if aggregate == nil {
		return fmt.Errorf("aggregate cannot be nil")
	}

	aggregate.EnsureID()
	aggregate.BeforeCreate()

	_, err := r.collection.InsertOne(ctx, aggregate)
	if err != nil {
		return fmt.Errorf("could not create List aggregate: %w", err)
	}

	return nil
}

// Get retrieves a complete List aggregate by ID from MongoDB.
// Returns the aggregate root with all its child entities loaded.
func (r *ListMongoRepo) Get(ctx context.Context, id uuid.UUID) (*todo.List, error) {
	var aggregate todo.List

	filter := bson.M{"_id": id.String()}
	err := r.collection.FindOne(ctx, filter).Decode(&aggregate)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("List aggregate with ID %s not found", id.String())
		}
		return nil, fmt.Errorf("could not get List aggregate: %w", err)
	}

	return &aggregate, nil
}

// Save performs a unit-of-work save operation on the List aggregate.
// In MongoDB, this is straightforward since the entire aggregate is replaced as one document.
func (r *ListMongoRepo) Save(ctx context.Context, aggregate *todo.List) error {
	if aggregate == nil {
		return fmt.Errorf("aggregate cannot be nil")
	}

	aggregate.BeforeUpdate()

	filter := bson.M{"_id": aggregate.GetID().String()}
	opts := options.Replace().SetUpsert(false)

	result, err := r.collection.ReplaceOne(ctx, filter, aggregate, opts)
	if err != nil {
		return fmt.Errorf("could not save List aggregate: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("List aggregate with ID %s not found for update", aggregate.GetID().String())
	}

	return nil
}

// Delete removes the entire List aggregate from MongoDB.
// This automatically removes all child entities since they're part of the same document.
func (r *ListMongoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id.String()}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("could not delete List aggregate: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("List aggregate with ID %s not found for deletion", id.String())
	}

	return nil
}

// List retrieves all List aggregates from MongoDB.
// Each document contains the complete aggregate (root + children).
func (r *ListMongoRepo) List(ctx context.Context) ([]*todo.List, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("could not list List aggregates: %w", err)
	}
	defer cursor.Close(ctx)

	var aggregates []*todo.List

	for cursor.Next(ctx) {
		var aggregate todo.List
		if err := cursor.Decode(&aggregate); err != nil {
			return nil, fmt.Errorf("could not decode List aggregate: %w", err)
		}
		aggregates = append(aggregates, &aggregate)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing List aggregates: %w", err)
	}

	return aggregates, nil
}
