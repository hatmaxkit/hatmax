// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package local

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"hatmax.adrianpk.com/image"
)

// Store implements image.Store using local filesystem.
type Store struct {
	basePath string
	baseURL  string
}

// NewStore creates a new local filesystem store.
// basePath is the root directory for file storage.
// The configured root and its ancestors must be controlled by the application.
// baseURL is the URL prefix for serving files (e.g., "/uploads").
func NewStore(basePath, baseURL string) *Store {
	return &Store{basePath: basePath, baseURL: baseURL}
}

// Put stores data at a local, non-root path confined to the configured directory.
func (s *Store) Put(ctx context.Context, path string, data io.Reader) error {
	root, err := s.openRoot(path, true)
	if err != nil {
		return err
	}
	defer root.Close()

	path = filepath.Clean(path)

	err = root.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	f, err := root.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, data)

	return err
}

// Get returns a reader for the file at the given path.
func (s *Store) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	root, err := s.openRoot(path, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	return root.Open(filepath.Clean(path))
}

// Delete removes the file at the given path.
func (s *Store) Delete(ctx context.Context, path string) error {
	root, err := s.openRoot(path, false)
	if err != nil {
		return err
	}
	defer root.Close()

	return root.Remove(filepath.Clean(path))
}

func (s *Store) openRoot(path string, create bool) (*os.Root, error) {
	if !filepath.IsLocal(path) || filepath.Clean(path) == "." {
		return nil, &os.PathError{Op: "image", Path: path, Err: fs.ErrInvalid}
	}

	basePath := filepath.Clean(s.basePath)

	if create {
		err := os.MkdirAll(basePath, 0755)
		if err != nil {
			return nil, err
		}
	}

	// A root handle confines traversal during the operation, including symlink
	// replacement races. Checking a resolved path before opening it is not enough.
	return os.OpenRoot(basePath)
}

// URL returns a servable URL for the image.
func (s *Store) URL(path string) string {
	return s.baseURL + "/" + path
}

// BasePath returns the base filesystem path.
func (s *Store) BasePath() string {
	return s.basePath
}

var _ image.Store = (*Store)(nil)
