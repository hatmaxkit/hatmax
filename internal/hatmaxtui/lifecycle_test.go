package hatmaxtui

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/execute"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/interaction"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

func TestModelOpensConversationAndReportsOpenFailures(t *testing.T) {
	session := &fakeSession{current: conversation.Conversation{ID: "conversation-1"}}
	value := newModel(context.Background(), "/project", func(
		context.Context,
		string,
		conversation.SessionOptions,
	) (Session, error) {
		return session, nil
	})

	opened := value.openSession()().(sessionOpenedMsg)
	updated, command := value.Update(opened)
	value = updated.(model)

	if command != nil || value.busy || value.session != session || value.value.ID != "conversation-1" {
		t.Fatalf("opened model = busy %t session %p value %#v command nil %t", value.busy, value.session, value.value, command == nil)
	}

	missing := newModel(context.Background(), "/project", nil).openSession()().(sessionOpenedMsg)
	if missing.err == nil {
		t.Fatal("missing session factory did not fail")
	}

	failedModel := newModel(context.Background(), "/project", func(
		context.Context,
		string,
		conversation.SessionOptions,
	) (Session, error) {
		return nil, errors.New("backend unavailable")
	})
	failed := failedModel.openSession()().(sessionOpenedMsg)
	updated, _ = failedModel.Update(failed)
	failedModel = updated.(model)

	if failedModel.status != "Unavailable" || !strings.Contains(failedModel.summary, "backend unavailable") {
		t.Fatalf("open failure status %q summary %q", failedModel.status, failedModel.summary)
	}
}

func TestModelSubmitsOneTurnAndFinishesItsResult(t *testing.T) {
	session := &fakeSession{turnResult: conversation.SessionResult{
		Conversation: conversation.Conversation{ID: "conversation-2"},
		Interaction:  interaction.Result{Outcome: interaction.OutcomeConversationResponse},
	}}
	value := readyModelWithSession(session)
	value.summary = "old summary"
	value.detail = "old detail"
	value.showDetail = true
	value.composer.SetValue("  Create invoices.  ")

	command := value.submit()
	if command == nil || !value.busy || value.status != "Planning" || value.composer.Value() != "" {
		t.Fatalf("submit state = command nil %t busy %t status %q composer %q", command == nil, value.busy, value.status, value.composer.Value())
	}

	message := command().(turnFinishedMsg)
	if message.err != nil || session.turnRequest.Content != "Create invoices." {
		t.Fatalf("turn result error %v request %#v", message.err, session.turnRequest)
	}

	updated, returned := value.Update(message)
	value = updated.(model)

	if returned != nil || value.busy || value.status != "Ready" || value.value.ID != "conversation-2" {
		t.Fatalf("finished model = busy %t status %q value %#v command nil %t", value.busy, value.status, value.value, returned == nil)
	}
}

func TestModelRejectsUnavailableSubmissionsAndReportsTurnFailure(t *testing.T) {
	value := newModel(context.Background(), "/project", nil)
	value.busy = false
	value.composer.SetValue("request")

	if value.submit() != nil {
		t.Fatal("submit accepted without a session")
	}

	value = readyModel()

	if value.submit() != nil {
		t.Fatal("submit accepted empty composer")
	}

	value.busy = true
	value.composer.SetValue("request")

	if value.submit() != nil {
		t.Fatal("submit accepted while busy")
	}

	value.busy = false
	value.finishWork(conversation.SessionResult{}, errors.New("interpretation failed"))

	if value.status != "Failed" || !strings.Contains(value.summary, "interpretation failed") || value.busy {
		t.Fatalf("turn failure status %q summary %q busy %t", value.status, value.summary, value.busy)
	}
}

func TestModelCancelsActiveAndPendingWork(t *testing.T) {
	value := readyModel()
	ctx, cancel := context.WithCancel(context.Background())
	value.busy = true
	value.cancel = cancel

	if command := value.cancelOperation(); command != nil {
		t.Fatal("active cancellation returned an asynchronous command")
	}

	if value.status != "Cancelling" || value.cancel != nil || ctx.Err() != context.Canceled {
		t.Fatalf("active cancellation status %q cancel nil %t context error %v", value.status, value.cancel == nil, ctx.Err())
	}

	session := &fakeSession{cancelResult: conversation.SessionResult{
		Interaction: interaction.Result{Outcome: interaction.OutcomeCancelled},
	}}
	value = readyModelWithSession(session)
	value.pendingID = "operation-1"

	command := value.cancelOperation()

	message := command().(turnFinishedMsg)
	if message.err != nil || session.cancelledOperation != "operation-1" || value.status != "Cancelling" {
		t.Fatalf("pending cancellation = operation %q status %q error %v", session.cancelledOperation, value.status, message.err)
	}

	value = readyModelWithSession(session)
	value.composer.SetValue("draft")

	if command := value.cancelOperation(); command != nil || value.composer.Value() != "" || value.status != "Ready" {
		t.Fatalf("idle cancellation = command nil %t composer %q status %q", command == nil, value.composer.Value(), value.status)
	}
}

func TestModelResetsConversationAndReportsResetFailure(t *testing.T) {
	session := &fakeSession{resetValue: conversation.Conversation{ID: "conversation-new"}}
	value := readyModelWithSession(session)
	value.pendingID = "operation-1"
	value.digest = "sha256:plan"
	value.detail = "details"
	value.showDetail = true

	command := value.reset()
	if command == nil || !value.busy || value.status != "Resetting" {
		t.Fatalf("reset start = command nil %t busy %t status %q", command == nil, value.busy, value.status)
	}

	message := command().(resetFinishedMsg)
	updated, returned := value.Update(message)

	value = updated.(model)
	if returned != nil || value.value.ID != "conversation-new" || value.pendingID != "" || value.digest != "" || value.showDetail {
		t.Fatalf("reset result = value %#v pending %q digest %q details %t", value.value, value.pendingID, value.digest, value.showDetail)
	}

	value = readyModelWithSession(&fakeSession{resetErr: errors.New("state unavailable")})
	message = value.reset()().(resetFinishedMsg)
	updated, _ = value.Update(message)

	value = updated.(model)
	if value.status != "Reset failed" || !strings.Contains(value.summary, "state unavailable") {
		t.Fatalf("reset failure status %q summary %q", value.status, value.summary)
	}

	value.busy = true
	if value.reset() != nil {
		t.Fatal("reset accepted while busy")
	}
}

func TestModelCloseCancelsWorkAndClosesSession(t *testing.T) {
	session := &fakeSession{closeErr: errors.New("ignored close error")}
	value := readyModelWithSession(session)
	ctx, cancel := context.WithCancel(context.Background())
	value.cancel = cancel

	value.close()

	if !session.closed || ctx.Err() != context.Canceled {
		t.Fatalf("close = session closed %t context error %v", session.closed, ctx.Err())
	}

	withoutSession := newModel(context.Background(), "/project", nil)
	withoutSession.close()
}

func TestPresentationHelpersCoverEveryOutcomeAndConciseFailure(t *testing.T) {
	outcomes := map[interaction.Outcome]string{
		interaction.OutcomeConversationResponse:  "Ready",
		interaction.OutcomeClarificationRequired: "Needs input",
		interaction.OutcomePlanReady:             "Plan ready",
		interaction.OutcomeCompleted:             "Completed",
		interaction.OutcomeValidationIncomplete:  "Validation incomplete",
		interaction.OutcomeUnsupported:           "Unsupported",
		interaction.OutcomeIntentRejected:        "Rejected",
		interaction.OutcomePlanStale:             "Plan stale",
		interaction.OutcomeExecutionFailed:       "Failed",
		interaction.OutcomeFailed:                "Failed",
		interaction.OutcomeCancelled:             "Cancelled",
		interaction.Outcome("unknown"):           "Failed",
	}
	for outcome, expected := range outcomes {
		if actual := outcomeStatus(outcome); actual != expected {
			t.Errorf("outcomeStatus(%q) = %q, want %q", outcome, actual, expected)
		}
	}

	results := map[string]string{
		"completed: generated invoice":          "Completed\ngenerated invoice",
		"execution_failed: apply failed":        "Execution failed\napply failed",
		"plan_stale: project changed":           "Plan stale\nproject changed",
		"intent_rejected: missing field":        "Request rejected\nmissing field",
		"unsupported: unrelated request":        "Unsupported request\nunrelated request",
		"cancelled: user cancelled":             "Cancelled\nuser cancelled",
		"failed: backend unavailable":           "Failed\nbackend unavailable",
		"validation_incomplete: command failed": "Validation incomplete\nThe generated changes were applied, but required external tests could not run.",
		"completed: ":                           "Completed",
		"plain_identifier":                      "Plain identifier",
		"custom: custom failure":                "custom: custom failure",
	}
	for input, expected := range results {
		if actual := resultTurnPresentation(input); actual != expected {
			t.Errorf("resultTurnPresentation(%q) = %q, want %q", input, actual, expected)
		}
	}

	diagnostics := []struct {
		code     string
		message  string
		expected string
	}{
		{"HMGEN-DOCKER", "daemon unavailable", "Docker-based tests could not run because Docker infrastructure is unavailable."},
		{"HMGEN-TOOL", "exec: executable file not found", "A required executable is not available on PATH."},
		{"HMGEN-VALIDATION", "observed false; expected true", "Validation did not meet the expected condition."},
		{"HMGEN-EXEC", "first line\nraw output", "first line"},
	}
	for _, diagnostic := range diagnostics {
		if actual := conciseDiagnosticMessage(diagnostic.code, diagnostic.message); actual != diagnostic.expected {
			t.Errorf("conciseDiagnosticMessage(%q) = %q, want %q", diagnostic.message, actual, diagnostic.expected)
		}
	}

	if failureSummary("Failed.", nil) != "Failed." {
		t.Fatal("nil failure unexpectedly added detail")
	}

	if actual := failureSummary("Failed.", errors.New("HMGEN-EXEC: apply failed")); actual != "Failed.\napply failed" {
		t.Fatalf("failure summary = %q", actual)
	}
}

func TestPlanSummaryCoversFallbacksAndMetadata(t *testing.T) {
	if planSummary(nil) != "" {
		t.Fatal("nil plan produced a summary")
	}

	tests := []struct {
		name     string
		value    plan.Plan
		expected []string
	}{
		{
			name:     "anonymous application",
			value:    plan.Plan{Intent: intent.OperationCreateApplication},
			expected: []string{"Create a Hatmax application"},
		},
		{
			name: "feature metadata",
			value: plan.Plan{
				Intent:  intent.OperationCreateFeature,
				Feature: "invoice",
				Domain: intent.Domain{
					Entity: "Invoice", Route: "/invoices",
					Rules: []intent.BusinessRule{{Description: "Numbers are unique"}},
				},
				Capabilities:     []string{"custom_capability", "custom_capability"},
				AffectedSurfaces: []string{"handler", "handler"},
				Documentation:    intent.DocumentationPlanned,
			},
			expected: []string{"Create the Invoice feature", "Entity: Invoice", "Route: /invoices", "Behavior: Numbers are unique", "Custom capability", "Changes: Handler", "Documentation: planned change"},
		},
		{
			name:     "field fallback",
			value:    plan.Plan{Intent: intent.OperationAddField, Feature: "invoice"},
			expected: []string{"Add a field to the Invoice feature"},
		},
		{
			name:     "validation fallback",
			value:    plan.Plan{Intent: intent.OperationAddValidation, Feature: "invoice"},
			expected: []string{"Add validation to the Invoice feature"},
		},
		{
			name:     "empty intent",
			value:    plan.Plan{},
			expected: []string{"Review the proposed Hatmax change"},
		},
		{
			name:     "unknown intent",
			value:    plan.Plan{Intent: intent.Operation("custom_operation")},
			expected: []string{"Custom operation"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			summary := planSummary(&test.value)
			for _, expected := range test.expected {
				if !strings.Contains(summary, expected) {
					t.Errorf("summary does not contain %q:\n%s", expected, summary)
				}
			}
		})
	}
}

func TestResultDetailIncludesInspectableExecutionEvidence(t *testing.T) {
	result := conversation.SessionResult{Interaction: interaction.Result{
		PlanYAML:         []byte("intent: create_feature"),
		FreshnessChanges: []project.Change{{Kind: project.ChangeModified, Path: "main.go"}},
		Report: &execute.ExecutionReport{
			Conformance: execute.ConformanceResult{Passed: true},
			Commands:    []execute.CommandEvidence{{Name: "test", ExitCode: 0}},
		},
	}}

	detail := resultDetail(result)
	for _, expected := range []string{
		"Typed plan\nintent: create_feature",
		"Stale modified: main.go",
		"Conformance: true",
		"Validation test: exit 0",
	} {
		if !strings.Contains(detail, expected) {
			t.Errorf("detail does not contain %q:\n%s", expected, detail)
		}
	}
}

func TestSmallPresentationBoundsRemainStable(t *testing.T) {
	if truncatePlain("anything", 0) != "" || truncatePlain("anything", 1) != "…" || truncatePlain("ok", 4) != "ok" {
		t.Fatal("plain truncation boundary changed")
	}

	long := strings.Repeat("a", 300)
	if actual := conciseDiagnosticMessage("", long); len([]rune(actual)) != 240 {
		t.Fatalf("diagnostic length = %d, want 240", len([]rune(actual)))
	}

	if humanizeIdentifier("") != "" || documentationLabel(intent.Documentation("custom_docs")) != "Custom docs" {
		t.Fatal("identifier fallback changed")
	}

	if actual := clarificationPresentation([]string{"Only one?"}); actual != "Only one?" {
		t.Fatalf("single clarification = %q", actual)
	}

	if actual := resultGroupPresentation("completed: done", []string{"HMGEN-EXEC: ignored"}); actual != "Completed\ndone" {
		t.Fatalf("non-validation result group = %q", actual)
	}

	value := readyModel()
	value.showHelp = true
	value.pendingID = "operation-1"
	value.detail = "details"
	value.showDetail = true
	value.composer.SetValue("draft")
	value.resize(200, defaultHeight)

	footer := value.footerView()
	for _, expected := range []string{"Ctrl+D summary", "Ctrl+J newline", "Ctrl+N new conversation", "Esc cancel", "Ctrl+C quit", "F1 close"} {
		if !strings.Contains(footer, expected) {
			t.Errorf("expanded footer does not contain %q: %s", expected, footer)
		}
	}
}

func TestModelRoutesQuitAndResizeBoundaries(t *testing.T) {
	value := readyModel()
	ctx, cancel := context.WithCancel(context.Background())
	value.cancel = cancel

	updated, command := value.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	value = updated.(model)
	if command == nil || ctx.Err() != context.Canceled || value.cancel != nil {
		t.Fatalf("quit = command nil %t context error %v cancel nil %t", command == nil, ctx.Err(), value.cancel == nil)
	}

	width, height := value.width, value.height
	value.resize(0, 10)

	if value.width != width || value.height != height {
		t.Fatalf("invalid resize changed size to %dx%d", value.width, value.height)
	}
}

func TestRunValidatesConfigurationAndHandlesTerminalQuit(t *testing.T) {
	err := Run(context.Background(), Config{})
	if err == nil {
		t.Fatal("Run() accepted missing terminal streams")
	}

	var output bytes.Buffer

	err = Run(context.Background(), Config{
		Input: bytes.NewBuffer([]byte{3}), Output: &output, ErrorOutput: &output,
		Root: "/project",
		Sessions: func(context.Context, string, conversation.SessionOptions) (Session, error) {
			return &fakeSession{}, nil
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestModelInitialCommandsAndCompactPresentationBranches(t *testing.T) {
	value := newModel(context.Background(), "/project", nil)
	if value.Init() == nil {
		t.Fatal("Init() command = nil")
	}

	message := activityTick(9)().(activityTickMsg)
	if message.id != 9 {
		t.Fatalf("activity tick ID = %d", message.id)
	}

	if value.approve() != nil {
		t.Fatal("approve accepted an unavailable plan")
	}

	busyView := value.View().Content
	if !strings.Contains(busyView, "Opening…") {
		t.Fatalf("busy view omitted active phase:\n%s", busyView)
	}

	if composerContentHeight("abcdefgh", 2) != 4 || composerContentHeight("content", 0) != 1 {
		t.Fatal("composer height boundary changed")
	}

	if lipgloss.Width(activityView(3, 5)) != 2 {
		t.Fatal("compact activity width changed")
	}

	empty := conversationPresentation(conversation.Conversation{}, "", "", false, 80)
	if !strings.Contains(empty, "Build and evolve Hatmax applications") {
		t.Fatalf("empty conversation = %q", empty)
	}

	turns := conversation.Conversation{Turns: []conversation.Turn{{
		Role: conversation.RoleHatmax, Kind: conversation.TurnResult, Content: "Plan ready: sha256:plan",
	}}}

	presentation := conversationPresentation(turns, "Plan ready", "details", true, 80)
	if strings.Count(presentation, "Plan ready") != 1 || !strings.Contains(presentation, "Technical details") {
		t.Fatalf("plan presentation = %q", presentation)
	}
}

func TestPlanSummaryCoversRequiredFieldAndNonFeatureUnit(t *testing.T) {
	value := plan.Plan{
		Intent: intent.OperationCreateApplication,
		Units: []plan.Unit{
			{Intent: intent.OperationAddField, Feature: "ignored"},
			{
				Intent: intent.OperationCreateFeature, Feature: "invoice",
				Domain: intent.Domain{Fields: []intent.Field{{
					Name: "number", Type: "string", Required: true,
				}}},
			},
		},
	}

	summary := planSummary(&value)
	if !strings.Contains(summary, "Number: string, required") || strings.Contains(summary, "Ignored") {
		t.Fatalf("application summary = %q", summary)
	}

	goal := fieldGoalPresentation(intent.Field{Name: "number", Type: "string", Required: true})
	if goal != "required string field Number" {
		t.Fatalf("required field goal = %q", goal)
	}
}
