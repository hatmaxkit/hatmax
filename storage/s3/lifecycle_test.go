//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"hatmax.adrianpk.com/storage"
	"hatmax.adrianpk.com/storage/local"
)

// One caller workflow must preserve completed bytes and uncertainty across adapters.
func TestObjectLifecycle(t *testing.T) {
	remote, options := liveStore(t)

	filesystem, err := local.NewStore(t.TempDir(), local.Options{MaxSpoolBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := filesystem.Close()
		if err != nil {
			t.Error(err)
		}
	})

	for _, adapter := range []struct {
		name  string
		store storage.Store
	}{{"local", filesystem}, {"remote", remote}} {
		t.Run(adapter.name, func(t *testing.T) {
			ctx := context.Background()
			staging, final := "acceptance/staging", "acceptance/final"
			original := []byte("complete private document\n")
			write := storage.WriteOptions{ContentType: "text/plain", MaxBytes: 4096}

			input := &inputOwner{Reader: bytes.NewReader(original)}

			_, err := adapter.store.Put(ctx, staging, input, write)
			if err != nil || input.closed {
				t.Fatalf("staging write or caller ownership: %v", err)
			}

			observed, err := adapter.store.Stat(ctx, staging)
			if err != nil || observed.SizeBytes != int64(len(original)) {
				t.Fatalf("staging metadata: %+v %v", observed, err)
			}

			opened, identity, err := adapter.store.Open(ctx, staging, storage.ReadOptions{MaxBytes: 4096})
			if err != nil {
				t.Fatal(err)
			}

			// A new staging version after Open must not change this validated snapshot.
			_, replaceErr := adapter.store.Put(ctx, staging, bytes.NewBufferString("later staging"), write)
			snapshot, readErr := io.ReadAll(opened)

			closeErr := opened.Close()
			if replaceErr != nil || readErr != nil || closeErr != nil || identity.SizeBytes != int64(len(snapshot)) || !bytes.Equal(snapshot, original) {
				t.Fatal("opened identity or complete snapshot changed")
			}

			_, err = adapter.store.Put(ctx, final, bytes.NewReader(snapshot), write)
			if err != nil {
				t.Fatal(err)
			}

			// Rejected replacement must leave the previous complete final bytes intact.
			_, err = adapter.store.Put(ctx, final, bytes.NewReader(bytes.Repeat([]byte{'x'}, 4097)), write)
			if !errors.Is(err, storage.ErrTooLarge) {
				t.Fatalf("oversize replacement: %v", err)
			}

			canceled, cancel := context.WithCancel(ctx)
			cancel()

			err = adapter.store.Delete(canceled, final)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled delete: %v", err)
			}

			reader, _, err := adapter.store.Open(ctx, final, storage.ReadOptions{MaxBytes: 4096})
			if err != nil {
				t.Fatal(err)
			}

			actual, readErr := io.ReadAll(reader)

			closeErr = reader.Close()
			if readErr != nil || closeErr != nil || !bytes.Equal(actual, snapshot) {
				t.Fatal("final differs from validated snapshot after rejected operations")
			}

			for _, key := range []string{staging, final} {
				for range 2 {
					err = adapter.store.Delete(ctx, key)
					if err != nil {
						t.Fatal(err)
					}
				}

				_, err = adapter.store.Stat(ctx, key)
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatalf("confirmed absence: %v", err)
				}
			}
		})
	}

	assertEmptySpool(t, options.SpoolDirectory)
	live.record("lifecycle: local and remote snapshots, failed replacement, cancellation and absence passed\n")
}
