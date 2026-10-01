// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package ingest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// Accept the byte boundary exactly, detect one byte beyond it, and never drain
// oversized input. The source generates bytes without allocating an input file.
func TestReadBudget(t *testing.T) {
	for _, tc := range []struct {
		name string
		size int64
	}{
		{name: "empty"},
		{name: "small", size: 10},
		{name: "exact limit", size: MaxBytes},
		{name: "over limit", size: MaxBytes + 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := &generatedInput{remaining: tc.size}

			data, err := ReadAll(context.Background(), input)
			if tc.size > MaxBytes {
				if err == nil || !strings.Contains(err.Error(), "exceeds") || data != nil || input.read != MaxBytes+1 {
					t.Fatalf("overflow: %v, %d bytes returned, %d read", err, len(data), input.read)
				}

				return
			}

			if err != nil || int64(len(data)) != tc.size || input.read != tc.size {
				t.Fatalf("ReadAll: %v, %d bytes returned, %d read", err, len(data), input.read)
			}
		})
	}
}

type generatedInput struct {
	remaining int64
	read      int64
}

func (r *generatedInput) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}

	n := min(int64(len(p)), r.remaining)
	clear(p[:n])
	r.remaining -= n
	r.read += n

	return int(n), nil
}

// Context error identity must survive reads, and input I/O failures must not expose
// partially buffered data as a usable upload.
func TestReadFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	data, err := ReadAll(ctx, bytes.NewReader([]byte("unused")))
	if !errors.Is(err, context.Canceled) || data != nil {
		t.Fatalf("cancellation: %v, %v", data, err)
	}

	ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancel()

	data, err = ReadAll(ctx, bytes.NewReader([]byte("unused")))
	if !errors.Is(err, context.DeadlineExceeded) || data != nil {
		t.Fatalf("deadline: %v, %v", data, err)
	}

	data, err = ReadAll(context.Background(), failedInput{})
	if !errors.Is(err, io.ErrUnexpectedEOF) || data != nil {
		t.Fatalf("read failure: %v, %v", data, err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	data, err = ReadAll(ctx, cancelInput{cancel: cancel})
	if !errors.Is(err, context.Canceled) || data != nil {
		t.Fatalf("canceled read: %v, %v", data, err)
	}
}

type cancelInput struct{ cancel context.CancelFunc }

func (r cancelInput) Read(p []byte) (int, error) {
	r.cancel()

	return copy(p, "partial"), nil
}

type failedInput struct{}

func (failedInput) Read(p []byte) (int, error) {
	return copy(p, "partial"), io.ErrUnexpectedEOF
}
