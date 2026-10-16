//go:build integration

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
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/credentials"

	"hatmax.adrianpk.com/storage"
)

// TestPrivateObjects exercises real private object lifecycles and opened identity.
func TestPrivateObjects(t *testing.T) {
	store, options := liveStore(t)
	ctx := context.Background()

	for _, content := range []string{"", "private document", "\x00\x01\xff"} {
		key := "lifecycle/object"
		input := &inputOwner{Reader: strings.NewReader(content)}

		object, err := store.Put(ctx, key, input, storage.WriteOptions{ContentType: "application/octet-stream", MaxBytes: 4096})
		if err != nil || input.closed || object.SizeBytes != int64(len(content)) {
			t.Fatalf("private put: %+v/%v", object, err)
		}

		observed, err := store.Stat(ctx, key)
		if err != nil || observed != object {
			t.Fatalf("private stat: %+v/%v", observed, err)
		}

		response, err := live.httpClient.Get(live.endpoint + "/" + live.bucket + "/" + key)
		if err != nil {
			t.Fatal(err)
		}

		_ = response.Body.Close()

		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("anonymous private access status=%d", response.StatusCode)
		}

		reader, opened, err := store.Open(ctx, key, storage.ReadOptions{MaxBytes: 4096})
		if err != nil || opened != object {
			t.Fatalf("private open: %+v/%v", opened, err)
		}

		data, readErr := io.ReadAll(reader)
		closeErr := reader.Close()

		if readErr != nil || closeErr != nil || string(data) != content {
			t.Fatalf("private bytes=%q errors=%v/%v", data, readErr, closeErr)
		}

		for range 2 {
			err = store.Delete(ctx, key)
			if err != nil {
				t.Fatal(err)
			}
		}

		_, err = store.Stat(ctx, key)
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("confirmed absence: %v", err)
		}

		assertEmptySpool(t, options.SpoolDirectory)
	}

	_, err := store.Put(ctx, "snapshot/staging", strings.NewReader("original"), storage.WriteOptions{MaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}

	reader, opened, err := store.Open(ctx, "snapshot/staging", storage.ReadOptions{MaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	_, err = store.Put(ctx, "snapshot/staging", strings.NewReader("later replacement"), storage.WriteOptions{MaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(reader)
	if err != nil || string(data) != "original" || opened.SizeBytes != 8 {
		t.Fatalf("opened snapshot changed: %q/%+v/%v", data, opened, err)
	}

	current, err := store.Stat(ctx, "snapshot/staging")
	if err != nil || current.SizeBytes != int64(len("later replacement")) {
		t.Fatalf("later observation: %+v/%v", current, err)
	}
}

type failedInput struct{}

func (failedInput) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

// TestRemoteBounds preserves old bytes and cleans real spools under input failure.
func TestRemoteBounds(t *testing.T) {
	store, options := liveStore(t)
	ctx := context.Background()
	key := "bounds/object"

	_, err := store.Put(ctx, key, strings.NewReader("original"), storage.WriteOptions{MaxBytes: 8})
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name  string
		input io.Reader
		want  error
	}{
		{"overflow", strings.NewReader(strings.Repeat("x", 100)), storage.ErrTooLarge},
		{"partial failure", io.MultiReader(strings.NewReader("part"), failedInput{}), io.ErrUnexpectedEOF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := &inputOwner{Reader: tc.input}

			object, err := store.Put(ctx, key, input, storage.WriteOptions{MaxBytes: 8})
			if !errors.Is(err, tc.want) || object != (storage.Object{}) || input.read > 9 || input.closed {
				t.Fatalf("failed input: %+v/%v bytes=%d closed=%t", object, err, input.read, input.closed)
			}

			reader, _, err := store.Open(ctx, key, storage.ReadOptions{MaxBytes: 8})
			if err != nil {
				t.Fatal(err)
			}

			data, readErr := io.ReadAll(reader)
			closeErr := reader.Close()

			if readErr != nil || closeErr != nil || string(data) != "original" {
				t.Fatalf("old content changed: %q/%v/%v", data, readErr, closeErr)
			}

			assertEmptySpool(t, options.SpoolDirectory)
		})
	}

	reader, object, err := store.Open(ctx, key, storage.ReadOptions{MaxBytes: 7})
	if !errors.Is(err, storage.ErrTooLarge) || reader != nil || object != (storage.Object{}) {
		t.Fatalf("real size bound: %+v/%v", object, err)
	}

	reading, cancel := context.WithCancel(ctx)

	reader, _, err = store.Open(reading, key, storage.ReadOptions{MaxBytes: 8})
	if err != nil {
		cancel()
		t.Fatal(err)
	}

	cancel()

	_, readErr := io.ReadAll(reader)
	closeErr := reader.Close()

	if !errors.Is(readErr, context.Canceled) || closeErr != nil {
		t.Fatalf("response cancellation: %v/%v", readErr, closeErr)
	}

	input := &inputOwner{Reader: strings.NewReader("replacement")}

	_, err = store.Put(reading, key, input, storage.WriteOptions{MaxBytes: 16})
	if !errors.Is(err, context.Canceled) || input.read != 0 || input.closed {
		t.Fatalf("pre-request cancellation: %v", err)
	}

	assertEmptySpool(t, options.SpoolDirectory)
}

// TestRemoteFailures distinguishes real authorization denial and committed lost replies.
func TestRemoteFailures(t *testing.T) {
	store, _ := liveStore(t)
	ctx := context.Background()
	options := testOptions(t, live.endpoint)
	options.Bucket = live.bucket
	options.Credentials = credentials.NewStaticCredentialsProvider("unknown-fixture-key", "unknown-fixture-secret", "")
	denied := newTestStore(t, options)

	_, putErr := denied.Put(ctx, "denied", strings.NewReader("data"), storage.WriteOptions{MaxBytes: 8})
	reader, _, openErr := denied.Open(ctx, "denied", storage.ReadOptions{MaxBytes: 8})
	_, statErr := denied.Stat(ctx, "denied")
	deleteErr := denied.Delete(ctx, "denied")

	for _, err := range []error{putErr, openErr, statErr, deleteErr} {
		if err == nil || errors.Is(err, storage.ErrNotFound) || errors.Is(err, storage.ErrIndeterminate) {
			t.Fatalf("real denial misclassified: %v", err)
		}
	}

	if reader != nil {
		t.Fatal("denied response transferred a reader")
	}

	assertEmptySpool(t, options.SpoolDirectory)

	t.Run("lost mutation acknowledgments", func(t *testing.T) {
		target, err := url.Parse(live.endpoint)
		if err != nil {
			t.Fatal(err)
		}

		var mutations atomic.Int32

		proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			forward := request.Clone(request.Context())
			forward.RequestURI = ""
			forward.URL.Scheme = target.Scheme
			forward.URL.Host = target.Host

			response, err := live.httpClient.Do(forward)
			if err != nil {
				t.Error("owned proxy forwarding failed:", err)
				w.WriteHeader(502)

				return
			}
			defer response.Body.Close()

			_, _ = io.Copy(io.Discard, response.Body)

			if response.StatusCode < 200 || response.StatusCode >= 300 {
				t.Errorf("owned proxy mutation rejected: status=%d", response.StatusCode)
				w.WriteHeader(502)

				return
			}

			mutations.Add(1)

			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)

				return
			}

			_ = connection.Close()
		}))
		defer proxy.Close()

		proxied := testOptions(t, proxy.URL)
		proxied.Bucket = live.bucket
		proxied.Credentials = live.provider
		uncertain := newTestStore(t, proxied)
		key := "unknown/mutation"

		_, err = uncertain.Put(ctx, key, strings.NewReader("committed"), storage.WriteOptions{MaxBytes: 16})
		if !errors.Is(err, storage.ErrIndeterminate) || mutations.Load() != 1 {
			t.Fatalf("lost write confirmation: error=%v mutations=%d", err, mutations.Load())
		}

		observed, err := store.Stat(ctx, key)
		if err != nil || observed.SizeBytes != 9 {
			t.Fatalf("write reconciliation: %+v/%v", observed, err)
		}

		err = uncertain.Delete(ctx, key)
		if !errors.Is(err, storage.ErrIndeterminate) || mutations.Load() != 2 {
			t.Fatalf("lost delete confirmation: error=%v mutations=%d", err, mutations.Load())
		}

		_, err = store.Stat(ctx, key)
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("delete reconciliation: %v", err)
		}

		assertEmptySpool(t, proxied.SpoolDirectory)
	})
}
