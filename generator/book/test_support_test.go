package book

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func defaultBookFS(t *testing.T) fstest.MapFS {
	t.Helper()

	result := make(fstest.MapFS)

	err := fs.WalkDir(defaultSource, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		data, readErr := fs.ReadFile(defaultSource, path)
		if readErr != nil {
			return readErr
		}

		result[path] = &fstest.MapFile{Data: data}

		return nil
	})
	if err != nil {
		t.Fatalf("copy embedded Book: %v", err)
	}

	return result
}

func replaceBookText(t *testing.T, source fstest.MapFS, path, oldValue, newValue string) {
	t.Helper()

	entry, exists := source[path]
	if !exists {
		t.Fatalf("Book fixture %q does not exist", path)
	}

	original := string(entry.Data)
	if strings.Count(original, oldValue) != 1 {
		t.Fatalf("Book fixture %q has %d occurrences of %q, want 1", path, strings.Count(original, oldValue), oldValue)
	}

	entry.Data = []byte(strings.Replace(original, oldValue, newValue, 1))
}

func requireValidationCode(t *testing.T, err error, code string) {
	t.Helper()

	if err == nil {
		t.Fatalf("Load() error = nil, want validation code %q", code)
	}

	var validationErr ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Load() error = %v, want ValidationError", err)
	}

	if validationErr.Code != code {
		t.Fatalf("Load() validation code = %q, want %q", validationErr.Code, code)
	}
}
