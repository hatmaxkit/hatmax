package local

import (
	"context"
	"io"
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
// baseURL is the URL prefix for serving files (e.g., "/uploads").
func NewStore(basePath, baseURL string) *Store {
	return &Store{basePath: basePath, baseURL: baseURL}
}

// Put stores data at the given path.
func (s *Store) Put(ctx context.Context, path string, data io.Reader) error {
	fullPath := filepath.Join(s.basePath, path)

	err := os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err != nil {
		return err
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, data)

	return err
}

// Get returns a reader for the file at the given path.
func (s *Store) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.basePath, path))
}

// Delete removes the file at the given path.
func (s *Store) Delete(ctx context.Context, path string) error {
	return os.Remove(filepath.Join(s.basePath, path))
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
