// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package conversation_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/execute"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/interaction"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
	"hatmax.adrianpk.com/internal/hatmaxstate"
)

const (
	testFingerprintA = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	testFingerprintB = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	testPlanDigest   = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
)

func TestCoordinatorResumesClarificationReinspectsAndReplacesThread(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeModule(t, root)

	clock := &testClock{value: time.Date(2026, time.September, 29, 20, 0, 0, 0, time.UTC)}
	engine := &fakeTurnEngine{fingerprint: testFingerprintA, previews: []interaction.Result{
		clarificationResult("thread-old"),
		featurePlanResult("thread-replacement"),
	}}
	coordinator := newConversationCoordinator(t, engine, clock)

	session, err := coordinator.Open(ctx, root, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	first, err := session.Turn(ctx, conversation.TurnRequest{Content: "Add an invoice feature."})
	if err != nil {
		t.Fatalf("Turn(goal) error = %v", err)
	}

	if len(first.Conversation.Operations) != 1 || first.Conversation.Operations[0].Status != conversation.OperationClarifying {
		t.Fatalf("first conversation = %#v, want clarifying operation", first.Conversation)
	}

	if first.Conversation.BackendThreadID != "thread-old" {
		t.Fatalf("backend thread = %q, want first managed thread", first.Conversation.BackendThreadID)
	}

	err = session.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	resumed, err := coordinator.Open(ctx, root, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("Open(resume) error = %v", err)
	}

	second, err := resumed.Turn(ctx, conversation.TurnRequest{Content: "/invoices"})
	if err != nil {
		t.Fatalf("Turn(answer) error = %v", err)
	}

	operation := second.Conversation.Operations[0]
	if operation.Status != conversation.OperationPlanned || len(operation.Decisions) != 1 || operation.Decisions[0].Answer != "/invoices" {
		t.Fatalf("planned operation = %#v, want retained explicit decision", operation)
	}

	if second.Conversation.BackendThreadID != "thread-replacement" {
		t.Fatalf("backend thread = %q, want replacement thread", second.Conversation.BackendThreadID)
	}

	if len(engine.requests) != 2 || len(engine.requests[1].Clarifications) != 1 || engine.requests[1].Clarifications[0].Answer != "/invoices" {
		t.Fatalf("interaction requests = %#v, want resumed structured decision", engine.requests)
	}

	err = resumed.Close()
	if err != nil {
		t.Fatalf("Close(resumed) error = %v", err)
	}

	engine.fingerprint = testFingerprintB

	drifted, err := coordinator.Open(ctx, root, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("Open(drifted) error = %v", err)
	}
	defer drifted.Close()

	if got := drifted.Current().Operations[0].Status; got != conversation.OperationStale {
		t.Fatalf("resumed operation status = %q, want stale after source drift", got)
	}
}

func TestCoordinatorBindsOneCombinedTurnToEveryPendingClarification(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeModule(t, root)

	clock := &testClock{value: time.Date(2026, time.September, 30, 9, 0, 0, 0, time.UTC)}
	engine := &fakeTurnEngine{fingerprint: testFingerprintA, previews: []interaction.Result{
		multipleClarificationResult("thread-old"),
		featurePlanResult("thread-replacement"),
	}}
	coordinator := newConversationCoordinator(t, engine, clock)

	session, err := coordinator.Open(ctx, root, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer session.Close()

	_, err = session.Turn(ctx, conversation.TurnRequest{Content: "Add an invoice feature."})
	if err != nil {
		t.Fatalf("Turn(goal) error = %v", err)
	}

	answer := "Use /invoices with a required number string field."

	result, err := session.Turn(ctx, conversation.TurnRequest{Content: answer})
	if err != nil {
		t.Fatalf("Turn(combined answer) error = %v", err)
	}

	if result.Interaction.Outcome != interaction.OutcomePlanReady {
		t.Fatalf("outcome = %q, want %q", result.Interaction.Outcome, interaction.OutcomePlanReady)
	}

	clarifications := engine.requests[1].Clarifications
	if len(clarifications) != 2 {
		t.Fatalf("clarifications = %#v, want two bound answers", clarifications)
	}

	for _, clarification := range clarifications {
		if clarification.Answer != answer {
			t.Errorf("answer for %q = %q, want combined answer", clarification.Field, clarification.Answer)
		}
	}
}

func TestCoordinatorRebindsApplicationConversationAndSupportsFreshReset(t *testing.T) {
	ctx := context.Background()
	parent := t.TempDir()
	target := filepath.Join(parent, "ledger")
	clock := &testClock{value: time.Date(2026, time.September, 29, 20, 0, 0, 0, time.UTC)}
	application := applicationPlanResult(target, "thread-new")
	engine := &fakeTurnEngine{
		fingerprint: testFingerprintA,
		previews:    []interaction.Result{application},
		executions:  []interaction.Result{completedApplicationResult(application)},
	}
	coordinator := newConversationCoordinator(t, engine, clock)

	session, err := coordinator.Open(ctx, parent, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	planned, err := session.Turn(ctx, conversation.TurnRequest{Content: "Create Ledger."})
	if err != nil {
		t.Fatalf("Turn() error = %v", err)
	}

	targetScope, err := conversation.ResolveScope(conversation.ScopePreProject, target)
	if err != nil {
		t.Fatalf("ResolveScope(target) error = %v", err)
	}

	if planned.Conversation.Scope != targetScope || planned.Conversation.Operations[0].Status != conversation.OperationPlanned {
		t.Fatalf("planned conversation = %#v, want target-bound plan", planned.Conversation)
	}

	completed, err := session.Approve(ctx, planned.OperationID, testPlanDigest)
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	projectScope, err := conversation.ResolveScope(conversation.ScopeProject, target)
	if err != nil {
		t.Fatalf("ResolveScope(project) error = %v", err)
	}

	if completed.Conversation.Scope != projectScope || completed.Conversation.BookContract.BookVersion != 1 || completed.Conversation.Operations[0].Status != conversation.OperationCompleted {
		t.Fatalf("completed conversation = %#v, want project-bound result", completed.Conversation)
	}

	previousID := completed.Conversation.ID

	fresh, err := session.Reset(ctx)
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	if fresh.ID == previousID || len(fresh.Turns) != 0 || len(fresh.Operations) != 0 {
		t.Fatalf("fresh conversation = %#v", fresh)
	}

	_, err = os.Stat(filepath.Join(parent, "unexpected"))
	if !os.IsNotExist(err) {
		t.Fatalf("conversation controls changed project source: %v", err)
	}
}

func TestCoordinatorPreservesCancelledAndFailedProposalsForSafeRetry(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeModule(t, root)

	before, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile(before) error = %v", err)
	}

	clock := &testClock{value: time.Date(2026, time.September, 29, 21, 0, 0, 0, time.UTC)}
	failed := featurePlanResult("thread-1")
	failed.Outcome = interaction.OutcomeExecutionFailed
	failed.State = interaction.StateFinished
	failed.Diagnostics = []interaction.Diagnostic{{
		Code: "HMGEN-VALIDATION-FAILED", Phase: interaction.PhaseValidation, Message: "generated validation failed",
	}}
	failed.RetainedChanges = []execute.Change{
		{Target: "internal/feat/invoice/model.go", Status: execute.ChangeApplied},
		{Target: "internal/feat/invoice/handler.go", Status: execute.ChangeApplied},
	}
	engine := &fakeTurnEngine{
		fingerprint: testFingerprintA,
		previews: []interaction.Result{
			clarificationResult("thread-1"), featurePlanResult("thread-1"), featurePlanResult("thread-2"),
		},
		executions: []interaction.Result{failed},
	}
	coordinator := newConversationCoordinator(t, engine, clock)

	session, err := coordinator.Open(ctx, root, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer session.Close()

	started, err := session.Turn(ctx, conversation.TurnRequest{Content: "Add invoices."})
	if err != nil {
		t.Fatalf("Turn(goal) error = %v", err)
	}

	planned, err := session.Turn(ctx, conversation.TurnRequest{Content: "/invoices"})
	if err != nil {
		t.Fatalf("Turn(decision) error = %v", err)
	}

	cancelled, err := session.Cancel(ctx, planned.OperationID)
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	operation := cancelled.Conversation.Operations[0]
	if operation.Status != conversation.OperationCancelled || len(operation.Decisions) != 1 || operation.PlanDigest != testPlanDigest {
		t.Fatalf("cancelled operation = %#v, want revisable context without approval", operation)
	}

	after, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile(after) error = %v", err)
	}

	if string(after) != string(before) {
		t.Fatal("cancellation changed project source")
	}

	if started.OperationID != planned.OperationID {
		t.Fatalf("clarification operation changed identity: %q then %q", started.OperationID, planned.OperationID)
	}

	retryPlan, err := session.Turn(ctx, conversation.TurnRequest{Content: "Retry the invoice feature."})
	if err != nil {
		t.Fatalf("Turn(retry) error = %v", err)
	}

	failedResult, err := session.Approve(ctx, retryPlan.OperationID, testPlanDigest)
	if err != nil {
		t.Fatalf("Approve(failing retry) error = %v", err)
	}

	failedOperation := failedResult.Conversation.Operations[1]
	if failedOperation.Status != conversation.OperationFailed ||
		!strings.Contains(failedOperation.ResultSummary, "internal/feat/invoice/model.go") ||
		!strings.Contains(failedOperation.ResultSummary, "internal/feat/invoice/handler.go") {
		t.Fatalf("failed operation = %#v, want every retained target", failedOperation)
	}
}

func TestCoordinatorRejectsInvalidApprovalWithoutConsumingRetry(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeModule(t, root)

	clock := &testClock{value: time.Date(2026, time.September, 29, 21, 0, 0, 0, time.UTC)}
	plannedResult := featurePlanResult("thread-1")
	completed := plannedResult
	completed.State = interaction.StateFinished
	completed.Outcome = interaction.OutcomeValidationIncomplete
	engine := &fakeTurnEngine{
		fingerprint: testFingerprintA,
		previews:    []interaction.Result{plannedResult},
		executions:  []interaction.Result{completed},
	}
	coordinator := newConversationCoordinator(t, engine, clock)

	session, err := coordinator.Open(ctx, root, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer session.Close()

	planned, err := session.Turn(ctx, conversation.TurnRequest{Content: "Add invoices."})
	if err != nil {
		t.Fatalf("Turn() error = %v", err)
	}

	invalid, err := session.Approve(ctx, planned.OperationID, testFingerprintB)
	if err != nil {
		t.Fatalf("Approve(invalid) error = %v", err)
	}

	if invalid.Conversation.Operations[0].Status != conversation.OperationPlanned || len(engine.executions) != 1 {
		t.Fatalf("invalid approval consumed proposal or execution: %#v", invalid)
	}

	completedResult, err := session.Approve(ctx, planned.OperationID, testPlanDigest)
	if err != nil {
		t.Fatalf("Approve(valid retry) error = %v", err)
	}

	if completedResult.Interaction.Outcome != interaction.OutcomeValidationIncomplete || completedResult.Conversation.Operations[0].Status != conversation.OperationCompleted || len(engine.executions) != 0 {
		t.Fatalf("valid retry = %#v, want one completed execution", completedResult)
	}
}

func TestCoordinatorContinuesInMemoryAfterPersistenceFailure(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeModule(t, root)

	clock := &testClock{value: time.Date(2026, time.September, 29, 21, 0, 0, 0, time.UTC)}
	engine := &fakeTurnEngine{
		fingerprint: testFingerprintA,
		previews: []interaction.Result{
			featurePlanResult("thread-1"),
			{
				State: interaction.StateConversation, Outcome: interaction.OutcomeConversationResponse,
				Response: &eval.ConversationResponse{Content: "The proposal is still available in this process."},
			},
		},
	}

	state, err := hatmaxstate.New(hatmaxstate.Config{
		Root: filepath.Join(t.TempDir(), "state"), Now: clock.now,
		NewID: func() string { return "stored-conversation" },
	})
	if err != nil {
		t.Fatalf("hatmaxstate.New() error = %v", err)
	}

	failing := &failingStore{Store: state, successfulReplacements: 1}
	coordinator := newConversationCoordinatorWithStore(t, failing, engine, clock)

	session, err := coordinator.Open(ctx, root, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	first, err := session.Turn(ctx, conversation.TurnRequest{Content: "Add invoices."})
	if err != nil {
		t.Fatalf("Turn() error = %v", err)
	}

	if !first.PersistenceFailed || first.Conversation.Operations[0].Status != conversation.OperationPlanned || !hasDiagnostic(first.Interaction.Diagnostics, "HMGEN-STATE-PERSISTENCE-FAILED") {
		t.Fatalf("persistence failure result = %#v", first)
	}

	second, err := session.Turn(ctx, conversation.TurnRequest{Content: "Can we keep discussing it?"})
	if err != nil {
		t.Fatalf("Turn(in-memory) error = %v", err)
	}

	if !second.PersistenceFailed || len(second.Conversation.Turns) <= len(first.Conversation.Turns) {
		t.Fatalf("in-memory continuation = %#v", second)
	}

	err = session.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	scope, err := conversation.ResolveScope(conversation.ScopeProject, root)
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}

	stored, err := state.Open(ctx, conversation.OpenRequest{
		Scope:           scope,
		BookContract:    conversation.BookContract{BookVersion: 1, InterpreterVersion: eval.CurrentContractVersion},
		BackendIdentity: conversation.BackendIdentity{Adapter: "codex-app-server", Version: "0.158.0"},
	})
	if err != nil {
		t.Fatalf("state.Open() error = %v", err)
	}
	defer stored.Close()

	if len(stored.Current().Operations) != 0 {
		t.Fatalf("failed persistence unexpectedly reached durable state: %#v", stored.Current())
	}
}

type fakeTurnEngine struct {
	fingerprint string
	previews    []interaction.Result
	executions  []interaction.Result
	requests    []interaction.TurnRequest
}

type failingStore struct {
	conversation.Store
	successfulReplacements int
}

func (s *failingStore) Open(ctx context.Context, request conversation.OpenRequest) (conversation.Session, error) {
	session, err := s.Store.Open(ctx, request)
	if err != nil {
		return nil, err
	}

	return &failingSession{Session: session, remaining: s.successfulReplacements}, nil
}

type failingSession struct {
	conversation.Session
	remaining int
}

func (s *failingSession) Replace(ctx context.Context, value conversation.Conversation) error {
	if s.remaining == 0 {
		return errors.New("injected persistence failure")
	}

	s.remaining--

	return s.Session.Replace(ctx, value)
}

func (f *fakeTurnEngine) PreviewTurn(_ context.Context, _ string, request interaction.TurnRequest) interaction.Result {
	f.requests = append(f.requests, request)
	result := f.previews[0]
	f.previews = f.previews[1:]

	return result
}

func (f *fakeTurnEngine) RunApprovedTurn(_ context.Context, _ string, request interaction.TurnRequest) interaction.Result {
	f.requests = append(f.requests, request)
	result := f.executions[0]
	f.executions = f.executions[1:]

	return result
}

func (f *fakeTurnEngine) ProjectFingerprint(context.Context, string) (project.Fingerprint, error) {
	return project.Fingerprint{Value: f.fingerprint}, nil
}

type testClock struct {
	value time.Time
}

func (c *testClock) now() time.Time {
	c.value = c.value.Add(time.Second)

	return c.value
}

func newConversationCoordinator(t *testing.T, engine *fakeTurnEngine, clock *testClock) *conversation.Coordinator {
	t.Helper()

	ids := 0

	state, err := hatmaxstate.New(hatmaxstate.Config{
		Root: filepath.Join(t.TempDir(), "state"),
		Now:  clock.now,
		NewID: func() string {
			ids++

			return "stored-conversation-" + strconv.Itoa(ids)
		},
	})
	if err != nil {
		t.Fatalf("hatmaxstate.New() error = %v", err)
	}

	return newConversationCoordinatorWithStore(t, state, engine, clock)
}

func newConversationCoordinatorWithStore(
	t *testing.T,
	state conversation.Store,
	engine *fakeTurnEngine,
	clock *testClock,
) *conversation.Coordinator {
	t.Helper()

	operationIDs := 0

	coordinator, err := conversation.NewCoordinator(conversation.CoordinatorConfig{
		Store: state, Engine: engine,
		BookContract:            conversation.BookContract{BookVersion: 1, InterpreterVersion: eval.CurrentContractVersion},
		ApplicationBookContract: conversation.BookContract{BookVersion: 2, InterpreterVersion: eval.CurrentContractVersion},
		BackendIdentity:         conversation.BackendIdentity{Adapter: "codex-app-server", Version: "0.158.0"},
		Clock:                   clock.now,
		IDSource: func(prefix string) (string, error) {
			operationIDs++

			return prefix + "-test-" + strconv.Itoa(operationIDs), nil
		},
	})
	if err != nil {
		t.Fatalf("NewCoordinator() error = %v", err)
	}

	return coordinator
}

func hasDiagnostic(values []interaction.Diagnostic, code string) bool {
	for _, diagnostic := range values {
		if diagnostic.Code == code {
			return true
		}
	}

	return false
}

func clarificationResult(threadID string) interaction.Result {
	return interaction.Result{
		State: interaction.StateClarifying, Outcome: interaction.OutcomeClarificationRequired,
		Clarifications: []intent.Clarification{{
			Field: "domain.route", Question: "Which route should expose invoices?",
		}},
		Provenance: interaction.Provenance{Interpretations: []eval.Provenance{{
			Adapter: "codex-app-server", ThreadID: threadID,
			ContractVersion: eval.CurrentContractVersion, ModelSelection: eval.ModelBackendDefault,
			Timing: eval.TimingUnderSecond,
		}}},
	}
}

func multipleClarificationResult(threadID string) interaction.Result {
	result := clarificationResult(threadID)
	result.Clarifications = append(result.Clarifications, intent.Clarification{
		Field: "domain.fields", Question: "Which fields belong to invoices?",
	})

	return result
}

func featurePlanResult(threadID string) interaction.Result {
	value := intent.Intent{
		SchemaVersion: intent.CurrentSchemaVersion, Operation: intent.OperationCreateFeature,
		ProjectFingerprint: testFingerprintA, HatmaxVersion: "v0.5.0", BookVersion: 1,
		Archetype: "server_rendered_crud", Feature: "invoice",
		Domain: intent.Domain{
			Entity: "Invoice", Route: "/invoices",
			Fields: []intent.Field{{Name: "number", Type: "string", Required: true}},
		},
		Capabilities:  []string{"postgres_persistence", "htmx_form", "runtime_validation"},
		Documentation: intent.DocumentationNotRequested, DocumentationTargets: []intent.DocumentationTarget{},
		Exceptions: []intent.Exception{},
	}

	return interaction.Result{
		State: interaction.StatePlanReady, Outcome: interaction.OutcomePlanReady,
		Intent:   &value,
		Plan:     &plan.Plan{Intent: value.Operation, ProjectFingerprint: testFingerprintA, Digest: testPlanDigest},
		PlanYAML: []byte("digest: " + testPlanDigest + "\n"),
		Provenance: interaction.Provenance{Interpretations: []eval.Provenance{{
			Adapter: "codex-app-server", ThreadID: threadID,
			ContractVersion: eval.CurrentContractVersion, ModelSelection: eval.ModelBackendDefault,
			Timing: eval.TimingUnderSecond,
		}}},
	}
}

func applicationPlanResult(target, threadID string) interaction.Result {
	value := intent.Intent{
		SchemaVersion: intent.ApplicationSchemaVersion, Operation: intent.OperationCreateApplication,
		SourceFingerprint: testFingerprintA, HatmaxVersion: "v0.5.0", BookVersion: 2,
		Archetype: "server_rendered_hatmax_application", Capabilities: []string{},
		Documentation: intent.DocumentationNotRequested, DocumentationTargets: []intent.DocumentationTarget{},
		Application:     &intent.ApplicationIdentity{DisplayName: "Ledger", ModulePath: "example.com/alex/ledger"},
		Target:          &intent.ApplicationTarget{Base: "session_directory", Directory: "ledger"},
		InitialFeatures: []intent.InitialFeature{}, Exceptions: []intent.Exception{},
	}

	return interaction.Result{
		State: interaction.StatePlanReady, Outcome: interaction.OutcomePlanReady,
		Intent: &value,
		Plan: &plan.Plan{
			Intent: value.Operation, SourceFingerprint: testFingerprintA, Digest: testPlanDigest,
			Target: &plan.ApplicationTarget{Directory: "ledger", Parent: filepath.Dir(target), Path: target},
		},
		PlanYAML: []byte("digest: " + testPlanDigest + "\n"),
		Provenance: interaction.Provenance{Interpretations: []eval.Provenance{{
			Adapter: "codex-app-server", ThreadID: threadID,
			ContractVersion: eval.CurrentContractVersion, ModelSelection: eval.ModelBackendDefault,
			Timing: eval.TimingUnderSecond,
		}}},
	}
}

func completedApplicationResult(planned interaction.Result) interaction.Result {
	return interaction.Result{
		State: interaction.StateFinished, Outcome: interaction.OutcomeCompleted,
		Intent: planned.Intent, Plan: planned.Plan, PlanYAML: planned.PlanYAML,
		Provenance: planned.Provenance,
	}
}

func writeModule(t *testing.T, root string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.27.1\n"), 0o600)
	if err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}
