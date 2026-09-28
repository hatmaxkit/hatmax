package hatmaxcli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/interaction"
)

func TestTerminalSurfaceExecutesCanonicalOperationsWithoutUndeclaredEffects(t *testing.T) {
	installCLICommands(t, false)

	root := copyCLIProject(t)
	writeCLICompositionRoot(t, root)

	interpreter := &cliInterpreter{interpret: interpretCLIOperation}

	requests := []struct {
		prompt    string
		operation intent.Operation
	}{
		{prompt: "Create an invoice feature.", operation: intent.OperationCreateFeature},
		{prompt: "Add a due date to invoices.", operation: intent.OperationAddField},
		{prompt: "Validate invoice numbers.", operation: intent.OperationAddValidation},
	}

	for _, request := range requests {
		before := snapshotCLIProject(t, root)
		exitCode, output, errorOutput, result := runCLIRequest(t, root, "yes\n", request.prompt, interpreter, nil)

		if exitCode != ExitSuccess || result == nil || result.Outcome != interaction.OutcomeCompleted {
			t.Fatalf("generate %q = exit %d, result %#v; want completed", request.prompt, exitCode, result)
		}

		if result.Plan == nil || result.Plan.Intent != request.operation {
			t.Fatalf("generate %q plan = %#v, want %q", request.prompt, result.Plan, request.operation)
		}

		for _, expected := range []string{"Plan:\n", "Approve plan ", "Outcome: completed", "Conformance passed: true"} {
			if !strings.Contains(output, expected) {
				t.Errorf("generate %q output does not contain %q:\n%s", request.prompt, expected, output)
			}
		}

		if errorOutput != "" {
			t.Errorf("generate %q error output = %q, want empty", request.prompt, errorOutput)
		}

		assertDeclaredCLIChanges(t, before, snapshotCLIProject(t, root), result)
	}

	_, err := os.Stat(filepath.Join(root, ".git"))
	if !os.IsNotExist(err) {
		t.Fatalf("terminal generation created Git state: %v", err)
	}
}

func TestTerminalSurfaceReportsPrincipalFailurePaths(t *testing.T) {
	t.Run("approval rejected", func(t *testing.T) {
		installCLICommands(t, false)

		root := copyCLIProject(t)
		writeCLICompositionRoot(t, root)
		before := snapshotCLIProject(t, root)
		exitCode, output, _, result := runCLIRequest(t, root, "no\n", "Create an invoice feature.", &cliInterpreter{interpret: interpretCLIOperation}, nil)

		if exitCode != ExitCancelled || result == nil || result.Outcome != interaction.OutcomeCancelled {
			t.Fatalf("rejected request = exit %d, result %#v; want cancelled", exitCode, result)
		}

		if !strings.Contains(output, "Outcome: cancelled") {
			t.Errorf("rejected request omitted outcome:\n%s", output)
		}

		assertEqualCLISnapshots(t, before, snapshotCLIProject(t, root))
	})

	t.Run("unsupported request", func(t *testing.T) {
		root := copyCLIProject(t)
		interpreter := &cliInterpreter{interpret: func(context.Context, eval.Request) (eval.InterpreterResult, error) {
			return cliInterpreterResult(eval.Interpretation{
				SchemaVersion: eval.CurrentInterpretationSchemaVersion,
				Kind:          eval.InterpretationUnsupported,
				Diagnostics: []intent.Diagnostic{{
					Code: "CAPABILITY-UNSUPPORTED", Field: "request", Message: "The requested capability is outside the Hatmax Book.",
				}},
			}), nil
		}}

		exitCode, output, _, result := runCLIRequest(t, root, "", "Add a message broker.", interpreter, nil)
		if exitCode != ExitUnsupported || result == nil || result.Outcome != interaction.OutcomeUnsupported {
			t.Fatalf("unsupported request = exit %d, result %#v; want unsupported", exitCode, result)
		}

		if !strings.Contains(output, "CAPABILITY-UNSUPPORTED") || !strings.Contains(output, "outside the Hatmax Book") {
			t.Errorf("unsupported request omitted bounded diagnostic:\n%s", output)
		}
	})

	t.Run("backend unavailable", func(t *testing.T) {
		root := copyCLIProject(t)
		interpreter := &cliInterpreter{interpret: func(context.Context, eval.Request) (eval.InterpreterResult, error) {
			return eval.InterpreterResult{}, eval.BackendError{
				Code: eval.BackendUnavailable, Operation: "turn_start", Message: "resident backend is unavailable",
			}
		}}

		exitCode, output, _, result := runCLIRequest(t, root, "", "Create an invoice feature.", interpreter, nil)
		if exitCode != ExitFailure || result == nil || result.Outcome != interaction.OutcomeFailed {
			t.Fatalf("backend failure = exit %d, result %#v; want failed", exitCode, result)
		}

		if !strings.Contains(output, "HMGEN-BACKEND-UNAVAILABLE") || !strings.Contains(output, "resident backend is unavailable") {
			t.Errorf("backend failure omitted diagnostic:\n%s", output)
		}
	})

	t.Run("approved plan becomes stale", func(t *testing.T) {
		installCLICommands(t, false)

		root := copyCLIProject(t)
		writeCLICompositionRoot(t, root)

		wrapApprover := func(approver interaction.Approver) interaction.Approver {
			return &mutatingCLIApprover{delegate: approver, mutate: func() error {
				path := filepath.Join(root, "main.go")

				file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
				if err != nil {
					return err
				}

				_, writeErr := file.WriteString("\n// concurrent project change\n")
				closeErr := file.Close()

				if writeErr != nil {
					return writeErr
				}

				return closeErr
			}}
		}

		exitCode, output, _, result := runCLIRequest(t, root, "yes\n", "Create an invoice feature.", &cliInterpreter{interpret: interpretCLIOperation}, wrapApprover)
		if exitCode != ExitPlanStale || result == nil || result.Outcome != interaction.OutcomePlanStale {
			t.Fatalf("stale request = exit %d, result %#v; want plan_stale", exitCode, result)
		}

		if !strings.Contains(output, "Outcome: plan_stale") || !strings.Contains(output, "Project drift:") {
			t.Errorf("stale request omitted drift evidence:\n%s", output)
		}
	})

	t.Run("repository validation fails", func(t *testing.T) {
		installCLICommands(t, true)

		root := copyCLIProject(t)
		writeCLICompositionRoot(t, root)
		exitCode, output, _, result := runCLIRequest(t, root, "yes\n", "Create an invoice feature.", &cliInterpreter{interpret: interpretCLIOperation}, nil)

		if exitCode != ExitExecutionFailed || result == nil || result.Outcome != interaction.OutcomeExecutionFailed {
			t.Fatalf("validation failure = exit %d, result %#v; want execution_failed", exitCode, result)
		}

		if !strings.Contains(output, "Command validation.check:") || !strings.Contains(output, "exit=7") {
			t.Errorf("validation failure omitted exact command evidence:\n%s", output)
		}
	})
}

type cliInterpreter struct {
	interpret func(context.Context, eval.Request) (eval.InterpreterResult, error)
}

func (interpreter *cliInterpreter) Interpret(ctx context.Context, request eval.Request) (eval.InterpreterResult, error) {
	return interpreter.interpret(ctx, request)
}

type recordingCLIRunner struct {
	delegate Runner
	result   *interaction.Result
}

func (runner *recordingCLIRunner) Run(ctx context.Context, root, prompt string) interaction.Result {
	result := runner.delegate.Run(ctx, root, prompt)
	runner.result = &result

	return result
}

type mutatingCLIApprover struct {
	delegate interaction.Approver
	mutate   func() error
}

func (approver *mutatingCLIApprover) Approve(ctx context.Context, request interaction.ApprovalRequest) (interaction.ApprovalDecision, error) {
	decision, err := approver.delegate.Approve(ctx, request)
	if err != nil || decision != interaction.ApprovalGranted {
		return decision, err
	}

	err = approver.mutate()

	return decision, err
}

func runCLIRequest(
	t *testing.T,
	root string,
	input string,
	prompt string,
	interpreter eval.Interpreter,
	wrapApprover func(interaction.Approver) interaction.Approver,
) (int, string, string, *interaction.Result) {
	t.Helper()

	var (
		output      bytes.Buffer
		errorOutput bytes.Buffer
		recorder    *recordingCLIRunner
	)

	factory := func(_ string, approver interaction.Approver, clarifier interaction.Clarifier) (Runner, error) {
		if wrapApprover != nil {
			approver = wrapApprover(approver)
		}

		coordinator, err := interaction.New(interaction.Config{
			Interpreter: interpreter,
			Approver:    approver,
			Clarifier:   clarifier,
		})
		if err != nil {
			return nil, err
		}

		recorder = &recordingCLIRunner{delegate: coordinator}

		return recorder, nil
	}

	app, err := New(Config{
		Input: strings.NewReader(input), Output: &output, ErrorOutput: &errorOutput,
		WorkingDirectory: func() (string, error) { return root, nil }, CoordinatorFactory: factory,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	exitCode := app.Run(context.Background(), []string{"generate", prompt})

	if recorder == nil {
		t.Fatal("coordinator was not assembled")
	}

	return exitCode, output.String(), errorOutput.String(), recorder.result
}

func interpretCLIOperation(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
	var operation intent.Operation

	switch request.Prompt {
	case "Create an invoice feature.":
		operation = intent.OperationCreateFeature
	case "Add a due date to invoices.":
		operation = intent.OperationAddField
	case "Validate invoice numbers.":
		operation = intent.OperationAddValidation
	default:
		return eval.InterpreterResult{}, fmt.Errorf("unexpected test prompt %q", request.Prompt)
	}

	value := cliIntent(request, operation)

	return cliInterpreterResult(eval.Interpretation{
		SchemaVersion: eval.CurrentInterpretationSchemaVersion,
		Kind:          eval.InterpretationIntent,
		Intent:        &value,
	}), nil
}

func cliIntent(request eval.Request, operation intent.Operation) intent.Intent {
	value := intent.Intent{
		SchemaVersion: intent.CurrentSchemaVersion, Operation: operation,
		ProjectFingerprint: request.Project.Fingerprint, HatmaxVersion: request.Project.HatmaxVersion,
		BookVersion: request.Book.Version, Archetype: "server_rendered_crud", Feature: "invoice",
		Capabilities: []string{"postgres_persistence"}, Documentation: intent.DocumentationNotRequested,
		Exceptions: []intent.Exception{},
	}

	switch operation {
	case intent.OperationCreateFeature:
		value.Capabilities = []string{"postgres_persistence", "runtime_validation", "htmx_form"}
		value.Domain = intent.Domain{
			Entity: "Invoice", Route: "/invoices",
			Fields: []intent.Field{
				{Name: "number", Type: "string", Label: "Number", Required: true},
				{Name: "notes", Type: "text", Label: "Notes"},
			},
		}
	case intent.OperationAddField:
		value.Domain.Field = &intent.Field{Name: "due_on", Type: "date", Label: "Due on"}
	case intent.OperationAddValidation:
		value.Capabilities = []string{"postgres_persistence", "runtime_validation"}
		value.Domain.Validation = &intent.ValidationRule{
			Field: "number", Kind: "min_length", Value: "3",
			Message: "Enter at least three characters.", Scope: intent.ValidationDurable,
		}
	}

	return value
}

func cliInterpreterResult(interpretation eval.Interpretation) eval.InterpreterResult {
	return eval.InterpreterResult{
		Interpretation: interpretation,
		Provenance: eval.Provenance{
			Adapter: "fixture", ContractVersion: eval.CurrentContractVersion,
			ModelSelection: eval.ModelNotApplicable, Timing: eval.TimingNotMeasured,
		},
	}
}

func copyCLIProject(t *testing.T) string {
	t.Helper()

	source := filepath.Join("..", "..", "generator", "project", "testdata", "supported")
	target := filepath.Join(t.TempDir(), "supported")

	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destination, content, 0o644)
	})
	if err != nil {
		t.Fatalf("copy supported project: %v", err)
	}

	return target
}

func writeCLICompositionRoot(t *testing.T, root string) {
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

func installCLICommands(t *testing.T, failValidation bool) {
	t.Helper()

	directory := t.TempDir()

	makeSource := "#!/bin/sh\nexit 0\n"
	if failValidation {
		makeSource = "#!/bin/sh\nif [ \"$1\" = \"check\" ]; then\n  echo validation failed\n  exit 7\nfi\nexit 0\n"
	}

	writeCLIExecutable(t, filepath.Join(directory, "make"), makeSource)
	writeCLIExecutable(t, filepath.Join(directory, "sqlc"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeCLIExecutable(t *testing.T, path, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0o755)
	if err != nil {
		t.Fatalf("write fixture executable: %v", err)
	}
}

func snapshotCLIProject(t *testing.T, root string) map[string][sha256.Size]byte {
	t.Helper()

	result := make(map[string][sha256.Size]byte)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		result[filepath.ToSlash(relative)] = sha256.Sum256(content)

		return nil
	})
	if err != nil {
		t.Fatalf("snapshot project: %v", err)
	}

	return result
}

func assertDeclaredCLIChanges(
	t *testing.T,
	before map[string][sha256.Size]byte,
	after map[string][sha256.Size]byte,
	result *interaction.Result,
) {
	t.Helper()

	if result.Manifest == nil {
		t.Fatal("completed interaction omitted execution manifest")
	}

	declared := make(map[string]struct{}, len(result.Manifest.Edits))
	for _, edit := range result.Manifest.Edits {
		declared[edit.Target] = struct{}{}
	}

	for path, digest := range after {
		if previous, exists := before[path]; exists && previous == digest {
			continue
		}

		if _, ok := declared[path]; !ok {
			t.Errorf("project changed undeclared target %q", path)
		}
	}

	for path := range before {
		if _, exists := after[path]; exists {
			continue
		}

		if _, ok := declared[path]; !ok {
			t.Errorf("project removed undeclared target %q", path)
		}
	}
}

func assertEqualCLISnapshots(t *testing.T, before, after map[string][sha256.Size]byte) {
	t.Helper()

	if fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatalf("project changed without approved execution\nbefore: %v\nafter: %v", before, after)
	}
}
