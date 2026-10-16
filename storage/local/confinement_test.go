// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package local

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"hatmax.adrianpk.com/storage"
)

func writeFile(t *testing.T, name, content string) {
	t.Helper()

	err := os.WriteFile(name, []byte(content), 0600)
	if err != nil {
		t.Fatal(err)
	}
}

// TestConfinement uses actual paths and atomic link swaps to protect outside bytes.
func TestConfinement(t *testing.T) {
	ctx := context.Background()

	t.Run("keys before I/O", func(t *testing.T) {
		directory := t.TempDir()
		store := newTestStore(t, directory)

		for _, key := range []string{"", "..", "../outside", "a/../outside", "/absolute", "a//b", `a\b`, "a/%2e/b", "a/", "a/./b"} {
			input := &ownedInput{Reader: strings.NewReader("private")}

			_, writeErr := store.Put(ctx, key, input, storage.WriteOptions{MaxBytes: 8})
			reader, object, readErr := store.Open(ctx, key, storage.ReadOptions{MaxBytes: 8})
			_, statErr := store.Stat(ctx, key)
			deleteErr := store.Delete(ctx, key)

			for _, err := range []error{writeErr, readErr, statErr, deleteErr} {
				if !errors.Is(err, storage.ErrInvalid) {
					t.Fatalf("key=%q, error=%v", key, err)
				}
			}

			if input.reads != 0 || input.closes != 0 || reader != nil || object != (storage.Object{}) {
				t.Fatalf("invalid key touched input or returned a reader: %q", key)
			}
		}

		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 0 {
			t.Fatalf("invalid keys changed root: %v/%v", entries, err)
		}
	})

	t.Run("external links", func(t *testing.T) {
		parent := t.TempDir()
		directory := filepath.Join(parent, "root")
		outside := filepath.Join(parent, "outside")

		for _, name := range []string{directory, outside} {
			err := os.Mkdir(name, 0700)
			if err != nil {
				t.Fatal(err)
			}
		}

		writeFile(t, filepath.Join(outside, "sentinel"), "outside")
		store := newTestStore(t, directory)

		err := os.Symlink("../outside", filepath.Join(directory, "link"))
		if err != nil {
			t.Fatal(err)
		}

		_, writeErr := store.Put(ctx, "link/sentinel", strings.NewReader("private"), storage.WriteOptions{MaxBytes: 8})
		reader, _, readErr := store.Open(ctx, "link/sentinel", storage.ReadOptions{MaxBytes: 8})
		_, statErr := store.Stat(ctx, "link/sentinel")
		deleteErr := store.Delete(ctx, "link/sentinel")

		for _, err := range []error{writeErr, readErr, statErr, deleteErr} {
			if err == nil || errors.Is(err, storage.ErrNotFound) {
				t.Fatalf("escaped or hid confinement failure: %v", err)
			}
		}

		if reader != nil {
			t.Fatal("external reader returned")
		}

		// A final link can be removed without deleting its target.
		err = store.Delete(ctx, "link")
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(filepath.Join(outside, "sentinel"))
		if err != nil || string(data) != "outside" {
			t.Fatalf("external sentinel=%q/%v", data, err)
		}

		assertNoSpools(t, directory)
	})

	t.Run("access failure", func(t *testing.T) {
		directory := t.TempDir()
		store := newTestStore(t, directory)
		name := filepath.Join(directory, "private")
		writeFile(t, name, "private")

		err := os.Chmod(name, 0000)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(name, 0600)

		_, err = store.Stat(ctx, "private")
		if !errors.Is(err, os.ErrPermission) || errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("permission failure misclassified: %v", err)
		}
	})

	t.Run("link replacement", testLinkReplacement)
}

func testLinkReplacement(t *testing.T) {
	parent := t.TempDir()
	directory := filepath.Join(parent, "root")
	inside := filepath.Join(directory, "inside")
	outside := filepath.Join(parent, "outside")

	for _, name := range []string{inside, outside} {
		err := os.MkdirAll(name, 0700)
		if err != nil {
			t.Fatal(err)
		}
	}

	writeFile(t, filepath.Join(inside, "sentinel"), "inside")
	writeFile(t, filepath.Join(outside, "sentinel"), "outside")
	link := filepath.Join(directory, "slot")

	err := os.Symlink("inside", link)
	if err != nil {
		t.Fatal(err)
	}

	stop := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		for index := range 1000 {
			select {
			case <-stop:
				done <- nil

				return
			default:
			}

			target := "inside"
			if index%2 == 0 {
				target = "../outside"
			}

			next := filepath.Join(directory, "next-link")

			err := os.Symlink(target, next)
			if err == nil {
				err = os.Rename(next, link)
			}

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
			t.Error("link replacement:", err)
		}

		assertNoSpools(t, directory)
	}()

	store := newTestStore(t, directory)
	ctx := context.Background()

	for range 200 {
		_, _ = store.Put(ctx, "slot/nested/object", strings.NewReader("private"), storage.WriteOptions{MaxBytes: 8})

		reader, _, err := store.Open(ctx, "slot/sentinel", storage.ReadOptions{MaxBytes: 8})
		if err == nil {
			data, readErr := io.ReadAll(reader)

			closeErr := reader.Close()
			if readErr != nil || closeErr != nil || string(data) != "inside" {
				t.Fatalf("read escaped root: %q/%v/%v", data, readErr, closeErr)
			}
		}

		_, _ = store.Stat(ctx, "slot/sentinel")
		_ = store.Delete(ctx, "slot/sentinel")
	}

	data, err := os.ReadFile(filepath.Join(outside, "sentinel"))
	if err != nil || string(data) != "outside" {
		t.Fatalf("outside sentinel changed: %q/%v", data, err)
	}

	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 1 {
		t.Fatalf("outside entries changed: %v/%v", entries, err)
	}
}
