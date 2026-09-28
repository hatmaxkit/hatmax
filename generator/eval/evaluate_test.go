package eval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

type promptCorpus struct {
	SchemaVersion int          `yaml:"schema_version"`
	Cases         []promptCase `yaml:"cases"`
}

type promptCase struct {
	ID             string                  `yaml:"id"`
	Prompt         string                  `yaml:"prompt"`
	Clarifications []ClarificationExchange `yaml:"clarifications,omitempty"`
	Intent         string                  `yaml:"intent,omitempty"`
	DirectResult   string                  `yaml:"direct_result,omitempty"`
	Expected       expectedCase            `yaml:"expected"`
}

type fixtureInterpreter struct {
	outputs  map[string]Interpretation
	requests []Request
	err      error
}

func (f *fixtureInterpreter) Interpret(_ context.Context, request Request) (InterpreterResult, error) {
	f.requests = append(f.requests, request)

	if f.err != nil {
		return InterpreterResult{}, f.err
	}

	result, exists := f.outputs[fixtureRequestKey(request)]
	if !exists {
		return InterpreterResult{}, errors.New("fixture output missing")
	}

	if result.SchemaVersion == 0 {
		result.SchemaVersion = CurrentInterpretationSchemaVersion
	}

	return InterpreterResult{
		Interpretation: result,
		Provenance: Provenance{
			Adapter:         "fixture",
			ContractVersion: CurrentContractVersion,
			ModelSelection:  ModelNotApplicable,
			Timing:          TimingNotMeasured,
		},
	}, nil
}

func TestNaturalLanguageCasesUseFixtureInterpreterOnly(t *testing.T) {
	loaded := loadPromptCorpus(t)
	evaluationContext := syntheticEvaluationContext(t)
	interpreter := &fixtureInterpreter{outputs: make(map[string]Interpretation, len(loaded.Cases))}

	for _, testCase := range loaded.Cases {
		interpreter.outputs[fixtureRequestKey(Request{
			Prompt:         testCase.Prompt,
			Clarifications: testCase.Clarifications,
		})] = promptInterpretation(t, testCase)
	}

	digests := make(map[string]string)

	for index, testCase := range loaded.Cases {
		t.Run(corpusCaseName(index, testCase.ID), func(t *testing.T) {
			result, err := EvaluateConversation(
				context.Background(),
				interpreter,
				testCase.Prompt,
				testCase.Clarifications,
				evaluationContext,
			)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}

			if string(result.Status) != testCase.Expected.Status {
				t.Errorf("Status = %q, want %q", result.Status, testCase.Expected.Status)
			}

			assertExpectedDiagnostics(t, result.Diagnostics, testCase.Expected.Diagnostics)
			assertExpectedClarifications(t, result.Clarifications, testCase.Expected.Clarifications)

			if result.Provenance.Adapter != "fixture" || result.Provenance.ModelSelection != ModelNotApplicable {
				t.Errorf("Provenance = %#v, want bounded fixture provenance", result.Provenance)
			}

			if testCase.Expected.EquivalentGroup != "" {
				if result.Plan == nil {
					t.Fatal("admitted equivalent case has no plan")
				}

				previous, exists := digests[testCase.Expected.EquivalentGroup]
				if exists && previous != result.Plan.Digest {
					t.Errorf("equivalent plan digest = %q, want %q", result.Plan.Digest, previous)
				}

				digests[testCase.Expected.EquivalentGroup] = result.Plan.Digest
			}
		})
	}

	if len(interpreter.requests) != len(loaded.Cases) {
		t.Errorf("fixture interpreter received %d requests, want %d", len(interpreter.requests), len(loaded.Cases))
	}

	for _, request := range interpreter.requests {
		assertBoundedRequest(t, request)
	}
}

func fixtureRequestKey(request Request) string {
	var result strings.Builder

	result.WriteString(request.Prompt)

	for _, clarification := range request.Clarifications {
		result.WriteByte(0)
		result.WriteString(clarification.Field)
		result.WriteByte(0)
		result.WriteString(clarification.Question)
		result.WriteByte(0)
		result.WriteString(clarification.Answer)
	}

	return result.String()
}

func TestEvaluateAcceptsDirectClarification(t *testing.T) {
	interpreter := &fixtureInterpreter{outputs: map[string]Interpretation{
		"Add validation.": {
			Kind: InterpretationClarification,
			Clarifications: []intent.Clarification{{
				Field:    "domain.validation",
				Question: "What validation should be added?",
			}},
		},
	}}

	result, err := Evaluate(context.Background(), interpreter, "Add validation.", syntheticEvaluationContext(t))
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if result.Status != intent.StatusClarificationRequired || len(result.Clarifications) != 1 {
		t.Errorf("Evaluate() = %#v, want direct clarification", result)
	}
}

func TestEvaluateConversationSuppliesDetachedClarificationHistory(t *testing.T) {
	clarifications := []ClarificationExchange{{
		Field:    "domain.validation.kind",
		Question: "Which validation should be added?",
		Answer:   "Minimum length.",
	}}
	interpreter := &fixtureInterpreter{outputs: map[string]Interpretation{
		fixtureRequestKey(Request{Prompt: "Add validation.", Clarifications: clarifications}): {
			Kind: InterpretationClarification,
			Clarifications: []intent.Clarification{{
				Field:    "domain.validation.value",
				Question: "What minimum length should be used?",
			}},
		},
	}}

	result, err := EvaluateConversation(
		context.Background(),
		interpreter,
		"Add validation.",
		clarifications,
		syntheticEvaluationContext(t),
	)
	if err != nil {
		t.Fatalf("EvaluateConversation() error = %v", err)
	}

	clarifications[0].Answer = "changed"
	if len(interpreter.requests) != 1 || interpreter.requests[0].Clarifications[0].Answer != "Minimum length." {
		t.Errorf("Interpreter request = %#v, want detached clarification history", interpreter.requests)
	}

	if result.Provenance.Adapter != "fixture" || result.Provenance.ContractVersion != CurrentContractVersion {
		t.Errorf("Provenance = %#v, want validated fixture provenance", result.Provenance)
	}
}

func TestEvaluateConversationBoundsClarificationHistory(t *testing.T) {
	valid := ClarificationExchange{Field: "domain.validation.kind", Question: "Which kind?", Answer: "required"}
	tests := []struct {
		name           string
		prompt         string
		clarifications []ClarificationExchange
		code           string
	}{
		{name: "large prompt", prompt: strings.Repeat("p", MaximumPromptBytes+1), code: "evaluation_prompt_too_large"},
		{name: "too many exchanges", prompt: "prompt", clarifications: repeatedExchanges(valid, MaximumClarificationExchanges+1), code: "evaluation_clarification_limit"},
		{name: "empty answer", prompt: "prompt", clarifications: []ClarificationExchange{{Field: valid.Field, Question: valid.Question}}, code: "evaluation_clarification_invalid"},
		{name: "large answer", prompt: "prompt", clarifications: []ClarificationExchange{{Field: valid.Field, Question: valid.Question, Answer: strings.Repeat("a", MaximumClarificationTextBytes+1)}}, code: "evaluation_clarification_too_large"},
		{name: "duplicate field", prompt: "prompt", clarifications: []ClarificationExchange{valid, valid}, code: "evaluation_clarification_duplicate"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := EvaluateConversation(
				context.Background(),
				&fixtureInterpreter{},
				test.prompt,
				test.clarifications,
				syntheticEvaluationContext(t),
			)
			requireEvaluationCode(t, err, test.code)
		})
	}
}

func TestEvaluatePreservesStableBackendFailure(t *testing.T) {
	interpreter := &fixtureInterpreter{err: BackendError{
		Code:      BackendTimeout,
		Operation: "turn/start",
		Message:   "deadline exceeded",
	}}

	_, err := Evaluate(context.Background(), interpreter, "prompt", syntheticEvaluationContext(t))
	if err == nil {
		t.Fatal("Evaluate() error = nil, want backend failure")
	}

	var backendErr BackendError
	if !errors.As(err, &backendErr) || backendErr.Code != BackendTimeout {
		t.Errorf("Evaluate() error = %#v, want backend timeout", err)
	}
}

func TestEvaluateRejectsInvalidBackendProvenance(t *testing.T) {
	interpreter := &invalidProvenanceInterpreter{}

	_, err := Evaluate(context.Background(), interpreter, "prompt", syntheticEvaluationContext(t))
	requireEvaluationCode(t, err, "evaluation_provenance_invalid")
}

type invalidProvenanceInterpreter struct{}

func (invalidProvenanceInterpreter) Interpret(_ context.Context, _ Request) (InterpreterResult, error) {
	return InterpreterResult{
		Interpretation: Interpretation{
			Kind: InterpretationUnsupported,
			Diagnostics: []intent.Diagnostic{{
				Code:    "HMGEN-CAPABILITY-UNSUPPORTED",
				Field:   "capabilities",
				Message: "unsupported",
			}},
		},
	}, nil
}

func repeatedExchanges(value ClarificationExchange, count int) []ClarificationExchange {
	result := make([]ClarificationExchange, count)
	for index := range result {
		result[index] = value
		result[index].Field = fmt.Sprintf("%s.%d", value.Field, index)
	}

	return result
}

func TestEvaluateRejectsInvalidBoundaryResults(t *testing.T) {
	validIntent := decodeCorpusIntent(t, "intents/create-invoice-canonical.yaml")
	tests := []struct {
		name          string
		interpreter   Interpreter
		prompt        string
		mutateContext func(*Context)
		code          string
	}{
		{name: "missing interpreter", prompt: "prompt", code: "evaluation_interpreter_required"},
		{name: "empty prompt", interpreter: &fixtureInterpreter{}, prompt: " ", code: "evaluation_prompt_required"},
		{name: "missing Book", interpreter: &fixtureInterpreter{}, prompt: "prompt", mutateContext: func(value *Context) { value.Book = nil }, code: "evaluation_book_required"},
		{name: "interpreter failure", interpreter: &fixtureInterpreter{err: errors.New("failed")}, prompt: "prompt", code: "evaluation_interpreter_failed"},
		{name: "unknown kind", interpreter: &fixtureInterpreter{outputs: map[string]Interpretation{"prompt": {Kind: "free_form"}}}, prompt: "prompt", code: "evaluation_kind_invalid"},
		{name: "intent with diagnostics", interpreter: &fixtureInterpreter{outputs: map[string]Interpretation{"prompt": {Kind: InterpretationIntent, Intent: &validIntent, Diagnostics: []intent.Diagnostic{{Code: "unexpected"}}}}}, prompt: "prompt", code: "evaluation_result_invalid"},
		{name: "empty clarification", interpreter: &fixtureInterpreter{outputs: map[string]Interpretation{"prompt": {Kind: InterpretationClarification}}}, prompt: "prompt", code: "evaluation_result_invalid"},
		{name: "empty unsupported", interpreter: &fixtureInterpreter{outputs: map[string]Interpretation{"prompt": {Kind: InterpretationUnsupported}}}, prompt: "prompt", code: "evaluation_result_invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			evaluationContext := syntheticEvaluationContext(t)
			if test.mutateContext != nil {
				test.mutateContext(&evaluationContext)
			}

			_, err := Evaluate(context.Background(), test.interpreter, test.prompt, evaluationContext)
			requireEvaluationCode(t, err, test.code)
		})
	}
}

func loadPromptCorpus(t *testing.T) promptCorpus {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(corpusRoot, "prompts.yaml"))
	if err != nil {
		t.Fatalf("read prompt corpus: %v", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var result promptCorpus

	err = decoder.Decode(&result)
	if err != nil {
		t.Fatalf("decode prompt corpus: %v", err)
	}

	if result.SchemaVersion != 1 || len(result.Cases) == 0 {
		t.Fatalf("prompt corpus = %#v, want schema 1 with cases", result)
	}

	return result
}

func promptInterpretation(t *testing.T, testCase promptCase) Interpretation {
	t.Helper()

	switch {
	case testCase.Intent != "":
		value := decodeCorpusIntent(t, testCase.Intent)

		return Interpretation{SchemaVersion: CurrentInterpretationSchemaVersion, Kind: InterpretationIntent, Intent: &value}
	case testCase.DirectResult == "unsupported":
		return Interpretation{
			SchemaVersion: CurrentInterpretationSchemaVersion,
			Kind:          InterpretationUnsupported,
			Diagnostics: []intent.Diagnostic{{
				Code:    "HMGEN-REQUEST-UNSUPPORTED",
				Field:   "prompt",
				Message: "request requires an architecture outside the Hatmax Book",
			}},
		}
	case testCase.DirectResult == "clarification":
		return Interpretation{
			SchemaVersion: CurrentInterpretationSchemaVersion,
			Kind:          InterpretationClarification,
			Clarifications: []intent.Clarification{{
				Field:    "documentation_targets",
				Question: "What should the reader be able to learn, do, look up, or understand?",
			}},
		}
	default:
		t.Fatalf("prompt case %q has no fixture result", testCase.ID)

		return Interpretation{}
	}
}

func decodeCorpusIntent(t *testing.T, path string) intent.Intent {
	t.Helper()

	value, err := intent.DecodeYAML(readCorpusIntent(t, path))
	if err != nil {
		t.Fatalf("DecodeYAML(%q) error = %v", path, err)
	}

	return value
}

func syntheticEvaluationContext(t *testing.T) Context {
	t.Helper()

	selectedBook, err := book.LoadDefault()
	if err != nil {
		t.Fatalf("book.LoadDefault() error = %v", err)
	}

	return Context{
		Inventory: project.Inventory{
			Module: project.Module{
				Hatmax: project.HatmaxModule{
					Version: "v0.4.0",
					Source:  project.HatmaxSourceModule,
				},
			},
			Layout: project.Layout{Features: []string{"internal/feat/property"}},
		},
		Fingerprint: project.Fingerprint{
			Value:                "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			BookVersion:          1,
			SelectedDependencies: []string{"github.com/sqlc-dev/sqlc/cmd/sqlc"},
			PlannedSurfaces:      []string{"migration", "model", "store", "service", "handler", "templates", "wiring", "tests", "documentation"},
			Observations: []project.Observation{{
				Key:     "file:internal/feat/property/model.go",
				Class:   project.ObservationPlannedSurface,
				Path:    "internal/feat/property/model.go",
				Surface: "model",
				Digest:  "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			}},
		},
		Book: selectedBook,
		DocumentationEvidence: &project.FeatureEvidence{
			Basis:   "existing",
			Feature: "property",
			Entity:  "Property",
			Label:   "Properties",
			Route:   "/properties",
			Table:   "properties",
			Fields: []project.FeatureFieldEvidence{{
				Name: "name", Type: "string", Label: "Name", Required: true,
			}},
			Validations: []project.FeatureValidationEvidence{{
				Field: "name", Kind: "required", Scope: "durable",
			}},
			Postgres:          true,
			HTMX:              true,
			RuntimeValidation: true,
			Wired:             true,
			Tested:            true,
			Sources: []project.FeatureEvidenceSource{{
				Role: "model", Path: "internal/feat/property/model.go", Digest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			}},
		},
	}
}

func assertExpectedDiagnostics(t *testing.T, got []intent.Diagnostic, want []string) {
	t.Helper()

	for _, code := range want {
		found := false

		for _, diagnostic := range got {
			if diagnostic.Code == code {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("Diagnostics does not contain %q: %#v", code, got)
		}
	}
}

func assertExpectedClarifications(t *testing.T, got []intent.Clarification, want []string) {
	t.Helper()

	for _, field := range want {
		found := false

		for _, clarification := range got {
			if clarification.Field == field {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("Clarifications does not contain %q: %#v", field, got)
		}
	}
}

func assertBoundedRequest(t *testing.T, request Request) {
	t.Helper()

	if request.ContractVersion != CurrentContractVersion {
		t.Errorf("Request.ContractVersion = %d, want %d", request.ContractVersion, CurrentContractVersion)
	}

	if request.Project.Fingerprint == "" || request.Project.HatmaxVersion == "" {
		t.Errorf("Request.Project = %#v, want versioned bounded context", request.Project)
	}

	if len(request.Project.ExistingFeatures) != 1 || request.Project.ExistingFeatures[0] != "property" {
		t.Errorf("ExistingFeatures = %v, want property", request.Project.ExistingFeatures)
	}

	if request.Book.Version != 1 || len(request.Book.Archetypes) != 1 || len(request.Book.Capabilities) != 3 {
		t.Errorf("Request.Book = %#v, want initial Book projection", request.Book)
	}
}

func requireEvaluationCode(t *testing.T, err error, code string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want evaluation code %q", code)
	}

	var evaluationErr Error
	if !errors.As(err, &evaluationErr) {
		t.Fatalf("error = %v, want eval.Error", err)
	}

	if evaluationErr.Code != code {
		t.Fatalf("evaluation code = %q, want %q", evaluationErr.Code, code)
	}
}
