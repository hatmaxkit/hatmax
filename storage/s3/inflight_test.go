//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"hatmax.adrianpk.com/storage"
)

type gatedPayload struct {
	ctx       context.Context
	remaining int64
	tail      int64
	paused    bool
	reached   chan struct{}
	release   chan struct{}
}

func (g *gatedPayload) Read(p []byte) (int, error) {
	if g.remaining == 0 {
		return 0, io.EOF
	}

	if !g.paused && g.remaining == g.tail {
		close(g.reached)

		select {
		case <-g.release:
			g.paused = true
		case <-g.ctx.Done():
			return 0, g.ctx.Err()
		}
	}

	n := min(int64(len(p)), g.remaining)
	if !g.paused {
		n = min(n, g.remaining-g.tail)
	}

	for i := range n {
		p[i] = 'x'
	}

	g.remaining -= n

	return int(n), nil
}

func testInFlight(t *testing.T, store *Store) {
	// Larger than the provider's inline threshold: policy admission occurs before
	// the file stream completes. The test still observes actual bytes after EOF.
	const size = (1 << 20) + (32 << 10)

	request := storage.UploadRequest{Key: "expiry/inflight", ContentType: "application/octet-stream", MinBytes: size, MaxBytes: size, ExpiresAt: time.Now().Add(6 * time.Second)}
	grant := signRequest(t, store, request)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gate := &gatedPayload{ctx: ctx, remaining: size, tail: 32 << 10, reached: make(chan struct{}), release: make(chan struct{})}

	form, err := formRequest(ctx, grant, gate, size)
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		status int
		err    error
	}

	done := make(chan result, 1)

	var release sync.Once

	joined := false
	client := *live.httpClient
	client.Timeout = 30 * time.Second

	go func() {
		response, err := client.Do(form)
		if err != nil {
			done <- result{err: err}

			return
		}

		_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))

		closeErr := response.Body.Close()
		done <- result{status: response.StatusCode, err: errors.Join(readErr, closeErr)}
	}()

	defer func() {
		cancel()
		release.Do(func() { close(gate.release) })

		if !joined {
			<-done
		}
	}()

	select {
	case <-gate.reached:
	case outcome := <-done:
		joined = true

		t.Fatalf("transfer finished before its stream pause: status=%d error=%v", outcome.status, outcome.err)
	case <-ctx.Done():
		t.Fatal("transfer did not reach its bounded stream pause")
	}

	if !time.Now().Before(grant.ExpiresAt) {
		t.Fatal("transfer did not reach the pause before policy expiry")
	}

	waitForExpiry(t, grant.ExpiresAt)
	release.Do(func() { close(gate.release) })

	outcome := <-done
	joined = true

	if outcome.err != nil || outcome.status < 200 || outcome.status >= 300 || time.Now().Before(grant.ExpiresAt) {
		t.Fatalf("in-flight completion boundary: status=%d error=%v", outcome.status, outcome.err)
	}

	reader, object, err := store.Open(context.Background(), request.Key, storage.ReadOptions{MaxBytes: size})
	if err != nil {
		t.Fatal(err)
	}

	hash := sha256.New()

	count, readErr := io.Copy(hash, reader)
	closeErr := reader.Close()
	want := sha256.Sum256(bytes.Repeat([]byte{'x'}, size))

	if readErr != nil || closeErr != nil || count != size || object.SizeBytes != size || !bytes.Equal(hash.Sum(nil), want[:]) {
		t.Fatal("completed in-flight object differs from the submitted bounded bytes")
	}
}
