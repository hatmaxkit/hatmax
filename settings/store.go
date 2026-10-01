// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package settings

import (
	"context"
	"errors"
)

// ErrNotFound identifies an absent setting, not a stored empty value or a read failure.
// Store adapters may wrap it; Service uses errors.Is to select registered defaults.
var ErrNotFound = errors.New("settings: setting not found")

// Store defines the persistence interface for settings.
type Store interface {
	// Get returns ErrNotFound only when key is absent. Present values, including
	// empty strings, return nil error. Propagate read and context errors instead
	// of reporting absence; translate backend-specific not-found errors here.
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	All(ctx context.Context) ([]Value, error)
	Delete(ctx context.Context, key string) error
}
