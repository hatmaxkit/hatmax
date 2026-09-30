// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/book"
)

func TestInspectSupportedProject(t *testing.T) {
	root := copyFixture(t, "supported")
	initializeGit(t, root)

	inventory, err := Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if inventory.Module.Path != "example.com/property" || inventory.Module.GoVersion != "1.24.0" {
		t.Errorf("Module = %#v, want example.com/property with Go 1.24.0", inventory.Module)
	}

	if inventory.Module.Hatmax.Source != HatmaxSourceModule || inventory.Module.Hatmax.Version != "v0.4.0" {
		t.Errorf("Module.Hatmax = %#v, want module v0.4.0", inventory.Module.Hatmax)
	}

	if !inventory.Repository.Available || inventory.Repository.Revision == "" || len(inventory.Repository.Dirty) != 0 {
		t.Errorf("Repository = %#v, want clean Git revision", inventory.Repository)
	}

	if len(inventory.Entrypoints) != 1 || inventory.Entrypoints[0].Path != "main.go" || !inventory.Entrypoints[0].CompositionRoot {
		t.Errorf("Entrypoints = %#v, want main.go composition root", inventory.Entrypoints)
	}

	assertSupportedLayouts(t, inventory.Layout)

	for _, command := range []string{"check", "format", "generate", "lint", "sqlc"} {
		if !hasCommand(inventory.Commands, command) {
			t.Errorf("Commands does not contain %q: %#v", command, inventory.Commands)
		}
	}

	if !hasString(inventory.Rules.Instructions, "AGENTS.md") {
		t.Errorf("Rules.Instructions = %v, want AGENTS.md", inventory.Rules.Instructions)
	}

	for _, path := range []string{".git", "internal/dal/generated.go"} {
		if !hasProtectedPath(inventory.Rules.ProtectedPaths, path) {
			t.Errorf("Rules.ProtectedPaths does not contain %q: %#v", path, inventory.Rules.ProtectedPaths)
		}
	}

	selectedBook, err := book.LoadDefault()
	if err != nil {
		t.Fatalf("book.LoadDefault() error = %v", err)
	}

	compatibility, err := inventory.CompatibilityWith(selectedBook)
	if err != nil {
		t.Fatalf("CompatibilityWith() error = %v", err)
	}

	if compatibility != BookCompatibilityCompatible {
		t.Errorf("CompatibilityWith() = %q, want %q", compatibility, BookCompatibilityCompatible)
	}
}

func TestInspectCompatibilityStates(t *testing.T) {
	selectedBook, err := book.LoadDefault()
	if err != nil {
		t.Fatalf("book.LoadDefault() error = %v", err)
	}

	tests := []struct {
		fixture           string
		wantSource        HatmaxSource
		wantEntrypoints   int
		wantCompatibility BookCompatibility
	}{
		{
			fixture:           "incompatible",
			wantSource:        HatmaxSourceModule,
			wantEntrypoints:   1,
			wantCompatibility: BookCompatibilityIncompatible,
		},
		{
			fixture:           "incomplete",
			wantSource:        HatmaxSourceMissing,
			wantEntrypoints:   0,
			wantCompatibility: BookCompatibilityUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.fixture, func(t *testing.T) {
			inventory, inspectErr := Inspect(context.Background(), copyFixture(t, test.fixture))
			if inspectErr != nil {
				t.Fatalf("Inspect() error = %v", inspectErr)
			}

			if inventory.Module.Hatmax.Source != test.wantSource {
				t.Errorf("Hatmax.Source = %q, want %q", inventory.Module.Hatmax.Source, test.wantSource)
			}

			if len(inventory.Entrypoints) != test.wantEntrypoints {
				t.Errorf("len(Entrypoints) = %d, want %d", len(inventory.Entrypoints), test.wantEntrypoints)
			}

			compatibility, compatibilityErr := inventory.CompatibilityWith(selectedBook)
			if compatibilityErr != nil {
				t.Fatalf("CompatibilityWith() error = %v", compatibilityErr)
			}

			if compatibility != test.wantCompatibility {
				t.Errorf("CompatibilityWith() = %q, want %q", compatibility, test.wantCompatibility)
			}
		})
	}
}

func TestInspectLocalWorkspace(t *testing.T) {
	workspace := copyFixture(t, "workspace")
	appRoot := filepath.Join(workspace, "app")

	inventory, err := Inspect(context.Background(), appRoot)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if inventory.Module.Hatmax.Source != HatmaxSourceWorkspace {
		t.Fatalf("Hatmax.Source = %q, want %q", inventory.Module.Hatmax.Source, HatmaxSourceWorkspace)
	}

	if inventory.Module.Hatmax.LocalPath != filepath.Join(workspace, "hatmax") {
		t.Errorf("Hatmax.LocalPath = %q, want workspace Hatmax path", inventory.Module.Hatmax.LocalPath)
	}

	if inventory.Module.GoWorkPath != filepath.Join(workspace, "go.work") {
		t.Errorf("Module.GoWorkPath = %q, want workspace go.work", inventory.Module.GoWorkPath)
	}
}

func TestInspectLocalReplacement(t *testing.T) {
	root := filepath.Join(t.TempDir(), "app")
	hatmaxRoot := filepath.Join(filepath.Dir(root), "hatmax")
	writeProjectFile(t, root, "go.mod", `module example.com/replaced

go 1.24.0

require hatmax.adrianpk.com v0.4.0

replace hatmax.adrianpk.com => ../hatmax
`)
	writeProjectFile(t, hatmaxRoot, "go.mod", "module hatmax.adrianpk.com\n\ngo 1.24.0\n")

	inventory, err := Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if inventory.Module.Hatmax.Source != HatmaxSourceLocalReplace {
		t.Fatalf("Hatmax.Source = %q, want %q", inventory.Module.Hatmax.Source, HatmaxSourceLocalReplace)
	}

	if inventory.Module.Hatmax.LocalPath != hatmaxRoot {
		t.Errorf("Hatmax.LocalPath = %q, want %q", inventory.Module.Hatmax.LocalPath, hatmaxRoot)
	}
}

func TestInspectDirtyOverlap(t *testing.T) {
	root := copyFixture(t, "dirty")
	initializeGit(t, root)
	appendProjectFile(t, root, "internal/feat/property/model.go", "\n// changed\n")
	appendProjectFile(t, root, "README.md", "\nChanged.\n")

	inventory, err := Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	overlap, err := inventory.DirtyOverlap([]string{"internal/feat/property"})
	if err != nil {
		t.Fatalf("DirtyOverlap() error = %v", err)
	}

	if len(overlap) != 1 || overlap[0].Path != "internal/feat/property/model.go" {
		t.Errorf("DirtyOverlap() = %#v, want only property model", overlap)
	}

	modelDirty, err := inventory.DirtyForSurfaces([]string{"model"})
	if err != nil {
		t.Fatalf("DirtyForSurfaces() error = %v", err)
	}

	if len(modelDirty) != 1 || modelDirty[0].Path != "internal/feat/property/model.go" {
		t.Errorf("DirtyForSurfaces() = %#v, want only property model", modelDirty)
	}
}

func TestInspectRejectsInvalidProjects(t *testing.T) {
	missingModule := t.TempDir()

	_, err := Inspect(context.Background(), missingModule)
	requireProjectError(t, err, "project_go_module_missing")

	invalidModule := t.TempDir()
	writeProjectFile(t, invalidModule, "go.mod", "not a module\n")

	_, err = Inspect(context.Background(), invalidModule)
	requireProjectError(t, err, "project_go_module_invalid")

	_, err = InspectWithOptions(context.Background(), copyFixture(t, "incomplete"), Options{MaximumFiles: -1})
	requireProjectError(t, err, "project_invalid_options")

	_, err = InspectWithOptions(context.Background(), copyFixture(t, "supported"), Options{MaximumFiles: 1})
	requireProjectError(t, err, "project_limit_exceeded")

	fileRoot := filepath.Join(t.TempDir(), "file")

	err = os.WriteFile(fileRoot, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("write root file: %v", err)
	}

	_, err = Inspect(context.Background(), fileRoot)
	requireProjectError(t, err, "project_root_invalid")
}

func assertSupportedLayouts(t *testing.T, layout Layout) {
	t.Helper()

	wants := []struct {
		name   string
		values []string
		path   string
	}{
		{name: "features", values: layout.Features, path: "internal/feat/property"},
		{name: "assets", values: layout.Assets, path: "assets"},
		{name: "migrations", values: layout.Migrations, path: "db/migrations"},
		{name: "templates", values: layout.Templates, path: "assets/templates"},
		{name: "queries", values: layout.Queries, path: "internal/dal/queries"},
		{name: "generated", values: layout.Generated, path: "internal/dal"},
	}

	for _, want := range wants {
		if !hasString(want.values, want.path) {
			t.Errorf("Layout.%s = %v, want %q", want.name, want.values, want.path)
		}
	}
}
