package local

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	store := NewStore("/tmp/images", "/uploads")

	if store.basePath != "/tmp/images" {
		t.Errorf("basePath = %q, want %q", store.basePath, "/tmp/images")
	}
	if store.baseURL != "/uploads" {
		t.Errorf("baseURL = %q, want %q", store.baseURL, "/uploads")
	}
}

func TestStore_URL(t *testing.T) {
	store := NewStore("/tmp/images", "/uploads")
	url := store.URL("2024/01/image.jpg")

	if url != "/uploads/2024/01/image.jpg" {
		t.Errorf("URL() = %q, want %q", url, "/uploads/2024/01/image.jpg")
	}
}

func TestStore_BasePath(t *testing.T) {
	store := NewStore("/tmp/images", "/uploads")
	if store.BasePath() != "/tmp/images" {
		t.Errorf("BasePath() = %q, want %q", store.BasePath(), "/tmp/images")
	}
}

func TestStore_PutGetDelete(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "hatmax-image-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewStore(tmpDir, "/uploads")
	ctx := context.Background()
	path := "test/image.txt"
	content := []byte("test content")

	// Test Put
	err = store.Put(ctx, path, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify file exists
	fullPath := filepath.Join(tmpDir, path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("file was not created")
	}

	// Test Get
	reader, err := store.Get(ctx, path)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(reader)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("content mismatch: got %q, want %q", buf.String(), string(content))
	}

	// Test Delete
	err = store.Delete(ctx, path)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("file was not deleted")
	}
}
