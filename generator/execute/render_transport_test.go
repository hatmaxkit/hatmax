package execute

import (
	"go/parser"
	"go/token"
	"html/template"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/htmx"
)

func TestRenderTransportRecipesUseHatmaxBoundaries(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Label:  "Invoices",
		Fields: []intent.Field{
			{Name: "number", Type: "string", Label: "Number", Required: true},
			{Name: "notes", Type: "text", Label: "Notes"},
		},
	}, []string{"postgres_persistence", "runtime_validation", "htmx_form"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := renderSelectedRecipes(value, manifest, inventory, transportRenderers)
	if err != nil {
		t.Fatalf("renderSelectedRecipes() error = %v", err)
	}

	if len(mutations) != 7 {
		t.Fatalf("rendered mutations = %d, want 7", len(mutations))
	}

	for _, mutation := range mutations {
		target := manifestEditTarget(manifest, mutation.EditID)
		switch {
		case strings.HasSuffix(target, ".go"):
			_, parseErr := parser.ParseFile(token.NewFileSet(), target, mutation.Content, parser.AllErrors)
			if parseErr != nil {
				t.Errorf("mutation %q is invalid Go: %v", mutation.EditID, parseErr)
			}
		case strings.HasSuffix(target, ".html"):
			_, parseErr := template.New(target).Funcs(htmx.FuncMap()).Parse(string(mutation.Content))
			if parseErr != nil {
				t.Errorf("mutation %q is invalid template: %v", mutation.EditID, parseErr)
			}
		}
	}

	assertRenderedContains(t, mutations, "create.migration", "-- +migrate Up", "CREATE TABLE invoices", "-- +migrate Down")
	assertRenderedContains(t, mutations, "create.queries", "-- name: ListInvoices :many", "-- name: UpdateInvoice :execrows")
	assertRenderedContains(t, mutations, "create.handler", "web.ParseForm(request)", "htmx.IsHTMXRequest(request)", "web.FormErrorsFrom")
	assertRenderedContains(t, mutations, "create.form_template", "hxAttrs", "hxPost", "required")
	assertRenderedContains(t, mutations, "create.row_template", "hxPut", "hxDelete", "hxSwapDelete")
	assertRenderedContains(t, mutations, "create.handler_tests", "HX-Request", "renderer.name != \"row\"")
}

func TestRenderTransportParsersCoverTypedFields(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{
			{Name: "active", Type: "boolean"},
			{Name: "due_on", Type: "date"},
			{Name: "sequence", Type: "integer"},
			{Name: "issued_at", Type: "timestamp"},
		},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := renderSelectedRecipes(value, manifest, inventory, transportRenderers)
	if err != nil {
		t.Fatalf("renderSelectedRecipes() error = %v", err)
	}

	assertCanonicalRenderedGo(t, manifest, mutations)

	assertRenderedContains(t, mutations, "create.handler", "values.Bool(\"active\")", "strconv.ParseInt", "time.Parse(\"2006-01-02\"", "time.Parse(\"2006-01-02T15:04\"")
	assertRenderedContains(t, mutations, "create.handler", "\"strconv\"", "\"time\"")
}
