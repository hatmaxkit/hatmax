// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package stdprocessor

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	goimage "image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
)

type countedInput struct {
	reader io.Reader
	read   int
	cancel context.CancelFunc
}

func (r *countedInput) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)

	r.read += n
	if r.cancel != nil {
		r.cancel()
	}

	return n, err
}

// Canceled operations must stop reading and must not report a usable image.
func TestProcessorCanceled(t *testing.T) {
	for _, operation := range []string{"resize", "dimensions"} {
		for _, when := range []string{"before", "during read"} {
			t.Run(operation+"/"+when, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				input := &countedInput{reader: bytes.NewReader(testImage(t, "png", 8, 4))}

				if when == "before" {
					cancel()
				} else {
					input.cancel = cancel
				}

				var err error

				if operation == "resize" {
					_, err = New().Resize(ctx, input, "image/png", 4, 4)
				} else {
					_, _, err = New().GetDimensions(ctx, input, "image/png")
				}

				if !errors.Is(err, context.Canceled) {
					t.Fatalf("got %v; want context.Canceled", err)
				}

				if when == "before" && input.read != 0 {
					t.Fatalf("canceled operation read %d bytes", input.read)
				}
			})
		}
	}
}

// The encoded input budget must stop a finite over-limit reader before decode.
func TestResizeByteLimit(t *testing.T) {
	const limit = 20 << 20

	input := &countedInput{reader: io.LimitReader(zeroInput{}, limit+100)}

	_, err := New().Resize(context.Background(), input, "image/png", 4, 4)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("got %v; want encoded-size rejection", err)
	}

	if input.read != limit+1 {
		t.Fatalf("read %d bytes; want only %d", input.read, limit+1)
	}
}

// JPEG metadata can require long header scans. Limit that scan without
// allocating pixels or draining the rest of the finite generated input.
func TestConfigByteLimit(t *testing.T) {
	segment := make([]byte, 65537)
	copy(segment, []byte{0xff, 0xe1, 0xff, 0xff})
	input := &countedInput{reader: io.MultiReader(
		bytes.NewReader([]byte{0xff, 0xd8}),
		io.LimitReader(&repeatedSegment{data: segment}, (20<<20)+100),
	)}

	_, _, err := New().GetDimensions(context.Background(), input, "image/jpeg")
	if err == nil || !strings.Contains(err.Error(), "configuration exceeds") || input.read != (20<<20)+1 {
		t.Fatalf("header budget: %v; read %d bytes", err, input.read)
	}
}

type repeatedSegment struct {
	data   []byte
	offset int
}

func (r *repeatedSegment) Read(p []byte) (int, error) {
	n := copy(p, r.data[r.offset:])
	r.offset = (r.offset + n) % len(r.data)

	return n, nil
}

type zeroInput struct{}

func (zeroInput) Read(p []byte) (int, error) {
	clear(p)

	return len(p), nil
}

// Small PNG headers advertise large dimensions without allocating pixel data.
// Resize must reject them before Decode; dimensions uses the same pixel budget.
func TestProcessorPixels(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height uint32
		want          string
	}{
		{name: "over budget", width: 5000, height: 5001, want: "pixels"},
		{name: "overflow product", width: 1 << 30, height: 1 << 30, want: "dimension overflow"},
	} {
		for _, operation := range []string{"resize", "dimensions"} {
			t.Run(tc.name+"/"+operation, func(t *testing.T) {
				input := bytes.NewReader(testPNGHeader(tc.width, tc.height))

				var err error

				if operation == "resize" {
					_, err = New().Resize(context.Background(), input, "image/png", 4, 4)
				} else {
					_, _, err = New().GetDimensions(context.Background(), input, "image/png")
				}

				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("got %v; want %q rejection", err, tc.want)
				}
			})
		}
	}

	width, height, err := New().GetDimensions(context.Background(), bytes.NewReader(testPNGHeader(5000, 5000)), "image/png")
	if err != nil || width != 5000 || height != 5000 {
		t.Fatalf("pixel boundary: %dx%d, %v", width, height, err)
	}
}

// Ordinary resize keeps aspect ratio and output formats; no-resize returns the
// exact original bytes and caller content type, without changing quality APIs.
func TestProcessorResize(t *testing.T) {
	for _, tc := range []struct {
		name       string
		format     string
		width      int
		height     int
		maxWidth   int
		maxHeight  int
		wantWidth  int
		wantHeight int
	}{
		{name: "PNG landscape", format: "png", width: 8, height: 4, maxWidth: 4, maxHeight: 4, wantWidth: 4, wantHeight: 2},
		{name: "JPEG portrait", format: "jpeg", width: 4, height: 8, maxWidth: 4, maxHeight: 4, wantWidth: 2, wantHeight: 4},
		{name: "large caller bound", format: "png", width: 4, height: 8, maxWidth: int(^uint(0) >> 1), maxHeight: 4, wantWidth: 2, wantHeight: 4},
		{name: "unchanged", format: "png", width: 8, height: 4, maxWidth: 8, maxHeight: 8, wantWidth: 8, wantHeight: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := testImage(t, tc.format, tc.width, tc.height)

			result, err := New().Resize(context.Background(), bytes.NewReader(input), "image/"+tc.format, tc.maxWidth, tc.maxHeight)
			if err != nil {
				t.Fatal(err)
			}

			output, err := io.ReadAll(result.Data)
			if err != nil {
				t.Fatal(err)
			}

			cfg, format, err := goimage.DecodeConfig(bytes.NewReader(output))
			if err != nil || format != tc.format || cfg.Width != tc.wantWidth || cfg.Height != tc.wantHeight {
				t.Fatalf("encoded result: %+v, %s, %v", cfg, format, err)
			}

			if result.Width != cfg.Width || result.Height != cfg.Height || result.SizeBytes != int64(len(output)) || result.ContentType != "image/"+tc.format {
				t.Fatalf("incorrect result metadata: %+v", result)
			}

			if tc.name == "unchanged" && !bytes.Equal(input, output) {
				t.Fatal("no-resize changed encoded bytes")
			}
		})
	}

	for _, tc := range []struct{ quality, want int }{{-1, 1}, {50, 50}, {101, 100}} {
		if NewWithQuality(tc.quality).jpegQuality != tc.want {
			t.Fatalf("quality %d was not clamped to %d", tc.quality, tc.want)
		}
	}
}

// Invalid resize bounds must fail before reads or output allocation; malformed
// image data must remain a decode error instead of a successful passthrough.
func TestProcessorInvalid(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
	}{
		{name: "zero width", height: 4},
		{name: "negative height", width: 4, height: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := &countedInput{reader: strings.NewReader("unused")}

			_, err := New().Resize(context.Background(), input, "image/png", tc.width, tc.height)
			if err == nil || input.read != 0 {
				t.Fatalf("invalid bounds: %v; read %d bytes", err, input.read)
			}
		})
	}

	_, err := New().Resize(context.Background(), strings.NewReader("invalid"), "image/png", 4, 4)
	if err == nil || !strings.Contains(err.Error(), "cannot decode image config") {
		t.Fatalf("malformed image: %v", err)
	}
}

func testImage(t *testing.T, format string, width, height int) []byte {
	t.Helper()

	src := goimage.NewRGBA(goimage.Rect(0, 0, width, height))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})

	var (
		buf bytes.Buffer
		err error
	)

	if format == "png" {
		err = png.Encode(&buf, src)
	} else {
		err = jpeg.Encode(&buf, src, nil)
	}

	if err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func testPNGHeader(width, height uint32) []byte {
	buf := bytes.NewBufferString("\x89PNG\r\n\x1a\n")
	header := make([]byte, 17)
	copy(header, "IHDR")
	binary.BigEndian.PutUint32(header[4:8], width)
	binary.BigEndian.PutUint32(header[8:12], height)
	header[12], header[13] = 8, 6

	buf.Write([]byte{0, 0, 0, 13})
	buf.Write(header)

	var checksum [4]byte
	binary.BigEndian.PutUint32(checksum[:], crc32.ChecksumIEEE(header))
	buf.Write(checksum[:])
	buf.Write([]byte{0, 0, 0, 0, 'I', 'D', 'A', 'T'})

	return buf.Bytes()
}
