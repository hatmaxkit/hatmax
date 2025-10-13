package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/google/uuid"
)

// MongoTagRepo implements the TagRepo interface for MongoDB.
type MongoTagRepo struct {
	collection *mongo.Collection
}

// NewMongoTagRepo creates a new MongoTagRepo.
func NewMongoTagRepo(db *mongo.Database) *MongoTagRepo {
	return &MongoTagRepo{
		collection: db.Collection("tags"),
	}
}

// Create inserts a new Tag into the database.
func (r *MongoTagRepo) Create(ctx context.Context, item *Tag) error {
	// TODO: Handle item.BeforeCreate() if applicable
	_, err := r.collection.InsertOne(ctx, item)
	if err != nil {
		return fmt.Errorf("cannot create Tag: %w", err)
	}
	return nil
}

// Get retrieves a Tag by its ID.
func (r *MongoTagRepo) Get(ctx context.Context, id uuid.UUID) (*Tag, error) {
	var item Tag
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("Tag not found")
		}
		return nil, fmt.Errorf("cannot get Tag: %w", err)
	}
	return &item, nil
}

// Update updates an existing Tag in the database.
func (r *MongoTagRepo) Update(ctx context.Context, item *Tag) error {
	// TODO: Handle item.BeforeUpdate() if applicable
	filter := bson.M{"_id": item.ID()}
	update := bson.M{"$set": item} // This will replace the entire document with the item's current state
	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("cannot update Tag: %w", err)
	}
	return nil
}

// Delete deletes a Tag from the database by its ID.
func (r *MongoTagRepo) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id}
	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("cannot delete Tag: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("Tag not found")
	}
	return nil
}

// List retrieves all Tag records from the database.
func (r *MongoTagRepo) List(ctx context.Context) ([]*Tag, error) {
	var items []*Tag
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find())
	if err != nil {
		return nil, fmt.Errorf("cannot list Tags: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &items); err != nil {
		return nil, fmt.Errorf("cannot decode Tags: %w", err)
	}

	return items, nil
}
