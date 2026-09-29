package interaction

import (
	"context"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

func TestInteractionContractsKeepApprovalAndClarificationExplicit(t *testing.T) {
	approval := ApprovalRequest{
		PlanDigest:         "sha256:plan",
		ProjectFingerprint: "sha256:project",
		PlanYAML:           []byte("digest: sha256:plan\n"),
	}
	approver := recordingApprover{decision: ApprovalGranted}

	decision, err := approver.Approve(context.Background(), approval)
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	if decision != ApprovalGranted {
		t.Fatalf("Approve() = %q, want %q", decision, ApprovalGranted)
	}

	clarifier := recordingClarifier{response: ClarificationResponse{
		Answers: []ClarificationAnswer{{Field: "domain.route", Answer: "/invoices"}},
	}}

	response, err := clarifier.Clarify(context.Background(), ClarificationRequest{
		Round: 1,
		Questions: []intent.Clarification{{
			Field:    "domain.route",
			Question: "Which route should expose the feature?",
		}},
	})
	if err != nil {
		t.Fatalf("Clarify() error = %v", err)
	}

	if len(response.Answers) != 1 || response.Answers[0].Field != "domain.route" {
		t.Fatalf("Clarify() = %#v, want one bound answer", response)
	}
}

func TestInteractionStatesAndOutcomesAreStable(t *testing.T) {
	states := []State{
		StateConversation,
		StateCandidateChange,
		StateInspecting,
		StateInterpreting,
		StateClarifying,
		StatePlanReady,
		StateAwaitingApproval,
		StateExecuting,
		StateValidating,
		StateFinished,
	}

	wantStates := []State{
		"conversation",
		"candidate_change",
		"inspecting",
		"interpreting",
		"clarifying",
		"plan_ready",
		"awaiting_approval",
		"executing",
		"validating",
		"finished",
	}
	for index := range wantStates {
		if states[index] != wantStates[index] {
			t.Errorf("state %d = %q, want %q", index, states[index], wantStates[index])
		}
	}

	outcomes := []Outcome{
		OutcomeConversationResponse,
		OutcomePlanReady,
		OutcomeCompleted,
		OutcomeCancelled,
		OutcomeUnsupported,
		OutcomeIntentRejected,
		OutcomeClarificationRequired,
		OutcomePlanStale,
		OutcomeExecutionFailed,
		OutcomeValidationIncomplete,
		OutcomeFailed,
	}

	wantOutcomes := []Outcome{
		"conversation_response",
		"plan_ready",
		"completed",
		"cancelled",
		"unsupported",
		"intent_rejected",
		"clarification_required",
		"plan_stale",
		"execution_failed",
		"validation_incomplete",
		"failed",
	}
	for index := range wantOutcomes {
		if outcomes[index] != wantOutcomes[index] {
			t.Errorf("outcome %d = %q, want %q", index, outcomes[index], wantOutcomes[index])
		}
	}
}

type recordingApprover struct {
	decision ApprovalDecision
}

func (a recordingApprover) Approve(context.Context, ApprovalRequest) (ApprovalDecision, error) {
	return a.decision, nil
}

type recordingClarifier struct {
	response ClarificationResponse
}

func (c recordingClarifier) Clarify(context.Context, ClarificationRequest) (ClarificationResponse, error) {
	return c.response, nil
}
