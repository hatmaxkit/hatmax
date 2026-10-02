// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package local

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func runStoreOperation(store *Store, operation, key string) error {
	ctx := context.Background()

	switch operation {
	case "put":
		return store.Put(ctx, key, strings.NewReader("inside"))
	case "get":
		reader, err := store.Get(ctx, key)
		if err != nil {
			return err
		}

		return reader.Close()
	default:
		return store.Delete(ctx, key)
	}
}

// TestStoreInvalidKeys rejects invalid paths before creating or removing the root.
func TestStoreInvalidKeys(t *testing.T) {
	for _, operation := range []string{"put", "get", "delete"} {
		for _, key := range []string{"", ".", "..", "../outside", "a/../../outside", "a/..", "/absolute"} {
			t.Run(operation+"/"+key, func(t *testing.T) {
				root := filepath.Join(t.TempDir(), "missing")

				err := runStoreOperation(NewStore(root, "/uploads"), operation, key)
				if !errors.Is(err, fs.ErrInvalid) {
					t.Fatalf("invalid key: got %v, want fs.ErrInvalid", err)
				}

				_, err = os.Stat(root)
				if !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("invalid key created the root: %v", err)
				}
			})
		}
	}
}

// TestStoreNestedKeys preserves normalized nested keys and lazy root creation.
func TestStoreNestedKeys(t *testing.T) {
	for _, key := range []string{"nested/image", "nested/../image", "nested//photo/image"} {
		t.Run(key, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "missing", "root")
			store := NewStore(root, "/uploads")
			ctx := context.Background()

			err := store.Put(ctx, key, strings.NewReader("inside"))
			if err != nil {
				t.Fatal(err)
			}

			reader, err := store.Get(ctx, key)
			if err != nil {
				t.Fatal(err)
			}

			// Get closes its root handle before returning; the file must stay readable.
			data, readErr := io.ReadAll(reader)
			closeErr := reader.Close()

			if readErr != nil || closeErr != nil || string(data) != "inside" {
				t.Fatalf("read = %q, read error = %v, close error = %v", data, readErr, closeErr)
			}

			err = store.Delete(ctx, key)
			if err != nil {
				t.Fatal(err)
			}

			_, err = os.Stat(filepath.Join(root, key))
			if !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("object still exists: %v", err)
			}
		})
	}
}

// TestStoreSymlinks allows contained relative links and only unlinks a final external link.
func TestStoreSymlinks(t *testing.T) {
	for _, operation := range []string{"put", "get", "delete"} {
		for _, link := range []string{"inside file", "outside file", "inside directory", "absolute inside"} {
			t.Run(operation+"/"+link, func(t *testing.T) {
				parent := t.TempDir()
				root := filepath.Join(parent, "root")
				inside := filepath.Join(root, "inside")
				outside := filepath.Join(parent, "outside")

				err := os.MkdirAll(inside, 0755)
				if err != nil {
					t.Fatal(err)
				}

				writeTestFile(t, filepath.Join(inside, "image"), "inside")
				writeTestFile(t, outside, "outside")

				target := "inside/image"
				key := "link"

				switch link {
				case "outside file":
					target = "../outside"
				case "inside directory":
					target = "inside"
					key = "link/image"
				case "absolute inside":
					target = filepath.Join(inside, "image")
				}

				err = os.Symlink(target, filepath.Join(root, "link"))
				if err != nil {
					t.Fatal(err)
				}

				err = runStoreOperation(NewStore(root, "/uploads"), operation, key)
				blocked := operation != "delete" && (link == "outside file" || link == "absolute inside")

				if (err != nil) != blocked {
					t.Fatalf("operation error = %v, want blocked = %v", err, blocked)
				}

				if operation == "delete" {
					_, err = os.Lstat(filepath.Join(root, key))
					if !errors.Is(err, fs.ErrNotExist) {
						t.Fatalf("link or object not deleted: %v", err)
					}
				}

				data, err := os.ReadFile(outside)
				if err != nil || string(data) != "outside" {
					t.Fatalf("outside target changed: %q, %v", data, err)
				}
			})
		}
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

// TestStoreRootErrors preserves missing-root behavior and reports filesystem failures.
func TestStoreRootErrors(t *testing.T) {
	for _, operation := range []string{"put", "get", "delete"} {
		t.Run(operation, func(t *testing.T) {
			parent := t.TempDir()
			baseFile := filepath.Join(parent, "file")
			writeTestFile(t, baseFile, "not a directory")

			err := runStoreOperation(NewStore(baseFile, "/uploads"), operation, "image")
			if err == nil {
				t.Fatal("accepted a file as storage root")
			}

			if operation != "put" {
				missing := filepath.Join(parent, "missing")

				err = runStoreOperation(NewStore(missing, "/uploads"), operation, "image")
				if !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("missing root: %v", err)
				}

				_, err = os.Stat(missing)
				if !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("read or delete created root: %v", err)
				}
			}
		})
	}

	t.Run("blocked parent", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, filepath.Join(root, "file"), "inside")

		err := runStoreOperation(NewStore(root, "/uploads"), "put", "file/image")
		if err == nil {
			t.Fatal("created object through a file parent")
		}
	})

	t.Run("directory object", func(t *testing.T) {
		root := t.TempDir()

		err := os.Mkdir(filepath.Join(root, "directory"), 0755)
		if err != nil {
			t.Fatal(err)
		}

		err = runStoreOperation(NewStore(root, "/uploads"), "put", "directory")
		if err == nil {
			t.Fatal("created file over a directory")
		}
	})
}

// TestStoreLinkSwap keeps reads, writes, and deletes confined during atomic link replacement.
func TestStoreLinkSwap(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	inside := filepath.Join(root, "inside")
	outside := filepath.Join(parent, "outside")

	for _, dir := range []string{inside, outside} {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatal(err)
		}
	}

	writeTestFile(t, filepath.Join(inside, "sentinel"), "inside")
	writeTestFile(t, filepath.Join(outside, "sentinel"), "outside")
	slot := filepath.Join(root, "slot")

	err := os.Symlink("inside", slot)
	if err != nil {
		t.Fatal(err)
	}

	stop := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		for i := 0; i < 1000; i++ {
			select {
			case <-stop:
				done <- nil

				return
			default:
			}

			target := "inside"

			if i%2 == 0 {
				target = "../outside"
			}

			next := filepath.Join(root, "next-link")

			err := os.Symlink(target, next)
			if err != nil {
				done <- err

				return
			}

			err = os.Rename(next, slot)
			if err != nil {
				done <- err

				return
			}

			runtime.Gosched()
		}

		done <- nil
	}()

	defer func() {
		close(stop)

		err := <-done
		if err != nil {
			t.Error("link replacement failed:", err)
		}
	}()

	store := NewStore(root, "/uploads")
	ctx := context.Background()

	for i := 0; i < 300; i++ {
		// Operations may succeed on the inside link or fail on the outside link.
		_ = store.Put(ctx, "slot/nested/image", strings.NewReader("inside"))

		reader, err := store.Get(ctx, "slot/sentinel")
		if err == nil {
			data, readErr := io.ReadAll(reader)
			closeErr := reader.Close()

			if readErr != nil || closeErr != nil || string(data) != "inside" {
				t.Fatalf("read escaped the root: %q, %v, %v", data, readErr, closeErr)
			}
		}

		_ = store.Delete(ctx, "slot/sentinel")
	}

	data, err := os.ReadFile(filepath.Join(outside, "sentinel"))
	if err != nil || string(data) != "outside" {
		t.Fatalf("outside sentinel changed: %q, %v", data, err)
	}

	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 1 {
		t.Fatalf("outside directory changed: %v, %v", entries, err)
	}
}

// FuzzStoreKeys checks that arbitrary object paths never modify an outside sentinel.
func FuzzStoreKeys(f *testing.F) {
	for _, key := range []string{"", ".", "../outside/sentinel", "/absolute", "link/sentinel", "link/new/image", "nested/image", "a/../image", "image\x00"} {
		f.Add(key)
	}

	f.Fuzz(func(t *testing.T, key string) {
		parent := t.TempDir()
		root := filepath.Join(parent, "root")
		outside := filepath.Join(parent, "outside")

		for _, dir := range []string{root, outside} {
			err := os.Mkdir(dir, 0755)
			if err != nil {
				t.Fatal(err)
			}
		}

		writeTestFile(t, filepath.Join(outside, "sentinel"), "outside")

		err := os.Symlink("../outside", filepath.Join(root, "link"))
		if err != nil {
			t.Fatal(err)
		}

		store := NewStore(root, "/uploads")
		ctx := context.Background()
		_ = store.Put(ctx, key, strings.NewReader("inside"))

		reader, err := store.Get(ctx, key)
		if err == nil {
			data, readErr := io.ReadAll(reader)
			closeErr := reader.Close()

			if closeErr != nil || (readErr == nil && string(data) == "outside") {
				t.Fatalf("read escaped root or did not close: %q, %v", data, closeErr)
			}
		}

		_ = store.Delete(ctx, key)

		data, err := os.ReadFile(filepath.Join(outside, "sentinel"))
		if err != nil || string(data) != "outside" {
			t.Fatalf("outside sentinel changed: %q, %v", data, err)
		}

		entries, err := os.ReadDir(outside)
		if err != nil || len(entries) != 1 {
			t.Fatalf("outside directory changed: %v, %v", entries, err)
		}
	})
}

// TestStoreEscape rejects traversal and directory symlinks without touching outside files.
func TestStoreEscape(t *testing.T) {
	for _, operation := range []string{"put", "get", "delete"} {
		for _, attack := range []string{"traversal", "directory symlink"} {
			t.Run(operation+"/"+attack, func(t *testing.T) {
				parent := t.TempDir()
				root := filepath.Join(parent, "root")
				outside := filepath.Join(parent, "outside")
				sentinel := filepath.Join(outside, "sentinel")

				err := os.MkdirAll(outside, 0755)
				if err != nil {
					t.Fatal(err)
				}

				err = os.Mkdir(root, 0755)
				if err != nil {
					t.Fatal(err)
				}

				err = os.WriteFile(sentinel, []byte("outside"), 0644)
				if err != nil {
					t.Fatal(err)
				}

				key := "../outside/sentinel"

				if attack == "directory symlink" {
					err = os.Symlink("../outside", filepath.Join(root, "link"))
					if err != nil {
						t.Fatal(err)
					}

					key = "link/sentinel"
				}

				err = runStoreOperation(NewStore(root, "/uploads"), operation, key)
				if err == nil {
					t.Error("outside-root operation succeeded")
				}

				data, err := os.ReadFile(sentinel)
				if err != nil {
					t.Fatal("outside sentinel removed:", err)
				}

				if string(data) != "outside" {
					t.Fatalf("outside sentinel changed: %q", data)
				}
			})
		}
	}
}
