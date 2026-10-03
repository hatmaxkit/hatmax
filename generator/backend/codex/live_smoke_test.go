// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package codex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/interaction"
	"hatmax.adrianpk.com/generator/project"
	"hatmax.adrianpk.com/internal/hatmaxstate"
)

func TestAuthenticatedLiveSmoke(t *testing.T) {
	// This opt-in observation exercises saved ChatGPT authentication, the
	// resident daemon, isolated thread creation, and one structured turn. It
	// never prints account data, raw events, reasoning, or model output.
	if os.Getenv("HATMAX_CODEX_LIVE_SMOKE") != "1" {
		t.Skip("set HATMAX_CODEX_LIVE_SMOKE=1 to run the authenticated Codex smoke test")
	}

	contextRoot := t.TempDir()

	interpreter, err := NewLocalInterpreter(InterpreterConfig{
		ContextRoot:     contextRoot,
		ProjectIdentity: "hatmax-authenticated-live-smoke-v1",
	})
	if err != nil {
		t.Fatalf("NewLocalInterpreter() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	first, err := interpreter.Interpret(ctx, liveSmokeRequest())
	if err != nil {
		t.Fatalf("initial authenticated Codex smoke failed: %v", err)
	}

	if first.Interpretation.Kind == "" || first.Provenance.EffectiveModel == "" || first.Provenance.ThreadReused {
		t.Fatalf("initial authenticated Codex smoke returned invalid bounded provenance: %#v", first.Provenance)
	}

	second, err := interpreter.Interpret(ctx, liveSmokeRequest())
	if err != nil {
		t.Fatalf("same-project authenticated Codex smoke failed: %v", err)
	}

	if !second.Provenance.RuntimeReused || !second.Provenance.ThreadReused {
		t.Fatalf("same-project request did not reuse runtime and thread: %#v", second.Provenance)
	}

	dialogueRequest := liveSmokeRequest()
	dialogueRequest.Prompt = "What is one plus one? Reply conversationally without proposing project changes."
	dialogueRequest.Conversation = []eval.DialogueTurn{
		{Role: eval.DialogueRoleUser, Content: "We are working on a Hatmax application."},
		{Role: eval.DialogueRoleHatmax, Content: "Understood."},
	}

	dialogue, err := interpreter.Interpret(ctx, dialogueRequest)
	if err != nil {
		t.Fatalf("authenticated conversational smoke failed: %v", err)
	}

	if dialogue.Interpretation.Kind != eval.InterpretationConversation || dialogue.Interpretation.Response == nil ||
		!dialogue.Provenance.RuntimeReused || !dialogue.Provenance.ThreadReused {
		t.Fatalf("authenticated conversational smoke returned invalid result: %#v %#v", dialogue.Interpretation, dialogue.Provenance)
	}

	documentation, err := interpreter.Interpret(ctx, liveDocumentationSmokeRequest())
	if err != nil {
		t.Fatalf("authenticated documentation smoke failed: %v", err)
	}

	value := documentation.Interpretation.Intent
	if documentation.Interpretation.Kind != eval.InterpretationIntent || value == nil ||
		value.Operation != "document_feature" || value.Documentation != "document_existing_behavior" ||
		len(value.DocumentationTargets) != 1 || value.DocumentationTargets[0].Quadrant != "reference" {
		t.Fatalf("authenticated documentation smoke returned invalid bounded intent: %#v", documentation.Interpretation)
	}

	applicationClarification, err := interpreter.Interpret(ctx, liveApplicationClarificationRequest())
	if err != nil {
		t.Fatalf("authenticated application clarification smoke failed: %v", err)
	}

	if applicationClarification.Interpretation.Kind != eval.InterpretationClarification ||
		len(applicationClarification.Interpretation.Clarifications) == 0 ||
		!applicationClarification.Provenance.RuntimeReused {
		t.Fatalf("authenticated application clarification returned invalid result: %#v %#v", applicationClarification.Interpretation, applicationClarification.Provenance)
	}

	runLiveApplicationConversation(t, ctx, interpreter)

	isolated, err := NewLocalInterpreter(InterpreterConfig{
		ContextRoot:     contextRoot,
		ProjectIdentity: "hatmax-authenticated-live-smoke-isolated-v1",
	})
	if err != nil {
		t.Fatalf("NewLocalInterpreter() for isolated project error = %v", err)
	}

	third, err := isolated.Interpret(ctx, liveSmokeRequest())
	if err != nil {
		t.Fatalf("different-project authenticated Codex smoke failed: %v", err)
	}

	if !third.Provenance.RuntimeReused || third.Provenance.ThreadReused {
		t.Fatalf("different-project request did not reuse only the runtime: %#v", third.Provenance)
	}

	t.Logf("authenticated Codex smoke passed: kind=%s model_selection=%s runtime_reused=%t same_project_thread_reused=%t different_project_thread_reused=%t timing=%s",
		second.Interpretation.Kind,
		second.Provenance.ModelSelection,
		second.Provenance.RuntimeReused,
		second.Provenance.ThreadReused,
		third.Provenance.ThreadReused,
		second.Provenance.Timing,
	)
}

func runLiveApplicationConversation(t *testing.T, ctx context.Context, interpreter *Interpreter) {
	t.Helper()

	parent := t.TempDir()

	state, err := hatmaxstate.New(hatmaxstate.Config{Root: filepath.Join(t.TempDir(), "state")})
	if err != nil {
		t.Fatalf("initialize live conversation state: %v", err)
	}

	kernel, err := interaction.New(interaction.Config{
		Interpreter: interpreter,
		Approver:    liveRejectingApprover{},
	})
	if err != nil {
		t.Fatalf("initialize live interaction kernel: %v", err)
	}

	engine := &liveApplicationEngine{kernel: kernel}

	coordinator, err := conversation.NewCoordinator(conversation.CoordinatorConfig{
		Store:                   state,
		Engine:                  engine,
		BookContract:            conversation.BookContract{BookVersion: 1, InterpreterVersion: eval.CurrentContractVersion},
		ApplicationBookContract: conversation.BookContract{BookVersion: 2, InterpreterVersion: eval.CurrentContractVersion},
		BackendIdentity:         conversation.BackendIdentity{Adapter: "codex-app-server", Model: string(eval.ModelBackendDefault)},
	})
	if err != nil {
		t.Fatalf("initialize live conversation coordinator: %v", err)
	}

	session, err := coordinator.Open(ctx, parent, conversation.SessionOptions{})
	if err != nil {
		t.Fatalf("open live pre-project conversation: %v", err)
	}
	defer session.Close()

	planned, err := session.Turn(ctx, conversation.TurnRequest{
		Content: "Create the Hatmax application now. Its application name is Ledger. Its target directory is ledger. Its Go module path is example.com/adrian/ledger. It has no initial features.",
	})
	if err != nil {
		t.Fatalf("plan live application: %v", err)
	}

	if planned.Interaction.Outcome != interaction.OutcomePlanReady || planned.Interaction.Plan == nil ||
		planned.Interaction.Intent == nil || planned.Interaction.Intent.Operation != "create_application" {
		t.Fatalf("live application planning result = %#v", planned.Interaction)
	}

	approved, err := session.Approve(ctx, planned.OperationID, planned.Interaction.Plan.Digest)
	if err != nil {
		t.Fatalf("approve live application plan: %v", err)
	}

	if engine.err != nil {
		t.Fatalf("complete live application approval: %v", engine.err)
	}

	if approved.Interaction.Outcome != interaction.OutcomeCompleted ||
		approved.Conversation.Scope.Kind != conversation.ScopeProject ||
		approved.Conversation.BackendThreadID != "" {
		t.Fatalf(
			"approved application outcome=%q scope=%q rebound_thread=%q",
			approved.Interaction.Outcome,
			approved.Conversation.Scope.Kind,
			approved.Conversation.BackendThreadID,
		)
	}

	if len(engine.approval.Provenance.Interpretations) == 0 {
		t.Fatal("approved live application omitted interpreter provenance")
	}

	provenance := engine.approval.Provenance.Interpretations[0]
	if !provenance.RuntimeReused || !provenance.ThreadReused {
		t.Fatalf("approved live application did not reuse runtime and isolated thread: %#v", provenance)
	}

	target := filepath.Join(parent, "ledger")

	projectScope, err := conversation.ResolveScope(conversation.ScopeProject, target)
	if err != nil {
		t.Fatalf("resolve rebound project scope: %v", err)
	}

	if approved.Conversation.Scope != projectScope {
		t.Fatalf("rebound scope = %#v, want %#v", approved.Conversation.Scope, projectScope)
	}
}

type liveRejectingApprover struct{}

func (liveRejectingApprover) Approve(
	context.Context,
	interaction.ApprovalRequest,
) (interaction.ApprovalDecision, error) {
	return interaction.ApprovalRejected, nil
}

type liveApplicationEngine struct {
	kernel   *interaction.Coordinator
	approval interaction.Result
	err      error
}

func (engine *liveApplicationEngine) PreviewTurn(
	ctx context.Context,
	root string,
	request interaction.TurnRequest,
) interaction.Result {
	return engine.kernel.PreviewTurn(ctx, root, request)
}

func (engine *liveApplicationEngine) RunApprovedTurn(
	ctx context.Context,
	root string,
	request interaction.TurnRequest,
) interaction.Result {
	approvedDigest := request.ApprovalPlanDigest
	request.ApprovalPlanDigest = ""
	recomputed := engine.kernel.PreviewTurn(ctx, root, request)
	engine.approval = recomputed

	if recomputed.Outcome != interaction.OutcomePlanReady || recomputed.Plan == nil {
		engine.err = fmt.Errorf("approval replan returned %q", recomputed.Outcome)

		return liveApplicationFailure(engine.err)
	}

	if recomputed.Plan.Digest == "" || recomputed.Plan.Digest != approvedDigest {
		engine.err = fmt.Errorf("approval replan changed the displayed digest")

		return liveApplicationFailure(engine.err)
	}

	if recomputed.Plan.Target == nil {
		engine.err = fmt.Errorf("application plan omitted target")

		return liveApplicationFailure(engine.err)
	}

	err := os.MkdirAll(recomputed.Plan.Target.Path, 0o700)
	if err == nil {
		err = os.WriteFile(
			filepath.Join(recomputed.Plan.Target.Path, "go.mod"),
			[]byte("module example.com/adrian/ledger\n\ngo 1.27.1\n\nrequire hatmax.adrianpk.com v0.5.0\n"),
			0o600,
		)
	}

	if err != nil {
		engine.err = fmt.Errorf("materialize smoke project identity: %w", err)

		return liveApplicationFailure(engine.err)
	}

	recomputed.State = interaction.StateFinished
	recomputed.Outcome = interaction.OutcomeCompleted

	return recomputed
}

func (engine *liveApplicationEngine) ProjectFingerprint(
	ctx context.Context,
	root string,
) (project.Fingerprint, error) {
	return engine.kernel.ProjectFingerprint(ctx, root)
}

func liveApplicationFailure(err error) interaction.Result {
	return interaction.Result{
		State:   interaction.StateFinished,
		Outcome: interaction.OutcomeFailed,
		Diagnostics: []interaction.Diagnostic{{
			Code: "HMGEN-LIVE-SMOKE", Phase: interaction.PhaseApplication, Message: err.Error(),
		}},
	}
}

func liveSmokeRequest() eval.Request {
	return eval.Request{
		ContractVersion: eval.CurrentContractVersion,
		Prompt:          "Create an invoice feature with a required reference string field",
		Project: eval.ProjectContext{
			Fingerprint:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			HatmaxVersion: "0.4.0",
		},
		Book: eval.BookContext{
			Version: 1,
			Archetypes: []eval.ArchetypeContext{{
				ID:                   "server_rendered_crud",
				Operations:           []string{"create_feature", "add_field", "add_validation", "document_feature"},
				RequiredCapabilities: []string{"postgres_persistence", "htmx_form", "runtime_validation"},
			}},
			Capabilities: []eval.CapabilityContext{
				{ID: "postgres_persistence", Intent: "Persist the feature in PostgreSQL"},
				{ID: "htmx_form", Intent: "Render server-owned HTMX forms"},
				{ID: "runtime_validation", Intent: "Validate input at runtime"},
			},
		},
	}
}

func liveDocumentationSmokeRequest() eval.Request {
	request := liveSmokeRequest()
	request.Prompt = "Document the existing invoice feature as reference for readers who need its exact contract"
	request.Project.ExistingFeatures = []string{"invoice"}

	return request
}

func liveApplicationClarificationRequest() eval.Request {
	return eval.Request{
		ContractVersion: eval.CurrentContractVersion,
		Prompt:          "Create a Hatmax application.",
		Target: &eval.TargetContext{
			SourceFingerprint: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			HatmaxVersion:     "0.5.0",
			Admission:         project.TargetAbsent,
		},
		Book: eval.BookContext{
			Version: 2,
			Archetypes: []eval.ArchetypeContext{{
				ID:                   "server_rendered_hatmax_application",
				Operations:           []string{"create_application"},
				RequiredCapabilities: []string{"postgres_persistence"},
			}},
			Capabilities: []eval.CapabilityContext{{
				ID: "postgres_persistence", Intent: "Persist feature state through the canonical Hatmax database lifecycle",
			}},
		},
	}
}
