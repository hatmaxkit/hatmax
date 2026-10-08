// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package project

import (
	"slices"
	"testing"
)

// Delegated composition remains separate from binary entrypoints, excludes
// test-only setup calls and exposes every ambiguous production root.
func TestCompositionRoots(t *testing.T) {
	for _, tc := range []struct {
		name   string
		direct bool
		want   []string
	}{
		{"delegated", false, []string{"internal/application/application.go"}},
		{"ambiguous", true, []string{"internal/application/application.go", "main.go"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := copyFixture(t, "supported")
			if !tc.direct {
				writeProjectFile(t, root, "main.go", "package main\nfunc main() {}\n")
			}

			composition := "package application\nimport \"hatmax.adrianpk.com/app\"\nfunc run() { app.Setup() }\n"
			writeProjectFile(t, root, "internal/application/application.go", composition)
			writeProjectFile(t, root, "internal/application/application_test.go", composition)

			inventory, err := Inspect(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}

			roots := inventory.CompositionRoots()
			if !slices.Equal(roots, tc.want) || len(inventory.Entrypoints) != 1 {
				t.Fatalf("roots=%v entrypoints=%v", roots, inventory.Entrypoints)
			}

			roots[0] = "changed"

			if !slices.Equal(inventory.CompositionRoots(), tc.want) {
				t.Fatal("composition root snapshot was mutable")
			}
		})
	}
}

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
