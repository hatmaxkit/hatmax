// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package modal

// Config holds configuration for a modal dialog.
type Config struct {
	ID           string
	Title        string
	Size         Size
	CloseOnEsc   bool
	CloseOnClick bool
}

// Size represents the modal width.
type Size string

const (
	SizeSmall  Size = "modal-sm"
	SizeMedium Size = "modal-md"
	SizeLarge  Size = "modal-lg"
)

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(id, title string) Config {
	return Config{
		ID:           id,
		Title:        title,
		Size:         SizeMedium,
		CloseOnEsc:   true,
		CloseOnClick: true,
	}
}
