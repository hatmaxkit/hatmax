package execute

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

func TestRenderDocumentationProducesDistinctDiataxisDocuments(t *testing.T) {
	root := generatedInvoiceProject(t)
	appendExecutionFile(t, root, "Makefile", "\ndocs-check:\n\t@echo docs\n")

	targets := []intent.DocumentationTarget{
		{Quadrant: intent.DocumentationTutorial, Subject: "invoice_basics", ReaderGoal: "Learn the invoice workflow."},
		{Quadrant: intent.DocumentationHowTo, Subject: "invoice_workflow", ReaderGoal: "Complete the invoice workflow."},
		{Quadrant: intent.DocumentationReference, Subject: "invoice", ReaderGoal: "Find invoice contracts."},
		{Quadrant: intent.DocumentationExplanation, Subject: "invoice_ownership", ReaderGoal: "Understand invoice ownership."},
	}
	value, inventory, selectedBook := documentationExecutionPlan(t, root, intent.OperationDocumentFeature, intent.Domain{}, targets)

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := RenderDocumentation(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderDocumentation() error = %v", err)
	}

	if len(mutations) != 9 {
		t.Fatalf("len(Mutations) = %d, want four targets and five indexes", len(mutations))
	}

	contents := documentationMutationContents(t, manifest, mutations)
	assertDocumentationContains(t, contents["docs/tutorials/invoice-basics/index.md"], "## Outcome", "## Guided Steps", "make docs-check", "[Back to Tutorials](../index.md)")
	assertDocumentationContains(t, contents["docs/how-to/invoice-workflow/index.md"], "## Goal", "## Procedure", "make docs-check", "[Back to How-to Guides](../index.md)")
	assertDocumentationContains(t, contents["docs/reference/invoice/index.md"], "## Contract", "## Fields", "`number`", "`/invoices`")
	assertDocumentationContains(t, contents["docs/explanation/invoice-ownership/index.md"], "## Context", "## Ownership", "## Tradeoffs", "canonical implementation path")
	assertDocumentationContains(t, contents["docs/index.md"], "tutorials/index.md", "how-to/index.md", "reference/index.md", "explanation/index.md")

	for target, content := range contents {
		if strings.Contains(content, "TODO") || strings.Count(content, string(managedMarkdownStart)) != 1 || strings.Count(content, string(managedMarkdownEnd)) != 1 {
			t.Errorf("documentation %q contains a placeholder or invalid markers: %q", target, content)
		}
	}
}

func TestRenderDocumentationUsesPlannedChangeEvidence(t *testing.T) {
	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := documentationExecutionPlan(
		t,
		root,
		intent.OperationAddField,
		intent.Domain{Field: &intent.Field{Name: "due_on", Type: "date", Label: "Due on"}},
		[]intent.DocumentationTarget{{
			Quadrant: intent.DocumentationReference, Subject: "invoice", ReaderGoal: "Find the updated invoice contract.",
		}},
	)

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := RenderDocumentation(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderDocumentation() error = %v", err)
	}

	implementation, err := RenderAddField(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderAddField() error = %v", err)
	}

	if len(implementation) != 14 {
		t.Errorf("len(RenderAddField()) = %d, want 14 implementation mutations", len(implementation))
	}

	contents := documentationMutationContents(t, manifest, mutations)
	assertDocumentationContains(t, contents["docs/reference/invoice/index.md"], "`due_on`", "`date`", "Due on")
}

func TestRenderDocumentationAppliesThroughAtomicWorkspace(t *testing.T) {
	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := documentationExecutionPlan(
		t,
		root,
		intent.OperationDocumentFeature,
		intent.Domain{},
		[]intent.DocumentationTarget{{
			Quadrant: intent.DocumentationReference, Subject: "invoice", ReaderGoal: "Find invoice contracts.",
		}},
	)

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := RenderDocumentation(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderDocumentation() error = %v", err)
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

	result, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	if result.Status != ExecutionApplied || len(result.Changes) != 3 {
		t.Errorf("Commit() = %#v, want three applied documentation changes", result)
	}

	current, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	file, exists := current.DocumentationFile("docs/reference/invoice/index.md")
	if !exists || file.ManagedState != project.DocumentationManaged {
		t.Errorf("generated reference = %#v, %v", file, exists)
	}
}

func documentationMutationContents(t *testing.T, manifest Manifest, mutations []Mutation) map[string]string {
	t.Helper()

	result := make(map[string]string, len(mutations))
	for _, mutation := range mutations {
		edit, exists := manifestEdit(manifest.Edits, mutation.EditID)
		if !exists {
			t.Fatalf("mutation %q has no manifest edit", mutation.EditID)
		}

		result[edit.Target] = string(mutation.Content)
	}

	return result
}

func assertDocumentationContains(t *testing.T, content string, markers ...string) {
	t.Helper()

	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			t.Errorf("documentation does not contain %q:\n%s", marker, content)
		}
	}
}

func readDocumentationFile(t *testing.T, root, target string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(target)))
	if err != nil {
		t.Fatalf("read %q: %v", target, err)
	}

	return string(content)
}
