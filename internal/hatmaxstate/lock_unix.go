// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package hatmaxstate

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

type fileLock struct {
	file *os.File
}

func acquireFileLock(path string) (*fileLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, stateError("state_persistence_failed", path, "open scope lock: %v", err)
	}

	err = file.Chmod(0o600)
	if err != nil {
		_ = file.Close()

		return nil, stateError("state_persistence_failed", path, "secure scope lock: %v", err)
	}

	err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		_ = file.Close()

		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return nil, stateError("state_busy", "scope", "another process owns the mutable scope session")
		}

		return nil, stateError("state_persistence_failed", path, "acquire scope lock: %v", err)
	}

	return &fileLock{file: file}, nil
}

func (l *fileLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}

	err := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil

	if err != nil {
		return stateError("state_persistence_failed", "lock", "release scope lock: %v", err)
	}

	if closeErr != nil {
		return stateError("state_persistence_failed", "lock", "close scope lock: %v", closeErr)
	}

	return nil
}
