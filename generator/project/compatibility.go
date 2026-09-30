// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package project

import "hatmax.adrianpk.com/generator/book"

// BookCompatibility describes whether an inventory can use a Book.
type BookCompatibility string

const (
	// BookCompatibilityUnknown means the project has no versioned Hatmax dependency.
	BookCompatibilityUnknown BookCompatibility = "unknown"
	// BookCompatibilityCompatible means the Book admits the project's Hatmax version.
	BookCompatibilityCompatible BookCompatibility = "compatible"
	// BookCompatibilityIncompatible means the Book excludes the project's Hatmax version.
	BookCompatibilityIncompatible BookCompatibility = "incompatible"
)

// CompatibilityWith checks the inventory's declared Hatmax version against a
// validated Book.
func (i Inventory) CompatibilityWith(selectedBook *book.Book) (BookCompatibility, error) {
	if selectedBook == nil {
		return BookCompatibilityUnknown, projectError("project_book_missing", "", "Book is required")
	}

	if i.Module.Hatmax.Version == "" {
		return BookCompatibilityUnknown, nil
	}

	supported, err := selectedBook.SupportsHatmax(i.Module.Hatmax.Version)
	if err != nil {
		return BookCompatibilityUnknown, projectError(
			"project_hatmax_version_invalid",
			"go.mod",
			"%v",
			err,
		)
	}

	if !supported {
		return BookCompatibilityIncompatible, nil
	}

	return BookCompatibilityCompatible, nil
}
