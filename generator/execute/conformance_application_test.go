// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplicationConformanceAcceptsCanonicalScaffold(t *testing.T) {
	value, _, _ := applicationExecutionPlan(t, nil)
	manifest := applicationManifestForRecipes(t, value, append(applicationFoundationOrder, applicationWebOrder...))

	mutations, err := RenderApplication(value, manifest)
	if err != nil {
		t.Fatalf("RenderApplication() error = %v", err)
	}

	root := t.TempDir()
	writeApplicationMutations(t, root, manifest, mutations)

	result, err := CheckApplicationConformance(value, manifest, root)
	if err != nil {
		t.Fatalf("CheckApplicationConformance() error = %v", err)
	}

	if !result.Passed || len(result.Diagnostics) != 0 {
		t.Errorf("conformance = %#v, want passed", result)
	}
}

func TestApplicationConformanceRejectsHelpersInMain(t *testing.T) {
	value, _, _ := applicationExecutionPlan(t, nil)
	manifest := applicationManifestForRecipes(t, value, append(applicationFoundationOrder, applicationWebOrder...))

	mutations, err := RenderApplication(value, manifest)
	if err != nil {
		t.Fatalf("RenderApplication() error = %v", err)
	}

	root := t.TempDir()
	writeApplicationMutations(t, root, manifest, mutations)
	mainPath := filepath.Join(root, "main.go")

	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	content = append(content, []byte("\nfunc buildApplication() {}\n")...)

	err = os.WriteFile(mainPath, content, 0o644)
	if err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	result, err := CheckApplicationConformance(value, manifest, root)
	if err != nil {
		t.Fatalf("CheckApplicationConformance() error = %v", err)
	}

	if result.Passed || !hasDiagnostic(result.Diagnostics, "HMGEN-APPLICATION-THIN-MAIN") {
		t.Errorf("conformance = %#v, want thin-main diagnostic", result)
	}

	if !strings.Contains(string(content), "buildApplication") {
		t.Fatal("test fixture does not contain helper")
	}
}
