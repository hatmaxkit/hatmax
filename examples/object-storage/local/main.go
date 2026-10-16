// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Command local demonstrates a complete bounded private document lifecycle.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"hatmax.adrianpk.com/storage"
	"hatmax.adrianpk.com/storage/local"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: local OWNED_DIRECTORY")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := run(ctx, os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "local object workflow failed")
		os.Exit(1)
	}

	fmt.Println("local object workflow complete")
}

func run(ctx context.Context, directory string) (result error) {
	const maximum = 1 << 20

	store, err := local.NewStore(directory, local.Options{MaxSpoolBytes: maximum})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, store.Close()) }()

	key := "guide/document.txt"
	data := []byte("Approved document\nPrivate bytes without image metadata.\n")

	_, err = store.Put(ctx, key, bytes.NewReader(data), storage.WriteOptions{ContentType: "text/plain", MaxBytes: maximum})
	if err != nil {
		return err
	}

	observed, err := store.Stat(ctx, key)
	if err != nil {
		return err
	}

	reader, opened, err := store.Open(ctx, key, storage.ReadOptions{MaxBytes: maximum})
	if err != nil {
		return err
	}

	snapshot, readErr := io.ReadAll(reader)
	err = errors.Join(readErr, reader.Close())
	if err != nil {
		return err
	}

	if opened.Key != key || observed.SizeBytes != int64(len(data)) || opened.SizeBytes != int64(len(snapshot)) || !bytes.Equal(snapshot, data) {
		return errors.New("document readback differs")
	}

	err = store.Delete(ctx, key)
	if err != nil {
		return err
	}

	_, err = store.Stat(ctx, key)
	if !errors.Is(err, storage.ErrNotFound) {
		return errors.New("document absence was not confirmed")
	}

	return nil
}
