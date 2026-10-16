// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"

	"hatmax.adrianpk.com/storage"
	"hatmax.adrianpk.com/storage/internal/stream"
)

// Open returns metadata from the same GET as its owned response reader. Observed
// size is checked before returning, and actual bytes are bounded independently.
// Only successful EOF establishes a complete snapshot; callers must close even
// on read failure. The operation timeout remains active until Close.
func (s *Store) Open(ctx context.Context, key string, options storage.ReadOptions) (io.ReadCloser, storage.Object, error) {
	err := storage.ValidateKey(key)
	if err != nil {
		return nil, storage.Object{}, err
	}

	err = options.Validate()
	if err != nil {
		return nil, storage.Object{}, err
	}

	operation, cancel := context.WithTimeout(ctx, s.timeout)

	response, err := s.client.GetObject(operation, &sdk.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}, s.readOptions)
	if err != nil {
		cancel()

		return nil, storage.Object{}, readError(err)
	}

	object, err := metadata(key, response.ContentLength, response.ContentType)
	if err == nil && object.SizeBytes > options.MaxBytes {
		err = storage.ErrTooLarge
	}

	if response.Body == nil && err == nil {
		err = errors.New("S3 response has no object body")
	}

	if err != nil {
		cancel()

		if response.Body != nil {
			err = errors.Join(err, response.Body.Close())
		}

		return nil, storage.Object{}, err
	}

	return &objectReader{Reader: stream.NewReader(operation, response.Body, options.MaxBytes), body: response.Body, cancel: cancel}, object, nil
}

// Stat observes validated provider metadata without reserving a later read.
func (s *Store) Stat(ctx context.Context, key string) (storage.Object, error) {
	err := storage.ValidateKey(key)
	if err != nil {
		return storage.Object{}, err
	}

	operation, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	response, err := s.client.HeadObject(operation, &sdk.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}, s.readOptions)
	if err != nil {
		return storage.Object{}, readError(err)
	}

	return metadata(key, response.ContentLength, response.ContentType)
}

func (s *Store) readOptions(options *sdk.Options) {
	options.Retryer = retry.NewStandard(func(o *retry.StandardOptions) {
		o.MaxAttempts = s.readAttempts
		o.MaxBackoff = time.Second
	})
}

func metadata(key string, size *int64, contentType *string) (storage.Object, error) {
	if size == nil || *size < 0 || *size == math.MaxInt64 || contentType == nil || *contentType == "" {
		return storage.Object{}, errors.New("S3 response has invalid object metadata")
	}

	err := (storage.WriteOptions{MaxBytes: 1, ContentType: *contentType}).Validate()
	if err != nil {
		return storage.Object{}, fmt.Errorf("S3 response has invalid content type")
	}

	return storage.Object{Key: key, SizeBytes: *size, ContentType: *contentType}, nil
}

type objectReader struct {
	*stream.Reader
	body   io.ReadCloser
	cancel context.CancelFunc
}

func (r *objectReader) Close() error {
	r.cancel()

	return r.body.Close()
}
