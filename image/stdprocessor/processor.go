// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

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

	"hatmax.adrianpk.com/image"
	"hatmax.adrianpk.com/image/internal/ingest"
)

// Limit full pixel decoding independently of the encoded input budget.
const maxDecodedPixels = 25_000_000

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

// Resize maintains aspect ratio with a 20 MiB encoded input limit and a
// 25-million-pixel decode limit. Cancellation is checked between CPU phases
// and input reads; codecs and scaling cannot be preempted during computation.
func (p *Processor) Resize(ctx context.Context, input io.Reader, contentType string, maxWidth, maxHeight int) (*image.ProcessedImage, error) {
	if maxWidth <= 0 || maxHeight <= 0 {
		return nil, fmt.Errorf("resize dimensions must be positive")
	}

	data, err := ingest.ReadAll(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("cannot read input: %w", err)
	}

	_, err = readConfig(ctx, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	src, format, err := goimage.Decode(ingest.NewReader(ctx, bytes.NewReader(data)))

	cause := ctx.Err()
	if cause != nil {
		return nil, cause
	}

	if err != nil {
		return nil, fmt.Errorf("cannot decode image: %w", err)
	}

	bounds := src.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Resize never enlarges input. Clamp requested bounds before ratio arithmetic
	// so very large caller limits cannot overflow output allocation dimensions.
	newWidth, newHeight := calculateDimensions(origWidth, origHeight, min(maxWidth, origWidth), min(maxHeight, origHeight))

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

	cause = ctx.Err()
	if cause != nil {
		return nil, cause
	}

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

	cause = ctx.Err()
	if cause != nil {
		return nil, cause
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

// GetDimensions reads image configuration without decoding pixels. Header reads
// are limited to 20 MiB, dimensions to 25 million pixels, and cancellation is
// checked between reads. It does not inspect or validate the remaining body.
func (p *Processor) GetDimensions(ctx context.Context, input io.Reader, contentType string) (int, int, error) {
	cfg, err := readConfig(ctx, input)
	if err != nil {
		return 0, 0, err
	}

	return cfg.Width, cfg.Height, nil
}

func readConfig(ctx context.Context, input io.Reader) (goimage.Config, error) {
	reader := &io.LimitedReader{R: ingest.NewReader(ctx, input), N: ingest.MaxBytes + 1}
	cfg, _, err := goimage.DecodeConfig(reader)

	cause := ctx.Err()
	if cause != nil {
		return goimage.Config{}, cause
	}

	if reader.N == 0 {
		return goimage.Config{}, fmt.Errorf("image configuration exceeds %d bytes", ingest.MaxBytes)
	}

	if err != nil {
		return goimage.Config{}, fmt.Errorf("cannot decode image config: %w", err)
	}

	// Division avoids overflow in a width*height check, including on 32-bit Go.
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxDecodedPixels/cfg.Height {
		return goimage.Config{}, fmt.Errorf("image dimensions exceed %d pixels", maxDecodedPixels)
	}

	return cfg, nil
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
