// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package hatmaxcli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
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

func TestTerminalSurfaceCreatesTerseAndDetailedApplications(t *testing.T) {
	tests := []struct {
		name      string
		prompt    string
		input     string
		composite bool
		turns     int
	}{
		{name: "terse clarified", prompt: "Create Ledger.", input: "example.com/alex/ledger\nyes\n", turns: 2},
		{name: "detailed composite", prompt: "Create Ledger with an invoice number.", input: "yes\n", composite: true, turns: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			installCLIApplicationCommands(t)
			parent := t.TempDir()
			turns := 0
			interpreter := &cliInterpreter{interpret: func(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
				turns++

				value := cliApplicationIntent(request, test.composite)
				if !test.composite && !request.Target.Resolved {
					value.Application.ModulePath = ""
				}

				if request.Target.Resolved && value.Application.ModulePath == "" {
					value.Application.ModulePath = request.Clarifications[0].Answer
				}

				return cliInterpreterResult(eval.Interpretation{
					SchemaVersion: eval.CurrentInterpretationSchemaVersion,
					Kind:          eval.InterpretationIntent,
					Intent:        &value,
				}), nil
			}}

			exitCode, output, errorOutput, result := runCLIRequest(t, parent, test.input, test.prompt, interpreter, nil)
			if exitCode != ExitSuccess || result == nil || result.Outcome != interaction.OutcomeCompleted {
				t.Fatalf("application request = exit %d result %#v\n%s", exitCode, result, output)
			}

			if turns != test.turns {
				t.Fatalf("interpreter turns = %d, want %d", turns, test.turns)
			}

			for _, expected := range []string{"Intent: create_application", "Application: Ledger", "Source fingerprint:", "Outcome: completed"} {
				if !strings.Contains(output, expected) {
					t.Errorf("output does not contain %q:\n%s", expected, output)
				}
			}

			if errorOutput != "" {
				t.Errorf("error output = %q", errorOutput)
			}

			_, err := os.Stat(filepath.Join(parent, "ledger", "main.go"))
			if err != nil {
				t.Fatalf("generated application: %v", err)
			}

			_, featureErr := os.Stat(filepath.Join(parent, "ledger", "internal", "feat", "invoice", "model.go"))
			if test.composite && featureErr != nil {
				t.Fatalf("generated initial feature: %v", featureErr)
			}

			if !test.composite && !errors.Is(featureErr, os.ErrNotExist) {
				t.Fatalf("terse application invented a feature: %v", featureErr)
			}
		})
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

func TestTerminalSurfaceGeneratesBoxedDocumentation(t *testing.T) {
	installCLICommands(t, false)

	root := copyCLIProject(t)
	writeCLICompositionRoot(t, root)
	appendCLIFile(t, root, "Makefile", "\ndocs-check:\n\t@echo docs\n")

	interpreter := &cliInterpreter{interpret: interpretCLIDocumentation}

	exitCode, _, _, created := runCLIRequest(t, root, "yes\n", "Create an invoice feature.", interpreter, nil)
	if exitCode != ExitSuccess || created == nil || created.Outcome != interaction.OutcomeCompleted {
		t.Fatalf("ordinary generation = exit %d, result %#v; want completed", exitCode, created)
	}

	_, err := os.Stat(filepath.Join(root, "docs"))
	if !os.IsNotExist(err) {
		t.Fatalf("ordinary generation created documentation without intent: %v", err)
	}

	exitCode, output, _, documented := runCLIRequest(t, root, "yes\n", "Document every invoice reader need.", interpreter, nil)
	if exitCode != ExitSuccess || documented == nil || documented.Outcome != interaction.OutcomeCompleted {
		t.Fatalf("documentation generation = exit %d, result %#v; want completed", exitCode, documented)
	}

	for _, target := range []string{
		"docs/tutorials/invoice-basics/README.md",
		"docs/how-to/invoice-workflow/README.md",
		"docs/reference/invoice/README.md",
		"docs/explanation/invoice-ownership/README.md",
	} {
		_, err = os.Stat(filepath.Join(root, filepath.FromSlash(target)))
		if err != nil {
			t.Errorf("generated documentation target %q: %v", target, err)
		}
	}

	for _, expected := range []string{
		"Documentation: document_existing_behavior",
		"tutorial: subject=invoice_basics",
		"how_to: subject=invoice_workflow",
		"reference: subject=invoice",
		"explanation: subject=invoice_ownership",
		"Conformance passed: true",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("documentation output does not contain %q:\n%s", expected, output)
		}
	}

	referencePath := filepath.Join(root, "docs", "reference", "invoice", "README.md")

	reference, err := os.ReadFile(referencePath)
	if err != nil {
		t.Fatalf("read generated reference: %v", err)
	}

	userPrefix := "User introduction.\n\n"
	userSuffix := "\nUser follow-up.\n"

	err = os.WriteFile(referencePath, append(append([]byte(userPrefix), reference...), []byte(userSuffix)...), 0o644)
	if err != nil {
		t.Fatalf("add user documentation: %v", err)
	}

	exitCode, output, _, regenerated := runCLIRequest(t, root, "yes\n", "Document every invoice reader need.", interpreter, nil)
	if exitCode != ExitSuccess || regenerated == nil || regenerated.Outcome != interaction.OutcomeCompleted {
		t.Fatalf("documentation regeneration = exit %d, result %#v; want completed", exitCode, regenerated)
	}

	regeneratedReference, err := os.ReadFile(referencePath)
	if err != nil {
		t.Fatalf("read regenerated reference: %v", err)
	}

	if !strings.HasPrefix(string(regeneratedReference), userPrefix) || !strings.HasSuffix(string(regeneratedReference), userSuffix) {
		t.Fatalf("regeneration did not preserve user-owned bytes: %q", regeneratedReference)
	}

	if !strings.Contains(output, "docs/reference/invoice/README.md: unchanged ownership=outside_content_preserved") {
		t.Errorf("regeneration output omitted preservation evidence:\n%s", output)
	}

	unmanaged := strings.ReplaceAll(string(regeneratedReference), "<!-- hatmax:generated:start -->\n", "")
	unmanaged = strings.ReplaceAll(unmanaged, "<!-- hatmax:generated:end -->", "")

	err = os.WriteFile(referencePath, []byte(unmanaged), 0o644)
	if err != nil {
		t.Fatalf("write unmanaged conflict: %v", err)
	}

	beforeConflict := snapshotCLIProject(t, root)

	exitCode, output, _, conflict := runCLIRequest(t, root, "", "Document every invoice reader need.", interpreter, nil)
	if exitCode != ExitFailure || conflict == nil || conflict.Outcome != interaction.OutcomeFailed {
		t.Fatalf("unmanaged conflict = exit %d, result %#v; want failed", exitCode, conflict)
	}

	if !strings.Contains(output, "plan_documentation_conflict") {
		t.Errorf("unmanaged conflict output omitted exact failure:\n%s", output)
	}

	assertEqualCLISnapshots(t, beforeConflict, snapshotCLIProject(t, root))
}

func TestTerminalSurfaceGeneratesCombinedDocumentationAndReportsGateFailure(t *testing.T) {
	t.Run("combined planned change", func(t *testing.T) {
		installCLICommands(t, false)

		root := copyCLIProject(t)
		writeCLICompositionRoot(t, root)
		appendCLIFile(t, root, "Makefile", "\ndocs-check:\n\t@echo docs\n")
		before := snapshotCLIProject(t, root)

		exitCode, output, _, result := runCLIRequest(
			t,
			root,
			"yes\n",
			"Create and document an invoice feature.",
			&cliInterpreter{interpret: interpretCLIDocumentation},
			nil,
		)
		if exitCode != ExitSuccess || result == nil || result.Outcome != interaction.OutcomeCompleted {
			t.Fatalf("combined generation = exit %d, result %#v; want completed", exitCode, result)
		}

		for _, target := range []string{
			"internal/feat/invoice/model.go",
			"docs/reference/invoice/README.md",
		} {
			_, err := os.Stat(filepath.Join(root, filepath.FromSlash(target)))
			if err != nil {
				t.Errorf("combined target %q: %v", target, err)
			}
		}

		if !strings.Contains(output, "Documentation: document_planned_change") || !strings.Contains(output, "Changed surfaces:") {
			t.Errorf("combined output omitted documentation plan or execution ownership:\n%s", output)
		}

		assertDeclaredCLIChanges(t, before, snapshotCLIProject(t, root), result)
	})

	t.Run("documentation gate failure", func(t *testing.T) {
		installCLIDocumentationCommands(t, true)

		root := copyCLIProject(t)
		writeCLICompositionRoot(t, root)
		appendCLIFile(t, root, "Makefile", "\ndocs-check:\n\t@echo docs\n")

		exitCode, output, _, result := runCLIRequest(
			t,
			root,
			"yes\n",
			"Create and document an invoice feature.",
			&cliInterpreter{interpret: interpretCLIDocumentation},
			nil,
		)
		if exitCode != ExitExecutionFailed || result == nil || result.Outcome != interaction.OutcomeExecutionFailed {
			t.Fatalf("documentation gate failure = exit %d, result %#v; want execution_failed", exitCode, result)
		}

		if !strings.Contains(output, "Command validation.docs-check:") || !strings.Contains(output, "exit=9") {
			t.Errorf("documentation gate failure omitted command evidence:\n%s", output)
		}

		_, err := os.Stat(filepath.Join(root, "docs", "reference", "invoice", "README.md"))
		if err != nil {
			t.Errorf("documentation gate failure removed retained documentation: %v", err)
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

func interpretCLIDocumentation(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
	var value intent.Intent

	switch request.Prompt {
	case "Create an invoice feature.":
		value = cliIntent(request, intent.OperationCreateFeature)
	case "Create and document an invoice feature.":
		value = cliIntent(request, intent.OperationCreateFeature)
		value.Documentation = intent.DocumentationPlanned
		value.DocumentationTargets = []intent.DocumentationTarget{{
			Quadrant: intent.DocumentationReference, Subject: "invoice", ReaderGoal: "Find the exact invoice contract.",
		}}
	case "Document every invoice reader need.":
		value = cliIntent(request, intent.OperationDocumentFeature)
		value.Documentation = intent.DocumentationExisting
		value.DocumentationTargets = []intent.DocumentationTarget{
			{Quadrant: intent.DocumentationTutorial, Subject: "invoice_basics", ReaderGoal: "Learn the invoice workflow."},
			{Quadrant: intent.DocumentationHowTo, Subject: "invoice_workflow", ReaderGoal: "Complete the invoice workflow."},
			{Quadrant: intent.DocumentationReference, Subject: "invoice", ReaderGoal: "Find the exact invoice contract."},
			{Quadrant: intent.DocumentationExplanation, Subject: "invoice_ownership", ReaderGoal: "Understand invoice ownership."},
		}
	default:
		return eval.InterpreterResult{}, fmt.Errorf("unexpected documentation test prompt %q", request.Prompt)
	}

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

func cliApplicationIntent(request eval.Request, composite bool) intent.Intent {
	value := intent.Intent{
		SchemaVersion:     intent.ApplicationSchemaVersion,
		Operation:         intent.OperationCreateApplication,
		SourceFingerprint: request.Target.SourceFingerprint,
		HatmaxVersion:     request.Target.HatmaxVersion,
		BookVersion:       request.Book.Version,
		Archetype:         "server_rendered_hatmax_application",
		Capabilities:      []string{},
		Documentation:     intent.DocumentationNotRequested,
		Application: &intent.ApplicationIdentity{
			DisplayName: "Ledger", ModulePath: "example.com/alex/ledger",
		},
		Target: &intent.ApplicationTarget{Base: "session_directory"},
	}
	if composite {
		value.InitialFeatures = []intent.InitialFeature{{
			Feature: "invoice",
			Domain:  intent.Domain{Fields: []intent.Field{{Name: "number", Type: "string", Required: true}}},
		}}
	}

	return value
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
	"embed"

	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/db"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/web"
)

func main() {
	var assets embed.FS

	configuration := &config.Config{}
	logger := log.NewNoopLogger()
	database := db.New(assets, db.Postgres, configuration, logger)
	migrator := db.NewMigrator(database, assets, db.Postgres, logger)
	tmplMgr := web.NewTemplateManager(assets, logger)
	router := app.NewRouter(logger)
	deps := []any{database, migrator, tmplMgr}
	app.Setup(context.Background(), router, deps...)
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

func installCLIApplicationCommands(t *testing.T) {
	t.Helper()

	if os.Getenv("HATMAX_REAL_APPLICATION_COMMANDS") == "1" {
		for _, name := range []string{"go", "sqlc"} {
			_, err := exec.LookPath(name)
			if err != nil {
				t.Fatalf("%s is required for real application acceptance: %v", name, err)
			}
		}

		return
	}

	directory := t.TempDir()
	writeCLIExecutable(t, filepath.Join(directory, "go"), "#!/bin/sh\nexit 0\n")
	writeCLIExecutable(t, filepath.Join(directory, "sqlc"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func installCLIDocumentationCommands(t *testing.T, failDocumentation bool) {
	t.Helper()

	directory := t.TempDir()
	makeSource := "#!/bin/sh\nexit 0\n"

	if failDocumentation {
		makeSource = "#!/bin/sh\nif [ \"$1\" = \"docs-check\" ]; then\n  echo documentation validation failed\n  exit 9\nfi\nexit 0\n"
	}

	writeCLIExecutable(t, filepath.Join(directory, "make"), makeSource)
	writeCLIExecutable(t, filepath.Join(directory, "sqlc"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func appendCLIFile(t *testing.T, root, relative, content string) {
	t.Helper()

	file, err := os.OpenFile(filepath.Join(root, filepath.FromSlash(relative)), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open CLI fixture file: %v", err)
	}

	_, writeErr := file.WriteString(content)
	closeErr := file.Close()

	if writeErr != nil {
		t.Fatalf("append CLI fixture file: %v", writeErr)
	}

	if closeErr != nil {
		t.Fatalf("close CLI fixture file: %v", closeErr)
	}
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
