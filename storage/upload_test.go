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
	"time"
)

// TestUploadBounds admits only canonical exact targets and finite upload windows.
func TestUploadBounds(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 500000000, time.UTC)
	cases := []struct {
		name   string
		change func(*UploadRequest)
		valid  bool
	}{
		{"valid", func(*UploadRequest) {}, true},
		{"empty allowed", func(r *UploadRequest) { r.MinBytes = 0 }, true},
		{"maximum lifetime", func(r *UploadRequest) { r.ExpiresAt = now.Add(15 * time.Minute) }, true},
		{"key traversal", func(r *UploadRequest) { r.Key = "a/../target" }, false},
		{"wildcard key", func(r *UploadRequest) { r.Key = "uploads/*" }, false},
		{"omitted type", func(r *UploadRequest) { r.ContentType = "" }, false},
		{"broken type", func(r *UploadRequest) { r.ContentType = "broken" }, false},
		{"long type", func(r *UploadRequest) { r.ContentType = "text/" + strings.Repeat("x", 252) }, false},
		{"negative minimum", func(r *UploadRequest) { r.MinBytes = -1 }, false},
		{"inverted range", func(r *UploadRequest) { r.MinBytes = 9 }, false},
		{"zero maximum", func(r *UploadRequest) { r.MaxBytes = 0 }, false},
		{"unsafe maximum", func(r *UploadRequest) { r.MaxBytes = math.MaxInt64 }, false},
		{"zero expiry", func(r *UploadRequest) { r.ExpiresAt = time.Time{} }, false},
		{"current expiry", func(r *UploadRequest) { r.ExpiresAt = now }, false},
		{"past expiry", func(r *UploadRequest) { r.ExpiresAt = now.Add(-time.Nanosecond) }, false},
		{"excess lifetime", func(r *UploadRequest) { r.ExpiresAt = now.Add(15*time.Minute + time.Nanosecond) }, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := UploadRequest{Key: "staging/document", ContentType: "text/plain", MinBytes: 1, MaxBytes: 8, ExpiresAt: now.Add(time.Minute)}
			tc.change(&request)

			err := request.Validate(now)
			if tc.valid && err != nil || !tc.valid && !errors.Is(err, ErrInvalid) {
				t.Fatalf("valid=%t error=%v", tc.valid, err)
			}
		})
	}
}
