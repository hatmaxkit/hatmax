package eval

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

type typedOutcome struct {
	status         intent.Status
	diagnostics    []intent.Diagnostic
	clarifications []intent.Clarification
	plan           *plan.Plan
}

func TestTypedCorpusExercisesPlanningKernel(t *testing.T) {
	loaded := loadCorpus(t)
	evaluationContext := syntheticEvaluationContext(t)
	digests := make(map[string]string)

	for index, testCase := range loaded.TypedCases {
		t.Run(corpusCaseName(index, testCase.ID), func(t *testing.T) {
			result := runTypedCase(t, testCase, evaluationContext)
			if string(result.status) != testCase.Expected.Status {
				t.Errorf("status = %q, want %q", result.status, testCase.Expected.Status)
			}

			assertExpectedDiagnostics(t, result.diagnostics, testCase.Expected.Diagnostics)
			assertExpectedClarifications(t, result.clarifications, testCase.Expected.Clarifications)

			if result.plan != nil {
				err := plan.VerifyDigest(*result.plan)
				if err != nil {
					t.Errorf("VerifyDigest() error = %v", err)
				}
			}

			group := testCase.Expected.EquivalentGroup
			if group == "" {
				return
			}

			if result.plan == nil {
				t.Fatal("equivalent admitted case has no plan")
			}

			previous, exists := digests[group]
			if exists && result.plan.Digest != previous {
				t.Errorf("equivalent plan digest = %q, want %q", result.plan.Digest, previous)
			}

			digests[group] = result.plan.Digest
		})
	}
}

func TestKernelFromProjectInventoryThroughDriftDiagnostic(t *testing.T) {
	root := copySupportedProject(t)
	selectedBook := loadDefaultBook(t)
	inventory, fingerprint := inspectAndFingerprint(t, root, selectedBook.Manifest().BookVersion)
	evaluationContext := Context{
		Inventory:   inventory,
		Fingerprint: fingerprint,
		Book:        selectedBook,
	}

	loaded := loadCorpus(t)
	testCase := loaded.StaleCases[0]
	value := decodeCorpusIntent(t, testCase.Intent)
	value.ProjectFingerprint = fingerprint.Value
	value.HatmaxVersion = inventory.Module.Hatmax.Version

	prompt := "Build the invoice feature from the corpus."
	interpreter := &fixtureInterpreter{outputs: map[string]Interpretation{
		prompt: {Kind: InterpretationIntent, Intent: &value},
	}}

	first, err := Evaluate(context.Background(), interpreter, prompt, evaluationContext)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if first.Status != intent.StatusAdmitted || first.Plan == nil {
		t.Fatalf("Evaluate() = %#v, want admitted sealed plan", first)
	}

	second, err := Evaluate(context.Background(), interpreter, prompt, evaluationContext)
	if err != nil {
		t.Fatalf("second Evaluate() error = %v", err)
	}

	if second.Plan == nil || second.Plan.Digest != first.Plan.Digest {
		t.Errorf("repeated evaluation digest = %#v, want %q", second.Plan, first.Plan.Digest)
	}

	afterEvaluation, afterFingerprint := inspectAndFingerprint(t, root, selectedBook.Manifest().BookVersion)
	if afterFingerprint.Value != fingerprint.Value || afterEvaluation.Module.Path != inventory.Module.Path {
		t.Error("planning evaluation modified or changed the inspected project")
	}

	appendFixtureDrift(t, root, testCase.ChangedPath)
	_, currentFingerprint := inspectAndFingerprint(t, root, selectedBook.Manifest().BookVersion)

	freshness, err := plan.CheckFingerprint(*first.Plan, currentFingerprint)
	if err != nil {
		t.Fatalf("CheckFingerprint() error = %v", err)
	}

	if !freshness.Stale || len(freshness.Changes) == 0 {
		t.Fatalf("CheckFingerprint() = %#v, want classified relevant drift", freshness)
	}

	for _, code := range testCase.ExpectedDiagnostics {
		if !hasPlanDiagnosticCode(freshness.Diagnostics, code) {
			t.Errorf("freshness diagnostics do not contain %q: %#v", code, freshness.Diagnostics)
		}
	}
}

func TestAdmittedPlanSerializationIsStable(t *testing.T) {
	loaded := loadCorpus(t)
	testCase := typedCaseByID(t, loaded, "create_invoice_canonical")

	result := runTypedCase(t, testCase, syntheticEvaluationContext(t))
	if result.plan == nil {
		t.Fatal("admitted corpus case has no plan")
	}

	first, err := plan.MarshalYAML(*result.plan)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	second, err := plan.MarshalYAML(*result.plan)
	if err != nil {
		t.Fatalf("second MarshalYAML() error = %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Error("sealed plan serialization is not stable")
	}
}

func runTypedCase(t *testing.T, testCase typedCase, evaluationContext Context) typedOutcome {
	t.Helper()

	value, err := intent.DecodeYAML(readCorpusIntent(t, testCase.Intent))
	if err != nil {
		return typedOutcome{
			status: intent.StatusInvalid,
			diagnostics: []intent.Diagnostic{{
				Code:    "HMGEN-INTENT-SCHEMA",
				Field:   "intent",
				Message: err.Error(),
			}},
			clarifications: []intent.Clarification{},
		}
	}

	validation := intent.Validate(value, intent.ValidationContext{
		Inventory:   evaluationContext.Inventory,
		Fingerprint: evaluationContext.Fingerprint,
		Book:        evaluationContext.Book,
	})
	result := typedOutcome{
		status:         validation.Status,
		diagnostics:    validation.Diagnostics,
		clarifications: validation.Clarifications,
	}

	if !validation.Admitted() {
		return result
	}

	expanded, err := plan.Expand(validation, plan.ExpansionContext{
		Book:        evaluationContext.Book,
		Fingerprint: evaluationContext.Fingerprint,
	})
	if err != nil {
		t.Fatalf("plan.Expand() error = %v", err)
	}

	result.plan = &expanded

	return result
}

func loadDefaultBook(t *testing.T) *book.Book {
	t.Helper()

	selectedBook, err := book.LoadDefault()
	if err != nil {
		t.Fatalf("book.LoadDefault() error = %v", err)
	}

	return selectedBook
}

func inspectAndFingerprint(t *testing.T, root string, bookVersion int) (project.Inventory, project.Fingerprint) {
	t.Helper()

	inventory, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	fingerprint, err := inventory.Fingerprint(project.FingerprintRequest{
		BookVersion:          bookVersion,
		SelectedDependencies: []string{"github.com/sqlc-dev/sqlc/cmd/sqlc"},
		PlannedSurfaces: []string{
			"migration",
			"model",
			"store",
			"service",
			"handler",
			"templates",
			"wiring",
			"tests",
			"documentation",
		},
	})
	if err != nil {
		t.Fatalf("Inventory.Fingerprint() error = %v", err)
	}

	return inventory, fingerprint
}

func copySupportedProject(t *testing.T) string {
	t.Helper()

	source := filepath.Join("..", "project", "testdata", "supported")
	target := filepath.Join(t.TempDir(), "supported")

	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, relErr := filepath.Rel(source, path)
		if relErr != nil {
			return relErr
		}

		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy supported project: %v", err)
	}

	return target
}

func appendFixtureDrift(t *testing.T, root, path string) {
	t.Helper()

	file, err := os.OpenFile(filepath.Join(root, filepath.FromSlash(path)), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open drift target: %v", err)
	}

	_, writeErr := file.WriteString("\n// Relevant corpus drift.\n")
	closeErr := file.Close()

	if writeErr != nil {
		t.Fatalf("append drift: %v", writeErr)
	}

	if closeErr != nil {
		t.Fatalf("close drift target: %v", closeErr)
	}
}

func hasPlanDiagnosticCode(diagnostics []plan.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}

	return false
}
