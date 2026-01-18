package image

import (
	"context"
	"io"
)

// Store defines image storage operations.
type Store interface {
	// Put stores an image and returns the storage path.
	Put(ctx context.Context, path string, data io.Reader) error

	// Get returns a reader for a stored image.
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes an image from storage.
	Delete(ctx context.Context, path string) error

	// URL returns a servable URL for the image.
	URL(path string) string
}
