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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/credentials"

	"hatmax.adrianpk.com/storage"
)

func testOptions(t *testing.T, endpoint string) Options {
	t.Helper()

	return Options{
		Bucket: "private-bucket", Region: "us-east-1", Endpoint: endpoint,
		Credentials:    credentials.NewStaticCredentialsProvider("fixture-key", "fixture-secret", ""),
		SpoolDirectory: t.TempDir(), MaxSpoolBytes: 4096, AllowInsecureLoopback: true,
	}
}

func newTestStore(t *testing.T, options Options) *Store {
	t.Helper()

	store, err := NewStore(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store
}

func assertEmptySpool(t *testing.T, directory string) {
	t.Helper()

	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("spool not empty: %d entries, error=%v", len(entries), err)
	}
}

// TestConfiguration rejects invalid private endpoint, capacity and retry settings.
func TestConfiguration(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Options)
	}{
		{"bucket", func(o *Options) { o.Bucket = "../outside" }},
		{"uppercase bucket", func(o *Options) { o.Bucket = "Private" }},
		{"region", func(o *Options) { o.Region = "" }},
		{"capacity", func(o *Options) { o.MaxSpoolBytes = 0 }},
		{"missing spool", func(o *Options) { o.SpoolDirectory = "" }},
		{"negative attempts", func(o *Options) { o.ReadAttempts = -1 }},
		{"excess attempts", func(o *Options) { o.ReadAttempts = 6 }},
		{"negative timeout", func(o *Options) { o.OperationTimeout = -time.Second }},
		{"excess timeout", func(o *Options) { o.OperationTimeout = 121 * time.Second }},
		{"insecure production", func(o *Options) { o.Endpoint = "http://objects.example.test" }},
		{"implicit insecure", func(o *Options) { o.AllowInsecureLoopback = false }},
		{"endpoint credentials", func(o *Options) { o.Endpoint = "https://user:password@objects.example.test" }},
		{"endpoint query", func(o *Options) { o.Endpoint = "https://objects.example.test/?other=target" }},
		{"endpoint path", func(o *Options) { o.Endpoint = "https://objects.example.test/other" }},
		{"endpoint scheme", func(o *Options) { o.Endpoint = "ftp://objects.example.test" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			options := testOptions(t, "http://127.0.0.1:1")
			tc.change(&options)

			store, err := NewStore(context.Background(), options)
			if !errors.Is(err, storage.ErrInvalid) || store != nil {
				t.Fatalf("constructor=%v/%v", store, err)
			}
		})
	}

	t.Run("default SDK credentials", func(t *testing.T) {
		t.Setenv("AWS_ACCESS_KEY_ID", "fixture-key")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "fixture-secret")
		t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
		options := testOptions(t, "https://objects.example.test")
		options.Credentials = nil
		store := newTestStore(t, options)

		credentials, err := store.client.Options().Credentials.Retrieve(context.Background())
		if err != nil || credentials.AccessKeyID != "fixture-key" {
			t.Fatal("default SDK credential chain not retained")
		}
	})
}

type inputOwner struct {
	io.Reader
	read   int
	closed bool
}

func (r *inputOwner) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.read += n

	return n, err
}

func (r *inputOwner) Close() error {
	r.closed = true

	return nil
}

// TestInputAdmission bounds real input and prevents HTTP mutations on invalid input.
func TestInputAdmission(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)

		_, _ = io.Copy(io.Discard, r.Body)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	options := testOptions(t, server.URL)
	store := newTestStore(t, options)
	cases := []struct {
		name    string
		key     string
		maximum int64
		body    string
		want    error
	}{
		{"overflow", "object", 4, "too long", storage.ErrTooLarge},
		{"invalid key", "a/../object", 4, "body", storage.ErrInvalid},
		{"invalid limit", "object", 0, "body", storage.ErrInvalid},
		{"spool capacity", "object", 4097, "body", storage.ErrInvalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := &inputOwner{Reader: strings.NewReader(tc.body)}

			object, err := store.Put(context.Background(), tc.key, input, storage.WriteOptions{MaxBytes: tc.maximum})
			if !errors.Is(err, tc.want) || object != (storage.Object{}) || input.closed || input.read > 5 || requests.Load() != 0 {
				t.Fatalf("input admission: object=%+v error=%v bytes=%d closed=%t requests=%d", object, err, input.read, input.closed, requests.Load())
			}

			assertEmptySpool(t, options.SpoolDirectory)
		})
	}

	input := &inputOwner{Reader: strings.NewReader("body")}

	object, err := store.Put(context.Background(), "nested/object", input, storage.WriteOptions{MaxBytes: 4})
	if err != nil || object.SizeBytes != 4 || object.ContentType != "text/plain; charset=utf-8" || input.closed || requests.Load() != 1 {
		t.Fatalf("complete upload: object=%+v error=%v closed=%t requests=%d", object, err, input.closed, requests.Load())
	}

	assertEmptySpool(t, options.SpoolDirectory)
}
