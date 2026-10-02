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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type uploadInput struct {
	reader io.Reader
	read   int
	cancel context.CancelFunc
}

func (r *uploadInput) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)

	r.read += n
	if r.cancel != nil {
		r.cancel()
	}

	return n, err
}

type zeroInput struct{}

func (zeroInput) Read(p []byte) (int, error) {
	clear(p)

	return len(p), nil
}

// Uploads must preserve binary bytes, MIME detection, bucket, and object key.
// Failed input reads and cancellation must prevent any HTTP upload.
func TestUploadInput(t *testing.T) {
	for _, name := range []string{"binary", "empty", "over limit", "canceled before", "canceled read", "read failure", "rejected"} {
		t.Run(name, func(t *testing.T) {
			body := []byte{0, 1, 255, 128, 'a', 'b'}
			if name == "empty" {
				body = nil
			}

			var requests atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)

				if name == "rejected" {
					w.WriteHeader(http.StatusForbidden)
					io.WriteString(w, "<Error><Code>AccessDenied</Code><Message>denied</Message></Error>")

					return
				}

				got, err := io.ReadAll(r.Body)
				if err != nil || !bytes.Equal(got, body) {
					t.Errorf("upload body: %x, %v; want %x", got, err, body)
				}

				if r.Method != http.MethodPut || r.URL.Path != "/test-bucket/photos/sample.bin" || r.Header.Get("Content-Type") != http.DetectContentType(body) {
					t.Errorf("upload metadata: %s %s %s", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			store, err := NewStore(context.Background(), Options{
				Bucket: "test-bucket", BaseURL: server.URL, Endpoint: server.URL,
				Region: "us-east-1", AccessKey: "test-key", SecretKey: "test-secret",
			})
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			input := &uploadInput{reader: bytes.NewReader(body)}

			var want error

			switch name {
			case "over limit":
				input.reader = io.LimitReader(zeroInput{}, (20<<20)+100)
			case "canceled before":
				cancel()

				want = context.Canceled
			case "canceled read":
				input.cancel = cancel
				want = context.Canceled
			case "read failure":
				want = io.ErrUnexpectedEOF
				input.reader = failedInput{}
			}

			err = store.Put(ctx, "photos/sample.bin", input)
			if want != nil {
				if !errors.Is(err, want) || requests.Load() != 0 {
					t.Fatalf("Put: %v; requests: %d; want %v before HTTP", err, requests.Load(), want)
				}

				if name == "canceled before" && input.read != 0 {
					t.Fatalf("canceled upload read %d bytes", input.read)
				}

				return
			}

			if name == "over limit" {
				if err == nil || !strings.Contains(err.Error(), "exceeds") || requests.Load() != 0 || input.read != (20<<20)+1 {
					t.Fatalf("unbounded upload: %v, %d requests, %d bytes", err, requests.Load(), input.read)
				}

				return
			}

			if name == "rejected" {
				if err == nil || !strings.Contains(err.Error(), "cannot put object") || requests.Load() != 1 {
					t.Fatalf("rejected upload: %v, %d requests", err, requests.Load())
				}

				return
			}

			if err != nil || requests.Load() != 1 {
				t.Fatalf("Put: %v; requests: %d", err, requests.Load())
			}
		})
	}
}

type failedInput struct{}

func (failedInput) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
