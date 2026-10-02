// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Package ingest owns bounded, context-aware reads for image adapters.
package ingest

import (
	"context"
	"fmt"
	"io"
)

// MaxBytes bounds encoded image buffering and configuration reads to 20 MiB.
const MaxBytes = 20 << 20

type reader struct {
	ctx context.Context
	r   io.Reader
}

// NewReader checks cancellation before and after each underlying read. It does
// not take ownership of r or interrupt an already-blocked arbitrary Read call.
func NewReader(ctx context.Context, r io.Reader) io.Reader {
	return &reader{ctx: ctx, r: r}
}

func (r *reader) Read(p []byte) (int, error) {
	err := r.ctx.Err()
	if err != nil {
		return 0, err
	}

	n, err := r.r.Read(p)

	cause := r.ctx.Err()
	if cause != nil {
		return n, cause
	}

	return n, err
}

// ReadAll reads at most MaxBytes plus one byte to detect overflow without
// draining the rest of an oversized input. Errors never return partial data.
func ReadAll(ctx context.Context, input io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(NewReader(ctx, input), MaxBytes+1))
	if err != nil {
		return nil, err
	}

	if len(data) > MaxBytes {
		return nil, fmt.Errorf("image input exceeds %d bytes", MaxBytes)
	}

	return data, nil
}
