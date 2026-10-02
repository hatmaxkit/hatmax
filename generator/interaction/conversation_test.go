// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package interaction

import (
	"context"
	"testing"

	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/intent"
)

func TestPreviewTurnClassifiesConversationWithoutApproval(t *testing.T) {
	root := copySupportedProject(t)
	approver := approvingPort()
	interpreter := &functionInterpreter{interpret: func(_ context.Context, request eval.Request) (eval.InterpreterResult, error) {
		if len(request.Conversation) != 1 || request.Conversation[0].Content != "We are building a billing tool." {
			t.Fatalf("conversation = %#v, want retained visible context", request.Conversation)
		}

		return fixtureInterpreterResult(eval.Interpretation{
			SchemaVersion: eval.CurrentInterpretationSchemaVersion,
			Kind:          eval.InterpretationConversation,
			Response:      &eval.ConversationResponse{Content: "That gives us useful product context."},
		}), nil
	}}
	coordinator := newTestCoordinator(t, interpreter, approver, nil)

	result := coordinator.PreviewTurn(context.Background(), root, TurnRequest{
		Prompt: "Nice, let us keep it focused.",
		Conversation: []eval.DialogueTurn{{
			Role: eval.DialogueRoleUser, Content: "We are building a billing tool.",
		}},
	})

	if result.Outcome != OutcomeConversationResponse || result.State != StateConversation || result.Response == nil {
		t.Fatalf("PreviewTurn() = %#v, want conversational response", result)
	}

	if result.Plan != nil || approver.calls != 0 {
		t.Fatalf("conversation plan = %#v approval calls = %d, want neither", result.Plan, approver.calls)
	}
}

func TestPreviewTurnClassifiesClarificationUnsupportedAndPlanReady(t *testing.T) {
	tests := []struct {
		name           string
		interpretation eval.Interpretation
		wantOutcome    Outcome
		wantState      State
	}{
		{
			name: "clarification",
			interpretation: eval.Interpretation{
				SchemaVersion: eval.CurrentInterpretationSchemaVersion,
				Kind:          eval.InterpretationClarification,
				Clarifications: []intent.Clarification{{
					Field: "domain.fields", Question: "Which fields belong to the invoice?",
				}},
			},
			wantOutcome: OutcomeClarificationRequired,
			wantState:   StateClarifying,
		},
		{
			name: "unsupported",
			interpretation: eval.Interpretation{
				SchemaVersion: eval.CurrentInterpretationSchemaVersion,
				Kind:          eval.InterpretationUnsupported,
				Diagnostics: []intent.Diagnostic{{
					Code: "HMGEN-FRAMEWORK-UNSUPPORTED", Field: "archetype", Message: "React applications are outside Hatmax",
				}},
			},
			wantOutcome: OutcomeUnsupported,
			wantState:   StateFinished,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := copySupportedProject(t)
			approver := approvingPort()
			coordinator := newTestCoordinator(t, &functionInterpreter{interpret: func(context.Context, eval.Request) (eval.InterpreterResult, error) {
				return fixtureInterpreterResult(test.interpretation), nil
			}}, approver, nil)

			result := coordinator.PreviewTurn(context.Background(), root, TurnRequest{Prompt: "Build this."})
			if result.Outcome != test.wantOutcome || result.State != test.wantState {
				t.Fatalf("PreviewTurn() outcome/state = %q/%q, want %q/%q", result.Outcome, result.State, test.wantOutcome, test.wantState)
			}

			if approver.calls != 0 || !containsState(result.Transitions, StateCandidateChange) {
				t.Fatalf("approval calls = %d transitions = %v, want visible candidate without approval", approver.calls, result.Transitions)
			}
		})
	}

	root := copySupportedProject(t)
	approver := approvingPort()
	coordinator := newTestCoordinator(t, &functionInterpreter{interpret: admittedCreateFeature}, approver, nil)

	result := coordinator.PreviewTurn(context.Background(), root, TurnRequest{Prompt: "Add an invoice feature."})
	if result.Outcome != OutcomePlanReady || result.State != StatePlanReady || result.Plan == nil || len(result.PlanYAML) == 0 {
		t.Fatalf("PreviewTurn() = %#v, want inspectable plan", result)
	}

	if approver.calls != 0 || !containsState(result.Transitions, StateCandidateChange) {
		t.Fatalf("approval calls = %d transitions = %v, want visible candidate without approval", approver.calls, result.Transitions)
	}
}

func TestApprovedTurnRequiresTheRecomputedPlanDigest(t *testing.T) {
	root := copySupportedProject(t)
	approver := approvingPort()
	interpreter := &functionInterpreter{interpret: func(ctx context.Context, request eval.Request) (eval.InterpreterResult, error) {
		if len(request.Clarifications) != 1 || request.Clarifications[0].Answer != "/invoices" {
			t.Fatalf("clarifications = %#v, want conversation-derived route", request.Clarifications)
		}

		return admittedCreateFeature(ctx, request)
	}}
	coordinator := newTestCoordinator(t, interpreter, approver, nil)
	request := TurnRequest{
		Prompt: "Add the invoice feature.",
		Clarifications: []eval.ClarificationExchange{{
			Field: "domain.route", Question: "Which route?", Answer: "/invoices",
		}},
	}

	preview := coordinator.PreviewTurn(context.Background(), root, request)
	request.ApprovalPlanDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	result := coordinator.RunApprovedTurn(context.Background(), root, request)

	if preview.Plan == nil || result.Outcome != OutcomePlanStale || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "HMGEN-APPROVAL-DIGEST-MISMATCH" {
		t.Fatalf("approved result = %#v, want digest-bound rejection", result)
	}

	if approver.calls != 0 || result.Execution != nil {
		t.Fatalf("approval calls = %d execution = %#v, want no execution", approver.calls, result.Execution)
	}
}

func containsState(values []State, expected State) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}
