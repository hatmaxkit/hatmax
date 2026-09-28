package interaction

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/execute"
	"hatmax.adrianpk.com/generator/intent"
)

func TestCoordinatorRunsEveryCanonicalOperationWithoutGitEffects(t *testing.T) {
	installFixtureCommands(t, false)

	root := copySupportedProject(t)
	writeCanonicalCompositionRoot(t, root)

	operations := []intent.Operation{
		intent.OperationCreateFeature,
		intent.OperationAddField,
		intent.OperationAddValidation,
	}

	for _, operation := range operations {
		coordinator := newTestCoordinator(t, operationInterpreter(operation), approvingPort(), nil)
		result := coordinator.Run(context.Background(), root, "Apply the next canonical invoice change.")

		if result.Outcome != OutcomeCompleted || result.State != StateFinished {
			t.Fatalf("Run(%q) = %#v, want completed", operation, result)
		}

		if result.Plan == nil || result.Plan.Intent != operation {
			t.Fatalf("Run(%q) plan = %#v, want matching operation", operation, result.Plan)
		}

		if result.Execution == nil || result.Execution.Status != execute.ExecutionApplied {
			t.Fatalf("Run(%q) execution = %#v, want applied", operation, result.Execution)
		}

		if result.Report == nil || !result.Report.Conformance.Passed || len(result.Report.Commands) == 0 {
			t.Fatalf("Run(%q) report = %#v, want conformance and command evidence", operation, result.Report)
		}

		if len(result.RetainedChanges) == 0 {
			t.Fatalf("Run(%q) retained no applied changes", operation)
		}
	}

	_, err := os.Stat(filepath.Join(root, ".git"))
	if !os.IsNotExist(err) {
		t.Fatalf("generator created or changed Git state: %v", err)
	}
}

func TestCoordinatorRunsPureDocumentationWithoutImplementationEffects(t *testing.T) {
	installFixtureCommands(t, false)

	root := copySupportedProject(t)
	writeCanonicalCompositionRoot(t, root)
	appendFixtureFile(t, root, "Makefile", "\ndocs-check:\n\t@echo docs\n")
	createCoordinator := newTestCoordinator(t, operationInterpreter(intent.OperationCreateFeature), approvingPort(), nil)

	created := createCoordinator.Run(context.Background(), root, "Create an invoice feature.")
	if created.Outcome != OutcomeCompleted {
		t.Fatalf("create prerequisite = %#v, want completed", created)
	}

	interpreter := &functionInterpreter{interpret: func(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
		value := baseIntent(request, intent.OperationDocumentFeature)
		value.Capabilities = []string{"postgres_persistence"}
		value.Documentation = intent.DocumentationExisting
		value.DocumentationTargets = []intent.DocumentationTarget{{
			Quadrant:   intent.DocumentationReference,
			Subject:    "invoice",
			ReaderGoal: "Find the exact invoice contract.",
		}}

		return intentInterpreterResult(value), nil
	}}
	coordinator := newTestCoordinator(t, interpreter, approvingPort(), nil)

	result := coordinator.Run(context.Background(), root, "Document the invoice reference.")
	if result.Outcome != OutcomeCompleted || result.Manifest == nil || result.Execution == nil {
		t.Fatalf("Run() = %#v, want completed documentation interaction", result)
	}

	for _, edit := range result.Manifest.Edits {
		if edit.Surface != "documentation" {
			t.Errorf("manifest edit %#v exceeds documentation-only scope", edit)
		}
	}

	for _, target := range []string{"docs/index.md", "docs/reference/index.md", "docs/reference/invoice/index.md"} {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(target)))
		if err != nil {
			t.Errorf("documentation target %q: %v", target, err)
		}
	}
}

func TestCoordinatorRunsCombinedImplementationAndDocumentationUnderOneApproval(t *testing.T) {
	installFixtureCommands(t, false)

	root := copySupportedProject(t)
	writeCanonicalCompositionRoot(t, root)
	appendFixtureFile(t, root, "Makefile", "\ndocs-check:\n\t@echo docs\n")

	interpreter := &functionInterpreter{interpret: func(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
		value := createFeatureIntent(request)
		value.Documentation = intent.DocumentationPlanned
		value.DocumentationTargets = []intent.DocumentationTarget{{
			Quadrant:   intent.DocumentationReference,
			Subject:    "invoice",
			ReaderGoal: "Find the exact invoice contract.",
		}}

		return intentInterpreterResult(value), nil
	}}
	approver := approvingPort()
	coordinator := newTestCoordinator(t, interpreter, approver, nil)

	result := coordinator.Run(context.Background(), root, "Create and document an invoice feature.")
	if result.Outcome != OutcomeCompleted || result.Report == nil || !result.Report.Conformance.Passed {
		t.Fatalf("Run() = %#v, want completed combined interaction", result)
	}

	if approver.calls != 1 {
		t.Fatalf("approval calls = %d, want one combined approval", approver.calls)
	}

	_, err := os.Stat(filepath.Join(root, "internal", "feat", "invoice", "model.go"))
	if err != nil {
		t.Errorf("generated implementation: %v", err)
	}

	_, err = os.Stat(filepath.Join(root, "docs", "reference", "invoice", "index.md"))
	if err != nil {
		t.Errorf("generated documentation: %v", err)
	}

	changed := make(map[string]bool)

	for _, change := range result.Execution.Changes {
		if change.Status == execute.ChangeApplied {
			for _, edit := range result.Manifest.Edits {
				if edit.ID == change.EditID {
					changed[edit.Surface] = true
				}
			}
		}
	}

	if !changed["model"] || !changed["documentation"] {
		t.Errorf("changed surfaces = %v, want implementation and documentation", changed)
	}
}

func TestCoordinatorRetainsChangesAndEvidenceAfterValidationFailure(t *testing.T) {
	installFixtureCommands(t, true)

	root := copySupportedProject(t)
	writeCanonicalCompositionRoot(t, root)
	coordinator := newTestCoordinator(t, operationInterpreter(intent.OperationCreateFeature), approvingPort(), nil)

	result := coordinator.Run(context.Background(), root, "Add invoices.")
	if result.Outcome != OutcomeExecutionFailed || result.State != StateFinished {
		t.Fatalf("Run() = %#v, want post-commit validation failure", result)
	}

	if result.Execution == nil || result.Execution.Status != execute.ExecutionApplied {
		t.Fatalf("execution = %#v, want successful source commit", result.Execution)
	}

	if len(result.RetainedChanges) == 0 {
		t.Fatal("post-commit failure omitted retained changes")
	}

	if result.Report == nil || len(result.Report.Commands) == 0 {
		t.Fatal("post-commit failure omitted command evidence")
	}

	failedCommand := result.Report.Commands[len(result.Report.Commands)-1]
	if failedCommand.Name != "validation.check" || failedCommand.ExitCode != 7 {
		t.Fatalf("failed command = %#v, want exact check failure", failedCommand)
	}

	_, err := os.Stat(filepath.Join(root, "internal", "feat", "invoice", "model.go"))
	if err != nil {
		t.Fatalf("retained source is missing: %v", err)
	}
}

func TestCoordinatorLeavesProjectUnchangedWhenRenderingFails(t *testing.T) {
	installFixtureCommands(t, false)

	root := copySupportedProject(t)

	before, err := os.ReadFile(filepath.Join(root, "internal", "feat", "property", "model.go"))
	if err != nil {
		t.Fatalf("read model before interaction: %v", err)
	}

	interpreter := &functionInterpreter{interpret: func(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
		value := incrementalIntent(request, intent.OperationAddValidation)
		value.Feature = "property"
		value.Capabilities = []string{"postgres_persistence", "runtime_validation"}
		value.Domain.Validation = &intent.ValidationRule{
			Field: "missing", Kind: "required", Scope: intent.ValidationDurable,
		}

		return intentInterpreterResult(value), nil
	}}
	coordinator := newTestCoordinator(t, interpreter, approvingPort(), nil)

	result := coordinator.Run(context.Background(), root, "Require a missing field.")
	if result.Outcome != OutcomeExecutionFailed || len(result.RetainedChanges) != 0 {
		t.Fatalf("Run() = %#v, want pre-mutation rendering failure", result)
	}

	after, err := os.ReadFile(filepath.Join(root, "internal", "feat", "property", "model.go"))
	if err != nil {
		t.Fatalf("read model after interaction: %v", err)
	}

	if string(after) != string(before) {
		t.Fatal("rendering failure changed the project")
	}
}

func operationInterpreter(operation intent.Operation) *functionInterpreter {
	return &functionInterpreter{interpret: func(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
		var value intent.Intent

		switch operation {
		case intent.OperationCreateFeature:
			value = createFeatureIntent(request)
		case intent.OperationAddField:
			value = incrementalIntent(request, operation)
			value.Domain.Field = &intent.Field{Name: "due_on", Type: "date", Label: "Due on"}
		case intent.OperationAddValidation:
			value = incrementalIntent(request, operation)
			value.Capabilities = []string{"postgres_persistence", "runtime_validation"}
			value.Domain.Validation = &intent.ValidationRule{
				Field: "number", Kind: "min_length", Value: "3",
				Message: "Enter at least three characters.", Scope: intent.ValidationDurable,
			}
		}

		return intentInterpreterResult(value), nil
	}}
}

func createFeatureIntent(request eval.Request) intent.Intent {
	value := baseIntent(request, intent.OperationCreateFeature)
	value.Domain = intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{
			{Name: "number", Type: "string", Label: "Number", Required: true},
			{Name: "notes", Type: "text", Label: "Notes"},
		},
	}
	value.Capabilities = []string{"postgres_persistence", "runtime_validation", "htmx_form"}

	return value
}

func incrementalIntent(request eval.Request, operation intent.Operation) intent.Intent {
	value := baseIntent(request, operation)
	value.Capabilities = []string{"postgres_persistence"}

	return value
}

func baseIntent(request eval.Request, operation intent.Operation) intent.Intent {
	return intent.Intent{
		SchemaVersion:      intent.CurrentSchemaVersion,
		Operation:          operation,
		ProjectFingerprint: request.Project.Fingerprint,
		HatmaxVersion:      request.Project.HatmaxVersion,
		BookVersion:        request.Book.Version,
		Archetype:          "server_rendered_crud",
		Feature:            "invoice",
		Documentation:      intent.DocumentationNotRequested,
		Exceptions:         []intent.Exception{},
	}
}

func intentInterpreterResult(value intent.Intent) eval.InterpreterResult {
	return fixtureInterpreterResult(eval.Interpretation{
		SchemaVersion: eval.CurrentInterpretationSchemaVersion,
		Kind:          eval.InterpretationIntent,
		Intent:        &value,
	})
}

func installFixtureCommands(t *testing.T, failValidation bool) {
	t.Helper()

	directory := t.TempDir()

	makeSource := "#!/bin/sh\nexit 0\n"
	if failValidation {
		makeSource = "#!/bin/sh\nif [ \"$1\" = \"check\" ]; then\n  echo validation failed\n  exit 7\nfi\nexit 0\n"
	}

	writeExecutable(t, filepath.Join(directory, "make"), makeSource)
	writeExecutable(t, filepath.Join(directory, "sqlc"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0o755)
	if err != nil {
		t.Fatalf("write fixture executable: %v", err)
	}
}

func writeCanonicalCompositionRoot(t *testing.T, root string) {
	t.Helper()

	const source = `package main

import (
	"context"

	"hatmax.adrianpk.com/app"
)

func main() {
	logger := buildLogger()
	database := buildDatabase()
	migrator := buildMigrator()
	tmplMgr := buildTemplates()
	deps := []any{database, migrator, tmplMgr}
app.Setup(context.Background(), nil, deps...)
}
`

	err := os.WriteFile(filepath.Join(root, "main.go"), []byte(source), 0o644)
	if err != nil {
		t.Fatalf("write canonical composition root: %v", err)
	}
}

func appendFixtureFile(t *testing.T, root, relative, content string) {
	t.Helper()

	file, err := os.OpenFile(filepath.Join(root, filepath.FromSlash(relative)), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open fixture file: %v", err)
	}

	_, writeErr := file.WriteString(content)
	closeErr := file.Close()

	if writeErr != nil {
		t.Fatalf("append fixture file: %v", writeErr)
	}

	if closeErr != nil {
		t.Fatalf("close fixture file: %v", closeErr)
	}
}
