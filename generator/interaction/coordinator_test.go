// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package interaction

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/intent"
)

func TestCoordinatorPlansPresentsAndClaimsApprovedIntent(t *testing.T) {
	root := copySupportedProject(t)
	interpreter := &functionInterpreter{interpret: admittedCreateFeature}
	approver := &functionApprover{approve: func(_ context.Context, request ApprovalRequest) (ApprovalDecision, error) {
		if request.PlanDigest == "" || request.ProjectFingerprint == "" {
			t.Fatal("approval request omitted plan identity")
		}

		if !strings.Contains(string(request.PlanYAML), "intent: create_feature") ||
			!strings.Contains(string(request.PlanYAML), "digest: "+request.PlanDigest) {
			t.Fatalf("approval plan YAML did not expose the canonical plan: %s", request.PlanYAML)
		}

		return ApprovalGranted, nil
	}}
	coordinator := newTestCoordinator(t, interpreter, approver, nil)

	prepared, terminal := coordinator.prepareApproved(context.Background(), root, "Add an invoice feature.")
	if terminal != nil {
		t.Fatalf("prepareApproved() terminal = %#v, want approved interaction", terminal)
	}

	if prepared.plan.Digest == "" || prepared.result.Plan == nil {
		t.Fatal("prepareApproved() omitted the sealed plan")
	}

	if prepared.result.State != StateAwaitingApproval || prepared.result.Outcome != "" {
		t.Errorf("prepared result = %#v, want non-terminal approved state", prepared.result)
	}

	if len(interpreter.requests) != 1 {
		t.Fatalf("interpreter calls = %d, want one", len(interpreter.requests))
	}

	if len(prepared.plan.FingerprintInputs.PlannedSurfaces) != 9 {
		t.Errorf("fingerprinted surfaces = %v, want complete bounded Book surface set", prepared.plan.FingerprintInputs.PlannedSurfaces)
	}

	if len(prepared.plan.FingerprintInputs.SelectedDependencies) != 1 ||
		prepared.plan.FingerprintInputs.SelectedDependencies[0] != "github.com/sqlc-dev/sqlc/cmd/sqlc" {
		t.Errorf("fingerprinted dependencies = %v, want Book dependency", prepared.plan.FingerprintInputs.SelectedDependencies)
	}
}

func TestCoordinatorContinuesFocusedClarification(t *testing.T) {
	root := copySupportedProject(t)
	interpreter := &functionInterpreter{interpret: func(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
		if len(request.Clarifications) == 0 {
			return fixtureInterpreterResult(eval.Interpretation{
				SchemaVersion: eval.CurrentInterpretationSchemaVersion,
				Kind:          eval.InterpretationClarification,
				Clarifications: []intent.Clarification{{
					Field:    "domain.route",
					Question: "Which route should expose invoices?",
				}},
			}), nil
		}

		if len(request.Clarifications) != 1 || request.Clarifications[0].Answer != "/invoices" {
			t.Fatalf("clarification history = %#v, want explicit route answer", request.Clarifications)
		}

		return admittedCreateFeature(context.Background(), request)
	}}
	clarifier := &functionClarifier{clarify: func(_ context.Context, request ClarificationRequest) (ClarificationResponse, error) {
		if request.Round != 1 || len(request.Questions) != 1 || request.Questions[0].Field != "domain.route" {
			t.Fatalf("clarification request = %#v, want focused first round", request)
		}

		return ClarificationResponse{Answers: []ClarificationAnswer{{
			Field:  "domain.route",
			Answer: "/invoices",
		}}}, nil
	}}
	coordinator := newTestCoordinator(t, interpreter, approvingPort(), clarifier)

	prepared, terminal := coordinator.prepareApproved(context.Background(), root, "Add invoices.")
	if terminal != nil {
		t.Fatalf("prepareApproved() terminal = %#v, want approved interaction", terminal)
	}

	if prepared.result.Provenance.ClarificationRounds != 1 || len(prepared.result.Provenance.Interpretations) != 2 {
		t.Errorf("provenance = %#v, want one clarification across two interpretations", prepared.result.Provenance)
	}
}

func TestCoordinatorReturnsUnsupportedAndUnansweredClarification(t *testing.T) {
	tests := []struct {
		name           string
		interpretation eval.Interpretation
		outcome        Outcome
	}{
		{
			name: "unsupported",
			interpretation: eval.Interpretation{
				SchemaVersion: eval.CurrentInterpretationSchemaVersion,
				Kind:          eval.InterpretationUnsupported,
				Diagnostics: []intent.Diagnostic{{
					Code: "HMGEN-DATABASE-UNSUPPORTED", Field: "capabilities", Message: "alternate SQL databases are unsupported",
				}},
			},
			outcome: OutcomeUnsupported,
		},
		{
			name: "clarification without port",
			interpretation: eval.Interpretation{
				SchemaVersion:  eval.CurrentInterpretationSchemaVersion,
				Kind:           eval.InterpretationClarification,
				Clarifications: []intent.Clarification{{Field: "domain.fields", Question: "Which fields belong to the invoice?"}},
			},
			outcome: OutcomeClarificationRequired,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			root := copySupportedProject(t)
			interpreter := &functionInterpreter{interpret: func(context.Context, eval.Request) (eval.InterpreterResult, error) {
				return fixtureInterpreterResult(testCase.interpretation), nil
			}}
			coordinator := newTestCoordinator(t, interpreter, approvingPort(), nil)

			_, terminal := coordinator.prepareApproved(context.Background(), root, "Use another database.")
			if terminal == nil || terminal.State != StateFinished || terminal.Outcome != testCase.outcome {
				t.Fatalf("prepareApproved() = %#v, want terminal %q", terminal, testCase.outcome)
			}

			if testCase.outcome == OutcomeClarificationRequired && len(terminal.Clarifications) != 1 {
				t.Errorf("clarifications = %v, want pending question", terminal.Clarifications)
			}
		})
	}
}

func TestCoordinatorRejectsStaleApprovedPlan(t *testing.T) {
	root := copySupportedProject(t)
	target := filepath.Join(root, "internal", "feat", "property", "model.go")
	approver := &functionApprover{approve: func(context.Context, ApprovalRequest) (ApprovalDecision, error) {
		content, err := os.ReadFile(target)
		if err != nil {
			return "", err
		}

		err = os.WriteFile(target, append(content, '\n'), 0o644)
		if err != nil {
			return "", err
		}

		return ApprovalGranted, nil
	}}
	coordinator := newTestCoordinator(t, &functionInterpreter{interpret: admittedCreateFeature}, approver, nil)

	_, terminal := coordinator.prepareApproved(context.Background(), root, "Add invoices.")
	if terminal == nil || terminal.Outcome != OutcomePlanStale {
		t.Fatalf("prepareApproved() = %#v, want stale plan", terminal)
	}

	if len(terminal.FreshnessChanges) == 0 {
		t.Fatal("stale result omitted changed observations")
	}
}

func TestCoordinatorClarifiesApplicationModuleAfterResolvingTarget(t *testing.T) {
	parent := t.TempDir()
	interpreter := &functionInterpreter{interpret: func(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
		value := applicationIntent(request, false)
		if !request.Target.Resolved {
			value.Application.ModulePath = ""
		}

		if request.Target.Resolved {
			if len(request.Clarifications) != 1 || request.Clarifications[0].Field != "application.module_path" {
				t.Fatalf("resolved request clarifications = %#v", request.Clarifications)
			}

			value.Application.ModulePath = request.Clarifications[0].Answer
		}

		return intentInterpreterResult(value), nil
	}}
	clarifier := &functionClarifier{clarify: func(_ context.Context, request ClarificationRequest) (ClarificationResponse, error) {
		if len(request.Questions) != 1 || request.Questions[0].Field != "application.module_path" {
			t.Fatalf("clarification = %#v, want module path", request)
		}

		return ClarificationResponse{Answers: []ClarificationAnswer{{
			Field: "application.module_path", Answer: "example.com/alex/ledger",
		}}}, nil
	}}
	coordinator := newTestCoordinator(t, interpreter, approvingPort(), clarifier)

	prepared, terminal := coordinator.prepareApproved(context.Background(), parent, "Create Ledger.")
	if terminal != nil {
		t.Fatalf("prepareApproved() terminal = %#v", terminal)
	}

	if len(interpreter.requests) != 2 || interpreter.requests[0].Target.Resolved || !interpreter.requests[1].Target.Resolved {
		t.Fatalf("target resolution sequence = %#v", interpreter.requests)
	}

	if prepared.plan.Application.ModulePath != "example.com/alex/ledger" || prepared.result.Provenance.ClarificationRounds != 1 {
		t.Fatalf("prepared application = %#v provenance = %#v", prepared.plan.Application, prepared.result.Provenance)
	}
}

func TestCoordinatorRejectsConflictingApplicationTargetBeforeApproval(t *testing.T) {
	parent := t.TempDir()

	target := filepath.Join(parent, "ledger")

	err := os.MkdirAll(target, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(target, "go.mod"), []byte("module example.com/other\n\ngo 1.27.1\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	approver := approvingPort()
	coordinator := newTestCoordinator(t, &functionInterpreter{interpret: func(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
		return intentInterpreterResult(applicationIntent(request, false)), nil
	}}, approver, nil)

	_, terminal := coordinator.prepareApproved(context.Background(), parent, "Create Ledger.")
	if terminal == nil || terminal.Outcome != OutcomeIntentRejected {
		t.Fatalf("prepareApproved() = %#v, want target rejection", terminal)
	}

	if approver.calls != 0 {
		t.Fatalf("approval calls = %d, want zero", approver.calls)
	}
}

func TestCoordinatorDetectsApplicationTargetDriftAndCancellation(t *testing.T) {
	tests := []struct {
		name    string
		approve func(string) *functionApprover
		outcome Outcome
	}{
		{
			name: "stale",
			approve: func(parent string) *functionApprover {
				return &functionApprover{approve: func(context.Context, ApprovalRequest) (ApprovalDecision, error) {
					return ApprovalGranted, os.Mkdir(filepath.Join(parent, "ledger"), 0o755)
				}}
			},
			outcome: OutcomePlanStale,
		},
		{
			name: "cancelled",
			approve: func(string) *functionApprover {
				return &functionApprover{approve: func(context.Context, ApprovalRequest) (ApprovalDecision, error) {
					return ApprovalRejected, nil
				}}
			},
			outcome: OutcomeCancelled,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parent := t.TempDir()
			coordinator := newTestCoordinator(t, &functionInterpreter{interpret: func(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
				return intentInterpreterResult(applicationIntent(request, false)), nil
			}}, test.approve(parent), nil)

			_, terminal := coordinator.prepareApproved(context.Background(), parent, "Create Ledger.")
			if terminal == nil || terminal.Outcome != test.outcome {
				t.Fatalf("prepareApproved() = %#v, want %q", terminal, test.outcome)
			}

			if test.outcome == OutcomeCancelled {
				_, err := os.Stat(filepath.Join(parent, "ledger"))
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("cancelled target exists: %v", err)
				}
			}
		})
	}
}

func TestCoordinatorRejectsInvalidClarificationBindingWithoutApproval(t *testing.T) {
	root := copySupportedProject(t)
	interpreter := &functionInterpreter{interpret: func(context.Context, eval.Request) (eval.InterpreterResult, error) {
		return fixtureInterpreterResult(eval.Interpretation{
			SchemaVersion:  eval.CurrentInterpretationSchemaVersion,
			Kind:           eval.InterpretationClarification,
			Clarifications: []intent.Clarification{{Field: "domain.route", Question: "Which route?"}},
		}), nil
	}}
	approver := approvingPort()
	clarifier := &functionClarifier{clarify: func(context.Context, ClarificationRequest) (ClarificationResponse, error) {
		return ClarificationResponse{Answers: []ClarificationAnswer{{Field: "domain.entity", Answer: "Invoice"}}}, nil
	}}
	coordinator := newTestCoordinator(t, interpreter, approver, clarifier)

	_, terminal := coordinator.prepareApproved(context.Background(), root, "Add invoices.")
	if terminal == nil || terminal.Outcome != OutcomeFailed || terminal.Diagnostics[0].Code != "HMGEN-CLARIFICATION-INVALID" {
		t.Fatalf("prepareApproved() = %#v, want invalid clarification failure", terminal)
	}

	if approver.calls != 0 {
		t.Fatalf("approval calls = %d, want zero", approver.calls)
	}
}

func TestNewRequiresInterpreterAndApprover(t *testing.T) {
	tests := []Config{
		{Approver: approvingPort()},
		{Interpreter: &functionInterpreter{interpret: admittedCreateFeature}},
	}

	for index, config := range tests {
		_, err := New(config)
		if err == nil {
			t.Errorf("New() case %d error = nil, want required-port error", index)
		}
	}
}

type functionInterpreter struct {
	interpret func(context.Context, eval.Request) (eval.InterpreterResult, error)
	requests  []eval.Request
}

func (f *functionInterpreter) Interpret(ctx context.Context, request eval.Request) (eval.InterpreterResult, error) {
	f.requests = append(f.requests, request)

	return f.interpret(ctx, request)
}

type functionApprover struct {
	approve func(context.Context, ApprovalRequest) (ApprovalDecision, error)
	calls   int
}

func (f *functionApprover) Approve(ctx context.Context, request ApprovalRequest) (ApprovalDecision, error) {
	f.calls++

	return f.approve(ctx, request)
}

type functionClarifier struct {
	clarify func(context.Context, ClarificationRequest) (ClarificationResponse, error)
}

func (f *functionClarifier) Clarify(ctx context.Context, request ClarificationRequest) (ClarificationResponse, error) {
	return f.clarify(ctx, request)
}

func admittedCreateFeature(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
	value := intent.Intent{
		SchemaVersion:      intent.CurrentSchemaVersion,
		Operation:          intent.OperationCreateFeature,
		ProjectFingerprint: request.Project.Fingerprint,
		HatmaxVersion:      request.Project.HatmaxVersion,
		BookVersion:        request.Book.Version,
		Archetype:          "server_rendered_crud",
		Feature:            "invoice",
		Domain: intent.Domain{
			Entity: "Invoice",
			Route:  "/invoices",
			Fields: []intent.Field{{Name: "title", Type: "string", Label: "Title"}},
		},
		Capabilities:  []string{"postgres_persistence", "runtime_validation", "htmx_form"},
		Documentation: intent.DocumentationNotRequested,
		Exceptions:    []intent.Exception{},
	}

	return fixtureInterpreterResult(eval.Interpretation{
		SchemaVersion: eval.CurrentInterpretationSchemaVersion,
		Kind:          eval.InterpretationIntent,
		Intent:        &value,
	}), nil
}

func applicationIntent(request eval.Request, composite bool) intent.Intent {
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

func fixtureInterpreterResult(interpretation eval.Interpretation) eval.InterpreterResult {
	return eval.InterpreterResult{
		Interpretation: interpretation,
		Provenance: eval.Provenance{
			Adapter:         "fixture",
			ContractVersion: eval.CurrentContractVersion,
			ModelSelection:  eval.ModelNotApplicable,
			Timing:          eval.TimingNotMeasured,
		},
	}
}

func approvingPort() *functionApprover {
	return &functionApprover{approve: func(context.Context, ApprovalRequest) (ApprovalDecision, error) {
		return ApprovalGranted, nil
	}}
}

func newTestCoordinator(
	t *testing.T,
	interpreter eval.Interpreter,
	approver Approver,
	clarifier Clarifier,
) *Coordinator {
	t.Helper()

	coordinator, err := New(Config{Interpreter: interpreter, Approver: approver, Clarifier: clarifier})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return coordinator
}

func copySupportedProject(t *testing.T) string {
	t.Helper()

	source := filepath.Join("..", "project", "testdata", "supported")
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
