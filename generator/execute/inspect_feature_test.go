package execute

import (
	"context"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

func TestInspectCanonicalFeatureDiscoversGeneratedSurfaces(t *testing.T) {
	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddField, intent.Domain{
		Field: &intent.Field{Name: "due_on", Type: "date", Label: "Due on"},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	feature, err := inspectCanonicalFeature(value, manifest, inventory)
	if err != nil {
		t.Fatalf("inspectCanonicalFeature() error = %v", err)
	}

	if feature.entity != "Invoice" || feature.table != "invoices" || feature.route != "/invoices" || feature.label != "Invoices" {
		t.Errorf("feature identity = %#v", feature)
	}

	if len(feature.fields) != 2 || feature.fields[0].Name != "number" || feature.fields[1].Name != "notes" {
		t.Errorf("feature fields = %#v", feature.fields)
	}
}

func TestInspectCanonicalFeatureRejectsMissingSemanticAnchor(t *testing.T) {
	root := generatedInvoiceProject(t)
	replaceExecutionText(t, root, "internal/feat/invoice/handler.go", "func parseInput", "func parseLegacyInput")
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddField, intent.Domain{
		Field: &intent.Field{Name: "due_on", Type: "date"},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	_, err = inspectCanonicalFeature(value, manifest, inventory)
	requireExecutionCode(t, err, "execution_feature_structure_invalid")
}

func generatedInvoiceProject(t *testing.T) string {
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
		t.Fatalf("Prepare(create) error = %v", err)
	}

	mutations, err := RenderCreateFeature(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderCreateFeature() error = %v", err)
	}

	workspace, err := OpenWorkspace(context.Background(), manifest, value, inventory)
	if err != nil {
		t.Fatalf("OpenWorkspace(create) error = %v", err)
	}

	for _, mutation := range mutations {
		_, err = workspace.Stage(mutation)
		if err != nil {
			t.Fatalf("Stage(create %q) error = %v", mutation.EditID, err)
		}
	}

	_, err = workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit(create) error = %v", err)
	}

	return root
}
