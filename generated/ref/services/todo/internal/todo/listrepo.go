package todo

import (
	"context"

	"github.com/google/uuid"
)

// ListRepo defines the interface for List aggregate operations.
// This repository manages the aggregate root and all its child entities as a single unit.
type ListRepo interface {
	// Create creates a new List aggregate with all its child entities.
	Create(ctx context.Context, aggregate *List) error

	// Get retrieves a complete List aggregate by ID, including all child entities.
	Get(ctx context.Context, id uuid.UUID) (*List, error)

	// Save performs a unit-of-work save operation on the aggregate.
	// This will compute differences and update/insert/delete child entities as needed.
	Save(ctx context.Context, aggregate *List) error

	// Delete removes the entire List aggregate and all its child entities.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all List aggregates with their child entities.
	List(ctx context.Context) ([]*List, error)
}