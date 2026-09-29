package execute

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

func TestRenderDomainRecipesDeterministically(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{
			{Name: "number", Type: "string", Required: true},
			{Name: "notes", Type: "text"},
		},
	}, []string{"postgres_persistence", "runtime_validation", "htmx_form"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	first, err := renderSelectedRecipes(value, manifest, inventory, domainRenderers)
	if err != nil {
		t.Fatalf("renderSelectedRecipes() error = %v", err)
	}

	second, err := renderSelectedRecipes(value, manifest, inventory, domainRenderers)
	if err != nil {
		t.Fatalf("second renderSelectedRecipes() error = %v", err)
	}

	if len(first) != 7 || len(second) != len(first) {
		t.Fatalf("rendered mutations = %d and %d, want 7", len(first), len(second))
	}

	for index := range first {
		if first[index].EditID != second[index].EditID || string(first[index].Content) != string(second[index].Content) {
			t.Fatalf("rendered mutation %d is not deterministic", index)
		}

		if strings.Contains(string(first[index].Content), "TODO") {
			t.Errorf("mutation %q contains a placeholder", first[index].EditID)
		}

		if strings.Contains(string(first[index].Content), "%!") {
			t.Errorf("mutation %q contains a formatting failure", first[index].EditID)
		}

		if strings.HasSuffix(manifestEditTarget(manifest, first[index].EditID), ".go") {
			_, parseErr := parser.ParseFile(token.NewFileSet(), first[index].EditID, first[index].Content, parser.AllErrors)
			if parseErr != nil {
				t.Errorf("mutation %q is invalid Go: %v", first[index].EditID, parseErr)
			}
		}
	}

	assertRenderedContains(t, first, "create.model", "model.NewID()", "validation.Field(\"number\"")
	assertRenderedContains(t, first, "create.store_contract", "type Store interface", "Create(context.Context")
	assertRenderedContains(t, first, "create.postgres_store", "dal.New(database)", "ErrNotFound")
	assertRenderedContains(t, first, "create.service", "NewInvoice(input)", "service.store.Create")
	assertRenderedContains(t, first, "create.model_tests", "missing required value")
	assertRenderedContains(t, first, "create.service_tests", "did not persist the valid entity")
}

func TestRenderDomainRecipesRejectUnresolvedBusinessRules(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{{Name: "number", Type: "string"}},
		Rules: []intent.BusinessRule{{
			Name:        "approved_total",
			Description: "Only approved invoices may expose a total.",
			Owner:       "Invoice",
		}},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	_, err = renderSelectedRecipes(value, manifest, inventory, domainRenderers)
	requireExecutionCode(t, err, "execution_slot_unresolved")
}

func TestRenderDomainRecipesCoverEveryAdmittedFieldType(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{
			{Name: "active", Type: "boolean"},
			{Name: "due_on", Type: "date", Required: true},
			{Name: "amount", Type: "decimal"},
			{Name: "sequence", Type: "integer"},
			{Name: "number", Type: "string"},
			{Name: "notes", Type: "text"},
			{Name: "issued_at", Type: "timestamp"},
			{Name: "customer_id", Type: "uuid"},
		},
	}, []string{"postgres_persistence", "runtime_validation"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := renderSelectedRecipes(value, manifest, inventory, domainRenderers)
	if err != nil {
		t.Fatalf("renderSelectedRecipes() error = %v", err)
	}

	assertCanonicalRenderedGo(t, manifest, mutations)

	assertRenderedContains(t, mutations, "create.model", "Active bool", "DueOn time.Time", "Amount string", "Sequence int64", "model.ParseID(value.CustomerID)")
	assertRenderedContains(t, mutations, "create.model_tests", "\"time\"")
	assertRenderedContains(t, mutations, "create.service_tests", "\"time\"")
}

func manifestEditTarget(manifest Manifest, id string) string {
	edit, _ := manifestEdit(manifest.Edits, id)

	return edit.Target
}

func assertRenderedContains(t *testing.T, mutations []Mutation, id string, expected ...string) {
	t.Helper()

	for _, mutation := range mutations {
		if mutation.EditID != id {
			continue
		}

		content := string(mutation.Content)

		normalizedContent := strings.Join(strings.Fields(content), " ")
		for _, value := range expected {
			normalizedValue := strings.Join(strings.Fields(value), " ")
			if !strings.Contains(normalizedContent, normalizedValue) {
				t.Errorf("mutation %q does not contain %q", id, value)
			}
		}

		return
	}

	t.Errorf("mutation %q was not rendered", id)
}
