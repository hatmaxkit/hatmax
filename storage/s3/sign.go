// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"

	"hatmax.adrianpk.com/storage"
)

// PostPolicyProfile selects an explicitly supported POST enforcement contract.
// It is a deployment assertion, not server discovery or a generic provider registry.
type PostPolicyProfile uint8

const (
	// PostPolicyDisabled leaves signing unavailable while retaining private I/O.
	PostPolicyDisabled PostPolicyProfile = iota
	// PostPolicyMinIO selects the unversioned MinIO RELEASE.2025-10-15T17-29-55Z
	// profile verified by real policy-enforcement tests. Configure its actual
	// endpoint explicitly. Other implementations are not enabled by inference.
	PostPolicyMinIO
)

// SignUpload returns an exact-target SigV4 POST with enforced type, size and
// expiry conditions. The configured deployment must match PostPolicyMinIO;
// disabled or unknown capabilities return ErrUnsupported, without weaker signing.
// This method does not create objects, authorize callers, or own completion.
func (s *Store) SignUpload(ctx context.Context, request storage.UploadRequest) (storage.UploadGrant, error) {
	now := time.Now().UTC()

	err := request.Validate(now)
	if err != nil {
		return storage.UploadGrant{}, err
	}

	if s.postPolicy != PostPolicyMinIO {
		return storage.UploadGrant{}, storage.ErrUnsupported
	}

	operation, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	err = operation.Err()
	if err != nil {
		return storage.UploadGrant{}, err
	}

	bound := &boundPostSigner{ctx: operation, bucket: s.bucket, request: request}
	conditions := []any{
		map[string]string{"key": request.Key},
		map[string]string{"Content-Type": request.ContentType},
		[]any{"content-length-range", request.MinBytes, request.MaxBytes},
	}

	post, err := s.presigner.PresignPostObject(operation, &sdk.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(request.Key), ContentType: aws.String(request.ContentType)}, func(o *sdk.PresignPostOptions) {
		// Never allow the SDK's zero-duration default. The bound signer below uses
		// the absolute requested deadline, not this duration added to a later clock.
		o.Expires = request.ExpiresAt.Sub(now)
		o.Conditions = conditions
		bound.delegate = o.PostPresigner
		o.PostPresigner = bound
	})
	if err != nil {
		return storage.UploadGrant{}, err
	}

	err = operation.Err()
	if err != nil {
		return storage.UploadGrant{}, err
	}

	if post == nil || post.URL == "" || bound.expiresAt.IsZero() {
		return storage.UploadGrant{}, fmt.Errorf("S3 POST policy binding: %w", storage.ErrUnsupported)
	}

	post.Values["Content-Type"] = request.ContentType

	return storage.UploadGrant{URL: post.URL, Fields: post.Values, ExpiresAt: bound.expiresAt}, nil
}

type boundPostSigner struct {
	ctx       context.Context
	bucket    string
	request   storage.UploadRequest
	delegate  sdk.PresignPost
	expiresAt time.Time
}

func (b *boundPostSigner) PresignPost(credentials aws.Credentials, bucket, key, region, service string, signingTime time.Time, conditions []any, _ time.Time, options ...func(*v4.SignerOptions)) (map[string]string, error) {
	err := b.ctx.Err()
	if err != nil {
		return nil, err
	}

	if b.delegate == nil || bucket != b.bucket || key != b.request.Key || service != "s3" || region == "" {
		return nil, storage.ErrUnsupported
	}

	expiration, err := uploadExpiration(b.request, credentials, signingTime)
	if err != nil {
		return nil, err
	}

	// Delegate policy construction and SigV4 to the SDK using its single actual
	// post-credential signing sample and our absolute, conservatively rounded expiry.
	fields, err := b.delegate.PresignPost(credentials, bucket, key, region, service, signingTime, conditions, expiration, options...)
	if err != nil {
		return nil, err
	}

	actual, err := policyExpiration(fields["policy"])
	if err != nil || !actual.Equal(expiration) || fields["X-Amz-Date"] != signingTime.UTC().Format("20060102T150405Z") {
		return nil, fmt.Errorf("S3 POST policy timestamp mismatch: %w", storage.ErrUnsupported)
	}

	b.expiresAt = actual

	return fields, nil
}

func uploadExpiration(request storage.UploadRequest, credentials aws.Credentials, signingTime time.Time) (time.Time, error) {
	err := request.Validate(signingTime)
	if err != nil {
		return time.Time{}, err
	}

	// Bound provider-owned signing material before the SDK creates its policy.
	if credentials.AccessKeyID == "" || credentials.SecretAccessKey == "" || len(credentials.AccessKeyID) > 256 || len(credentials.SecretAccessKey) > 256 || len(credentials.SessionToken) > 64<<10 {
		return time.Time{}, fmt.Errorf("S3 upload credentials: %w", storage.ErrInvalid)
	}

	expiration := request.ExpiresAt
	if credentials.CanExpire && credentials.Expires.Before(expiration) {
		expiration = credentials.Expires
	}

	// SDK policy serialization has second precision. Round down, never extend a
	// requested or credential deadline, and reject windows rounded into the past.
	expiration = expiration.UTC().Truncate(time.Second)
	if !expiration.After(signingTime) {
		return time.Time{}, fmt.Errorf("S3 upload expiry has no effective lifetime: %w", storage.ErrInvalid)
	}

	return expiration, nil
}

func policyExpiration(encoded string) (time.Time, error) {
	if len(encoded) == 0 || len(encoded) > 128<<10 {
		return time.Time{}, storage.ErrUnsupported
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return time.Time{}, storage.ErrUnsupported
	}

	var policy struct {
		Expiration time.Time `json:"expiration"`
	}

	err = json.Unmarshal(data, &policy)
	if err != nil || policy.Expiration.IsZero() {
		return time.Time{}, storage.ErrUnsupported
	}

	return policy.Expiration, nil
}

var _ storage.UploadSigner = (*Store)(nil)
