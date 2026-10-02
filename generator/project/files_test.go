// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package project

import (
	"context"
	"testing"
)

func TestInventoryFilesAreDetachedAndAddressable(t *testing.T) {
	inventory, err := Inspect(context.Background(), copyFixture(t, "supported"))
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	model, exists := inventory.File("internal/feat/property/model.go")
	if !exists || model.Path != "internal/feat/property/model.go" || !containsValue(model.Surfaces, "model") {
		t.Fatalf("File(model.go) = %#v, %v", model, exists)
	}

	files := inventory.Files()
	if len(files) == 0 {
		t.Fatal("Files() returned no inspected files")
	}

	files[0].Path = "changed"
	if inventory.Files()[0].Path == "changed" {
		t.Error("Files() exposed inventory storage")
	}
}
