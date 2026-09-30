package conversation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/execute"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/interaction"
)

func TestCoordinatorOperationHelpersRejectInvalidApprovalState(t *testing.T) {
	value := Conversation{Operations: []ProposedOperation{{
		ID: "operation-1", Status: OperationCandidate,
	}}}

	_, err := plannedOperation(value, "missing")
	if err == nil {
		t.Fatal("plannedOperation() accepted a missing operation")
	}

	_, err = plannedOperation(value, "operation-1")
	if err == nil {
		t.Fatal("plannedOperation() accepted a candidate operation")
	}

	value.Operations[0].Status = OperationPlanned

	operation, err := plannedOperation(value, "operation-1")
	if err != nil || operation.ID != "operation-1" {
		t.Fatalf("plannedOperation() = %#v, %v", operation, err)
	}

	if operationByID(value, "missing") != nil {
		t.Fatal("operationByID() found a missing operation")
	}
}

func TestCoordinatorDecisionBindingRejectsIncompleteAnswers(t *testing.T) {
	operation := ProposedOperation{Pending: []intent.Clarification{
		{Field: "domain.route", Question: "Which route?"},
		{Field: "domain.fields", Question: "Which fields?"},
	}}

	tests := []struct {
		name    string
		request TurnRequest
	}{
		{"wrong count", TurnRequest{Decisions: []Decision{{Field: "domain.route", Answer: "/invoices"}}}},
		{"blank answer", TurnRequest{Decisions: []Decision{{Field: "domain.route", Answer: "/invoices"}, {Field: "domain.fields", Answer: " "}}}},
		{"wrong field", TurnRequest{Decisions: []Decision{{Field: "domain.route", Answer: "/invoices"}, {Field: "other", Answer: "number"}}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := bindTurnDecisions(operation, test.request)
			if err == nil {
				t.Fatal("bindTurnDecisions() error = nil")
			}
		})
	}

	decisions, err := bindTurnDecisions(operation, TurnRequest{Content: "Use /invoices and a number field."})
	if err != nil || len(decisions) != 2 || decisions[1].Question != "Which fields?" {
		t.Fatalf("combined decisions = %#v, %v", decisions, err)
	}
}

func TestCoordinatorDecisionAndTurnHelpersPreserveOrder(t *testing.T) {
	previous := []Decision{
		{Field: "domain.route", Answer: "/old"},
		{Field: "domain.entity", Answer: "Invoice"},
	}
	additional := []Decision{
		{Field: "domain.route", Answer: "/invoices"},
		{Field: "domain.fields", Answer: "number"},
	}

	merged := mergeDecisions(previous, additional)
	if len(merged) != 3 || merged[0].Answer != "/invoices" || merged[2].Field != "domain.fields" {
		t.Fatalf("merged decisions = %#v", merged)
	}

	exchanges := operationDecisions(&ProposedOperation{Decisions: previous}, additional)
	if len(exchanges) != 3 || exchanges[0].Answer != "/invoices" {
		t.Fatalf("clarification exchanges = %#v", exchanges)
	}

	exchanges = operationDecisions(nil, additional)
	if len(exchanges) != 2 || exchanges[1].Field != "domain.fields" {
		t.Fatalf("additional exchanges = %#v", exchanges)
	}

	value := Conversation{}
	bindLastTurn(&value, "operation-1", TurnGoal)
	value.Turns = []Turn{{Kind: TurnDialogue}}
	bindLastTurn(&value, "operation-1", TurnGoal)

	if value.Turns[0].OperationID != "operation-1" || value.Turns[0].Kind != TurnGoal {
		t.Fatalf("bound turn = %#v", value.Turns[0])
	}
}

func TestCoordinatorBackendAndOutcomeHelpersCoverTerminalStates(t *testing.T) {
	value := Conversation{BackendThreadID: "old"}
	updateBackendIdentity(&value, interaction.Result{Provenance: interaction.Provenance{Interpretations: []eval.Provenance{
		{ThreadID: "thread-old"},
		{ThreadID: ""},
		{ThreadID: "thread-new", Adapter: "codex", BackendVersion: "1.0", EffectiveModel: "model"},
	}}})

	if value.BackendThreadID != "thread-new" || value.BackendIdentity.Adapter != "codex" || value.BackendIdentity.Model != "model" {
		t.Fatalf("backend identity = %#v thread %q", value.BackendIdentity, value.BackendThreadID)
	}

	updateBackendIdentity(&value, interaction.Result{})

	if value.BackendThreadID != "thread-new" {
		t.Fatal("empty provenance changed backend identity")
	}

	tests := map[interaction.Outcome]OperationStatus{
		interaction.OutcomeCompleted:            OperationCompleted,
		interaction.OutcomeValidationIncomplete: OperationCompleted,
		interaction.OutcomeCancelled:            OperationCancelled,
		interaction.OutcomePlanStale:            OperationStale,
		interaction.OutcomeFailed:               OperationFailed,
	}
	for outcome, expected := range tests {
		if actual := operationStatusForResult(interaction.Result{Outcome: outcome}); actual != expected {
			t.Errorf("operationStatusForResult(%q) = %q, want %q", outcome, actual, expected)
		}
	}
}

func TestCoordinatorResultSummaryKeepsOnlyConciseUsefulEvidence(t *testing.T) {
	if actual := resultSummary(interaction.Result{}); actual != "interaction failed" {
		t.Fatalf("empty summary = %q", actual)
	}

	validation := resultSummary(interaction.Result{
		Outcome: interaction.OutcomeValidationIncomplete,
		Diagnostics: []interaction.Diagnostic{{
			Code: "HMGEN-DOCKER", Message: "permission denied connecting to Docker",
		}},
		RetainedChanges: []execute.Change{{Target: "main.go"}},
	})
	if !strings.Contains(validation, "generated changes were applied") || strings.Contains(validation, "main.go") {
		t.Fatalf("validation summary = %q", validation)
	}

	failed := resultSummary(interaction.Result{
		Outcome:         interaction.OutcomeExecutionFailed,
		Diagnostics:     []interaction.Diagnostic{{Code: "HMGEN-EXEC", Message: "apply failed\nraw stack"}},
		RetainedChanges: []execute.Change{{Target: "main.go"}, {Target: "go.mod"}},
	})
	if !strings.Contains(failed, "apply failed") || !strings.Contains(failed, "retained changes: main.go, go.mod") || strings.Contains(failed, "raw stack") {
		t.Fatalf("failure summary = %q", failed)
	}

	changes := make([]execute.Change, 0, 1000)
	for index := 0; index < 1000; index++ {
		changes = append(changes, execute.Change{Target: strings.Repeat("x", 32)})
	}

	bounded := resultSummary(interaction.Result{Outcome: interaction.OutcomeCompleted, RetainedChanges: changes})
	if len(bounded) != MaximumResultSummaryBytes {
		t.Fatalf("bounded summary length = %d, want %d", len(bounded), MaximumResultSummaryBytes)
	}
}

func TestCoordinatorDiagnosticMessagesHideRawInfrastructureOutput(t *testing.T) {
	tests := []struct {
		name       string
		diagnostic interaction.Diagnostic
		expected   string
	}{
		{"docker permission", interaction.Diagnostic{Code: "HMGEN-DOCKER", Message: "permission denied"}, "Docker-based tests could not run because Docker is not accessible to the current user."},
		{"docker unavailable", interaction.Diagnostic{Message: "testcontainers daemon missing"}, "Docker-based tests could not run because Docker infrastructure is unavailable."},
		{"observed", interaction.Diagnostic{Message: "observed false; expected true"}, "validation did not meet the expected condition"},
		{"first line", interaction.Diagnostic{Message: "apply failed\nraw stack"}, "apply failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := conversationDiagnosticMessage(test.diagnostic); actual != test.expected {
				t.Fatalf("diagnostic message = %q, want %q", actual, test.expected)
			}
		})
	}

	long := conversationDiagnosticMessage(interaction.Diagnostic{Message: strings.Repeat("x", 600)})
	if len(long) != 512 {
		t.Fatalf("bounded diagnostic length = %d, want 512", len(long))
	}
}

func TestCoordinatorAppendsResultAndDiagnosticTurns(t *testing.T) {
	now := time.Date(2026, time.September, 30, 18, 0, 0, 0, time.UTC)
	value := testConversation(t, "conversation-results", ScopeProject, now)

	err := value.AddOperation("operation-1", "Create invoices", now)
	if err != nil {
		t.Fatalf("AddOperation() error = %v", err)
	}

	result := interaction.Result{Diagnostics: []interaction.Diagnostic{{
		Code: "HMGEN-EXEC", Message: "apply failed\nraw stack",
	}}}

	err = appendResultTurns(&value, "operation-1", result, "execution_failed: apply failed", now)
	if err != nil {
		t.Fatalf("appendResultTurns() error = %v", err)
	}

	if len(value.Turns) != 2 || value.Turns[0].Kind != TurnResult || value.Turns[1].Kind != TurnDiagnostic || strings.Contains(value.Turns[1].Content, "raw stack") {
		t.Fatalf("result turns = %#v", value.Turns)
	}

	err = appendResultTurns(&value, "operation-1", interaction.Result{}, "", now)
	if err != nil {
		t.Fatalf("append empty result error = %v", err)
	}
}

func TestResolveSessionRootAndLocalIdentifiers(t *testing.T) {
	_, _, err := ResolveSessionRoot("")
	if err == nil {
		t.Fatal("ResolveSessionRoot() accepted an empty root")
	}

	_, _, err = ResolveSessionRoot(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("ResolveSessionRoot() accepted a missing root")
	}

	file := filepath.Join(t.TempDir(), "file")

	err = os.WriteFile(file, []byte("content"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, _, err = ResolveSessionRoot(file)
	if err == nil {
		t.Fatal("ResolveSessionRoot() accepted a file")
	}

	parent := t.TempDir()

	absolute, kind, err := ResolveSessionRoot(parent)
	if err != nil || !filepath.IsAbs(absolute) || kind != ScopePreProject {
		t.Fatalf("pre-project root = %q %q %v", absolute, kind, err)
	}

	err = os.WriteFile(filepath.Join(parent, "go.mod"), []byte("module example.com/project\n"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	_, kind, err = ResolveSessionRoot(parent)
	if err != nil || kind != ScopeProject {
		t.Fatalf("project root kind = %q, %v", kind, err)
	}

	first, err := randomLocalID("conversation")
	if err != nil || !strings.HasPrefix(first, "conversation-") || len(first) != len("conversation-")+32 {
		t.Fatalf("random local ID = %q, %v", first, err)
	}
}
