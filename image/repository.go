package image

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines image persistence operations.
type Repository interface {
	Create(ctx context.Context, img *Image) error
	Get(ctx context.Context, id uuid.UUID) (*Image, error)
	Delete(ctx context.Context, id uuid.UUID) error

	CreateVariant(ctx context.Context, v *Variant) error
	GetVariants(ctx context.Context, originalID uuid.UUID) ([]*Variant, error)
	GetVariant(ctx context.Context, originalID uuid.UUID, variantType VariantType) (*Variant, error)
	DeleteVariants(ctx context.Context, originalID uuid.UUID) error
}
