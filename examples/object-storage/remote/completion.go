// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"bytes"
	"context"
	"errors"
	"io"

	"hatmax.adrianpk.com/storage"
)

func complete(ctx context.Context, store storage.Store) error {
	reader, opened, err := store.Open(ctx, stagingKey, storage.ReadOptions{MaxBytes: maximum})
	if err != nil {
		return err
	}

	// Only successful EOF establishes a complete bounded snapshot.
	snapshot, readErr := io.ReadAll(reader)
	err = errors.Join(readErr, reader.Close())
	if err != nil {
		return err
	}

	expected := []byte("Approved document\nPrivate bytes without image metadata.\n")
	if opened.ContentType != "text/plain" || opened.SizeBytes != int64(len(snapshot)) || !bytes.Equal(snapshot, expected) {
		return errors.New("document validation failed")
	}

	// The application rechecks current authority before this step in production.
	_, err = store.Put(ctx, finalKey, bytes.NewReader(snapshot), storage.WriteOptions{ContentType: "text/plain", MaxBytes: maximum})
	if err != nil {
		return err
	}

	final, _, err := store.Open(ctx, finalKey, storage.ReadOptions{MaxBytes: maximum})
	if err != nil {
		return err
	}

	accepted, readErr := io.ReadAll(final)
	err = errors.Join(readErr, final.Close())
	if err != nil {
		return err
	}

	if !bytes.Equal(accepted, snapshot) {
		return errors.New("final readback differs from validated snapshot")
	}

	return nil
}
