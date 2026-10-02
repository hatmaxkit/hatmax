// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package conversation

import (
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/intent"
)

func TestValidateRejectsEachConversationBoundary(t *testing.T) {
	now := time.Date(2026, time.September, 30, 18, 0, 0, 0, time.UTC)
	base := testConversation(t, "conversation-validation", ScopeProject, now)

	validTurn := Turn{
		ID: 1, Role: RoleUser, Kind: TurnDialogue, Content: "hello", CreatedAt: now,
	}
	validOperation := ProposedOperation{
		ID: "operation-1", ConversationID: base.ID, Status: OperationCandidate,
		RequestSummary: "Create invoices", CreatedAt: now, UpdatedAt: now,
	}

	tests := []struct {
		name   string
		mutate func(*Conversation)
		code   string
	}{
		{"schema", func(value *Conversation) { value.SchemaVersion = 0 }, "conversation_schema_unsupported"},
		{"identity", func(value *Conversation) { value.ID = " " }, "conversation_identity_invalid"},
		{"scope kind", func(value *Conversation) { value.Scope.Kind = ScopeKind("other") }, "conversation_scope_invalid"},
		{"scope key", func(value *Conversation) { value.Scope.Key = "invalid" }, "conversation_scope_invalid"},
		{"contract", func(value *Conversation) { value.BookContract.BookVersion = 0 }, "conversation_contract_invalid"},
		{"backend", func(value *Conversation) { value.BackendIdentity.Adapter = "" }, "conversation_backend_invalid"},
		{"status", func(value *Conversation) { value.Status = Status("other") }, "conversation_status_invalid"},
		{"time", func(value *Conversation) { value.UpdatedAt = value.CreatedAt.Add(-time.Second) }, "conversation_time_invalid"},
		{"turn count", func(value *Conversation) { value.Turns = make([]Turn, MaximumTurns+1) }, "state_limit_exceeded"},
		{"operation count", func(value *Conversation) { value.Operations = make([]ProposedOperation, MaximumOperations+1) }, "state_limit_exceeded"},
		{"turn identity", func(value *Conversation) {
			value.Turns = []Turn{{Role: RoleUser, Kind: TurnDialogue, Content: "hello", CreatedAt: now}}
		}, "conversation_turn_invalid"},
		{"turn order", func(value *Conversation) {
			value.Turns = []Turn{validTurn, validTurn}
			value.NextTurnID = 2
		}, "conversation_turn_invalid"},
		{"turn role", func(value *Conversation) {
			turn := validTurn
			turn.Role = Role("other")
			value.Turns = []Turn{turn}
			value.NextTurnID = 2
		}, "conversation_turn_invalid"},
		{"turn kind", func(value *Conversation) {
			turn := validTurn
			turn.Kind = TurnKind("other")
			value.Turns = []Turn{turn}
			value.NextTurnID = 2
		}, "conversation_turn_invalid"},
		{"turn content", func(value *Conversation) {
			turn := validTurn
			turn.Content = " "
			value.Turns = []Turn{turn}
			value.NextTurnID = 2
		}, "conversation_turn_invalid"},
		{"turn size", func(value *Conversation) {
			turn := validTurn
			turn.Content = strings.Repeat("x", MaximumTurnBytes+1)
			value.Turns = []Turn{turn}
			value.NextTurnID = 2
		}, "state_limit_exceeded"},
		{"turn operation", func(value *Conversation) {
			turn := validTurn
			turn.OperationID = "missing"
			value.Turns = []Turn{turn}
			value.NextTurnID = 2
		}, "conversation_turn_invalid"},
		{"diagnostic size", func(value *Conversation) {
			turn := validTurn
			turn.Kind = TurnDiagnostic
			turn.Content = strings.Repeat("x", MaximumDiagnosticBytes+1)
			value.Turns = []Turn{turn}
			value.NextTurnID = 2
		}, "state_limit_exceeded"},
		{"next turn", func(value *Conversation) {
			value.Turns = []Turn{validTurn}
			value.NextTurnID = 1
		}, "conversation_turn_invalid"},
		{"operation identity", func(value *Conversation) {
			operation := validOperation
			operation.ID = ""
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"operation duplicate", func(value *Conversation) {
			value.Operations = []ProposedOperation{validOperation, validOperation}
		}, "conversation_operation_invalid"},
		{"operation status", func(value *Conversation) {
			operation := validOperation
			operation.Status = OperationStatus("other")
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"operation summary size", func(value *Conversation) {
			operation := validOperation
			operation.RequestSummary = strings.Repeat("x", MaximumTurnBytes+1)
			value.Operations = []ProposedOperation{operation}
		}, "state_limit_exceeded"},
		{"decision count", func(value *Conversation) {
			operation := validOperation
			operation.Decisions = make([]Decision, MaximumOperationDecisions+1)
			value.Operations = []ProposedOperation{operation}
		}, "state_limit_exceeded"},
		{"clarification count", func(value *Conversation) {
			operation := validOperation
			operation.Pending = make([]intent.Clarification, MaximumOperationDecisions)
			operation.Decisions = []Decision{{}}
			value.Operations = []ProposedOperation{operation}
		}, "state_limit_exceeded"},
		{"clarification required", func(value *Conversation) {
			operation := validOperation
			operation.Pending = []intent.Clarification{{Field: "domain.route"}}
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"clarification size", func(value *Conversation) {
			operation := validOperation
			operation.Pending = []intent.Clarification{{Field: strings.Repeat("x", MaximumDecisionBytes+1), Question: "Route?"}}
			value.Operations = []ProposedOperation{operation}
		}, "state_limit_exceeded"},
		{"clarification duplicate", func(value *Conversation) {
			operation := validOperation
			question := intent.Clarification{Field: "domain.route", Question: "Route?"}
			operation.Pending = []intent.Clarification{question, question}
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"decision required", func(value *Conversation) {
			operation := validOperation
			operation.Decisions = []Decision{{Field: "domain.route", Question: "Route?"}}
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"decision size", func(value *Conversation) {
			operation := validOperation
			operation.Decisions = []Decision{{Field: "domain.route", Question: "Route?", Answer: strings.Repeat("x", MaximumDecisionBytes+1)}}
			value.Operations = []ProposedOperation{operation}
		}, "state_limit_exceeded"},
		{"decision duplicate", func(value *Conversation) {
			operation := validOperation
			decision := Decision{Field: "domain.route", Question: "Route?", Answer: "/invoices"}
			operation.Decisions = []Decision{decision, decision}
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"decision pending", func(value *Conversation) {
			operation := validOperation
			operation.Pending = []intent.Clarification{{Field: "domain.route", Question: "Route?"}}
			operation.Decisions = []Decision{{Field: "domain.route", Question: "Route?", Answer: "/invoices"}}
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"operation time", func(value *Conversation) {
			operation := validOperation
			operation.UpdatedAt = operation.CreatedAt.Add(-time.Second)
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"intent contract", func(value *Conversation) {
			operation := validOperation
			intentValue := testIntent()
			operation.Intent = &intentValue
			operation.IntentContract = operation.Intent.SchemaVersion + 1
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"intent schema", func(value *Conversation) {
			operation := validOperation
			intentValue := testIntent()
			operation.Intent = &intentValue
			operation.Intent.HatmaxVersion = ""
			operation.IntentContract = operation.Intent.SchemaVersion
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"plan digest", func(value *Conversation) {
			operation := validOperation
			operation.PlanDigest = "invalid"
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
		{"source fingerprint", func(value *Conversation) {
			operation := validOperation
			operation.SourceFingerprint = "invalid"
			value.Operations = []ProposedOperation{operation}
		}, "conversation_operation_invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := clone(base)
			test.mutate(&value)

			err := Validate(value)
			if err == nil {
				t.Fatal("Validate() error = nil")
			}

			model, ok := err.(Error)
			if !ok || model.Code != test.code {
				t.Fatalf("Validate() error = %#v, want code %q", err, test.code)
			}
		})
	}
}

func TestNonTerminalOperationsReturnsDetachedActiveProposals(t *testing.T) {
	value := Conversation{Operations: []ProposedOperation{
		{ID: "candidate", Status: OperationCandidate, Decisions: []Decision{{Field: "domain.route", Answer: "/invoices"}}},
		{ID: "clarifying", Status: OperationClarifying},
		{ID: "planned", Status: OperationPlanned},
		{ID: "completed", Status: OperationCompleted},
	}}

	operations := value.NonTerminalOperations()
	if len(operations) != 3 {
		t.Fatalf("non-terminal operations = %#v", operations)
	}

	operations[0].Decisions[0].Answer = "/changed"
	if value.Operations[0].Decisions[0].Answer != "/invoices" {
		t.Fatal("NonTerminalOperations() aliased retained decisions")
	}
}

func TestErrorFormattingAndInvalidConstruction(t *testing.T) {
	if actual := (Error{Code: "failure", Message: "failed"}).Error(); actual != "failure: failed" {
		t.Fatalf("fieldless error = %q", actual)
	}

	_, err := New("", Scope{}, BookContract{}, BackendIdentity{}, time.Time{})
	if err == nil {
		t.Fatal("New() accepted an invalid conversation")
	}
}

func TestLifecycleRejectsMissingOperationsAndInvalidScopeChanges(t *testing.T) {
	now := time.Date(2026, time.September, 30, 18, 30, 0, 0, time.UTC)
	value := testConversation(t, "conversation-errors", ScopePreProject, now)

	err := value.SetClarifyingOperation("missing", nil, nil, now)
	if err == nil {
		t.Fatal("SetClarifyingOperation() accepted a missing operation")
	}

	err = value.TransitionOperation("missing", OperationFailed, "failed", now)
	if err == nil {
		t.Fatal("TransitionOperation() accepted a missing operation")
	}

	planned := testIntent()

	err = value.SetPlannedOperation("missing", planned.SchemaVersion, planned, testFingerprint, testFingerprint, now)
	if err == nil {
		t.Fatal("SetPlannedOperation() accepted a missing operation")
	}

	_, _, err = value.Reset("", now)
	if err == nil {
		t.Fatal("Reset() accepted an empty replacement identity")
	}

	invalidTarget := Scope{Kind: ScopeProject, Key: "invalid"}

	err = value.Rebind(invalidTarget, value.BookContract, now)
	if err == nil {
		t.Fatal("Rebind() accepted an invalid target scope")
	}

	validTarget, err := ResolveScope(ScopeProject, t.TempDir())
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}

	err = value.Rebind(validTarget, BookContract{}, now)
	if err == nil {
		t.Fatal("Rebind() accepted an invalid contract")
	}

	project := testConversation(t, "conversation-project", ScopeProject, now)

	err = project.Rebind(validTarget, project.BookContract, now)
	if err == nil {
		t.Fatal("Rebind() accepted a project source")
	}

	_, err = ResolveScope(ScopeKind("other"), t.TempDir())
	if err == nil {
		t.Fatal("ResolveScope() accepted an invalid kind")
	}

	_, err = ResolveScope(ScopeProject, "")
	if err == nil {
		t.Fatal("ResolveScope() accepted an empty path")
	}
}

func TestLifecycleRetentionAndTransitionBoundaries(t *testing.T) {
	now := time.Date(2026, time.September, 30, 18, 45, 0, 0, time.UTC)

	value := testConversation(t, "conversation-retention", ScopeProject, now)

	err := value.AppendTurn(RoleUser, TurnDialogue, "hello", "", now)
	if err != nil {
		t.Fatalf("AppendTurn() error = %v", err)
	}

	if len(value.RecentTurns(0, 10)) != 0 || len(value.RecentTurns(1, 1)) != 0 {
		t.Fatal("RecentTurns() ignored zero or byte bounds")
	}

	protected := clone(value)

	protected.Operations = make([]ProposedOperation, 0, MaximumOperations+1)
	for index := 0; index <= MaximumOperations; index++ {
		protected.Operations = append(protected.Operations, ProposedOperation{
			ID: "operation-" + strings.Repeat("x", index+1), ConversationID: protected.ID,
			Status: OperationCandidate, RequestSummary: "request", CreatedAt: now, UpdatedAt: now,
		})
	}

	_, err = Prune(protected)
	if err == nil {
		t.Fatal("Prune() accepted too many protected operations")
	}

	transitions := []struct {
		from OperationStatus
		to   OperationStatus
		want bool
	}{
		{OperationCandidate, OperationClarifying, true},
		{OperationCandidate, OperationPlanned, true},
		{OperationCandidate, OperationCancelled, true},
		{OperationCandidate, OperationFailed, true},
		{OperationClarifying, OperationPlanned, true},
		{OperationClarifying, OperationCancelled, true},
		{OperationClarifying, OperationFailed, true},
		{OperationPlanned, OperationCompleted, true},
		{OperationPlanned, OperationStale, true},
		{OperationPlanned, OperationCancelled, true},
		{OperationPlanned, OperationFailed, true},
		{OperationCompleted, OperationCandidate, false},
	}
	for _, transition := range transitions {
		if actual := allowedTransition(transition.from, transition.to); actual != transition.want {
			t.Errorf("allowedTransition(%q, %q) = %t, want %t", transition.from, transition.to, actual, transition.want)
		}
	}
}

func TestCloneIntentDetachesOptionalApplicationState(t *testing.T) {
	field := intent.Field{Name: "notes", Type: "text"}
	validation := intent.ValidationRule{Field: "number", Kind: "required"}
	value := intent.Intent{
		Application: &intent.ApplicationIdentity{DisplayName: "Ledger"},
		Target:      &intent.ApplicationTarget{Directory: "ledger"},
		Domain:      intent.Domain{Field: &field, Validation: &validation},
		InitialFeatures: []intent.InitialFeature{{
			Feature: "invoice", Domain: intent.Domain{Fields: []intent.Field{{Name: "number", Type: "string"}}},
		}},
	}

	cloned := cloneIntent(value)
	cloned.Application.DisplayName = "Changed"
	cloned.Target.Directory = "changed"
	cloned.Domain.Field.Name = "changed"
	cloned.Domain.Validation.Kind = "changed"
	cloned.InitialFeatures[0].Domain.Fields[0].Name = "changed"

	if value.Application.DisplayName != "Ledger" || value.Target.Directory != "ledger" || value.Domain.Field.Name != "notes" || value.Domain.Validation.Kind != "required" || value.InitialFeatures[0].Domain.Fields[0].Name != "number" {
		t.Fatal("cloneIntent() aliased optional application state")
	}
}
