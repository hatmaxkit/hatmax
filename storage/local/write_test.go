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
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/storage"
)

type ownedInput struct {
	io.Reader
	reads  int
	bytes  int
	closes int
}

func (r *ownedInput) Read(p []byte) (int, error) {
	r.reads++
	n, err := r.Reader.Read(p)
	r.bytes += n

	return n, err
}

func (r *ownedInput) Close() error {
	r.closes++

	return nil
}

type failedInput struct {
	err error
}

func (r failedInput) Read([]byte) (int, error) {
	return 0, r.err
}

type cancelInput struct {
	cancel context.CancelFunc
}

func (r cancelInput) Read(p []byte) (int, error) {
	r.cancel()

	return copy(p, "partial"), nil
}

type blockedInput struct {
	started chan struct{}
	release chan struct{}
}

func (r blockedInput) Read([]byte) (int, error) {
	close(r.started)
	<-r.release

	return 0, io.EOF
}

func assertNoSpools(t *testing.T, directory string) {
	t.Helper()

	err := filepath.WalkDir(directory, func(name string, entry os.DirEntry, err error) error {
		if err == nil && strings.HasPrefix(entry.Name(), ".object-") {
			t.Errorf("owned spool left behind: %s", name)
		}

		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestWriteFailure proves real byte limits, input ownership, rollback and cancellation.
func TestWriteFailure(t *testing.T) {
	ctx := context.Background()

	t.Run("input failures", func(t *testing.T) {
		broken := errors.New("input failed")
		cases := []struct {
			name  string
			input io.Reader
			want  error
		}{
			{"overflow", strings.NewReader(strings.Repeat("x", 100)), storage.ErrTooLarge},
			{"partial", io.MultiReader(strings.NewReader("part"), failedInput{broken}), broken},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				directory := t.TempDir()
				store := newTestStore(t, directory)

				_, err := store.Put(ctx, "document", strings.NewReader("old"), storage.WriteOptions{MaxBytes: 8})
				if err != nil {
					t.Fatal(err)
				}

				input := &ownedInput{Reader: tc.input}

				object, err := store.Put(ctx, "document", input, storage.WriteOptions{MaxBytes: 8})
				if !errors.Is(err, tc.want) || object != (storage.Object{}) || input.closes != 0 || input.bytes > 9 {
					t.Fatalf("object=%+v, error=%v, input=%+v", object, err, input)
				}

				assertObject(t, store, "document", "old")
				assertNoSpools(t, directory)
			})
		}
	})

	t.Run("exact limit", func(t *testing.T) {
		directory := t.TempDir()
		store := newTestStore(t, directory)
		input := &ownedInput{Reader: strings.NewReader("12345678")}

		object, err := store.Put(ctx, "document", input, storage.WriteOptions{MaxBytes: 8})
		if err != nil || object.SizeBytes != 8 || input.bytes != 8 || input.closes != 0 {
			t.Fatalf("object=%+v, error=%v, input=%+v", object, err, input)
		}

		reader, object, err := store.Open(ctx, "document", storage.ReadOptions{MaxBytes: 7})
		if !errors.Is(err, storage.ErrTooLarge) || reader != nil || object != (storage.Object{}) {
			t.Fatalf("metadata overflow=%v/%+v/%v", reader, object, err)
		}

		reader, _, err = store.Open(ctx, "document", storage.ReadOptions{MaxBytes: 8})
		if err != nil {
			t.Fatal(err)
		}

		data, readErr := io.ReadAll(reader)

		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || string(data) != "12345678" {
			t.Fatalf("exact read=%q/%v/%v", data, readErr, closeErr)
		}

		assertNoSpools(t, directory)
	})

	t.Run("growing file", func(t *testing.T) {
		directory := t.TempDir()
		store := newTestStore(t, directory)

		_, err := store.Put(ctx, "document", strings.NewReader("small"), storage.WriteOptions{MaxBytes: 8})
		if err != nil {
			t.Fatal(err)
		}

		reader, _, err := store.Open(ctx, "document", storage.ReadOptions{MaxBytes: 8})
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()

		file, err := os.OpenFile(filepath.Join(directory, "document"), os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}

		_, writeErr := file.WriteString("now too long")

		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			t.Fatalf("growth=%v/%v", writeErr, closeErr)
		}

		data, err := io.ReadAll(reader)
		if !errors.Is(err, storage.ErrTooLarge) || len(data) != 8 {
			t.Fatalf("actual overflow=%q/%v", data, err)
		}
	})

	t.Run("invalid before I/O", func(t *testing.T) {
		directory := t.TempDir()
		store := newTestStore(t, directory)

		for _, options := range []storage.WriteOptions{{MaxBytes: 0}, {MaxBytes: -1}, {MaxBytes: 1025}, {MaxBytes: 8, ContentType: "broken"}} {
			input := &ownedInput{Reader: strings.NewReader("data")}

			_, err := store.Put(ctx, "nested/document", input, options)
			if !errors.Is(err, storage.ErrInvalid) || input.reads != 0 || input.closes != 0 {
				t.Fatalf("invalid options=%+v, error=%v, input=%+v", options, err, input)
			}
		}

		_, err := store.Put(ctx, "nested/document", nil, storage.WriteOptions{MaxBytes: 8})
		if !errors.Is(err, storage.ErrInvalid) {
			t.Fatalf("nil input=%v", err)
		}

		reader, object, err := store.Open(ctx, "missing", storage.ReadOptions{})
		if !errors.Is(err, storage.ErrInvalid) || reader != nil || object != (storage.Object{}) {
			t.Fatalf("invalid read=%v/%+v/%v", reader, object, err)
		}

		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 0 {
			t.Fatalf("invalid options performed I/O: %v/%v", entries, err)
		}
	})

	t.Run("cancelled phases", func(t *testing.T) {
		directory := t.TempDir()
		store := newTestStore(t, directory)
		cancelled, cancel := context.WithCancel(ctx)
		cancel()

		input := &ownedInput{Reader: strings.NewReader("private")}

		_, err := store.Put(cancelled, "nested/document", input, storage.WriteOptions{MaxBytes: 8})
		if !errors.Is(err, context.Canceled) || input.reads != 0 {
			t.Fatalf("pre-cancelled write=%v, input=%+v", err, input)
		}

		_, err = store.Stat(cancelled, "missing")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-cancelled stat=%v", err)
		}

		err = store.Delete(cancelled, "missing")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-cancelled delete=%v", err)
		}

		during, stop := context.WithCancel(ctx)
		defer stop()

		_, err = store.Put(during, "document", cancelInput{stop}, storage.WriteOptions{MaxBytes: 8})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("in-read cancellation=%v", err)
		}

		_, err = store.Stat(ctx, "document")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("cancelled write became visible: %v", err)
		}

		_, err = store.Put(ctx, "document", strings.NewReader("valid"), storage.WriteOptions{MaxBytes: 8})
		if err != nil {
			t.Fatal(err)
		}

		reading, stopReading := context.WithCancel(ctx)

		reader, _, err := store.Open(reading, "document", storage.ReadOptions{MaxBytes: 8})
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()

		stopReading()

		_, err = io.ReadAll(reader)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("reader cancellation=%v", err)
		}

		deadline, finish := context.WithDeadline(ctx, time.Now().Add(-time.Second))
		defer finish()

		_, err = store.Stat(deadline, "document")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("deadline=%v", err)
		}

		assertNoSpools(t, directory)
	})

	t.Run("blocking input", testBlockingInput)

	t.Run("commit failure", func(t *testing.T) {
		directory := t.TempDir()
		store := newTestStore(t, directory)

		err := os.Mkdir(filepath.Join(directory, "object"), 0700)
		if err != nil {
			t.Fatal(err)
		}

		object, err := store.Put(ctx, "object", strings.NewReader("data"), storage.WriteOptions{MaxBytes: 8})
		if !errors.Is(err, storage.ErrIndeterminate) || object != (storage.Object{}) {
			t.Fatalf("commit failure=%+v/%v", object, err)
		}

		_, err = store.Stat(ctx, "object")
		if !errors.Is(err, storage.ErrInvalid) || errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("non-object classified as absent: %v", err)
		}

		err = store.Delete(ctx, "object")
		if !errors.Is(err, storage.ErrInvalid) {
			t.Fatalf("deleted directory: %v", err)
		}

		assertNoSpools(t, directory)
	})
}

func testBlockingInput(t *testing.T) {
	directory := t.TempDir()
	store := newTestStore(t, directory)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	input := blockedInput{make(chan struct{}), make(chan struct{})}
	done := make(chan error, 1)

	go func() {
		_, err := store.Put(ctx, "document", input, storage.WriteOptions{MaxBytes: 8})
		done <- err
	}()

	// Always release and join the test-owned operation before its fixture closes.
	defer func() {
		close(input.release)

		err := <-done
		if !errors.Is(err, context.Canceled) {
			t.Errorf("released blocked input=%v", err)
		}

		assertNoSpools(t, directory)
	}()

	select {
	case <-input.started:
	case <-time.After(time.Second):
		t.Fatal("write did not reach caller input")
	}

	cancel()

	select {
	case err := <-done:
		done <- err

		t.Fatal("operation returned while caller read was still blocked:", err)
	case <-time.After(20 * time.Millisecond):
	}
}
