//go:build windows

package hatmaxstate

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

type fileLock struct {
	file       *os.File
	overlapped windows.Overlapped
}

func acquireFileLock(path string) (*fileLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, stateError("state_persistence_failed", path, "open scope lock: %v", err)
	}

	lock := &fileLock{file: file}
	err = windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		&lock.overlapped,
	)
	if err != nil {
		_ = file.Close()

		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, stateError("state_busy", "scope", "another process owns the mutable scope session")
		}

		return nil, stateError("state_persistence_failed", path, "acquire scope lock: %v", err)
	}

	return lock, nil
}

func (l *fileLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}

	err := windows.UnlockFileEx(windows.Handle(l.file.Fd()), 0, 1, 0, &l.overlapped)
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
