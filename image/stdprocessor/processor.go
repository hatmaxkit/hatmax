package stdprocessor

import (
	"bytes"
	"context"
	"fmt"
	goimage "image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"

	"golang.org/x/image/draw"

	"github.com/hatmaxkit/hatmax/image"
)

// Processor implements image.Processor using Go's standard library and x/image.
type Processor struct {
	jpegQuality int
}

// New creates a new standard library processor.
func New() *Processor {
	return &Processor{
		jpegQuality: 85,
	}
}

// NewWithQuality creates a new processor with custom JPEG quality (1-100).
func NewWithQuality(jpegQuality int) *Processor {
	if jpegQuality < 1 {
		jpegQuality = 1
	}

	if jpegQuality > 100 {
		jpegQuality = 100
	}

	return &Processor{jpegQuality: jpegQuality}
}

// Resize resizes an image maintaining aspect ratio.
func (p *Processor) Resize(ctx context.Context, input io.Reader, contentType string, maxWidth, maxHeight int) (*image.ProcessedImage, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("cannot read input: %w", err)
	}

	src, format, err := goimage.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("cannot decode image: %w", err)
	}

	bounds := src.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	newWidth, newHeight := calculateDimensions(origWidth, origHeight, maxWidth, maxHeight)

	if newWidth >= origWidth && newHeight >= origHeight {
		return &image.ProcessedImage{
			Data:        bytes.NewReader(data),
			Width:       origWidth,
			Height:      origHeight,
			SizeBytes:   int64(len(data)),
			ContentType: contentType,
		}, nil
	}

	dst := goimage.NewRGBA(goimage.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var buf bytes.Buffer

	outputContentType := contentType

	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: p.jpegQuality})
		outputContentType = "image/jpeg"
	case "png":
		err = png.Encode(&buf, dst)
		outputContentType = "image/png"
	default:
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: p.jpegQuality})
		outputContentType = "image/jpeg"
	}

	if err != nil {
		return nil, fmt.Errorf("cannot encode resized image: %w", err)
	}

	return &image.ProcessedImage{
		Data:        bytes.NewReader(buf.Bytes()),
		Width:       newWidth,
		Height:      newHeight,
		SizeBytes:   int64(buf.Len()),
		ContentType: outputContentType,
	}, nil
}

// GetDimensions returns width and height of an image.
func (p *Processor) GetDimensions(ctx context.Context, input io.Reader, contentType string) (int, int, error) {
	cfg, _, err := goimage.DecodeConfig(input)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot decode image config: %w", err)
	}

	return cfg.Width, cfg.Height, nil
}

// DetectContentType returns the detected content type from image data.
func (p *Processor) DetectContentType(data []byte) string {
	return http.DetectContentType(data)
}

// calculateDimensions calculates new dimensions maintaining aspect ratio.
func calculateDimensions(origWidth, origHeight, maxWidth, maxHeight int) (int, int) {
	if origWidth <= maxWidth && origHeight <= maxHeight {
		return origWidth, origHeight
	}

	ratio := float64(origWidth) / float64(origHeight)

	newWidth := maxWidth
	newHeight := int(float64(newWidth) / ratio)

	if newHeight > maxHeight {
		newHeight = maxHeight
		newWidth = int(float64(newHeight) * ratio)
	}

	return newWidth, newHeight
}

var _ image.Processor = (*Processor)(nil)
