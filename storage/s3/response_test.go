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
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/storage"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type trackedBody struct {
	io.Reader
	closes atomic.Int32
}

func (b *trackedBody) Close() error {
	b.closes.Add(1)

	return nil
}

// TestResponseOwnership rejects malformed metadata and bounds the actual body.
func TestResponseOwnership(t *testing.T) {
	cases := []struct {
		name    string
		size    int64
		typeOf  string
		body    string
		maximum int64
		reject  bool
		over    bool
	}{
		{"owned", 4, "text/plain", "data", 4, false, false},
		{"metadata limit", 8, "text/plain", "too long", 4, true, false},
		{"missing length", -1, "text/plain", "data", 4, true, false},
		{"missing type", 4, "", "data", 4, true, false},
		{"malformed type", 4, "broken", "data", 4, true, false},
		{"actual limit", 4, "text/plain", "too long", 4, false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newTestStore(t, testOptions(t, "http://127.0.0.1:1"))
			body := &trackedBody{Reader: strings.NewReader(tc.body)}
			store.httpClient.Transport = roundTrip(func(request *http.Request) (*http.Response, error) {
				header := make(http.Header)
				header.Set("Content-Type", tc.typeOf)

				if tc.size >= 0 {
					header.Set("Content-Length", strconv.FormatInt(tc.size, 10))
				}

				return &http.Response{StatusCode: 200, Header: header, ContentLength: tc.size, Body: body, Request: request}, nil
			})

			reader, object, err := store.Open(context.Background(), "object", storage.ReadOptions{MaxBytes: tc.maximum})
			if tc.reject {
				if err == nil || reader != nil || object != (storage.Object{}) || body.closes.Load() != 1 || errors.Is(err, storage.ErrNotFound) {
					t.Fatalf("rejected response: metadata=%+v error=%v closes=%d", object, err, body.closes.Load())
				}

				return
			}

			if err != nil || body.closes.Load() != 0 {
				t.Fatalf("ownership transferred incorrectly: %v, closes=%d", err, body.closes.Load())
			}

			data, readErr := io.ReadAll(reader)
			closeErr := reader.Close()

			if tc.over && !errors.Is(readErr, storage.ErrTooLarge) || !tc.over && readErr != nil || closeErr != nil || len(data) > int(tc.maximum) || body.closes.Load() != 1 {
				t.Fatalf("bounded response: bytes=%d error=%v/%v closes=%d", len(data), readErr, closeErr, body.closes.Load())
			}
		})
	}
}

// TestProviderErrors preserves denial and missing-bucket errors, with finite retries.
func TestProviderErrors(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		code      string
		operation string
		attempts  int
		absent    bool
		unknown   bool
	}{
		{"absent get", 404, "NoSuchKey", "get", 1, true, false},
		{"missing bucket", 404, "NoSuchBucket", "get", 1, false, false},
		{"denied get", 403, "AccessDenied", "get", 1, false, false},
		{"false absence", 403, "NoSuchKey", "get", 1, false, false},
		{"denied put", 403, "AccessDenied", "put", 1, false, false},
		{"unknown put", 503, "ServiceUnavailable", "put", 1, false, true},
		{"unknown delete", 503, "ServiceUnavailable", "delete", 1, false, true},
		{"read retries", 503, "ServiceUnavailable", "get", 2, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)

				_, _ = io.Copy(io.Discard, r.Body)

				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, "<Error><Code>"+tc.code+"</Code><Message>fixture failure</Message></Error>")
			}))
			defer server.Close()

			options := testOptions(t, server.URL)
			options.ReadAttempts = 2
			store := newTestStore(t, options)
			ctx := context.Background()

			var err error

			switch tc.operation {
			case "get":
				_, _, err = store.Open(ctx, "object", storage.ReadOptions{MaxBytes: 8})
			case "put":
				_, err = store.Put(ctx, "object", strings.NewReader("data"), storage.WriteOptions{MaxBytes: 8})
			case "delete":
				err = store.Delete(ctx, "object")
			}

			if err == nil || errors.Is(err, storage.ErrNotFound) != tc.absent || errors.Is(err, storage.ErrIndeterminate) != tc.unknown || int(requests.Load()) != tc.attempts {
				t.Fatalf("classification: error=%v requests=%d", err, requests.Load())
			}

			assertEmptySpool(t, options.SpoolDirectory)
		})
	}
}

// TestTransportPolicy verifies TLS certificates and preserves operation deadlines.
func TestTransportPolicy(t *testing.T) {
	t.Run("certificate verification", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
		defer server.Close()

		store := newTestStore(t, testOptions(t, server.URL))

		_, err := store.Stat(context.Background(), "object")
		if err == nil || errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("untrusted TLS response admitted: %v", err)
		}
	})

	t.Run("deadline", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) { <-request.Context().Done() }))
		defer server.Close()

		store := newTestStore(t, testOptions(t, server.URL))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()

		_, err := store.Stat(ctx, "object")
		if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("deadline cause lost: %v", err)
		}
	})
}
