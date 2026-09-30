// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

func TestWorkspaceCommitsAllStagedTargets(t *testing.T) {
	workspace, root := createFeatureWorkspace(t)
	stageCreateFeature(t, workspace, root)

	result, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v, result = %#v", err, result)
	}

	if result.Status != ExecutionApplied || len(result.Changes) != 15 || len(result.Diagnostics) != 0 {
		t.Errorf("Commit() = %#v, want 15 applied or satisfied changes", result)
	}

	content, err := os.ReadFile(filepath.Join(root, "internal", "feat", "invoice", "model.go"))
	if err != nil {
		t.Fatalf("read committed model: %v", err)
	}

	if string(content) != "package invoice\n" {
		t.Errorf("committed model = %q, want rendered source", content)
	}

	stagedFiles, err := filepath.Glob(filepath.Join(root, "**", ".hatmax-stage-*"))
	if err != nil {
		t.Fatalf("find temporary files: %v", err)
	}

	if len(stagedFiles) != 0 {
		t.Errorf("temporary files remain after commit: %v", stagedFiles)
	}
}

func TestWorkspaceRejectsDriftBeforeCommit(t *testing.T) {
	workspace, root := createFeatureWorkspace(t)
	stageCreateFeature(t, workspace, root)
	appendExecutionFile(t, root, "internal/feat/property/model.go", "\n// drift after staging\n")

	result, err := workspace.Commit(context.Background())
	requireExecutionCode(t, err, "execution_plan_stale")

	if result.Status != ExecutionFailed || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "HMGEN-EXEC-DRIFT" {
		t.Errorf("Commit() = %#v, want stable drift diagnostic", result)
	}

	_, statErr := os.Stat(filepath.Join(root, "internal", "feat", "invoice", "model.go"))
	if !os.IsNotExist(statErr) {
		t.Errorf("failed Commit() changed create target, stat error = %v", statErr)
	}
}

func createFeatureWorkspace(t *testing.T) (*Workspace, string) {
	t.Helper()

	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{{Name: "number", Type: "string"}},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	workspace, err := OpenWorkspace(context.Background(), manifest, value, inventory)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	return workspace, root
}

func stageCreateFeature(t *testing.T, workspace *Workspace, root string) {
	t.Helper()

	for _, edit := range workspace.manifest.Edits {
		mutation := Mutation{EditID: edit.ID}
		if edit.Kind == EditCreateFile {
			mutation.Kind = MutationCreate
			mutation.Content = fixtureContent(edit.Target)
		} else {
			mutation.Kind = MutationReplace

			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(edit.Target)))
			if err != nil {
				t.Fatalf("read update target %q: %v", edit.Target, err)
			}

			mutation.Content = content
		}

		_, err := workspace.Stage(mutation)
		if err != nil {
			t.Fatalf("Stage(%q) error = %v", edit.ID, err)
		}
	}
}

func fixtureContent(target string) []byte {
	switch filepath.Ext(target) {
	case ".go":
		return []byte("package invoice\n")
	case ".sql":
		return []byte("-- generated for invoice\n")
	case ".html":
		name := strings.TrimSuffix(filepath.Base(target), filepath.Ext(target))

		return []byte("{{define \"invoice/" + name + "\"}}{{end}}\n")
	default:
		return []byte("generated for invoice\n")
	}
}
