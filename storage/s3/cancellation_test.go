// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/storage"
)

type blockedInput struct {
	started chan struct{}
	release chan struct{}
}

func (b blockedInput) Read([]byte) (int, error) {
	close(b.started)
	<-b.release

	return 0, io.EOF
}

// TestInputCancellation does not return while an arbitrary caller read still owns I/O.
func TestInputCancellation(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(200)
	}))
	defer server.Close()

	options := testOptions(t, server.URL)
	store := newTestStore(t, options)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	input := &inputOwner{Reader: blockedInput{make(chan struct{}), make(chan struct{})}}
	blocked := input.Reader.(blockedInput)
	done := make(chan error, 1)

	go func() {
		_, err := store.Put(ctx, "object", input, storage.WriteOptions{MaxBytes: 8})
		done <- err
	}()

	defer func() {
		close(blocked.release)

		err := <-done
		if !errors.Is(err, context.Canceled) || input.closed || requests.Load() != 0 {
			t.Errorf("blocked cancellation: %v, closed=%t requests=%d", err, input.closed, requests.Load())
		}

		assertEmptySpool(t, options.SpoolDirectory)
	}()

	select {
	case <-blocked.started:
	case <-time.After(time.Second):
		t.Fatal("caller input was not reached")
	}

	entries, err := os.ReadDir(options.SpoolDirectory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("private spool admission: %d/%v", len(entries), err)
	}

	info, err := os.Stat(filepath.Join(options.SpoolDirectory, entries[0].Name()))
	if err != nil || info.Mode().Perm() != 0600 || info.Size() > 8 {
		t.Fatalf("private spool bounds: %v/%v", info, err)
	}

	cancel()

	select {
	case err := <-done:
		done <- err

		t.Fatal("returned while caller read was blocked:", err)
	case <-time.After(20 * time.Millisecond):
	}
}

// TestMutationCancellation retains the context cause after dispatch without retry.
func TestMutationCancellation(t *testing.T) {
	for _, operation := range []string{"put", "delete"} {
		t.Run(operation, func(t *testing.T) {
			accepted := make(chan struct{})

			server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				_, _ = io.Copy(io.Discard, request.Body)

				close(accepted)
				<-request.Context().Done()
			}))
			defer server.Close()

			options := testOptions(t, server.URL)
			store := newTestStore(t, options)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			done := make(chan error, 1)

			go func() {
				if operation == "delete" {
					done <- store.Delete(ctx, "object")

					return
				}

				_, err := store.Put(ctx, "object", strings.NewReader("data"), storage.WriteOptions{MaxBytes: 8})
				done <- err
			}()

			select {
			case <-accepted:
			case <-time.After(time.Second):
				cancel()
				<-done
				t.Fatal("mutation did not reach provider")
			}

			cancel()

			err := <-done

			if !errors.Is(err, storage.ErrIndeterminate) || !errors.Is(err, context.Canceled) {
				t.Fatalf("mutation cancellation lost cause or uncertainty: %v", err)
			}

			assertEmptySpool(t, options.SpoolDirectory)
		})
	}
}
