// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package hatmaxtui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/execute"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/interaction"
	"hatmax.adrianpk.com/generator/plan"
)

func TestModelResizesEveryInteractiveSurface(t *testing.T) {
	initial := readyModel()

	updated, _ := initial.Update(tea.WindowSizeMsg{Width: 112, Height: 38})
	value := updated.(model)

	if value.width != 112 || value.height != 38 {
		t.Fatalf("size = %dx%d, want 112x38", value.width, value.height)
	}

	if value.viewport.Width() != 112 || value.viewport.Height() != 35 {
		t.Fatalf("viewport = %dx%d, want 112x35", value.viewport.Width(), value.viewport.Height())
	}

	value.composer.SetValue("first\nsecond\nthird")
	value.resizeComposer()
	value.resizeSurfaces()

	if value.composer.Height() != 3 || value.viewport.Height() != 33 {
		t.Fatalf("dynamic layout = composer %d viewport %d, want 3 and 33", value.composer.Height(), value.viewport.Height())
	}
}

func TestModelExposesAccessibleControlsAndRestorableView(t *testing.T) {
	value := readyModel()
	view := value.View()

	if !view.AltScreen || view.WindowTitle != "Hatmax" {
		t.Fatalf("view terminal contract = alt %t title %q", view.AltScreen, view.WindowTitle)
	}

	for _, expected := range []string{"Hatmax", "F1", "help"} {
		if !strings.Contains(view.Content, expected) {
			t.Errorf("view does not contain %q:\n%s", expected, view.Content)
		}
	}

	for _, unexpected := range []string{"Ready", "Needs input", "Enter send", "Ctrl+C quit", "Ctrl+A"} {
		if strings.Contains(view.Content, unexpected) {
			t.Errorf("ready view contains redundant %q:\n%s", unexpected, view.Content)
		}
	}
}

func TestF1AndControlHToggleHelpWithoutEditing(t *testing.T) {
	value := readyModel()
	value.composer.SetValue("request")

	updated, _ := value.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	value = updated.(model)

	if !value.showHelp || value.composer.Value() != "request" {
		t.Fatalf("F1 help = %t composer = %q", value.showHelp, value.composer.Value())
	}

	updated, _ = value.Update(tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl})
	value = updated.(model)

	if value.showHelp || value.composer.Value() != "request" {
		t.Fatalf("Ctrl+H help = %t composer = %q", value.showHelp, value.composer.Value())
	}
}

func TestComposerUsesOnePromptAcrossMultipleLines(t *testing.T) {
	value := readyModel()
	value.composer.SetValue("Create invoices\nwith a required number\nand optional notes")
	value.resizeComposer()
	value.resizeSurfaces()

	view := value.View().Content
	if count := strings.Count(view, "> "); count != 1 {
		t.Fatalf("composer prompt count = %d, want 1:\n%s", count, view)
	}
}

func TestConversationSurfacesUseSymmetricHorizontalMargins(t *testing.T) {
	composer := ansi.Strip(composerView("content", 80))
	user := ansi.Strip(conversationTurn(conversation.Turn{
		Role: conversation.RoleUser, Kind: conversation.TurnDialogue, Content: "content",
	}, 80))
	activity := ansi.Strip(activityView(80, 0))

	for name, surface := range map[string]string{
		"composer": composer,
		"user":     user,
		"activity": activity,
	} {
		firstLine, _, _ := strings.Cut(surface, "\n")
		if !strings.HasPrefix(firstLine, " ") || lipgloss.Width(firstLine) != 79 {
			t.Errorf("%s first line width/prefix = %d %q, want one-cell left and right margins", name, lipgloss.Width(firstLine), firstLine)
		}
	}
}

func TestActivityIndicatorIsAnimatedWithoutClaimingPercentage(t *testing.T) {
	first := activityView(80, 0)
	second := activityView(80, 8)

	if first == second {
		t.Fatal("activity indicator did not change between frames")
	}

	if lipgloss.Width(first) != 79 || lipgloss.Width(second) != 79 {
		t.Fatalf("activity widths = %d and %d, want 79 with an implicit right margin", lipgloss.Width(first), lipgloss.Width(second))
	}

	if strings.Contains(first, "%") || strings.Contains(second, "%") {
		t.Fatalf("indeterminate activity exposes a percentage: %q %q", first, second)
	}
}

func TestActivityTicksOnlyForCurrentWork(t *testing.T) {
	value := readyModel()
	value.busy = true
	value.activityID = 7

	updated, command := value.Update(activityTickMsg{id: 7})
	current := updated.(model)

	if current.activityFrame != 1 || command == nil {
		t.Fatalf("current activity frame = %d command nil = %t", current.activityFrame, command == nil)
	}

	updated, command = current.Update(activityTickMsg{id: 6})
	stale := updated.(model)

	if stale.activityFrame != 1 || command != nil {
		t.Fatalf("stale activity frame = %d command nil = %t", stale.activityFrame, command == nil)
	}
}

func TestBusyFooterShowsOnlyContextualActionAndActivePhase(t *testing.T) {
	value := readyModel()
	value.busy = true
	value.status = "Planning"

	footer := value.footerView()
	for _, expected := range []string{"Esc cancel", "Planning…"} {
		if !strings.Contains(footer, expected) {
			t.Errorf("busy footer does not contain %q: %s", expected, footer)
		}
	}

	for _, unexpected := range []string{"Enter send", "Ctrl+C quit", "F1 help"} {
		if strings.Contains(footer, unexpected) {
			t.Errorf("busy footer contains redundant %q: %s", unexpected, footer)
		}
	}
}

func TestEscapeClearsPendingInput(t *testing.T) {
	value := readyModel()
	value.composer.SetValue("Create invoices")

	updated, _ := value.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := updated.(model)

	if result.composer.Value() != "" || result.status != "Ready" {
		t.Fatalf("escape left input %q status %q", result.composer.Value(), result.status)
	}
}

func TestPlanResultIsInspectableAndApprovalUsesDisplayedIdentity(t *testing.T) {
	session := &fakeSession{}
	value := readyModelWithSession(session)
	conversationValue := conversation.Conversation{Turns: []conversation.Turn{{
		Role: conversation.RoleHatmax, Kind: conversation.TurnResult,
		Content: "Plan ready: sha256:plan",
	}}}
	result := conversation.SessionResult{
		Conversation: conversationValue,
		OperationID:  "operation-1",
		Interaction: interaction.Result{
			Outcome: interaction.OutcomePlanReady,
			Plan: &plan.Plan{
				Intent: intent.OperationCreateApplication,
				Application: &intent.ApplicationIdentity{
					DisplayName: "Ledger", ModulePath: "example.com/alex/ledger",
				},
				Target: &plan.ApplicationTarget{Directory: "ledger"},
				Units: []plan.Unit{{
					Intent: intent.OperationCreateFeature, Feature: "invoice",
					Domain: intent.Domain{Fields: []intent.Field{
						{Name: "number", Type: "string", Required: true},
						{Name: "notes", Type: "text"},
					}},
					Capabilities:     []string{"postgres_persistence", "htmx_form", "runtime_validation"},
					AffectedSurfaces: []string{"migration", "model", "store", "service", "handler", "templates", "wiring", "tests"},
				}},
				Documentation: intent.DocumentationNotRequested,
				Digest:        "sha256:plan",
			},
			PlanYAML: []byte("intent: create_feature\ndigest: sha256:plan\n"),
		},
	}

	value.finishWork(result, nil)

	view := value.View().Content
	for _, expected := range []string{
		"Plan ready", "Create the Ledger Hatmax application", "example.com/alex/ledger",
		"Initial feature: Invoice", "Number: string, required", "Notes: text, optional",
		"PostgreSQL persistence", "runtime validation", "Ctrl+A approve", "Ctrl+D details",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("plan view does not contain %q:\n%s", expected, view)
		}
	}

	if strings.Contains(view, "intent: create_feature") || strings.Contains(view, "sha256:plan") {
		t.Errorf("default plan view exposes technical serialization:\n%s", view)
	}

	updated, _ := value.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	value = updated.(model)

	view = value.View().Content
	for _, expected := range []string{"Technical details", "intent: create_feature", "sha256:plan", "Ctrl+D summary"} {
		if !strings.Contains(view, expected) {
			t.Errorf("detailed plan view does not contain %q:\n%s", expected, view)
		}
	}

	command := value.approve()
	if command == nil {
		t.Fatal("approve command is nil")
	}

	message := command().(turnFinishedMsg)
	if message.err != nil || session.approvedOperation != "operation-1" || session.approvedDigest != "sha256:plan" {
		t.Fatalf("approval = %q %q error %v", session.approvedOperation, session.approvedDigest, message.err)
	}
}

func TestConversationPresentationKeepsRolesHumanAndDiagnosticsSecondary(t *testing.T) {
	value := conversation.Conversation{Turns: []conversation.Turn{
		{Role: conversation.RoleUser, Kind: conversation.TurnDialogue, Content: "What is 1+1?"},
		{Role: conversation.RoleHatmax, Kind: conversation.TurnDialogue, Content: "2"},
		{Role: conversation.RoleHatmax, Kind: conversation.TurnDiagnostic, Content: "HMGEN-EXEC: apply failed"},
	}}

	presentation := conversationPresentation(value, "", "", false, 80)
	for _, expected := range []string{"You", "What is 1+1?", "Hatmax", "2", "apply failed"} {
		if !strings.Contains(presentation, expected) {
			t.Errorf("conversation does not contain %q:\n%s", expected, presentation)
		}
	}

	for _, unexpected := range []string{"[dialogue]", "[diagnostic]", "HMGEN-EXEC"} {
		if strings.Contains(presentation, unexpected) {
			t.Errorf("conversation exposes internal marker %q:\n%s", unexpected, presentation)
		}
	}
}

func TestConversationPresentationCondensesValidationInfrastructureFailure(t *testing.T) {
	stack := "observed \"panic: permission denied while trying to connect to the docker API\n" +
		"github.com/testcontainers/testcontainers-go.Run()\nmore stack\"; expected \"declared test infrastructure is available\""
	value := conversation.Conversation{Turns: []conversation.Turn{
		{Role: conversation.RoleHatmax, Kind: conversation.TurnResult, OperationID: "operation-1", Content: "validation_incomplete: " + stack + "; retained changes: main.go, go.mod"},
		{Role: conversation.RoleHatmax, Kind: conversation.TurnDiagnostic, OperationID: "operation-1", Content: "HMGEN-VALIDATION-INCOMPLETE: " + stack},
	}}

	presentation := conversationPresentation(value, "", "", false, 200)
	for _, expected := range []string{
		"Validation incomplete",
		"The generated changes were applied.",
		"Docker-based tests could not run because Docker is not accessible to the current user.",
	} {
		if !strings.Contains(presentation, expected) {
			t.Errorf("condensed validation result does not contain %q:\n%s", expected, presentation)
		}
	}

	for _, unexpected := range []string{"panic", "testcontainers", "more stack", "HMGEN-", "retained changes"} {
		if strings.Contains(presentation, unexpected) {
			t.Errorf("condensed validation result contains noise %q:\n%s", unexpected, presentation)
		}
	}

	if count := strings.Count(presentation, "Hatmax"); count != 1 {
		t.Fatalf("Hatmax label count = %d, want 1:\n%s", count, presentation)
	}
}

func TestConversationPresentationGroupsConsecutiveClarifications(t *testing.T) {
	value := conversation.Conversation{Turns: []conversation.Turn{
		{Role: conversation.RoleUser, Kind: conversation.TurnGoal, Content: "Create an invoice application."},
		{Role: conversation.RoleHatmax, Kind: conversation.TurnClarification, Content: "What Go module path should it use?"},
		{Role: conversation.RoleHatmax, Kind: conversation.TurnClarification, Content: "Which fields belong to the initial entity?"},
	}}

	presentation := conversationPresentation(value, "", "", false, 80)
	for _, expected := range []string{
		"Before I can continue, I need these details:",
		"1. What Go module path should it use?",
		"2. Which fields belong to the initial entity?",
	} {
		if !strings.Contains(presentation, expected) {
			t.Errorf("grouped clarification does not contain %q:\n%s", expected, presentation)
		}
	}

	if count := strings.Count(presentation, "Hatmax"); count != 1 {
		t.Fatalf("Hatmax label count = %d, want 1:\n%s", count, presentation)
	}
}

func TestPlanSummaryDescribesSupportedChanges(t *testing.T) {
	tests := []struct {
		name     string
		value    plan.Plan
		expected []string
	}{
		{
			name: "feature",
			value: plan.Plan{
				Intent: intent.OperationCreateFeature, Feature: "invoice",
				Domain: intent.Domain{
					Route:  "/invoices",
					Fields: []intent.Field{{Name: "number", Type: "string", Required: true}},
				},
			},
			expected: []string{"Create the Invoice feature", "Number: string, required", "Route: /invoices"},
		},
		{
			name: "field",
			value: plan.Plan{
				Intent: intent.OperationAddField, Feature: "invoice",
				Domain: intent.Domain{Field: &intent.Field{Name: "notes", Type: "text"}},
			},
			expected: []string{"Add the optional text field Notes to the Invoice feature"},
		},
		{
			name: "validation",
			value: plan.Plan{
				Intent: intent.OperationAddValidation, Feature: "invoice",
				Domain: intent.Domain{Validation: &intent.ValidationRule{
					Field: "number", Kind: "required", Scope: intent.ValidationDurable,
				}},
			},
			expected: []string{"Add Required validation to Number in the Invoice feature"},
		},
		{
			name: "documentation",
			value: plan.Plan{
				Intent: intent.OperationDocumentFeature, Feature: "invoice",
				Documentation: intent.DocumentationExisting,
				DocumentationTargets: []intent.DocumentationTarget{{
					Quadrant: intent.DocumentationReference, Subject: "invoice feature",
				}},
			},
			expected: []string{"Document the Invoice feature", "Documentation: existing behavior", "Reference for invoice feature"},
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

func TestResultDetailPresentsFailuresRetainedChangesAndPersistence(t *testing.T) {
	result := conversation.SessionResult{
		PersistenceFailed: true,
		Interaction: interaction.Result{
			Outcome: interaction.OutcomeExecutionFailed,
			Diagnostics: []interaction.Diagnostic{{
				Code: "HMGEN-EXEC", Phase: interaction.PhaseApplication, Message: "apply failed",
			}},
			RetainedChanges: []execute.Change{{Kind: "write_file", Target: "main.go"}},
		},
	}

	detail := resultDetail(result)
	for _, expected := range []string{"HMGEN-EXEC", "apply failed", "Retained write_file: main.go", "continues in memory only"} {
		if !strings.Contains(detail, expected) {
			t.Errorf("detail does not contain %q:\n%s", expected, detail)
		}
	}
}

func TestResultDetailCondensesRawInfrastructureOutput(t *testing.T) {
	result := conversation.SessionResult{Interaction: interaction.Result{
		Outcome: interaction.OutcomeValidationIncomplete,
		Diagnostics: []interaction.Diagnostic{{
			Code: "HMGEN-VALIDATION-INCOMPLETE", Phase: interaction.PhaseValidation,
			Message: "observed \"panic: permission denied connecting to Docker\nlong stack\"; expected infrastructure",
		}},
	}}

	detail := resultDetail(result)
	if !strings.Contains(detail, "Docker-based tests could not run because Docker is not accessible to the current user.") {
		t.Fatalf("technical detail omitted concise cause:\n%s", detail)
	}

	for _, unexpected := range []string{"panic", "long stack"} {
		if strings.Contains(detail, unexpected) {
			t.Errorf("technical detail contains raw output %q:\n%s", unexpected, detail)
		}
	}
}

func readyModel() model {
	return readyModelWithSession(&fakeSession{})
}

func readyModelWithSession(session *fakeSession) model {
	value := newModel(context.Background(), "/project", nil)
	value.busy = false
	value.session = session
	value.status = "Ready"
	value.refreshViewport()

	return value
}

type fakeSession struct {
	current            conversation.Conversation
	turnResult         conversation.SessionResult
	turnErr            error
	turnRequest        conversation.TurnRequest
	approveResult      conversation.SessionResult
	approveErr         error
	approvedOperation  string
	approvedDigest     string
	cancelResult       conversation.SessionResult
	cancelErr          error
	cancelledOperation string
	resetValue         conversation.Conversation
	resetErr           error
	closed             bool
	closeErr           error
}

func (session *fakeSession) Current() conversation.Conversation {
	return session.current
}

func (session *fakeSession) Turn(
	_ context.Context,
	request conversation.TurnRequest,
) (conversation.SessionResult, error) {
	session.turnRequest = request

	return session.turnResult, session.turnErr
}

func (session *fakeSession) Approve(
	_ context.Context,
	operationID string,
	digest string,
) (conversation.SessionResult, error) {
	session.approvedOperation = operationID
	session.approvedDigest = digest

	return session.approveResult, session.approveErr
}

func (session *fakeSession) Cancel(
	_ context.Context,
	operationID string,
) (conversation.SessionResult, error) {
	session.cancelledOperation = operationID

	return session.cancelResult, session.cancelErr
}

func (session *fakeSession) Reset(context.Context) (conversation.Conversation, error) {
	return session.resetValue, session.resetErr
}

func (session *fakeSession) Close() error {
	session.closed = true

	return session.closeErr
}
