// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package storage

import (
	"context"
	"fmt"
	"time"
)

// UploadRequest admits one exact key and declared MIME type, an inclusive byte
// range, and an absolute expiry no more than 15 minutes after trusted signing time.
type UploadRequest struct {
	Key         string
	ContentType string
	MinBytes    int64
	MaxBytes    int64
	ExpiresAt   time.Time
}

// Validate checks the request against a trusted time sample without performing I/O.
// Signers must also validate at their actual signing time after retrieving credentials.
func (r UploadRequest) Validate(now time.Time) error {
	err := ValidateKey(r.Key)
	if err != nil {
		return err
	}

	err = (WriteOptions{ContentType: r.ContentType, MaxBytes: r.MaxBytes}).Validate()
	if err != nil {
		return err
	}

	if r.ContentType == "" || r.MinBytes < 0 || r.MinBytes > r.MaxBytes || !r.ExpiresAt.After(now) || r.ExpiresAt.Sub(now) > 15*time.Minute {
		return fmt.Errorf("upload bounds or expiry: %w", ErrInvalid)
	}

	return nil
}

// UploadGrant is a sensitive bearer capability for multipart POST. Copy Fields
// into form data followed by one file part. Do not log or persist grant fields.
// ExpiresAt is the actual policy expiry, possibly shortened by credential expiry
// and provider timestamp precision. A grant may be reused until expiry; it does
// not revoke in-flight uploads or perform application authorization or completion.
type UploadGrant struct {
	URL       string
	Fields    map[string]string
	ExpiresAt time.Time
}

// UploadSigner is an optional capability, independent of Store. Local stores do
// not implement it. Applications admit requests before signing, use disposable
// staging keys, and validate one complete bounded snapshot before final storage.
type UploadSigner interface {
	SignUpload(context.Context, UploadRequest) (UploadGrant, error)
}
