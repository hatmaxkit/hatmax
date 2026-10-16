// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package local

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"os"
	"path"

	"hatmax.adrianpk.com/storage"
	"hatmax.adrianpk.com/storage/internal/stream"
)

func put(ctx context.Context, parent *os.Root, key string, input io.Reader, maximum int64) (object storage.Object, result error) {
	name := ".object-" + rand.Text()

	file, err := parent.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return storage.Object{}, err
	}

	closed := false
	committed := false

	defer func() {
		if !closed {
			result = errors.Join(result, file.Close())
		}

		if !committed {
			result = errors.Join(result, parent.Remove(name))
		}
	}()

	_, err = io.CopyBuffer(file, stream.NewReader(ctx, input, maximum), make([]byte, 32<<10))
	if err != nil {
		return storage.Object{}, err
	}

	object, err = describe(ctx, file, key, maximum)
	if err != nil {
		return storage.Object{}, err
	}

	err = file.Close()
	closed = true

	if err != nil {
		return storage.Object{}, err
	}

	err = ctx.Err()
	if err != nil {
		return storage.Object{}, err
	}

	err = parent.Rename(name, path.Base(key))
	if err != nil {
		// A filesystem (including a remote mount) can lose mutation confirmation.
		return storage.Object{}, errors.Join(storage.ErrIndeterminate, err)
	}

	committed = true

	return object, nil
}

func describe(ctx context.Context, file *os.File, key string, maximum int64) (storage.Object, error) {
	err := ctx.Err()
	if err != nil {
		return storage.Object{}, err
	}

	info, err := file.Stat()
	if err != nil {
		return storage.Object{}, err
	}

	if !info.Mode().IsRegular() || info.Size() < 0 {
		return storage.Object{}, storage.ErrInvalid
	}

	if info.Size() > maximum {
		return storage.Object{}, storage.ErrTooLarge
	}

	var head [512]byte

	n, err := file.ReadAt(head[:], 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return storage.Object{}, err
	}

	err = ctx.Err()
	if err != nil {
		return storage.Object{}, err
	}

	return storage.Object{Key: key, SizeBytes: info.Size(), ContentType: http.DetectContentType(head[:n])}, nil
}
