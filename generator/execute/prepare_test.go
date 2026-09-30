// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

var executionSurfaces = []string{
	"migration",
	"model",
	"store",
	"service",
	"handler",
	"templates",
	"wiring",
	"tests",
}

func TestPrepareCreateFeatureManifest(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{
			{Name: "number", Type: "string", Required: true},
			{Name: "notes", Type: "text"},
		},
	}, []string{"postgres_persistence", "runtime_validation", "htmx_form"})

	result, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	if len(result.Edits) != 15 {
		t.Fatalf("len(Edits) = %d, want 15", len(result.Edits))
	}

	assertEditTarget(t, result.Edits, "create.migration", "db/migrations/002-invoice.sql")
	assertEditTarget(t, result.Edits, "create.queries", "internal/dal/queries/invoice.sql")
	assertEditTarget(t, result.Edits, "create.wiring", "main.go")

	err = VerifyDigest(result)
	if err != nil {
		t.Errorf("VerifyDigest() error = %v", err)
	}

	_, statErr := os.Stat(filepath.Join(root, "internal", "feat", "invoice"))
	if !os.IsNotExist(statErr) {
		t.Errorf("Prepare() changed project, stat error = %v", statErr)
	}
}

func TestPrepareIncrementalManifests(t *testing.T) {
	root := copyExecutionFixture(t)
	completeCanonicalProperty(t, root)

	tests := []struct {
		name         string
		operation    intent.Operation
		domain       intent.Domain
		capabilities []string
		wantEdits    int
	}{
		{
			name:      "add field",
			operation: intent.OperationAddField,
			domain: intent.Domain{
				Field: &intent.Field{Name: "available_from", Type: "date"},
			},
			capabilities: []string{"postgres_persistence"},
			wantEdits:    14,
		},
		{
			name:      "add durable validation",
			operation: intent.OperationAddValidation,
			domain: intent.Domain{
				Validation: &intent.ValidationRule{
					Field: "name",
					Kind:  "min_length",
					Value: "3",
					Scope: intent.ValidationDurable,
				},
			},
			capabilities: []string{"postgres_persistence", "runtime_validation"},
			wantEdits:    6,
		},
		{
			name:      "add client validation",
			operation: intent.OperationAddValidation,
			domain: intent.Domain{
				Validation: &intent.ValidationRule{
					Field: "name",
					Kind:  "min_length",
					Value: "3",
					Scope: intent.ValidationClientOnly,
				},
			},
			capabilities: []string{"postgres_persistence", "runtime_validation"},
			wantEdits:    4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, inventory, selectedBook := executionPlan(t, root, test.operation, test.domain, test.capabilities)

			result, err := Prepare(value, inventory, selectedBook)
			if err != nil {
				t.Fatalf("Prepare() error = %v", err)
			}

			if len(result.Edits) != test.wantEdits {
				t.Errorf("len(Edits) = %d, want %d", len(result.Edits), test.wantEdits)
			}

			if test.name == "add client validation" && hasSurface(result.Edits, "migration") {
				t.Error("client-only validation prepared a migration edit")
			}
		})
	}
}

func TestPrepareRejectsStalePlan(t *testing.T) {
	root := copyExecutionFixture(t)
	value, _, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{{Name: "number", Type: "string"}},
	}, []string{"postgres_persistence"})

	appendExecutionFile(t, root, "internal/feat/property/model.go", "\n// drift\n")

	current, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	_, err = Prepare(value, current, selectedBook)
	requireExecutionCode(t, err, "execution_plan_stale")
}

func TestPrepareEditRejectsProtectedAndGeneratedTargets(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, _ := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{{Name: "number", Type: "string"}},
	}, []string{"postgres_persistence"})

	operations := make(map[string]plan.Operation, len(value.Operations))
	for _, operation := range value.Operations {
		operations[operation.ID] = operation
	}

	protected := inventory
	protected.Rules.ProtectedPaths = append(protected.Rules.ProtectedPaths, project.ProtectedPath{
		Path:   "main.go",
		Reason: "fixture protection",
	})

	_, err := prepareEdit(value, protected, operations, updateTarget(
		"test.protected",
		EditUpdateGo,
		"wiring",
		"main.go",
		"server_rendered_crud.wiring",
		[]string{explicitWiringOperation},
	))
	requireExecutionCode(t, err, "execution_target_protected")

	_, err = prepareEdit(value, inventory, operations, updateTarget(
		"test.generated",
		EditUpdateGo,
		"store",
		"internal/dal/generated.go",
		"server_rendered_crud.postgres_store",
		[]string{postgresQueryAdapter},
	))
	requireExecutionCode(t, err, "execution_target_generated")
}

func executionPlan(
	t *testing.T,
	root string,
	operation intent.Operation,
	domain intent.Domain,
	capabilities []string,
) (plan.Plan, project.Inventory, *book.Book) {
	return executionPlanForFeature(t, root, featureForOperation(operation), operation, domain, capabilities)
}

func executionPlanForFeature(
	t *testing.T,
	root string,
	feature string,
	operation intent.Operation,
	domain intent.Domain,
	capabilities []string,
) (plan.Plan, project.Inventory, *book.Book) {
	t.Helper()

	selectedBook, err := book.LoadDefault()
	if err != nil {
		t.Fatalf("book.LoadDefault() error = %v", err)
	}

	inventory, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	fingerprint, err := inventory.Fingerprint(project.FingerprintRequest{
		BookVersion:          selectedBook.Manifest().BookVersion,
		SelectedDependencies: []string{"github.com/sqlc-dev/sqlc/cmd/sqlc"},
		PlannedSurfaces:      executionSurfaces,
	})
	if err != nil {
		t.Fatalf("Inventory.Fingerprint() error = %v", err)
	}

	value := intent.Intent{
		SchemaVersion:      intent.CurrentSchemaVersion,
		Operation:          operation,
		ProjectFingerprint: fingerprint.Value,
		HatmaxVersion:      inventory.Module.Hatmax.Version,
		BookVersion:        selectedBook.Manifest().BookVersion,
		Archetype:          "server_rendered_crud",
		Feature:            feature,
		Domain:             domain,
		Capabilities:       capabilities,
		Documentation:      intent.DocumentationNotRequested,
		Exceptions:         []intent.Exception{},
	}

	admission := intent.Validate(value, intent.ValidationContext{
		Inventory:   inventory,
		Fingerprint: fingerprint,
		Book:        selectedBook,
	})
	if !admission.Admitted() {
		t.Fatalf("intent.Validate() = %#v, want admitted", admission)
	}

	result, err := plan.Expand(admission, plan.ExpansionContext{
		Book:        selectedBook,
		Fingerprint: fingerprint,
	})
	if err != nil {
		t.Fatalf("plan.Expand() error = %v", err)
	}

	return result, inventory, selectedBook
}

func featureForOperation(operation intent.Operation) string {
	if operation == intent.OperationCreateFeature {
		return "invoice"
	}

	return "property"
}

func copyExecutionFixture(t *testing.T) string {
	t.Helper()

	source := filepath.Join("..", "project", "testdata", "supported")
	target := filepath.Join(t.TempDir(), "supported")

	err := filepath.WalkDir(source, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, relErr := filepath.Rel(source, sourcePath)
		if relErr != nil {
			return relErr
		}

		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		data, readErr := os.ReadFile(sourcePath)
		if readErr != nil {
			return readErr
		}

		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy execution fixture: %v", err)
	}

	return target
}

func completeCanonicalProperty(t *testing.T, root string) {
	t.Helper()

	files := map[string]string{
		"internal/feat/property/postgres_store.go":      "package property\n",
		"internal/feat/property/model_test.go":          "package property\n",
		"internal/feat/property/service_test.go":        "package property\n",
		"internal/feat/property/postgres_store_test.go": "package property\n",
		"internal/dal/queries/property.sql":             "-- name: ListProperties :many\nSELECT 1;\n",
		"assets/templates/property/page.html":           "{{define \"property/page\"}}{{end}}\n",
		"assets/templates/property/form.html":           "{{define \"property/form\"}}{{end}}\n",
		"assets/templates/property/row.html":            "{{define \"property/row\"}}{{end}}\n",
	}

	for path, content := range files {
		writeExecutionFile(t, root, path, content)
	}
}

func writeExecutionFile(t *testing.T, root, path, content string) {
	t.Helper()

	absolute := filepath.Join(root, filepath.FromSlash(path))

	err := os.MkdirAll(filepath.Dir(absolute), 0o755)
	if err != nil {
		t.Fatalf("create parent for %q: %v", path, err)
	}

	err = os.WriteFile(absolute, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func appendExecutionFile(t *testing.T, root, path, content string) {
	t.Helper()

	file, err := os.OpenFile(filepath.Join(root, filepath.FromSlash(path)), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}

	_, writeErr := file.WriteString(content)
	closeErr := file.Close()

	if writeErr != nil {
		t.Fatalf("append %q: %v", path, writeErr)
	}

	if closeErr != nil {
		t.Fatalf("close %q: %v", path, closeErr)
	}
}

func assertEditTarget(t *testing.T, edits []Edit, id, target string) {
	t.Helper()

	for _, edit := range edits {
		if edit.ID == id {
			if edit.Target != target {
				t.Errorf("edit %q target = %q, want %q", id, edit.Target, target)
			}

			return
		}
	}

	t.Errorf("edit %q not found", id)
}

func hasSurface(edits []Edit, surface string) bool {
	for _, edit := range edits {
		if edit.Surface == surface {
			return true
		}
	}

	return false
}
