//go:build acceptance

package conversation_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/interaction"
)

func TestConversationalBuilderAcceptance(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeModule(t, root)

	before, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read initial project: %v", err)
	}

	completed := featurePlanResult("thread-evolution")
	completed.State = interaction.StateFinished
	completed.Outcome = interaction.OutcomeCompleted

	incomplete := featurePlanResult("thread-infrastructure")
	incomplete.State = interaction.StateFinished
	incomplete.Outcome = interaction.OutcomeValidationIncomplete
	incomplete.Diagnostics = []interaction.Diagnostic{{
		Code:    "HMGEN-VALIDATION-INFRASTRUCTURE",
		Phase:   interaction.PhaseValidation,
		Message: "database infrastructure is unavailable",
	}}

	unsupported := interaction.Result{
		State:   interaction.StateFinished,
		Outcome: interaction.OutcomeUnsupported,
		Diagnostics: []interaction.Diagnostic{{
			Code:    "HMGEN-REQUEST-UNSUPPORTED",
			Phase:   interaction.PhaseInterpretation,
			Message: "the request is outside the Hatmax Book",
		}},
	}

	engine := &fakeTurnEngine{
		fingerprint: testFingerprintA,
		previews: []interaction.Result{
			featurePlanResult("thread-initial"),
			featurePlanResult("thread-revision"),
			featurePlanResult("thread-drift"),
			unsupported,
			featurePlanResult("thread-infrastructure"),
		},
		executions: []interaction.Result{completed, incomplete},
	}
	clock := &testClock{value: time.Date(2026, time.September, 29, 22, 0, 0, 0, time.UTC)}
	coordinator := newConversationCoordinator(t, engine, clock)

	session, err := coordinator.Open(ctx, root, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("open project conversation: %v", err)
	}

	initial, err := session.Turn(ctx, conversation.TurnRequest{Content: "Add invoices."})
	if err != nil {
		t.Fatalf("plan initial feature: %v", err)
	}

	cancelled, err := session.Cancel(ctx, initial.OperationID)
	if err != nil {
		t.Fatalf("cancel initial plan: %v", err)
	}

	if cancelled.Conversation.Operations[0].Status != conversation.OperationCancelled {
		t.Fatalf("cancelled operation = %#v", cancelled.Conversation.Operations[0])
	}

	revised, err := session.Turn(ctx, conversation.TurnRequest{Content: "Use invoice number as the required identifier."})
	if err != nil {
		t.Fatalf("revise feature plan: %v", err)
	}

	evolved, err := session.Approve(ctx, revised.OperationID, testPlanDigest)
	if err != nil {
		t.Fatalf("approve revised feature: %v", err)
	}

	if evolved.Interaction.Outcome != interaction.OutcomeCompleted || evolved.Conversation.Operations[1].Status != conversation.OperationCompleted {
		t.Fatalf("evolved result = %#v", evolved)
	}

	err = session.Close()
	if err != nil {
		t.Fatalf("close evolved conversation: %v", err)
	}

	resumed, err := coordinator.Open(ctx, root, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("reopen evolved conversation: %v", err)
	}

	driftCandidate, err := resumed.Turn(ctx, conversation.TurnRequest{Content: "Add invoice notes."})
	if err != nil {
		t.Fatalf("plan drift candidate: %v", err)
	}

	err = resumed.Close()
	if err != nil {
		t.Fatalf("close drift candidate: %v", err)
	}

	engine.fingerprint = testFingerprintB

	drifted, err := coordinator.Open(ctx, root, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("reopen after drift: %v", err)
	}

	if status := operationStatus(drifted.Current(), driftCandidate.OperationID); status != conversation.OperationStale {
		t.Fatalf("drifted operation status = %q, want stale", status)
	}

	fresh, err := drifted.Reset(ctx)
	if err != nil {
		t.Fatalf("reset conversation: %v", err)
	}

	if len(fresh.Turns) != 0 || len(fresh.Operations) != 0 {
		t.Fatalf("fresh conversation retained prior work: %#v", fresh)
	}

	rejected, err := drifted.Turn(ctx, conversation.TurnRequest{Content: "Build this service with another framework."})
	if err != nil {
		t.Fatalf("reject off-domain work: %v", err)
	}

	if rejected.Interaction.Outcome != interaction.OutcomeUnsupported || rejected.Conversation.Operations[0].Status != conversation.OperationFailed {
		t.Fatalf("unsupported result = %#v", rejected)
	}

	planned, err := drifted.Turn(ctx, conversation.TurnRequest{Content: "Add invoice reminders."})
	if err != nil {
		t.Fatalf("plan infrastructure-bound work: %v", err)
	}

	validation, err := drifted.Approve(ctx, planned.OperationID, testPlanDigest)
	if err != nil {
		t.Fatalf("approve infrastructure-bound work: %v", err)
	}

	if validation.Interaction.Outcome != interaction.OutcomeValidationIncomplete ||
		validation.Conversation.Operations[1].Status != conversation.OperationCompleted ||
		len(validation.Interaction.Diagnostics) != 1 {
		t.Fatalf("validation-incomplete result = %#v", validation)
	}

	after, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read final project: %v", err)
	}

	if string(after) != string(before) {
		t.Fatal("conversation control flow changed project source outside execution results")
	}

	if len(engine.requests) != 7 {
		t.Fatalf("kernel requests = %d, want 7 previews and approvals", len(engine.requests))
	}

	if len(validation.Interaction.RetainedChanges) != 0 {
		t.Fatalf("incomplete validation reported retained changes: %#v", validation.Interaction.RetainedChanges)
	}

}

func operationStatus(value conversation.Conversation, operationID string) conversation.OperationStatus {
	for _, operation := range value.Operations {
		if operation.ID == operationID {
			return operation.Status
		}
	}

	return ""
}
