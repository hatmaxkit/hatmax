// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/project"
)

func TestPrepareApplicationCoversExactPlannedScaffold(t *testing.T) {
	value, target, selectedBook := applicationExecutionPlan(t, nil)

	manifest, err := PrepareApplication(value, target, selectedBook)
	if err != nil {
		t.Fatalf("PrepareApplication() error = %v", err)
	}

	if manifest.SourceFingerprint != value.SourceFingerprint || manifest.TargetPath != target.Target {
		t.Errorf("manifest identity = fingerprint %q target %q", manifest.SourceFingerprint, manifest.TargetPath)
	}

	if len(manifest.Edits) != len(value.AllowedEffects.Files) {
		t.Fatalf("manifest edits = %d, want %d", len(manifest.Edits), len(value.AllowedEffects.Files))
	}

	for _, edit := range manifest.Edits {
		if applicationRecipeID(edit.Target) != edit.Recipe {
			t.Errorf("edit %s recipe = %q, want canonical recipe", edit.Target, edit.Recipe)
		}
	}
}

func TestApplicationWorkspacePublishesAtomicallyAndIsIdempotent(t *testing.T) {
	workspace, target := preparedApplicationWorkspace(t, nil)

	result, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	if result.Status != ExecutionApplied || len(result.Changes) == 0 {
		t.Fatalf("Commit() = %#v, want applied changes", result)
	}

	assertApplicationPublished(t, target)

	repeated, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("repeated Commit() error = %v", err)
	}

	if repeated.Status != ExecutionAlreadySatisfied {
		t.Errorf("repeated status = %q, want %q", repeated.Status, ExecutionAlreadySatisfied)
	}
}

func TestApplicationWorkspacePreservesAdmittedRepositoryFiles(t *testing.T) {
	prepare := func(target string) {
		t.Helper()

		err := os.MkdirAll(target, 0o755)
		if err != nil {
			t.Fatalf("create target: %v", err)
		}

		command := exec.Command("git", "init", "-q")
		command.Dir = target

		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git init: %v\n%s", err, output)
		}

		err = os.WriteFile(filepath.Join(target, "NOTICE"), []byte("preserve me\n"), 0o644)
		if err != nil {
			t.Fatalf("write NOTICE: %v", err)
		}
	}

	workspace, target := preparedApplicationWorkspace(t, prepare)

	_, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(target, "NOTICE"))
	if err != nil || string(content) != "preserve me\n" {
		t.Errorf("preserved NOTICE = %q, %v", content, err)
	}

	info, err := os.Stat(filepath.Join(target, ".git"))
	if err != nil || !info.IsDir() {
		t.Errorf("preserved .git = %#v, %v", info, err)
	}
}

func TestApplicationWorkspaceRejectsConcurrentTargetDrift(t *testing.T) {
	workspace, target := preparedApplicationWorkspace(t, nil)
	workspace.validator = func(context.Context, string, []Command) ([]ValidationCommandResult, error) {
		err := os.MkdirAll(target, 0o755)
		if err != nil {
			return nil, err
		}

		err = os.WriteFile(filepath.Join(target, "CONCURRENT"), []byte("changed\n"), 0o644)
		if err != nil {
			return nil, err
		}

		return passedApplicationValidation(), nil
	}

	result, err := workspace.Commit(context.Background())
	requireExecutionCode(t, err, "execution_plan_stale")

	if result.Status != ExecutionFailed {
		t.Errorf("status = %q, want %q", result.Status, ExecutionFailed)
	}

	_, err = os.Stat(filepath.Join(target, "main.go"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("main.go exists after rejected drift: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(target, "CONCURRENT"))
	if err != nil || string(content) != "changed\n" {
		t.Errorf("concurrent content = %q, %v", content, err)
	}
}

func TestApplicationWorkspaceCancellationLeavesTargetAbsent(t *testing.T) {
	workspace, target := preparedApplicationWorkspace(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	workspace.validator = func(context.Context, string, []Command) ([]ValidationCommandResult, error) {
		cancel()

		return passedApplicationValidation(), nil
	}

	_, err := workspace.Commit(ctx)
	requireExecutionCode(t, err, "execution_cancelled")

	_, err = os.Stat(target)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("target exists after cancellation: %v", err)
	}
}

func TestApplicationPublicationRestoresExistingTargetOnSwapFailure(t *testing.T) {
	parent := t.TempDir()

	target := filepath.Join(parent, "real-estate")

	err := os.Mkdir(target, 0o755)
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	notice := filepath.Join(target, "NOTICE")

	err = os.WriteFile(notice, []byte("original\n"), 0o644)
	if err != nil {
		t.Fatalf("write NOTICE: %v", err)
	}

	err = publishApplicationTarget(filepath.Join(parent, "missing-stage"), project.TargetInventory{
		Parent: parent, Target: target, Exists: true,
	})
	requireExecutionCode(t, err, "execution_publish_failed")

	content, readErr := os.ReadFile(notice)
	if readErr != nil || string(content) != "original\n" {
		t.Errorf("restored NOTICE = %q, %v", content, readErr)
	}
}

func TestApplicationValidationClassifiesMissingInfrastructure(t *testing.T) {
	for _, output := range []string{
		"dial tcp 127.0.0.1:5432: connect: connection refused",
		"Cannot connect to the Docker daemon",
		"testcontainers: Docker daemon is not running",
	} {
		if !testInfrastructureUnavailable(output) {
			t.Errorf("testInfrastructureUnavailable(%q) = false", output)
		}
	}

	if testInfrastructureUnavailable("model contract assertion failed") {
		t.Error("generated behavior failure classified as infrastructure")
	}

	commands := []Command{{
		Name: "validation.test", Args: []string{"sh", "-c", "echo connection refused; exit 1"}, WorkingDirectory: ".",
	}}

	results, err := validateApplicationStaging(context.Background(), t.TempDir(), commands)
	if err != nil {
		t.Fatalf("validateApplicationStaging() error = %v", err)
	}

	if len(results) != 1 || results[0].Status != ValidationCommandIncomplete {
		t.Errorf("validation results = %#v, want incomplete", results)
	}
}

func preparedApplicationWorkspace(t *testing.T, prepare func(string)) (*ApplicationWorkspace, string) {
	t.Helper()

	value, target, selectedBook := applicationExecutionPlan(t, prepare)

	manifest, err := PrepareApplication(value, target, selectedBook)
	if err != nil {
		t.Fatalf("PrepareApplication() error = %v", err)
	}

	mutations, err := RenderApplication(value, manifest)
	if err != nil {
		t.Fatalf("RenderApplication() error = %v", err)
	}

	workspace, err := OpenApplicationWorkspace(context.Background(), manifest, value, target, selectedBook)
	if err != nil {
		t.Fatalf("OpenApplicationWorkspace() error = %v", err)
	}

	for _, mutation := range mutations {
		_, err = workspace.Stage(mutation)
		if err != nil {
			t.Fatalf("Stage(%s) error = %v", mutation.EditID, err)
		}
	}

	workspace.validator = func(context.Context, string, []Command) ([]ValidationCommandResult, error) {
		return passedApplicationValidation(), nil
	}

	return workspace, target.Target
}

func passedApplicationValidation() []ValidationCommandResult {
	return []ValidationCommandResult{
		{Name: "validation.build", Status: ValidationCommandPassed},
		{Name: "validation.test", Status: ValidationCommandPassed},
	}
}

func assertApplicationPublished(t *testing.T, root string) {
	t.Helper()

	for _, target := range []string{
		"go.mod", "go.sum", "main.go", "config.yaml", "internal/application/application.go",
		"internal/application/web.go", "internal/application/assets/templates/pages/home.html",
	} {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(target)))
		if err != nil {
			t.Errorf("published %s: %v", target, err)
		}
	}

	_, err := os.Stat(filepath.Join(root, "README.md"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("README.md should not be generated: %v", err)
	}
}
