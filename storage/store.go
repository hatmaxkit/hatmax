// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Package storage defines bounded private object I/O without image or URL semantics.
// Callers own authorization, quotas, retries and durable completion.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"strings"
)

var (
	ErrInvalid       = errors.New("invalid storage input")
	ErrNotFound      = errors.New("object not found")
	ErrTooLarge      = errors.New("object exceeds byte limit")
	ErrUnsupported   = errors.New("storage capability unavailable")
	ErrIndeterminate = errors.New("storage mutation outcome unknown")
)

// Object describes observed bytes. ContentType is metadata, not format validation.
// Stat is an observation; it does not reserve a later read of the same bytes.
type Object struct {
	Key         string
	SizeBytes   int64
	ContentType string
}

// WriteOptions requires a positive MaxBytes below math.MaxInt64. ContentType is
// optional MIME metadata of at most 256 bytes; adapters document its provenance.
type WriteOptions struct {
	ContentType string
	MaxBytes    int64
}

// Validate checks options before any input is read or storage is accessed.
func (o WriteOptions) Validate() error {
	err := (ReadOptions{MaxBytes: o.MaxBytes}).Validate()
	if err != nil {
		return err
	}

	if o.ContentType == "" {
		return nil
	}

	if len(o.ContentType) > 256 || strings.ContainsAny(o.ContentType, "\r\n\x00") {
		return fmt.Errorf("content type: %w", ErrInvalid)
	}

	mediaType, _, err := mime.ParseMediaType(o.ContentType)
	if err != nil || !strings.Contains(mediaType, "/") {
		return fmt.Errorf("content type: %w", ErrInvalid)
	}

	return nil
}

// ReadOptions bounds actual streamed bytes as well as observed object size.
type ReadOptions struct {
	MaxBytes int64
}

// Validate rejects bounds that cannot safely accommodate an overflow probe.
func (o ReadOptions) Validate() error {
	if o.MaxBytes <= 0 || o.MaxBytes == math.MaxInt64 {
		return fmt.Errorf("maximum bytes: %w", ErrInvalid)
	}

	return nil
}

// Store operates on canonical keys. Put never closes the caller's reader. Open
// transfers ownership of its returned reader to the caller, who must close it.
// Only successful EOF establishes a complete bounded read. Delete is idempotent
// for confirmed absence. Mutations are not retried automatically.
type Store interface {
	Put(context.Context, string, io.Reader, WriteOptions) (Object, error)
	Open(context.Context, string, ReadOptions) (io.ReadCloser, Object, error)
	Stat(context.Context, string) (Object, error)
	Delete(context.Context, string) error
}

// ValidateKey admits 1-512 ASCII bytes in slash-separated, non-empty segments.
// Segments contain letters, digits, dots, hyphens or underscores, but cannot be
// "." or "..". Rejected keys are never normalized into another target.
func ValidateKey(key string) error {
	if len(key) == 0 || len(key) > 512 {
		return fmt.Errorf("key length: %w", ErrInvalid)
	}

	for _, segment := range strings.Split(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("key segment: %w", ErrInvalid)
		}

		for i := range len(segment) {
			c := segment[i]
			if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == '_' {
				continue
			}

			return fmt.Errorf("key character: %w", ErrInvalid)
		}
	}

	return nil
}
