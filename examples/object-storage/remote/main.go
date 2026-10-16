// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Command remote demonstrates signed staging and bounded private completion.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"hatmax.adrianpk.com/storage"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := run(ctx)
	if err != nil {
		// Grant fields, credentials and provider replies are deliberately not logged.
		fmt.Fprintln(os.Stderr, "remote object workflow failed")
		os.Exit(1)
	}

	fmt.Println("remote object workflow complete")
}

func run(ctx context.Context) (result error) {
	store, err := newStore(ctx)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, store.Close()) }()
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result = errors.Join(result, store.Delete(cleanup, stagingKey), store.Delete(cleanup, finalKey))
	}()

	grant, err := store.SignUpload(ctx, storage.UploadRequest{
		Key: stagingKey, ContentType: "text/plain", MinBytes: 1,
		MaxBytes: maximum, ExpiresAt: time.Now().Add(2 * time.Minute),
	})
	if err != nil {
		return err
	}

	err = submit(ctx, grant)
	if err != nil {
		return err
	}

	return complete(ctx, store)
}
