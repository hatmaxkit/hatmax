// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

func TestDocumentationConformanceAcceptsRenderedSurface(t *testing.T) {
	root, value, manifest, inventory := appliedDocumentationProject(t)

	result, err := CheckConformance(value, manifest, inventory)
	if err != nil {
		t.Fatalf("CheckConformance() error = %v", err)
	}

	if !result.Passed || len(result.Diagnostics) != 0 {
		t.Errorf("CheckConformance() = %#v, want passed", result)
	}

	for _, target := range []string{"docs/README.md", "docs/reference/README.md", "docs/reference/invoice/README.md"} {
		_, err = os.Stat(filepath.Join(root, filepath.FromSlash(target)))
		if err != nil {
			t.Errorf("generated target %q: %v", target, err)
		}
	}
}

func TestDocumentationConformanceRejectsMixedIntentBrokenLinkAndEvidenceDrift(t *testing.T) {
	root, value, manifest, _ := appliedDocumentationProject(t)
	target := "docs/reference/invoice/README.md"
	replaceExecutionText(t, root, target, "## Contract", "## Guided Steps")
	replaceExecutionText(t, root, target, "`/invoices`", "`/alternate`")
	replaceExecutionText(t, root, target, "../README.md", "../missing/README.md")

	inventory, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	result, err := CheckConformance(value, manifest, inventory)
	if err != nil {
		t.Fatalf("CheckConformance() error = %v", err)
	}

	for _, code := range []string{"HMGEN-DOCUMENTATION-DIATAXIS", "HMGEN-DOCUMENTATION-INDEX", "HMGEN-DOCUMENTATION-EVIDENCE"} {
		if !hasDiagnostic(result.Diagnostics, code) {
			t.Errorf("diagnostics = %#v, want %q", result.Diagnostics, code)
		}
	}
}

func TestDocumentationConformanceDetectsUserContentReplacement(t *testing.T) {
	root := generatedInvoiceProject(t)
	writeExecutionFile(t, root, "docs/reference/invoice/README.md", "User prefix.\n\n"+managedExecutionDocumentation("Old reference.")+"\nUser suffix.\n")
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

	stageMutations(t, workspace, mutations)

	_, err = workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	content := readDocumentationFile(t, root, "docs/reference/invoice/README.md")
	if !strings.HasPrefix(content, "User prefix.\n\n# Wrapper\n\n") || !strings.HasSuffix(content, "\n\nUser suffix.\n") {
		t.Errorf("user-owned content was not preserved: %q", content)
	}

	replaceExecutionText(t, root, "docs/reference/invoice/README.md", "User prefix.", "Changed prefix.")

	current, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	result, err := CheckConformance(value, manifest, current)
	if err != nil {
		t.Fatalf("CheckConformance() error = %v", err)
	}

	if !hasDiagnostic(result.Diagnostics, "HMGEN-DOCUMENTATION-MANAGED-SECTION") {
		t.Errorf("diagnostics = %#v, want preservation failure", result.Diagnostics)
	}
}

func TestDocumentationTransactionRollsBackAllTargets(t *testing.T) {
	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := documentationExecutionPlan(
		t,
		root,
		intent.OperationDocumentFeature,
		intent.Domain{},
		[]intent.DocumentationTarget{
			{Quadrant: intent.DocumentationTutorial, Subject: "invoice_basics", ReaderGoal: "Learn the invoice workflow."},
			{Quadrant: intent.DocumentationHowTo, Subject: "invoice_workflow", ReaderGoal: "Complete the invoice workflow."},
			{Quadrant: intent.DocumentationReference, Subject: "invoice", ReaderGoal: "Find invoice contracts."},
			{Quadrant: intent.DocumentationExplanation, Subject: "invoice_ownership", ReaderGoal: "Understand invoice ownership."},
		},
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

	stageMutations(t, workspace, mutations)
	workspace.hooks.beforeApply = func(index int, _ Edit) error {
		if index == 1 {
			return errors.New("forced documentation apply failure")
		}

		return nil
	}

	result, err := workspace.Commit(context.Background())
	requireExecutionCode(t, err, "execution_commit_failed")

	if !hasDiagnostic(result.Diagnostics, "HMGEN-EXEC-ROLLED-BACK") {
		t.Errorf("Commit() diagnostics = %#v, want rollback", result.Diagnostics)
	}

	for _, edit := range manifest.Edits {
		_, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(edit.Target)))
		if !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("rolled-back documentation target %q remains: %v", edit.Target, statErr)
		}
	}
}

func TestDocumentationGateFailureRetainsDeclaredChanges(t *testing.T) {
	root, value, manifest, inventory := appliedDocumentationProject(t)
	runner := func(_ context.Context, _ string, command Command) (CommandEvidence, error) {
		return CommandEvidence{
			Name: command.Name, Kind: command.Kind, Args: command.Args,
			WorkingDirectory: command.WorkingDirectory, ExitCode: 2,
		}, errors.New("documentation gate failed")
	}

	report, err := validateExecutionWithRunner(context.Background(), value, manifest, inventory, runner)
	requireExecutionCode(t, err, "execution_command_failed")

	if !report.Conformance.Passed || len(report.Commands) != 1 || report.Commands[0].Name != "validation.docs-check" {
		t.Errorf("ExecutionReport = %#v, want passed conformance and failed documentation gate evidence", report)
	}

	_, statErr := os.Stat(filepath.Join(root, "docs", "reference", "invoice", "README.md"))
	if statErr != nil {
		t.Errorf("documentation gate failure removed committed target: %v", statErr)
	}
}

func appliedDocumentationProject(t *testing.T) (string, plan.Plan, Manifest, project.Inventory) {
	t.Helper()

	root := generatedInvoiceProject(t)
	appendExecutionFile(t, root, "Makefile", "\ndocs-check:\n\t@echo docs\n")
	value, inventory, selectedBook := documentationExecutionPlan(
		t,
		root,
		intent.OperationDocumentFeature,
		intent.Domain{},
		[]intent.DocumentationTarget{
			{Quadrant: intent.DocumentationTutorial, Subject: "invoice_basics", ReaderGoal: "Learn the invoice workflow."},
			{Quadrant: intent.DocumentationHowTo, Subject: "invoice_workflow", ReaderGoal: "Complete the invoice workflow."},
			{Quadrant: intent.DocumentationReference, Subject: "invoice", ReaderGoal: "Find invoice contracts."},
			{Quadrant: intent.DocumentationExplanation, Subject: "invoice_ownership", ReaderGoal: "Understand invoice ownership."},
		},
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

	stageMutations(t, workspace, mutations)

	_, err = workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	current, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	return root, value, manifest, current
}

func stageMutations(t *testing.T, workspace *Workspace, mutations []Mutation) {
	t.Helper()

	for _, mutation := range mutations {
		_, err := workspace.Stage(mutation)
		if err != nil {
			t.Fatalf("Stage(%q) error = %v", mutation.EditID, err)
		}
	}
}
