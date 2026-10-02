// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

func TestPrepareDocumentationOnlyManifest(t *testing.T) {
	root := generatedInvoiceProject(t)
	appendExecutionFile(t, root, "Makefile", "\ndocs-check:\n\t@echo docs\n")

	value, inventory, selectedBook := documentationExecutionPlan(
		t,
		root,
		intent.OperationDocumentFeature,
		intent.Domain{},
		[]intent.DocumentationTarget{{
			Quadrant: intent.DocumentationReference, Subject: "invoice", ReaderGoal: "Find the exact invoice contract.",
		}},
	)

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	if len(manifest.Edits) != 3 {
		t.Fatalf("len(Edits) = %d, want target, root index, and quadrant index", len(manifest.Edits))
	}

	for _, edit := range manifest.Edits {
		if edit.Surface != "documentation" {
			t.Errorf("edit %q surface = %q, want documentation", edit.ID, edit.Surface)
		}
	}

	assertEditTarget(t, manifest.Edits, "documentation.target.reference.invoice", "docs/reference/invoice/README.md")
	assertEditTarget(t, manifest.Edits, "documentation.index.root", "docs/README.md")
	assertEditTarget(t, manifest.Edits, "documentation.index.quadrant.reference", "docs/reference/README.md")

	rootIndex, _ := manifestEdit(manifest.Edits, "documentation.index.root")
	quadrantIndex, _ := manifestEdit(manifest.Edits, "documentation.index.quadrant.reference")

	if !sameStrings(rootIndex.DependsOn, []string{"documentation.target.reference.invoice"}) || !sameStrings(quadrantIndex.DependsOn, []string{"documentation.target.reference.invoice", "documentation.index.root"}) {
		t.Errorf("documentation dependencies = root %v quadrant %v", rootIndex.DependsOn, quadrantIndex.DependsOn)
	}

	if len(manifest.Commands) != 1 || manifest.Commands[0].Name != "validation.docs-check" {
		t.Errorf("Commands = %#v, want only the repository documentation gate", manifest.Commands)
	}

	_, statErr := os.Stat(filepath.Join(root, "docs", "README.md"))
	if !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("Prepare() changed docs/README.md: %v", statErr)
	}
}

func TestPrepareCombinedManifestIncludesDocumentationEffects(t *testing.T) {
	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := documentationExecutionPlan(
		t,
		root,
		intent.OperationAddField,
		intent.Domain{Field: &intent.Field{Name: "due_on", Type: "date", Label: "Due on"}},
		[]intent.DocumentationTarget{{
			Quadrant: intent.DocumentationHowTo, Subject: "invoice_due_on", ReaderGoal: "Add and use the invoice due date.",
		}},
	)

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	if len(manifest.Edits) != 17 || !hasSurface(manifest.Edits, "documentation") || !hasSurface(manifest.Edits, "model") {
		t.Errorf("Edits = %d across %v, want implementation plus three documentation effects", len(manifest.Edits), manifest.AllowedSurfaces)
	}

	assertEditTarget(t, manifest.Edits, "documentation.target.how_to.invoice_due_on", "docs/how-to/invoice-due-on/README.md")
}

func TestPrepareManagedDocumentationUpdateUsesSnapshot(t *testing.T) {
	root := generatedInvoiceProject(t)
	writeExecutionFile(t, root, "docs/reference/invoice/README.md", managedExecutionDocumentation("Existing generated reference."))

	value, inventory, selectedBook := documentationExecutionPlan(
		t,
		root,
		intent.OperationDocumentFeature,
		intent.Domain{},
		[]intent.DocumentationTarget{{
			Quadrant: intent.DocumentationReference, Subject: "invoice", ReaderGoal: "Find the exact invoice contract.",
		}},
	)

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	edit, exists := manifestEdit(manifest.Edits, "documentation.target.reference.invoice")
	if !exists || edit.Kind != EditUpdateMarkdown || !conditionPresent(edit.Preconditions, ConditionPathDigest) || !conditionPresent(edit.Postconditions, ConditionManagedOutsideDigest) {
		t.Errorf("managed target edit = %#v, want digest-bound Markdown update", edit)
	}
}

func TestPrepareRejectsDocumentationDrift(t *testing.T) {
	root := generatedInvoiceProject(t)
	writeExecutionFile(t, root, "docs/reference/invoice/README.md", managedExecutionDocumentation("Existing generated reference."))

	value, _, selectedBook := documentationExecutionPlan(
		t,
		root,
		intent.OperationDocumentFeature,
		intent.Domain{},
		[]intent.DocumentationTarget{{
			Quadrant: intent.DocumentationReference, Subject: "invoice", ReaderGoal: "Find the exact invoice contract.",
		}},
	)
	appendExecutionFile(t, root, "docs/reference/invoice/README.md", "\nExternal drift.\n")

	current, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	_, err = Prepare(value, current, selectedBook)
	requireExecutionCode(t, err, "execution_plan_stale")
}

func documentationExecutionPlan(
	t *testing.T,
	root string,
	operation intent.Operation,
	domain intent.Domain,
	targets []intent.DocumentationTarget,
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

	surfaces := append([]string{}, executionSurfaces...)
	surfaces = append(surfaces, "documentation")

	fingerprint, err := inventory.Fingerprint(project.FingerprintRequest{
		BookVersion:          selectedBook.Manifest().BookVersion,
		SelectedDependencies: []string{"github.com/sqlc-dev/sqlc/cmd/sqlc"},
		PlannedSurfaces:      surfaces,
	})
	if err != nil {
		t.Fatalf("Inventory.Fingerprint() error = %v", err)
	}

	documentationMode := intent.DocumentationPlanned
	if operation == intent.OperationDocumentFeature {
		documentationMode = intent.DocumentationExisting
	}

	value := intent.Intent{
		SchemaVersion:        intent.CurrentSchemaVersion,
		Operation:            operation,
		ProjectFingerprint:   fingerprint.Value,
		HatmaxVersion:        inventory.Module.Hatmax.Version,
		BookVersion:          selectedBook.Manifest().BookVersion,
		Archetype:            "server_rendered_crud",
		Feature:              "invoice",
		Domain:               domain,
		Capabilities:         []string{"postgres_persistence"},
		Documentation:        documentationMode,
		DocumentationTargets: targets,
		Exceptions:           []intent.Exception{},
	}

	admission := intent.Validate(value, intent.ValidationContext{
		Inventory: inventory, Fingerprint: fingerprint, Book: selectedBook,
	})
	if !admission.Admitted() {
		t.Fatalf("intent.Validate() = %#v, want admitted", admission)
	}

	result, err := plan.Expand(admission, plan.ExpansionContext{
		Book: selectedBook, Fingerprint: fingerprint, Inventory: inventory,
	})
	if err != nil {
		t.Fatalf("plan.Expand() error = %v", err)
	}

	return result, inventory, selectedBook
}

func managedExecutionDocumentation(content string) string {
	return "# Wrapper\n\n<!-- hatmax:generated:start -->\n" + content + "\n<!-- hatmax:generated:end -->\n"
}
