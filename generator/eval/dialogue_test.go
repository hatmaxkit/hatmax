// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package eval

import (
	"context"
	"strings"
	"testing"
)

func TestEvaluateDialogueReturnsBoundedConversationWithoutMutationAuthority(t *testing.T) {
	response := &ConversationResponse{Content: "Two."}
	interpreter := &fixtureInterpreter{outputs: map[string]Interpretation{
		"What is one plus one?": {
			Kind:     InterpretationConversation,
			Response: response,
		},
	}}
	turns := []DialogueTurn{
		{Role: DialogueRoleUser, Content: "Let us discuss the domain first."},
		{Role: DialogueRoleHatmax, Content: "What matters most to the users?"},
	}

	result, err := EvaluateDialogue(
		context.Background(),
		interpreter,
		"What is one plus one?",
		turns,
		nil,
		syntheticEvaluationContext(t),
	)
	if err != nil {
		t.Fatalf("EvaluateDialogue() error = %v", err)
	}

	if result.Response == nil || result.Response.Content != "Two." {
		t.Fatalf("Response = %#v, want bounded dialogue", result.Response)
	}

	if result.Intent != nil || result.Plan != nil || len(result.Diagnostics) != 0 || len(result.Clarifications) != 0 {
		t.Errorf("conversation result contains mutation authority: %#v", result)
	}

	turns[0].Content = "mutated"
	response.Content = "mutated"

	if interpreter.requests[0].Conversation[0].Content != "Let us discuss the domain first." {
		t.Errorf("request conversation was mutated: %#v", interpreter.requests[0].Conversation)
	}

	if result.Response.Content != "Two." {
		t.Errorf("response was mutated through interpreter storage: %#v", result.Response)
	}
}

func TestEvaluateDialogueRejectsInvalidConversationBounds(t *testing.T) {
	validTurn := DialogueTurn{Role: DialogueRoleUser, Content: "hello"}
	tests := []struct {
		name  string
		turns []DialogueTurn
		code  string
	}{
		{name: "too many turns", turns: repeatedDialogueTurns(validTurn, MaximumDialogueTurns+1), code: "evaluation_dialogue_limit"},
		{name: "unknown role", turns: []DialogueTurn{{Role: "assistant", Content: "hello"}}, code: "evaluation_dialogue_invalid"},
		{name: "empty content", turns: []DialogueTurn{{Role: DialogueRoleUser, Content: " "}}, code: "evaluation_dialogue_invalid"},
		{name: "oversized turn", turns: []DialogueTurn{{Role: DialogueRoleUser, Content: strings.Repeat("x", MaximumDialogueTurnBytes+1)}}, code: "evaluation_dialogue_too_large"},
		{name: "oversized aggregate", turns: repeatedDialogueTurns(DialogueTurn{Role: DialogueRoleUser, Content: strings.Repeat("x", MaximumDialogueTurnBytes)}, 5), code: "evaluation_dialogue_too_large"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := EvaluateDialogue(
				context.Background(),
				&fixtureInterpreter{},
				"continue",
				test.turns,
				nil,
				syntheticEvaluationContext(t),
			)
			requireEvaluationCode(t, err, test.code)
		})
	}
}

func TestEvaluateDialogueRejectsInvalidConversationResults(t *testing.T) {
	validIntent := validOutputIntent()
	tests := []struct {
		name           string
		interpretation Interpretation
	}{
		{name: "missing response", interpretation: Interpretation{Kind: InterpretationConversation}},
		{name: "empty response", interpretation: Interpretation{Kind: InterpretationConversation, Response: &ConversationResponse{Content: " "}}},
		{name: "oversized response", interpretation: Interpretation{Kind: InterpretationConversation, Response: &ConversationResponse{Content: strings.Repeat("x", MaximumDialogueTurnBytes+1)}}},
		{name: "response with intent", interpretation: Interpretation{Kind: InterpretationConversation, Response: &ConversationResponse{Content: "hello"}, Intent: &validIntent}},
		{name: "intent with response", interpretation: Interpretation{Kind: InterpretationIntent, Response: &ConversationResponse{Content: "hello"}, Intent: &validIntent}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			interpreter := &fixtureInterpreter{outputs: map[string]Interpretation{"prompt": test.interpretation}}

			_, err := Evaluate(context.Background(), interpreter, "prompt", syntheticEvaluationContext(t))
			requireEvaluationCode(t, err, "evaluation_result_invalid")
		})
	}
}

func repeatedDialogueTurns(value DialogueTurn, count int) []DialogueTurn {
	result := make([]DialogueTurn, count)
	for index := range result {
		result[index] = value
	}

	return result
}
