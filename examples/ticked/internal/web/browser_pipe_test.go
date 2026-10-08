//go:build browser

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// Broken/private protocol replies must close the owned transport or return a
// redacted error rather than leaking remote values or hanging a fixture call.
func TestBrowserPipe(t *testing.T) {
	for _, tc := range []struct {
		name, frame string
		valid       bool
	}{
		{"success", `{"id":1,"result":{"owned":true}}` + "\x00", true},
		{"rejected", `{"id":1,"error":{"message":"private-response-value"}}` + "\x00", false},
		{"invalid JSON", "private-response-value\x00", false},
		{"truncated", `{"id":1,"result":"private-response-value"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader, writer := io.Pipe()

			t.Cleanup(func() { reader.Close(); writer.Close() })

			pipe := &browserPipe{writer: io.Discard, pending: make(map[int]chan browserMessage), closed: make(chan struct{})}
			go pipe.read(reader)

			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()

			result := make(chan error, 1)

			go func() {
				_, err := pipe.call(ctx, "Runtime.evaluate", map[string]any{}, "")
				result <- err
			}()

			// Wait for the request's pending slot before supplying the actual
			// protocol frame. The wait is bounded by this test's context.
			for {
				pipe.mu.Lock()
				ready := len(pipe.pending) == 1
				pipe.mu.Unlock()

				if ready {
					break
				}

				select {
				case <-ctx.Done():
					t.Fatal("browser request not installed")
				case <-time.After(time.Millisecond):
				}
			}

			_, err := io.WriteString(writer, tc.frame)
			if err != nil {
				t.Fatal(err)
			}

			// An incomplete frame is rejected at EOF. A valid reply stays open
			// until it has been delivered, preventing an artificial close race.
			if !json.Valid([]byte(strings.TrimSuffix(tc.frame, "\x00"))) || !strings.HasSuffix(tc.frame, "\x00") {
				writer.Close()
			}

			err = <-result
			if (err == nil) != tc.valid || err != nil && strings.Contains(err.Error(), "private-response-value") {
				t.Fatalf("valid=%v; safe protocol result=%v", tc.valid, err == nil)
			}
		})
	}
}
