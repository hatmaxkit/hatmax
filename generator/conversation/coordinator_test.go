package conversation_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/eval"
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

type fakeTurnEngine struct {
	fingerprint string
	previews    []interaction.Result
	executions  []interaction.Result
	requests    []interaction.TurnRequest
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

	err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.24\n"), 0o600)
	if err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}
