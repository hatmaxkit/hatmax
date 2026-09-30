// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

func TestExecutionConformanceCorpusAcceptsEveryAdmittedOperation(t *testing.T) {
	tests := []struct {
		name  string
		build func(*testing.T) (string, plan.Plan, Manifest)
	}{
		{name: "create feature", build: appliedCreateFeatureProject},
		{name: "add field", build: appliedAddFieldProject},
		{name: "add durable validation", build: func(t *testing.T) (string, plan.Plan, Manifest) {
			return appliedValidationProject(t, intent.ValidationDurable)
		}},
		{name: "add client validation", build: func(t *testing.T) (string, plan.Plan, Manifest) {
			return appliedValidationProject(t, intent.ValidationClientOnly)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, value, manifest := test.build(t)

			inventory, err := project.Inspect(context.Background(), root)
			if err != nil {
				t.Fatalf("project.Inspect() error = %v", err)
			}

			result, err := CheckConformance(value, manifest, inventory)
			if err != nil {
				t.Fatalf("CheckConformance() error = %v", err)
			}

			if !result.Passed {
				t.Errorf("diagnostics = %#v, want passed", result.Diagnostics)
			}
		})
	}
}

func TestExecutionConformanceCorpusRejectsNoncanonicalStructures(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*testing.T, string)
		wantCode string
	}{
		{
			name: "missing feature package surface",
			mutate: func(t *testing.T, root string) {
				t.Helper()

				err := os.Remove(filepath.Join(root, "internal", "feat", "invoice", "service.go"))
				if err != nil {
					t.Fatalf("remove service: %v", err)
				}
			},
			wantCode: "HMGEN-FEATURE-OWNERSHIP",
		},
		{
			name: "alternate validation dependency",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				replaceExecutionText(t, root, "internal/feat/invoice/model.go", "hatmax.adrianpk.com/validation", "github.com/go-playground/validator/v10")
			},
			wantCode: "HMGEN-DEPENDENCY-SUBSTITUTION",
		},
		{
			name: "hidden wiring",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				replaceExecutionText(t, root, "main.go", "invoiceStore :=", "store :=")
			},
			wantCode: "HMGEN-WIRING-ORDER",
		},
		{
			name: "missing SQLC query",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				replaceExecutionText(t, root, "internal/dal/queries/invoice.sql", "-- name: DeleteInvoice", "-- legacy: DeleteInvoice")
			},
			wantCode: "HMGEN-SQLC-QUERY",
		},
		{
			name: "missing boundary test",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				replaceExecutionText(t, root, "internal/feat/invoice/service_test.go", "func Test", "func Check")
			},
			wantCode: "HMGEN-TEST-BOUNDARY",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, value, manifest := appliedAddFieldProject(t)
			test.mutate(t, root)

			inventory, err := project.Inspect(context.Background(), root)
			if err != nil {
				t.Fatalf("project.Inspect() error = %v", err)
			}

			result, err := CheckConformance(value, manifest, inventory)
			if err != nil {
				t.Fatalf("CheckConformance() error = %v", err)
			}

			if result.Passed || !hasDiagnostic(result.Diagnostics, test.wantCode) {
				t.Errorf("diagnostics = %#v, want %s", result.Diagnostics, test.wantCode)
			}
		})
	}
}

func appliedCreateFeatureProject(t *testing.T) (string, plan.Plan, Manifest) {
	t.Helper()

	root := copyExecutionFixture(t)
	writeExecutionFile(t, root, "main.go", canonicalCompositionRoot)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice", Route: "/invoices", Label: "Invoices",
		Fields: []intent.Field{
			{Name: "number", Type: "string", Label: "Number", Required: true},
			{Name: "notes", Type: "text", Label: "Notes"},
		},
	}, []string{"postgres_persistence", "runtime_validation", "htmx_form"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := RenderCreateFeature(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderCreateFeature() error = %v", err)
	}

	applyMutations(t, value, manifest, inventory, mutations)

	return root, value, manifest
}

func appliedValidationProject(t *testing.T, scope intent.ValidationScope) (string, plan.Plan, Manifest) {
	t.Helper()

	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddValidation, intent.Domain{
		Validation: &intent.ValidationRule{
			Field: "number", Kind: "min_length", Value: "3", Message: "Use at least three characters.", Scope: scope,
		},
	}, []string{"postgres_persistence", "runtime_validation"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := RenderAddValidation(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderAddValidation() error = %v", err)
	}

	applyMutations(t, value, manifest, inventory, mutations)

	return root, value, manifest
}

func applyMutations(t *testing.T, value plan.Plan, manifest Manifest, inventory project.Inventory, mutations []Mutation) {
	t.Helper()

	workspace, err := OpenWorkspace(context.Background(), manifest, value, inventory)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	for _, mutation := range mutations {
		_, err = workspace.Stage(mutation)
		if err != nil {
			t.Fatalf("Stage(%q) error = %v", mutation.EditID, err)
		}
	}

	_, err = workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
}
