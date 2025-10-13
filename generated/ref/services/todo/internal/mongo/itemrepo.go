package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/google/uuid"
)

// MongoItemRepo implements the ItemRepo interface for MongoDB.
type MongoItemRepo struct {
	collection *mongo.Collection
}

// NewMongoItemRepo creates a new MongoItemRepo.
func NewMongoItemRepo(db *mongo.Database) *MongoItemRepo {
	return &MongoItemRepo{
		collection: db.Collection("items"),
	}
}

// Create inserts a new Item into the database.
func (r *MongoItemRepo) Create(ctx context.Context, item *Item) error {
	// TODO: Handle item.BeforeCreate() if applicable
	_, err := r.collection.InsertOne(ctx, item)
	if err != nil {
		return fmt.Errorf("cannot create Item: %w", err)
	}
	return nil
}

// Get retrieves a Item by its ID.
func (r *MongoItemRepo) Get(ctx context.Context, id uuid.UUID) (*Item, error) {
	var item Item
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("Item not found")
		}
		return nil, fmt.Errorf("cannot get Item: %w", err)
	}
	return &item, nil
}

// Update updates an existing Item in the database.
func (r *MongoItemRepo) Update(ctx context.Context, item *Item) error {
	// TODO: Handle item.BeforeUpdate() if applicable
	filter := bson.M{"_id": item.ID()}
	update := bson.M{"$set": item} // This will replace the entire document with the item's current state
	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("cannot update Item: %w", err)
	}
	return nil
}

// Delete deletes a Item from the database by its ID.
func (r *MongoItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	filter := bson.M{"_id": id}
	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("cannot delete Item: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("Item not found")
	}
	return nil
}

// List retrieves all Item records from the database.
func (r *MongoItemRepo) List(ctx context.Context) ([]*Item, error) {
	var items []*Item
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find())
	if err != nil {
		return nil, fmt.Errorf("cannot list Items: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &items); err != nil {
		return nil, fmt.Errorf("cannot decode Items: %w", err)
	}

	return items, nil
}
