// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"

	"hatmax.adrianpk.com/storage"
	"hatmax.adrianpk.com/storage/internal/stream"
)

// Put validates and completely spools bounded input before one remote mutation.
// It does not close the caller's input or allocate MaxBytes in memory. Supplied
// content type is stored as metadata; omitted type is derived from 512 bytes.
// Lost acknowledgments preserve their cause and classify as ErrIndeterminate.
func (s *Store) Put(ctx context.Context, key string, input io.Reader, options storage.WriteOptions) (object storage.Object, result error) {
	err := storage.ValidateKey(key)
	if err != nil {
		return storage.Object{}, err
	}

	err = options.Validate()
	if err != nil {
		return storage.Object{}, err
	}

	if input == nil || options.MaxBytes > s.capacity {
		return storage.Object{}, storage.ErrInvalid
	}

	operation, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	err = operation.Err()
	if err != nil {
		return storage.Object{}, err
	}

	name := ".object-" + rand.Text()

	file, err := s.spool.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return storage.Object{}, err
	}

	defer func() {
		cleanup := errors.Join(file.Close(), s.spool.Remove(name))
		if cleanup != nil {
			object = storage.Object{}
			result = errors.Join(result, cleanup)
		}
	}()

	size, err := io.CopyBuffer(struct{ io.Writer }{file}, stream.NewReader(operation, input, options.MaxBytes), make([]byte, 32<<10))
	if err != nil {
		return storage.Object{}, err
	}

	contentType := options.ContentType
	if contentType == "" {
		var head [512]byte

		n, readErr := file.ReadAt(head[:], 0)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return storage.Object{}, readErr
		}

		contentType = http.DetectContentType(head[:n])
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return storage.Object{}, err
	}

	err = operation.Err()
	if err != nil {
		return storage.Object{}, err
	}

	_, err = s.client.PutObject(operation, &sdk.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), Body: file, ContentLength: aws.Int64(size), ContentType: aws.String(contentType)})
	if err != nil {
		return storage.Object{}, mutationError(err)
	}

	return storage.Object{Key: key, SizeBytes: size, ContentType: contentType}, nil
}

// Delete performs one request. Confirmed absence succeeds. Permission failures
// remain errors, and lost confirmations are indeterminate rather than absence.
func (s *Store) Delete(ctx context.Context, key string) error {
	err := storage.ValidateKey(key)
	if err != nil {
		return err
	}

	operation, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	err = operation.Err()
	if err != nil {
		return err
	}

	_, err = s.client.DeleteObject(operation, &sdk.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err == nil || absent(err) {
		return nil
	}

	return mutationError(err)
}
