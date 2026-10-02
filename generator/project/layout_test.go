// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package project

import "testing"

func TestClassifyCanonicalPersistenceSurfaces(t *testing.T) {
	tests := []struct {
		path    string
		surface string
	}{
		{path: "internal/feat/invoice/postgres_store.go", surface: "store"},
		{path: "db/queries/invoice.sql", surface: "store"},
		{path: "assets/migration/postgres/001-invoice.sql", surface: "migration"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			if !containsValue(classifySurfaces(test.path), test.surface) {
				t.Errorf("classifySurfaces(%q) does not contain %q", test.path, test.surface)
			}
		})
	}
}
