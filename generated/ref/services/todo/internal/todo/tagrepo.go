package todo

import (
	"context"

	"github.com/google/uuid"
)

// TagRepo defines the interface for Tag data operations.
type TagRepo interface {
	Create(ctx context.Context, item *Tag) error
	Get(ctx context.Context, id uuid.UUID) (*Tag, error)
	Update(ctx context.Context, item *Tag) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*Tag, error)
}