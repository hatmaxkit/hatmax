package execute

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/plan"
)

var applicationWebOrder = []string{
	applicationWebRecipe,
	applicationLayoutRecipe,
	applicationHomeRecipe,
	applicationStylesRecipe,
	applicationWebTestsRecipe,
}

func TestRenderApplicationWebUsesHatmaxPrimitivesAndNeutralContent(t *testing.T) {
	value := applicationRenderPlan(t)
	manifest := applicationManifestForRecipes(t, value, applicationWebOrder)

	mutations, err := RenderApplicationWeb(value, manifest)
	if err != nil {
		t.Fatalf("RenderApplicationWeb() error = %v", err)
	}

	contents := mutationContents(manifest, mutations)
	assertContains(
		t,
		contents["internal/application/web.go"],
		"app.NewRouter",
		"middleware.DefaultStack()",
		"web.NewTemplateManager",
		"RegisterRoutes(router chi.Router)",
	)
	assertContains(t, contents["internal/application/assets/templates/pages/home.html"], "Hatmax application", "{{.ApplicationName}}")
	assertContains(t, contents["internal/application/web_test.go"], "TestNeutralWebSurface", "httptest.NewRequest")

	for path, content := range contents {
		for _, prohibited := range []string{"type Customer", "type Invoice", "application/json", "React", "Vue"} {
			if strings.Contains(content, prohibited) {
				t.Errorf("neutral scaffold contains %q in %s", prohibited, path)
			}
		}
	}
}

func TestRenderedApplicationCompilesAndPassesGeneratedTests(t *testing.T) {
	value := applicationRenderPlan(t)
	manifest := applicationManifestForRecipes(t, value, append(applicationFoundationOrder, applicationWebOrder...))

	mutations, err := RenderApplication(value, manifest)
	if err != nil {
		t.Fatalf("RenderApplication() error = %v", err)
	}

	root := t.TempDir()
	writeApplicationMutations(t, root, manifest, mutations)

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
		t.Fatalf("generated application validation failed: %v\n%s", err, output)
	}
}

func applicationManifestForRecipes(t *testing.T, value plan.Plan, recipeIDs []string) Manifest {
	t.Helper()

	edits := make([]Edit, 0, len(recipeIDs))
	for index, recipeID := range recipeIDs {
		_, recipe, exists := applicationRecipeForPath(applicationRecipeTarget(t, recipeID))
		if !exists {
			t.Fatalf("application recipe %q does not exist", recipeID)
		}

		operation := applicationOperationForPath(t, value, recipe.target)
		surface := operation.Surfaces[0]

		postconditions := []Condition{{Kind: ConditionPathPresent}}
		if recipe.goSource {
			postconditions = append(postconditions, Condition{Kind: ConditionGoParses})
		}

		dependsOn := []string{}
		if index > 0 {
			dependsOn = []string{edits[index-1].ID}
		}

		edits = append(edits, Edit{
			ID:   "application.scaffold." + strings.ReplaceAll(recipeID, "application.", ""),
			Kind: EditCreateFile, Surface: surface, Target: recipe.target, Recipe: recipeID,
			Obligations: []Obligation{{Operation: operation.ID, Owner: operation.Owner, Rules: append([]string{}, operation.Rules...)}},
			DependsOn:   dependsOn, Preconditions: []Condition{{Kind: ConditionPathAbsent}}, Postconditions: postconditions,
		})
	}

	result := Manifest{
		SchemaVersion: CurrentSchemaVersion, PlanDigest: value.Digest, SourceFingerprint: value.SourceFingerprint,
		Intent: value.Intent, AllowedSurfaces: append([]string{}, value.AllowedEffects.Surfaces...), Edits: edits, Commands: []Command{},
	}

	result, err := Seal(result)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	return result
}

func applicationRecipeTarget(t *testing.T, recipeID string) string {
	t.Helper()

	if recipe, exists := applicationFoundationRecipes[recipeID]; exists {
		return recipe.target
	}

	if recipe, exists := applicationWebRecipes[recipeID]; exists {
		return recipe.target
	}

	t.Fatalf("application recipe %q does not exist", recipeID)

	return ""
}
