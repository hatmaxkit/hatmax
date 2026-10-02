// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"context"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

func TestCheckConformanceAcceptsCanonicalIncrementalResult(t *testing.T) {
	root, value, manifest := appliedAddFieldProject(t)

	inventory, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	result, err := CheckConformance(value, manifest, inventory)
	if err != nil {
		t.Fatalf("CheckConformance() error = %v", err)
	}

	if !result.Passed || len(result.Diagnostics) != 0 {
		t.Errorf("CheckConformance() = %#v, want passed", result)
	}
}

func TestCheckConformanceRejectsRawHTMXAndHandlerPersistence(t *testing.T) {
	root, value, manifest := appliedAddFieldProject(t)
	replaceExecutionText(t, root, "assets/templates/invoice/form.html", "<form method=", `<form hx-post="/alternate" method=`)
	replaceExecutionText(t, root, "internal/feat/invoice/handler.go", `"context"`, "\"context\"\n\t\"database/sql\"")

	inventory, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	result, err := CheckConformance(value, manifest, inventory)
	if err != nil {
		t.Fatalf("CheckConformance() error = %v", err)
	}

	if result.Passed || !hasDiagnostic(result.Diagnostics, "HMGEN-HTMX-HELPER") || !hasDiagnostic(result.Diagnostics, "HMGEN-HANDLER-BOUNDARY") {
		t.Errorf("CheckConformance() = %#v, want HTMX and handler diagnostics", result)
	}
}

func appliedAddFieldProject(t *testing.T) (string, plan.Plan, Manifest) {
	t.Helper()

	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddField, intent.Domain{
		Field: &intent.Field{Name: "due_on", Type: "date", Label: "Due on"},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := RenderAddField(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderAddField() error = %v", err)
	}

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

	return root, value, manifest
}
