// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

var applicationFoundationOrder = []string{
	applicationModuleRecipe,
	applicationChecksumsRecipe,
	applicationIgnoreRecipe,
	applicationCommandsRecipe,
	applicationMainRecipe,
	applicationConfigFileRecipe,
	applicationConfigRecipe,
	applicationLoggingRecipe,
	applicationDatabaseRecipe,
	applicationCompositionRecipe,
	applicationTestsRecipe,
}

func TestRenderApplicationFoundationUsesCanonicalIdentityAndBoundaries(t *testing.T) {
	value := applicationRenderPlan(t)
	manifest := applicationFoundationManifest(t, value)

	mutations, err := RenderApplicationFoundation(value, manifest)
	if err != nil {
		t.Fatalf("RenderApplicationFoundation() error = %v", err)
	}

	if len(mutations) != len(applicationFoundationOrder) {
		t.Fatalf("mutations = %d, want %d", len(mutations), len(applicationFoundationOrder))
	}

	contents := mutationContents(manifest, mutations)
	assertContains(t, contents["go.mod"], "module example.com/alex/real-estate", "go 1.26.0", "hatmax.adrianpk.com v0.5.0")
	assertContains(t, contents["main.go"], "func main()", "application.Run(context.Background(), os.Args)")

	if strings.Contains(contents["main.go"], "func build") || strings.Count(contents["main.go"], "func ") != 1 {
		t.Errorf("main.go must contain only main:\n%s", contents["main.go"])
	}

	assertContains(t, contents["internal/application/database.go"], "db.New(assets, db.Postgres, cfg, logger)")
	assertContains(t, contents["internal/application/application.go"], "app.Setup", "app.Start", "app.Shutdown")

	for path, content := range contents {
		if strings.Contains(content, "migration") || strings.Contains(path, "migration") {
			t.Errorf("foundation unexpectedly activates migrations in %s", path)
		}
	}
}

func TestRenderedApplicationFoundationCompilesWithWebBoundary(t *testing.T) {
	value := applicationRenderPlan(t)
	manifest := applicationFoundationManifest(t, value)

	mutations, err := RenderApplicationFoundation(value, manifest)
	if err != nil {
		t.Fatalf("RenderApplicationFoundation() error = %v", err)
	}

	root := t.TempDir()
	writeApplicationMutations(t, root, manifest, mutations)
	writeExecutionFile(t, root, "internal/application/assets/stub.txt", "stub\n")
	writeExecutionFile(t, root, "internal/application/web.go", `package application

import (
	"embed"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/log"
)

//go:embed assets
var assetsFS embed.FS

func newWeb(log.Logger) (chi.Router, []any) {
	return chi.NewRouter(), nil
}
`)

	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	goModPath := filepath.Join(root, "go.mod")

	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("read generated go.mod: %v", err)
	}

	goMod = append(goMod, []byte("\nreplace hatmax.adrianpk.com => "+repositoryRoot+"\n")...)

	err = os.WriteFile(goModPath, goMod, 0o600)
	if err != nil {
		t.Fatalf("write generated go.mod: %v", err)
	}

	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = root

	command.Env = append(os.Environ(), "GOWORK=off")

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("generated foundation does not compile: %v\n%s", err, output)
	}
}

func TestRenderApplicationFoundationRejectsRecipeDrift(t *testing.T) {
	value := applicationRenderPlan(t)
	manifest := applicationFoundationManifest(t, value)
	manifest.Edits[0].Target = "alternate.mod"
	manifest.Digest = ""

	manifest, err := Seal(manifest)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	_, err = RenderApplicationFoundation(value, manifest)
	requireExecutionCode(t, err, "execution_recipe_target_invalid")
}

func applicationRenderPlan(t *testing.T) plan.Plan {
	t.Helper()

	result, _, _ := applicationExecutionPlan(t, nil)

	return result
}

func applicationExecutionPlan(
	t *testing.T,
	prepare func(string),
) (plan.Plan, project.TargetInventory, *book.Book) {
	t.Helper()

	selectedBook, err := book.LoadRelease(2)
	if err != nil {
		t.Fatalf("book.LoadRelease(2) error = %v", err)
	}

	value := intent.Intent{
		SchemaVersion: intent.ApplicationSchemaVersion,
		Operation:     intent.OperationCreateApplication,
		HatmaxVersion: "0.5.0",
		BookVersion:   2,
		Archetype:     "server_rendered_hatmax_application",
		Capabilities:  []string{},
		Documentation: intent.DocumentationNotRequested,
		Application: &intent.ApplicationIdentity{
			DisplayName: "Real Estate", ProjectSlug: "real-estate", ModulePath: "example.com/alex/real-estate",
		},
		Target: &intent.ApplicationTarget{Base: "session_directory", Directory: "real-estate"},
	}

	plannedPaths, err := plan.ApplicationTargetPaths(value, selectedBook)
	if err != nil {
		t.Fatalf("plan.ApplicationTargetPaths() error = %v", err)
	}

	parent := t.TempDir()

	targetPath := filepath.Join(parent, "real-estate")
	if prepare != nil {
		prepare(targetPath)
	}

	target, err := project.InspectTarget(context.Background(), project.TargetRequest{
		Parent: parent, Target: targetPath, PlannedPaths: plannedPaths, Book: selectedBook,
	})
	if err != nil {
		t.Fatalf("project.InspectTarget() error = %v", err)
	}

	fingerprint, err := target.Fingerprint(2)
	if err != nil {
		t.Fatalf("Target.Fingerprint() error = %v", err)
	}

	value.SourceFingerprint = fingerprint.Value

	admission := intent.Validate(value, intent.ValidationContext{Target: &target, Fingerprint: fingerprint, Book: selectedBook})
	if !admission.Admitted() {
		t.Fatalf("intent validation status = %q, diagnostics = %#v", admission.Status, admission.Diagnostics)
	}

	result, err := plan.Expand(admission, plan.ExpansionContext{Book: selectedBook, Target: &target, Fingerprint: fingerprint})
	if err != nil {
		t.Fatalf("plan.Expand() error = %v", err)
	}

	return result, target, selectedBook
}

func applicationFoundationManifest(t *testing.T, value plan.Plan) Manifest {
	t.Helper()

	return applicationManifestForRecipes(t, value, applicationFoundationOrder)
}

func applicationOperationForPath(t *testing.T, value plan.Plan, target string) plan.Operation {
	t.Helper()

	for _, operation := range value.Operations {
		for _, effect := range operation.Files {
			if effect.Path == target {
				return operation
			}
		}
	}

	t.Fatalf("plan has no operation for %s", target)

	return plan.Operation{}
}

func mutationContents(manifest Manifest, mutations []Mutation) map[string]string {
	edits := make(map[string]Edit, len(manifest.Edits))
	for _, edit := range manifest.Edits {
		edits[edit.ID] = edit
	}

	result := make(map[string]string, len(mutations))
	for _, mutation := range mutations {
		result[edits[mutation.EditID].Target] = string(mutation.Content)
	}

	return result
}

func writeApplicationMutations(t *testing.T, root string, manifest Manifest, mutations []Mutation) {
	t.Helper()

	edits := make(map[string]Edit, len(manifest.Edits))
	for _, edit := range manifest.Edits {
		edits[edit.ID] = edit
	}

	for _, mutation := range mutations {
		writeExecutionFile(t, root, edits[mutation.EditID].Target, string(mutation.Content))
	}
}

func assertContains(t *testing.T, content string, expected ...string) {
	t.Helper()

	for _, value := range expected {
		if !strings.Contains(content, value) {
			t.Errorf("content does not contain %q:\n%s", value, content)
		}
	}
}
