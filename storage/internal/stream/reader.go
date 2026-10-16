// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Package stream enforces actual byte and cancellation limits for storage adapters.
package stream

import (
	"context"
	"io"

	"hatmax.adrianpk.com/storage"
)

// Reader checks cancellation around synchronous reads. It never closes the input
// or starts a goroutine to interrupt an arbitrary blocking caller-owned reader.
type Reader struct {
	ctx       context.Context
	input     io.Reader
	remaining int64
	empty     int
	err       error
}

// NewReader requires an already validated positive maximum.
func NewReader(ctx context.Context, input io.Reader, maximum int64) *Reader {
	return &Reader{ctx: ctx, input: input, remaining: maximum}
}

// Read delivers at most the maximum. An extra byte is consumed only to prove
// overflow, which returns ErrTooLarge instead of a misleading successful EOF.
func (r *Reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.err != nil {
		return 0, r.err
	}

	r.err = r.ctx.Err()
	if r.err != nil {
		return 0, r.err
	}

	probe := r.remaining == 0
	if probe {
		p = p[:1]
	} else if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}

	n, err := r.input.Read(p)
	if n < 0 || n > len(p) {
		r.err = storage.ErrInvalid

		return 0, r.err
	}

	r.err = r.ctx.Err()
	if r.err != nil {
		return 0, r.err
	}

	if probe && n > 0 {
		r.err = storage.ErrTooLarge

		return 0, r.err
	}

	r.remaining -= int64(n)
	r.err = err

	if n == 0 && err == nil {
		r.empty++
		if r.empty >= 100 {
			r.err = io.ErrNoProgress
		}
	} else {
		r.empty = 0
	}

	return n, r.err
}
