package todo

import (
	"context"

	"github.com/google/uuid"
)

// ItemService defines the interface for Item service operations.
type ItemService interface {
	Create(ctx context.Context, item *Item) error
	Get(ctx context.Context, id uuid.UUID) (*Item, error)
	Update(ctx context.Context, item *Item) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*Item, error)
}
