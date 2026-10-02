// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

//go:build acceptance

package execute

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Published consumer checks must resolve the corrected module graph without
// local replacements and leave a reproducible readonly test surface.
func TestApplicationScaffoldAcceptance(t *testing.T) {
	root := publishedScaffold(t)
	assertPublishedModules(t, root)
	publishedCommand(t, root, "go", "test", "-mod=readonly", "./...")
}

// Scan the generated application, not Hatmax's checkout: their module graphs
// differ while the scaffold uses the existing published Hatmax release.
func TestScaffoldSecurity(t *testing.T) {
	_, err := exec.LookPath("govulncheck")
	if err != nil {
		t.Fatal("govulncheck is required for scaffold security acceptance")
	}

	root := publishedScaffold(t)
	assertPublishedModules(t, root)
	output := publishedCommand(t, root, "govulncheck", "./...")
	t.Log(strings.TrimSpace(string(output)))
}

func publishedScaffold(t *testing.T) string {
	t.Helper()

	value, target, selectedBook := applicationExecutionPlan(t, nil)

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

	result, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v, result = %#v", err, result)
	}

	if result.Status != ExecutionApplied || len(result.Validation) != 3 {
		t.Fatalf("Commit() = %#v, want applied with module, build, and test validation", result)
	}

	for _, validation := range result.Validation {
		if validation.Status != ValidationCommandPassed {
			t.Errorf("validation %s = %q, want passed", validation.Name, validation.Status)
		}
	}

	return target.Target
}

func publishedCommand(t *testing.T, root, name string, args ...string) []byte {
	t.Helper()

	command := exec.CommandContext(t.Context(), name, args...)
	command.Dir = root

	command.Env = append(os.Environ(), "GOWORK=off")

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("published scaffold %s %v: %v\n%s", name, args, err, output)
	}

	return output
}

func assertPublishedModules(t *testing.T, root string) {
	t.Helper()

	output := publishedCommand(t, root, "go", "list", "-mod=readonly", "-m", "-json", "all")
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	expected := map[string]string{
		"hatmax.adrianpk.com":     "v0.5.0",
		"github.com/jackc/pgx/v5": "v5.11.0",
		"golang.org/x/text":       "v0.42.0",
	}

	for {
		var module struct {
			Path    string
			Version string
			Replace *struct{}
		}

		err := decoder.Decode(&module)
		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		if module.Replace != nil {
			t.Fatalf("published scaffold replaces %s", module.Path)
		}

		if version, exists := expected[module.Path]; exists {
			if module.Version != version {
				// Continue through the graph to report each incorrect requirement.
				t.Errorf("%s resolves to %s; want %s", module.Path, module.Version, version)
			}

			delete(expected, module.Path)
		}
	}

	if len(expected) != 0 {
		t.Fatalf("published scaffold lacks required modules: %v", expected)
	}
}
