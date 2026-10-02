// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package project

import "path/filepath"

// Files returns a detached, path-ordered copy of inspected file metadata.
func (i Inventory) Files() []File {
	result := make([]File, 0, len(i.files))
	for _, file := range i.files {
		result = append(result, File{
			Path:      file.path,
			Size:      file.size,
			Generated: file.generated,
			Surfaces:  cloneStrings(file.surfaces),
		})
	}

	return result
}

// File returns detached metadata for one clean project-relative path.
func (i Inventory) File(path string) (File, bool) {
	normalized := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	for _, file := range i.files {
		if file.path != normalized {
			continue
		}

		return File{
			Path:      file.path,
			Size:      file.size,
			Generated: file.generated,
			Surfaces:  cloneStrings(file.surfaces),
		}, true
	}

	return File{}, false
}
