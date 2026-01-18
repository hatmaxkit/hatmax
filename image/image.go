package image

import (
	"time"

	"github.com/google/uuid"
)

// Image represents a stored image without domain knowledge.
type Image struct {
	ID          uuid.UUID
	Filename    string         // original filename
	ContentType string         // image/jpeg, image/png, image/webp
	SizeBytes   int64
	Width       int
	Height      int
	StoragePath string         // relative path in storage
	Metadata    map[string]any // EXIF, etc.
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
