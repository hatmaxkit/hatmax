// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

//go:build acceptance

package execute

import (
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

// SQLC must keep table and query row types compatible when ALTER TABLE appends
// a required timestamp after the original identity and audit columns.
func TestRequiredTimestamp(t *testing.T) {
	root := nativeFeatureProject(t)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddField, intent.Domain{
		Field: &intent.Field{Name: "issued_at", Type: "timestamp", Required: true},
	}, []string{"postgres_persistence", "runtime_validation"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatal(err)
	}

	mutations, err := RenderAddField(value, manifest, inventory)
	if err != nil {
		t.Fatal(err)
	}

	workspace, err := OpenWorkspace(t.Context(), manifest, value, inventory)
	if err != nil {
		t.Fatal(err)
	}

	for _, mutation := range mutations {
		_, err = workspace.Stage(mutation)
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = workspace.Commit(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	writeExecutionFile(t, root, "internal/feat/invoice/timestamp_test.go", `package invoice

import (
	"context"
	"testing"
	"time"

	"hatmax.adrianpk.com/testhelper"
)

// Required timestamps survive both SQLC query paths and subsequent updates.
func TestStoredTimestamp(t *testing.T) {
	database, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	_, err := database.Exec("CREATE TABLE invoices (id TEXT PRIMARY KEY, number TEXT NOT NULL, notes TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, issued_at TIMESTAMPTZ NOT NULL)")
	if err != nil {
		t.Fatal(err)
	}

	store := NewPostgresStore(testDBProvider{database: database})

	err = store.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	stamp := time.Unix(1700000000, 123000000).UTC()

	value, err := NewInvoice(InvoiceInput{Number: "INV-001", Notes: "Example", IssuedAt: stamp})
	if err != nil {
		t.Fatal(err)
	}

	err = store.Create(context.Background(), value)
	if err != nil {
		t.Fatal(err)
	}

	for attempt := range 2 {
		loaded, err := store.Get(context.Background(), value.ID)
		if err != nil || !loaded.IssuedAt.Equal(stamp) {
			t.Fatalf("Get timestamp: %#v, %v", loaded, err)
		}

		values, err := store.List(context.Background())
		if err != nil || len(values) != 1 || !values[0].IssuedAt.Equal(stamp) {
			t.Fatalf("List timestamp: %#v, %v", values, err)
		}

		if attempt == 0 {
			stamp = stamp.Add(time.Hour)
			value.IssuedAt = stamp

			err = store.Update(context.Background(), value)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
`)

	publishedCommand(t, root, "make", "generate")
	publishedCommand(t, root, "make", "format")
	publishedCommand(t, root, "make", "check")
}

func nativeFeatureProject(t *testing.T) string {
	t.Helper()
	t.Setenv("GOLANGCI_LINT_CACHE", t.TempDir())

	root := generatedInvoiceProject(t)
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	publishedCommand(t, root, "go", "mod", "edit", "-replace=hatmax.adrianpk.com="+repository)
	publishedCommand(t, root, "make", "generate")
	publishedCommand(t, root, "make", "format")
	publishedCommand(t, root, "make", "check")

	return root
}

// A durable validation change must survive the consumer's real lint rules and
// reject an invalid number in its generated executable model test.
func TestValidationRule(t *testing.T) {
	root := nativeFeatureProject(t)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddValidation, intent.Domain{
		Validation: &intent.ValidationRule{Field: "number", Kind: "min_length", Value: "3", Scope: intent.ValidationDurable},
	}, []string{"postgres_persistence", "runtime_validation"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatal(err)
	}

	mutations, err := RenderAddValidation(value, manifest, inventory)
	if err != nil {
		t.Fatal(err)
	}

	workspace, err := OpenWorkspace(t.Context(), manifest, value, inventory)
	if err != nil {
		t.Fatal(err)
	}

	for _, mutation := range mutations {
		_, err = workspace.Stage(mutation)
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = workspace.Commit(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	publishedCommand(t, root, "make", "generate")
	publishedCommand(t, root, "make", "format")
	publishedCommand(t, root, "make", "check")
}
