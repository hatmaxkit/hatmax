package image

import (
	"time"

	"github.com/google/uuid"
)

// VariantType represents the type of image variant.
type VariantType string

const (
	VariantOriginal  VariantType = "original"
	VariantLarge     VariantType = "large"     // ~1200px
	VariantMedium    VariantType = "medium"    // ~800px
	VariantThumbnail VariantType = "thumbnail" // ~300px
)

// ValidVariantTypes returns all valid variant types.
func ValidVariantTypes() []VariantType {
	return []VariantType{VariantOriginal, VariantLarge, VariantMedium, VariantThumbnail}
}

// Variant represents a derived version of an image.
type Variant struct {
	ID          uuid.UUID
	OriginalID  uuid.UUID
	Type        VariantType
	Width       int
	Height      int
	StoragePath string
	SizeBytes   int64
	CreatedAt   time.Time
}

// VariantSpec defines dimensions for a variant type.
type VariantSpec struct {
	Type      VariantType
	MaxWidth  int
	MaxHeight int
}

// DefaultVariantSpecs returns the default variant specifications.
func DefaultVariantSpecs() []VariantSpec {
	return []VariantSpec{
		{Type: VariantLarge, MaxWidth: 1200, MaxHeight: 1200},
		{Type: VariantMedium, MaxWidth: 800, MaxHeight: 800},
		{Type: VariantThumbnail, MaxWidth: 300, MaxHeight: 300},
	}
}
