package image

import (
	"context"
	"io"
)

// ProcessedImage contains the result of image processing.
type ProcessedImage struct {
	Data        io.Reader
	Width       int
	Height      int
	SizeBytes   int64
	ContentType string
}

// Processor generates image variants.
type Processor interface {
	// Resize resizes an image maintaining aspect ratio.
	// Returns the resized image data, new dimensions, and size.
	Resize(ctx context.Context, input io.Reader, contentType string, maxWidth, maxHeight int) (*ProcessedImage, error)

	// GetDimensions returns width and height of an image.
	GetDimensions(ctx context.Context, input io.Reader, contentType string) (width, height int, err error)

	// DetectContentType returns the detected content type from image data.
	DetectContentType(data []byte) string
}
