package hatmaxtui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/execute"
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

	if value.viewport.Width() != 112 || value.viewport.Height() != 28 {
		t.Fatalf("viewport = %dx%d, want 112x28", value.viewport.Width(), value.viewport.Height())
	}
}

func TestModelExposesAccessibleControlsAndRestorableView(t *testing.T) {
	value := readyModel()
	view := value.View()

	if !view.AltScreen || view.WindowTitle != "Hatmax" {
		t.Fatalf("view terminal contract = alt %t title %q", view.AltScreen, view.WindowTitle)
	}

	for _, expected := range []string{
		"Hatmax", "Status: Ready", "enter", "send", "ctrl+a", "approve plan", "ctrl+c", "quit",
	} {
		if !strings.Contains(view.Content, expected) {
			t.Errorf("view does not contain %q:\n%s", expected, view.Content)
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
			Outcome:  interaction.OutcomePlanReady,
			Plan:     &plan.Plan{Digest: "sha256:plan"},
			PlanYAML: []byte("intent: create_feature\ndigest: sha256:plan\n"),
		},
	}

	value.finishWork(result, nil)

	view := value.View().Content
	for _, expected := range []string{"Status: Plan ready for approval", "Plan", "intent: create_feature", "sha256:plan"} {
		if !strings.Contains(view, expected) {
			t.Errorf("plan view does not contain %q:\n%s", expected, view)
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
	current           conversation.Conversation
	turnResult        conversation.SessionResult
	approveResult     conversation.SessionResult
	approvedOperation string
	approvedDigest    string
}

func (session *fakeSession) Current() conversation.Conversation {
	return session.current
}

func (session *fakeSession) Turn(
	context.Context,
	conversation.TurnRequest,
) (conversation.SessionResult, error) {
	return session.turnResult, nil
}

func (session *fakeSession) Approve(
	_ context.Context,
	operationID string,
	digest string,
) (conversation.SessionResult, error) {
	session.approvedOperation = operationID
	session.approvedDigest = digest

	return session.approveResult, nil
}

func (session *fakeSession) Cancel(
	context.Context,
	string,
) (conversation.SessionResult, error) {
	return conversation.SessionResult{}, nil
}

func (session *fakeSession) Reset(context.Context) (conversation.Conversation, error) {
	return conversation.Conversation{}, nil
}

func (session *fakeSession) Close() error {
	return nil
}
