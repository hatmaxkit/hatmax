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
	"strings"
	"testing"

	"hatmax.adrianpk.com/storage"
)

func newTestStore(t *testing.T, directory string) *Store {
	t.Helper()

	store, err := NewStore(directory, Options{MaxSpoolBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store
}

func assertObject(t *testing.T, store *Store, key, want string) {
	t.Helper()

	reader, object, err := store.Open(context.Background(), key, storage.ReadOptions{MaxBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}

	data, readErr := io.ReadAll(reader)

	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || string(data) != want || object.Key != key || object.SizeBytes != int64(len(want)) {
		t.Fatalf("object=%+v, data=%q, errors=%v/%v", object, data, readErr, closeErr)
	}
}

// TestObjects exercises real object lifecycles, metadata restart and opened snapshots.
func TestObjects(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		content string
		typeOf  string
	}{
		{"empty", "", "text/plain; charset=utf-8"},
		{"document", "private document", "text/plain; charset=utf-8"},
		{"binary", "\x00\x01\xff", "application/octet-stream"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			store := newTestStore(t, directory)
			key := "nested/document.fake"

			object, err := store.Put(ctx, key, strings.NewReader(tc.content), storage.WriteOptions{ContentType: "image/png", MaxBytes: 1024})
			if err != nil || object.ContentType != tc.typeOf {
				t.Fatalf("put=%+v, error=%v", object, err)
			}

			info, err := os.Stat(filepath.Join(directory, key))
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("private object mode: %v, %v", info, err)
			}

			info, err = os.Stat(filepath.Join(directory, "nested"))
			if err != nil || info.Mode().Perm() != 0700 {
				t.Fatalf("private directory mode: %v, %v", info, err)
			}

			assertObject(t, store, key, tc.content)
			// A new handle derives metadata from bytes rather than filename or memory.
			reopened := newTestStore(t, directory)

			observed, err := reopened.Stat(ctx, key)
			if err != nil || observed != object {
				t.Fatalf("restart metadata=%+v, error=%v", observed, err)
			}

			err = reopened.Delete(ctx, key)
			if err != nil {
				t.Fatal(err)
			}

			err = reopened.Delete(ctx, key)
			if err != nil {
				t.Fatal("delete absent:", err)
			}

			_, err = store.Stat(ctx, key)
			if !errors.Is(err, storage.ErrNotFound) {
				t.Fatalf("absence=%v", err)
			}

			reader, empty, err := store.Open(ctx, key, storage.ReadOptions{MaxBytes: 1024})
			if !errors.Is(err, storage.ErrNotFound) || reader != nil || empty != (storage.Object{}) {
				t.Fatalf("absent open=%v/%+v/%v", reader, empty, err)
			}
		})
	}

	t.Run("snapshot", func(t *testing.T) {
		store := newTestStore(t, t.TempDir())
		options := storage.WriteOptions{MaxBytes: 1024}

		_, err := store.Put(ctx, "document", strings.NewReader("original"), options)
		if err != nil {
			t.Fatal(err)
		}

		reader, observed, err := store.Open(ctx, "document", storage.ReadOptions{MaxBytes: 1024})
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()

		_, err = store.Put(ctx, "document", strings.NewReader("replacement"), options)
		if err != nil {
			t.Fatal(err)
		}

		assertObject(t, store, "document", "replacement")

		err = store.Close()
		if err != nil {
			t.Fatal(err)
		}

		data, err := io.ReadAll(reader)
		if err != nil || string(data) != "original" || observed.SizeBytes != 8 {
			t.Fatalf("opened snapshot=%q, metadata=%+v, error=%v", data, observed, err)
		}
	})

	t.Run("configuration", func(t *testing.T) {
		for _, capacity := range []int64{0, -1} {
			_, err := NewStore(t.TempDir(), Options{capacity})
			if !errors.Is(err, storage.ErrInvalid) {
				t.Fatalf("capacity=%d: %v", capacity, err)
			}
		}

		_, err := NewStore("", Options{1024})
		if !errors.Is(err, storage.ErrInvalid) {
			t.Fatalf("empty root: %v", err)
		}

		missing := filepath.Join(t.TempDir(), "missing")

		_, err = NewStore(missing, Options{1024})
		if err == nil || errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("missing root must be configuration failure: %v", err)
		}

		_, err = os.Stat(missing)
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("constructor created root: %v", err)
		}
	})
}
