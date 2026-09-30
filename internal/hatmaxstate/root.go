// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

// Package hatmaxstate persists bounded Hatmax-owned conversation snapshots in
// the operating system's per-user state location.
package hatmaxstate

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolveRoot resolves the platform state root from explicit environment
// values. It performs no filesystem writes.
func ResolveRoot(goos, home, localAppData, xdgStateHome string) (string, error) {
	switch goos {
	case "darwin":
		if strings.TrimSpace(home) == "" {
			return "", stateError("state_root_unavailable", "HOME", "home directory is unavailable")
		}

		return filepath.Join(home, "Library", "Application Support", "Hatmax", "State"), nil
	case "windows":
		if strings.TrimSpace(localAppData) == "" {
			return "", stateError("state_root_unavailable", "LOCALAPPDATA", "local application data directory is unavailable")
		}

		return filepath.Join(localAppData, "Hatmax", "State"), nil
	default:
		if filepath.IsAbs(xdgStateHome) {
			return filepath.Join(xdgStateHome, "hatmax"), nil
		}

		if strings.TrimSpace(home) == "" {
			return "", stateError("state_root_unavailable", "HOME", "home directory is unavailable")
		}

		return filepath.Join(home, ".local", "state", "hatmax"), nil
	}
}

// DefaultRoot resolves the current process state root.
func DefaultRoot() (string, error) {
	home, _ := userHomeDir()

	return ResolveRoot(
		runtime.GOOS,
		home,
		lookupEnvironment("LOCALAPPDATA"),
		lookupEnvironment("XDG_STATE_HOME"),
	)
}

var (
	userHomeDir       = defaultUserHomeDir
	lookupEnvironment = defaultLookupEnvironment
)

func defaultUserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func defaultLookupEnvironment(name string) string {
	return os.Getenv(name)
}

// Error is one stable local-state failure.
type Error struct {
	Code    string
	Field   string
	Message string
}

func (e Error) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	return fmt.Sprintf("%s at %s: %s", e.Code, e.Field, e.Message)
}

func stateError(code, field, format string, arguments ...any) error {
	return Error{Code: code, Field: field, Message: fmt.Sprintf(format, arguments...)}
}
