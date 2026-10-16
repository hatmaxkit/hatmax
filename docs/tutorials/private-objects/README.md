<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Store a Private Document

This tutorial writes a document, inspects it, reads all its bytes and deletes it
using `storage/local`. Run from a repository checkout with Go 1.27.1. The example
does not create an image record or generate a public URL.

## Create an owned directory

The store opens an existing directory. The application must control the directory,
its initial ancestors and its contents, including hard links and mounts.
Create a disposable directory for this exercise:

```sh
object_root=$(mktemp -d)
trap 'rm -rf -- "$object_root"' EXIT
GOWORK=off go run ./examples/object-storage/local "$object_root"
```

Expected output is `local object workflow complete`. The command verifies the
returned bytes and confirmed absence after deletion. It leaves no object or
spool file; the shell removes the owned root when it exits.

## Follow the lifecycle

This is the exact [executable](../../../examples/object-storage/local/main.go).
Each read and write admits at most 1 MiB; the application may choose a lower
limit. `Stat` is used for inspection, and `Open` supplies metadata for its own
reader. Successful `io.ReadAll` reaches EOF; closing the reader remains required
on both success and failure.

<!-- object-storage-source: examples/object-storage/local/main.go -->
```go
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
```

The local adapter derives content type from at most the first 512 file bytes.
The write's declared `text/plain` value is validated but not persisted. It does
not certify document safety. The successful read verifies actual bytes, not
just metadata or the filename suffix.

## Continue with remote storage

Use [private S3 configuration](../../how-to/store-private-objects/README.md)
and [signed uploads](../../how-to/sign-object-uploads/README.md) for the remote
workflow. The [reference](../../reference/storage/README.md) lists exact bounds
and error predicates. [Snapshot and bearer ownership](../../explanation/private-object-ownership/README.md)
explains why complete bytes and application completion have separate owners.
