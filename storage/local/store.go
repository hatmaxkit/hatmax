// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Package local stores bounded private objects beneath an application-owned root.
package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"

	"hatmax.adrianpk.com/storage"
	"hatmax.adrianpk.com/storage/internal/stream"
)

// Options bounds each concurrent write's private disk use. A write's MaxBytes
// must not exceed MaxSpoolBytes. N concurrent writes use at most N times this
// capacity plus filesystem overhead, and a fixed-size copy buffer per write.
type Options struct {
	MaxSpoolBytes int64
}

// Store owns a directory handle. Call Close when done. The caller must control
// the root, its initial ancestors and its contents, including hard links and
// mounts. Directory handles confine later traversal under symlink replacement.
// Objects and spool files use mode 0600; created directories use mode 0700.
type Store struct {
	root     *os.Root
	capacity int64
}

// NewStore opens an existing directory and rejects invalid configuration.
// It starts no workers and does not create the root. Local content types are
// derived from at most 512 object bytes using http.DetectContentType, including
// after reopening the store. WriteOptions.ContentType is validated but is not
// persisted; returned metadata always describes this local derivation.
func NewStore(directory string, options Options) (*Store, error) {
	err := (storage.ReadOptions{MaxBytes: options.MaxSpoolBytes}).Validate()
	if err != nil || directory == "" {
		return nil, fmt.Errorf("local configuration: %w", storage.ErrInvalid)
	}

	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open local root: %w", err)
	}

	return &Store{root: root, capacity: options.MaxSpoolBytes}, nil
}

// Close releases the store's directory handle. Previously opened readers remain
// owned by the caller and can still be read and closed independently.
func (s *Store) Close() error {
	return s.root.Close()
}

// Put validates and spools complete bounded input before atomic replacement.
// Input ownership stays with the caller. Cancellation is checked between reads
// and phases; an arbitrary already-blocked input Read cannot be interrupted.
// The replacement is atomic, not compare-and-swap or crash-durable publication.
func (s *Store) Put(ctx context.Context, key string, input io.Reader, options storage.WriteOptions) (storage.Object, error) {
	err := storage.ValidateKey(key)
	if err != nil {
		return storage.Object{}, err
	}

	err = options.Validate()
	if err != nil {
		return storage.Object{}, err
	}

	if input == nil || options.MaxBytes > s.capacity {
		return storage.Object{}, fmt.Errorf("local write configuration: %w", storage.ErrInvalid)
	}

	err = ctx.Err()
	if err != nil {
		return storage.Object{}, err
	}

	parent, err := s.writeParent(key)
	if err != nil {
		return storage.Object{}, err
	}
	defer parent.Close()

	return put(ctx, parent, key, input, options.MaxBytes)
}

func (s *Store) writeParent(key string) (*os.Root, error) {
	directory := path.Dir(key)

	err := s.root.MkdirAll(directory, 0700)
	if err != nil {
		return nil, fmt.Errorf("create object parent: %w", err)
	}

	// Pin one parent for creation, rename and cleanup, even if its path changes.
	return s.root.OpenRoot(directory)
}

// Open returns an owned reader and metadata from the same opened file. It checks
// observed size before returning and actual streamed bytes independently.
func (s *Store) Open(ctx context.Context, key string, options storage.ReadOptions) (io.ReadCloser, storage.Object, error) {
	err := storage.ValidateKey(key)
	if err != nil {
		return nil, storage.Object{}, err
	}

	err = options.Validate()
	if err != nil {
		return nil, storage.Object{}, err
	}

	file, object, err := s.open(ctx, key, options.MaxBytes)
	if err != nil {
		return nil, storage.Object{}, err
	}

	return &objectReader{Reader: stream.NewReader(ctx, file, options.MaxBytes), file: file}, object, nil
}

// Stat derives metadata from one opened file without reserving a future read.
func (s *Store) Stat(ctx context.Context, key string) (storage.Object, error) {
	err := storage.ValidateKey(key)
	if err != nil {
		return storage.Object{}, err
	}

	file, object, err := s.open(ctx, key, math.MaxInt64)
	if err != nil {
		return storage.Object{}, err
	}

	err = file.Close()
	if err != nil {
		return storage.Object{}, err
	}

	return object, nil
}

func (s *Store) open(ctx context.Context, key string, maximum int64) (*os.File, storage.Object, error) {
	err := ctx.Err()
	if err != nil {
		return nil, storage.Object{}, err
	}

	file, err := s.root.Open(key)
	if err != nil {
		return nil, storage.Object{}, classifyRead(err)
	}

	object, err := describe(ctx, file, key, maximum)
	if err != nil {
		err = errors.Join(err, file.Close())

		return nil, storage.Object{}, err
	}

	return file, object, nil
}

// Delete unlinks the key without following a final symbolic link. Confirmed
// absence succeeds; a failed mutation is conservatively indeterminate, with its
// filesystem cause preserved. Directories are not objects.
func (s *Store) Delete(ctx context.Context, key string) error {
	err := storage.ValidateKey(key)
	if err != nil {
		return err
	}

	err = ctx.Err()
	if err != nil {
		return err
	}

	info, err := s.root.Lstat(key)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("directory is not an object: %w", storage.ErrInvalid)
	}

	err = ctx.Err()
	if err != nil {
		return err
	}

	err = s.root.Remove(key)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return errors.Join(storage.ErrIndeterminate, err)
}

func classifyRead(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return errors.Join(storage.ErrNotFound, err)
	}

	return err
}

type objectReader struct {
	*stream.Reader
	file *os.File
}

func (r *objectReader) Close() error {
	return r.file.Close()
}

var _ storage.Store = (*Store)(nil)
