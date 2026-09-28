package execute

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

func TestRenderAddFieldUpdatesEveryCanonicalRepresentation(t *testing.T) {
	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddField, intent.Domain{
		Field: &intent.Field{Name: "due_on", Type: "date", Label: "Due on", Required: true},
	}, []string{"postgres_persistence", "runtime_validation"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := RenderAddField(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderAddField() error = %v", err)
	}

	if len(mutations) != 14 {
		t.Fatalf("rendered mutations = %d, want 14", len(mutations))
	}

	assertRenderedContains(t, mutations, "add_field.migration", "ALTER TABLE invoices ADD COLUMN due_on DATE NOT NULL", "DROP COLUMN due_on")
	assertRenderedContains(t, mutations, "add_field.model", "DueOn time.Time")
	assertRenderedContains(t, mutations, "add_field.postgres_store", "DueOn: value.DueOn")
	assertRenderedContains(t, mutations, "add_field.queries", "due_on", "sqlc.arg(due_on)")
	assertRenderedContains(t, mutations, "add_field.handler", `values.String("due_on")`, `time.Parse("2006-01-02"`)
	assertRenderedContains(t, mutations, "add_field.form_template", `name="due_on"`, "Due on", "required")
	assertRenderedContains(t, mutations, "add_field.row_template", `name="due_on"`, ".DueOn")
	assertRenderedContains(t, mutations, "add_field.model_tests", "DueOn: time.Unix")

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

	result, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	if result.Status != ExecutionApplied || len(result.Changes) != len(manifest.Edits) {
		t.Fatalf("Commit() = %#v, want applied", result)
	}

	model, err := os.ReadFile(filepath.Join(root, "internal", "feat", "invoice", "model.go"))
	if err != nil || !strings.Contains(strings.Join(strings.Fields(string(model)), " "), "DueOn time.Time") {
		t.Errorf("applied model = %q, %v", model, err)
	}
}

func TestRenderAddFieldRejectsDuplicateField(t *testing.T) {
	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddField, intent.Domain{
		Field: &intent.Field{Name: "number", Type: "string"},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	_, err = RenderAddField(value, manifest, inventory)
	requireExecutionCode(t, err, "execution_semantic_conflict")
}
