// Package interaction coordinates the backend-neutral Hatmax generator
// lifecycle without granting an interpreter authority over project changes.
package interaction

import (
	"context"

	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/execute"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

const (
	// MaximumClarificationRounds bounds model and user clarification rounds in
	// one interaction.
	MaximumClarificationRounds = 8
	// MaximumClarificationExchanges bounds individual answered questions across
	// every clarification round.
	MaximumClarificationExchanges = eval.MaximumClarificationExchanges
	// MaximumPlanPresentationBytes bounds the canonical plan supplied to an
	// approval surface.
	MaximumPlanPresentationBytes = 256 << 10
)

// State identifies one stable stage of the interactive lifecycle.
type State string

const (
	// StateConversation means a turn produced dialogue without mutation authority.
	StateConversation State = "conversation"
	// StateCandidateChange means a turn has been recognized as possible Hatmax work.
	StateCandidateChange State = "candidate_change"
	// StateInspecting means Hatmax is reading bounded project metadata.
	StateInspecting State = "inspecting"
	// StateInterpreting means the backend is resolving natural-language intent.
	StateInterpreting State = "interpreting"
	// StateClarifying means Hatmax is obtaining explicit product decisions.
	StateClarifying State = "clarifying"
	// StatePlanReady means a sealed plan is available for inspection but has not
	// been approved.
	StatePlanReady State = "plan_ready"
	// StateAwaitingApproval means a sealed plan has been presented without edits.
	StateAwaitingApproval State = "awaiting_approval"
	// StateExecuting means canonical renderers and the atomic workspace are active.
	StateExecuting State = "executing"
	// StateValidating means committed changes are undergoing conformance checks.
	StateValidating State = "validating"
	// StateFinished means the interaction has reached a terminal outcome.
	StateFinished State = "finished"
)

// Outcome classifies one terminal interaction result.
type Outcome string

const (
	// OutcomeConversationResponse means the turn produced bounded dialogue only.
	OutcomeConversationResponse Outcome = "conversation_response"
	// OutcomePlanReady means a sealed plan is available for explicit approval.
	OutcomePlanReady Outcome = "plan_ready"
	// OutcomeCompleted means execution and validation succeeded.
	OutcomeCompleted Outcome = "completed"
	// OutcomeCancelled means the user declined approval or clarification.
	OutcomeCancelled Outcome = "cancelled"
	// OutcomeUnsupported means the request is outside the admitted Hatmax Book.
	OutcomeUnsupported Outcome = "unsupported"
	// OutcomeIntentRejected means deterministic intent validation rejected the request.
	OutcomeIntentRejected Outcome = "intent_rejected"
	// OutcomeClarificationRequired means the request still needs a product decision.
	OutcomeClarificationRequired Outcome = "clarification_required"
	// OutcomePlanStale means relevant project state changed after planning.
	OutcomePlanStale Outcome = "plan_stale"
	// OutcomeExecutionFailed means rendering, mutation, conformance, or validation failed.
	OutcomeExecutionFailed Outcome = "execution_failed"
	// OutcomeFailed means the interaction could not reach a domain outcome.
	OutcomeFailed Outcome = "failed"
)

// TurnRequest contains the bounded visible context for one conversational
// interpretation. ApprovalPlanDigest authorizes only the newly recomputed plan
// with that exact digest; an empty digest requests preview only.
type TurnRequest struct {
	Prompt             string
	Conversation       []eval.DialogueTurn
	Clarifications     []eval.ClarificationExchange
	ApprovalPlanDigest string
}

// Phase identifies the lifecycle boundary that produced a diagnostic.
type Phase string

const (
	// PhaseInspection covers Book loading, project inspection, and fingerprinting.
	PhaseInspection Phase = "inspection"
	// PhaseInterpretation covers backend interpretation and intent admission.
	PhaseInterpretation Phase = "interpretation"
	// PhaseClarification covers explicit user answers.
	PhaseClarification Phase = "clarification"
	// PhaseApproval covers canonical plan presentation and explicit approval.
	PhaseApproval Phase = "approval"
	// PhaseFreshness covers the post-approval project reinspection.
	PhaseFreshness Phase = "freshness"
	// PhasePreparation covers deterministic execution-manifest preparation.
	PhasePreparation Phase = "preparation"
	// PhaseRendering covers canonical renderer dispatch.
	PhaseRendering Phase = "rendering"
	// PhaseApplication covers staging and atomic workspace commit.
	PhaseApplication Phase = "application"
	// PhaseValidation covers conformance and repository-owned commands.
	PhaseValidation Phase = "validation"
)

// Diagnostic is one stable product-level failure or cancellation detail.
type Diagnostic struct {
	Code    string `json:"code" yaml:"code"`
	Phase   Phase  `json:"phase" yaml:"phase"`
	Field   string `json:"field,omitempty" yaml:"field,omitempty"`
	Message string `json:"message" yaml:"message"`
}

// ApprovalRequest presents one canonical sealed plan. Approval is valid only
// for the supplied digest and source fingerprint.
type ApprovalRequest struct {
	PlanDigest         string
	ProjectFingerprint string
	SourceFingerprint  string
	PlanYAML           []byte
}

// ApprovalDecision is an explicit response to one presented plan.
type ApprovalDecision string

const (
	// ApprovalGranted authorizes one freshness check and execution attempt.
	ApprovalGranted ApprovalDecision = "approved"
	// ApprovalRejected cancels the interaction without project changes.
	ApprovalRejected ApprovalDecision = "rejected"
)

// Approver obtains explicit approval for one canonical plan projection.
type Approver interface {
	Approve(context.Context, ApprovalRequest) (ApprovalDecision, error)
}

// ClarificationRequest presents focused product questions from one bounded
// interpretation round.
type ClarificationRequest struct {
	Round     int
	Questions []intent.Clarification
}

// ClarificationAnswer binds one explicit answer to a requested field.
type ClarificationAnswer struct {
	Field  string
	Answer string
}

// ClarificationResponse either supplies every requested answer or cancels the
// interaction.
type ClarificationResponse struct {
	Answers   []ClarificationAnswer
	Cancelled bool
}

// Clarifier obtains product decisions without selecting Hatmax architecture.
type Clarifier interface {
	Clarify(context.Context, ClarificationRequest) (ClarificationResponse, error)
}

// Provenance records bounded interpreter facts across one interaction.
type Provenance struct {
	Interpretations     []eval.Provenance
	ClarificationRounds int
}

// Result is the complete backend-neutral interaction outcome. A failed
// post-commit validation retains Execution.Changes and the partial Report.
type Result struct {
	State            State
	Transitions      []State
	Outcome          Outcome
	Response         *eval.ConversationResponse
	Plan             *plan.Plan
	PlanYAML         []byte
	Provenance       Provenance
	Manifest         *execute.Manifest
	Execution        *execute.Result
	Report           *execute.ExecutionReport
	Diagnostics      []Diagnostic
	Clarifications   []intent.Clarification
	FreshnessChanges []project.Change
	RetainedChanges  []execute.Change
}
