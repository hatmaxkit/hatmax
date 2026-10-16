// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package storage

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// TestKeys binds the portable contract without silently repairing target names.
func TestKeys(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		valid bool
	}{
		{"simple", "document", true},
		{"nested", "records/2026/report_1-v2.pdf", true},
		{"hidden", ".private/...", true},
		{"maximum", strings.Repeat("a", 512), true},
		{"empty", "", false},
		{"long", strings.Repeat("a", 513), false},
		{"absolute", "/file", false},
		{"empty segment", "a//b", false},
		{"trailing slash", "a/", false},
		{"dot", "a/./b", false},
		{"parent", "a/../b", false},
		{"backslash", `a\b`, false},
		{"encoded", "a/%2e%2e/b", false},
		{"unicode", "caf\u00e9", false},
		{"nul", "file\x00", false},
		{"space", "a b", false},
		{"drive", "C:/file", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateKey(tc.key)
			if tc.valid && err != nil || !tc.valid && !errors.Is(err, ErrInvalid) {
				t.Fatalf("valid=%t, error=%v", tc.valid, err)
			}
		})
	}
}

// TestBounds rejects unsafe arithmetic and malformed metadata before adapter I/O.
func TestBounds(t *testing.T) {
	for _, maximum := range []int64{-1, 0, 1, math.MaxInt64 - 1, math.MaxInt64} {
		valid := maximum > 0 && maximum < math.MaxInt64

		for _, err := range []error{(ReadOptions{maximum}).Validate(), (WriteOptions{MaxBytes: maximum}).Validate()} {
			if valid && err != nil || !valid && !errors.Is(err, ErrInvalid) {
				t.Fatalf("maximum=%d, error=%v", maximum, err)
			}
		}
	}

	cases := []struct {
		name  string
		value string
		valid bool
	}{
		{"omitted", "", true},
		{"text", "text/plain; charset=utf-8", true},
		{"opaque", "application/octet-stream", true},
		{"missing subtype", "text", false},
		{"broken", ";broken", false},
		{"injected", "text/plain\r\nother: value", false},
		{"long", "text/" + strings.Repeat("x", 252), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := (WriteOptions{ContentType: tc.value, MaxBytes: 1}).Validate()
			if tc.valid && err != nil || !tc.valid && !errors.Is(err, ErrInvalid) {
				t.Fatalf("valid=%t, error=%v", tc.valid, err)
			}
		})
	}
}
