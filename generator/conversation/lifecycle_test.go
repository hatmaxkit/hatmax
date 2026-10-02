// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package conversation

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/intent"
)

const testFingerprint = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

func TestResolveScopeIsOpaqueStableAndKindSpecific(t *testing.T) {
	root := t.TempDir()

	project, err := ResolveScope(ScopeProject, filepath.Join(root, "."))
	if err != nil {
		t.Fatalf("ResolveScope(project) error = %v", err)
	}

	again, err := ResolveScope(ScopeProject, root)
	if err != nil {
		t.Fatalf("ResolveScope(project again) error = %v", err)
	}

	preProject, err := ResolveScope(ScopePreProject, root)
	if err != nil {
		t.Fatalf("ResolveScope(pre-project) error = %v", err)
	}

	if project.Key != again.Key || project.Key == preProject.Key {
		t.Fatalf("scope keys = project %q again %q pre-project %q", project.Key, again.Key, preProject.Key)
	}

	if strings.Contains(project.Key, filepath.Base(root)) || len(project.Key) != 64 {
		t.Errorf("scope key exposes path or is not SHA-256: %q", project.Key)
	}
}

func TestConversationLifecycleRetainsNoApprovalAuthority(t *testing.T) {
	now := time.Date(2026, time.September, 29, 19, 0, 0, 0, time.UTC)
	value := testConversation(t, "conversation-1", ScopePreProject, now)
	value.BackendThreadID = "thread-1"

	err := value.AddOperation("operation-1", "Create an invoice feature", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("AddOperation() error = %v", err)
	}

	err = value.AppendTurn(RoleUser, TurnGoal, "Create an invoice feature", "operation-1", now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("AppendTurn() error = %v", err)
	}

	err = value.TransitionOperation("operation-1", OperationClarifying, "", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("TransitionOperation(clarifying) error = %v", err)
	}

	plannedIntent := testIntent()

	err = value.SetPlannedOperation(
		"operation-1",
		plannedIntent.SchemaVersion,
		plannedIntent,
		testFingerprint,
		testFingerprint,
		now.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatalf("SetPlannedOperation() error = %v", err)
	}

	value.ReconcileResume(value.BookContract, testFingerprint, now.Add(5*time.Minute))

	if value.Operations[0].Status != OperationPlanned {
		t.Fatalf("operation status = %q, want planned", value.Operations[0].Status)
	}

	value.ReconcileResume(value.BookContract, "sha256:"+strings.Repeat("1", 64), now.Add(6*time.Minute))

	if value.Operations[0].Status != OperationStale {
		t.Fatalf("operation status = %q, want stale", value.Operations[0].Status)
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if strings.Contains(strings.ToLower(string(encoded)), "approval") {
		t.Fatalf("durable conversation contains approval authority: %s", encoded)
	}

	operationType := reflect.TypeOf(ProposedOperation{})
	for index := 0; index < operationType.NumField(); index++ {
		if strings.Contains(strings.ToLower(operationType.Field(index).Name), "approval") {
			t.Fatalf("ProposedOperation field %q persists approval", operationType.Field(index).Name)
		}
	}
}

func TestClarifyingOperationRetainsExplicitDecisions(t *testing.T) {
	now := time.Date(2026, time.September, 29, 19, 0, 0, 0, time.UTC)
	value := testConversation(t, "conversation-1", ScopeProject, now)

	err := value.AddOperation("operation-1", "Add invoices", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("AddOperation() error = %v", err)
	}

	decisions := []Decision{{
		Field: "domain.route", Question: "Which route should expose invoices?", Answer: "/invoices",
	}}

	err = value.SetClarifyingOperation("operation-1", []intent.Clarification{{
		Field: "domain.fields", Question: "Which fields belong to invoices?",
	}}, decisions, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("SetClarifyingOperation() error = %v", err)
	}

	operation := value.Operations[0]
	if operation.Status != OperationClarifying || len(operation.Decisions) != 1 || operation.Decisions[0].Answer != "/invoices" {
		t.Fatalf("operation = %#v, want retained clarification decision", operation)
	}

	decisions[0].Answer = "/changed"
	if operation.Decisions[0].Answer != "/invoices" {
		t.Fatal("operation decisions alias caller storage")
	}
}

func TestConversationResetAndRebindReplaceAuthority(t *testing.T) {
	now := time.Date(2026, time.September, 29, 19, 0, 0, 0, time.UTC)
	value := testConversation(t, "conversation-1", ScopePreProject, now)
	value.BackendThreadID = "thread-1"

	err := value.AddOperation("operation-1", "Create billing", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("AddOperation() error = %v", err)
	}

	plannedIntent := testIntent()

	err = value.SetPlannedOperation("operation-1", plannedIntent.SchemaVersion, plannedIntent, testFingerprint, testFingerprint, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("SetPlannedOperation() error = %v", err)
	}

	projectScope, err := ResolveScope(ScopeProject, t.TempDir())
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}

	projectContract := BookContract{BookVersion: 1, InterpreterVersion: 3}

	err = value.Rebind(projectScope, projectContract, now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Rebind() error = %v", err)
	}

	if value.Scope != projectScope || value.BookContract != projectContract || value.BackendThreadID != "" || value.Operations[0].Status != OperationStale {
		t.Fatalf("rebound conversation = %#v", value)
	}

	previous, next, err := value.Reset("conversation-2", now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	if previous.Status != StatusReset || next.Status != StatusActive || next.ID != "conversation-2" || next.Scope != projectScope {
		t.Fatalf("reset = previous %#v next %#v", previous, next)
	}

	if len(next.Turns) != 0 || len(next.Operations) != 0 || next.BackendThreadID != "" {
		t.Fatalf("new conversation retained prior authority: %#v", next)
	}
}

func TestConversationRetentionPrunesOldEligibleRecords(t *testing.T) {
	now := time.Date(2026, time.September, 29, 19, 0, 0, 0, time.UTC)
	value := testConversation(t, "conversation-1", ScopeProject, now)

	for index := 0; index < MaximumTurns+1; index++ {
		err := value.AppendTurn(RoleUser, TurnDialogue, fmt.Sprintf("turn-%03d", index), "", now.Add(time.Duration(index+1)*time.Second))
		if err != nil {
			t.Fatalf("AppendTurn(%d) error = %v", index, err)
		}
	}

	if len(value.Turns) != MaximumTurns || value.Turns[0].Content != "turn-001" {
		t.Fatalf("retained turns = %d first %q", len(value.Turns), value.Turns[0].Content)
	}

	replay := value.RecentTurns(MaximumReplayTurns, MaximumReplayBytes)
	if len(replay) != MaximumReplayTurns || replay[0].Content != "turn-169" || replay[len(replay)-1].Content != "turn-200" {
		t.Fatalf("replay window = %d %#v", len(replay), replay)
	}

	for index := 0; index < MaximumOperations; index++ {
		operationID := fmt.Sprintf("operation-%03d", index)

		err := value.AddOperation(operationID, "candidate", now.Add(time.Duration(index+300)*time.Second))
		if err != nil {
			t.Fatalf("AddOperation(%d) error = %v", index, err)
		}
	}

	before := Clone(value)
	err := value.AddOperation("operation-over-limit", "candidate", now.Add(time.Hour))
	requireCode(t, err, "state_limit_exceeded")

	if !reflect.DeepEqual(value, before) {
		t.Fatal("failed retention mutation changed the conversation")
	}
}

func TestConversationRejectsIncompatibleContractAndInvalidTransitions(t *testing.T) {
	now := time.Date(2026, time.September, 29, 19, 0, 0, 0, time.UTC)

	value := testConversation(t, "conversation-1", ScopeProject, now)

	err := value.AddOperation("operation-1", "Create billing", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("AddOperation() error = %v", err)
	}

	err = value.TransitionOperation("operation-1", OperationCompleted, "done", now.Add(2*time.Minute))
	if err == nil {
		t.Fatal("candidate operation transitioned directly to completed")
	}

	value.ReconcileResume(BookContract{BookVersion: 2, InterpreterVersion: 3}, testFingerprint, now.Add(3*time.Minute))

	if value.Status != StatusIncompatible || value.BackendThreadID != "" {
		t.Fatalf("incompatible conversation = %#v", value)
	}
}

func testConversation(t *testing.T, id string, kind ScopeKind, now time.Time) Conversation {
	t.Helper()

	scope, err := ResolveScope(kind, t.TempDir())
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}

	value, err := New(
		id,
		scope,
		BookContract{BookVersion: 1, InterpreterVersion: 3},
		BackendIdentity{Adapter: "codex-app-server", Version: "0.158.0", Model: "backend_default"},
		now,
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return value
}

func testIntent() intent.Intent {
	return intent.Intent{
		SchemaVersion:      intent.CurrentSchemaVersion,
		Operation:          intent.OperationCreateFeature,
		ProjectFingerprint: testFingerprint,
		HatmaxVersion:      "v0.5.0",
		BookVersion:        1,
		Archetype:          "server_rendered_crud",
		Feature:            "invoice",
		Domain: intent.Domain{
			Entity: "Invoice",
			Fields: []intent.Field{{Name: "number", Type: "string", Required: true}},
		},
		Capabilities:         []string{"postgres_persistence", "htmx_form", "runtime_validation"},
		Documentation:        intent.DocumentationNotRequested,
		DocumentationTargets: []intent.DocumentationTarget{},
		Exceptions:           []intent.Exception{},
	}
}

func requireCode(t *testing.T, err error, code string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want %q", code)
	}

	value, ok := err.(Error)
	if !ok {
		t.Fatalf("error = %T %v, want conversation.Error", err, err)
	}

	if value.Code != code {
		t.Fatalf("error code = %q, want %q", value.Code, code)
	}
}
