// Package conversation defines bounded durable conversation state without
// granting stored text, intent, plans, or backend history project authority.
package conversation

import (
	"context"
	"fmt"
	"time"

	"hatmax.adrianpk.com/generator/intent"
)

const (
	// CurrentSchemaVersion is the durable conversation snapshot version.
	CurrentSchemaVersion = 1
	// MaximumTurnBytes bounds one retained user-visible turn.
	MaximumTurnBytes = 32 << 10
	// MaximumTurns bounds retained turns in one conversation.
	MaximumTurns = 200
	// MaximumTurnContentBytes bounds aggregate retained turn content.
	MaximumTurnContentBytes = 8 << 20
	// MaximumOperations bounds retained operation summaries in one conversation.
	MaximumOperations = 50
	// MaximumResultSummaryBytes bounds one retained execution result.
	MaximumResultSummaryBytes = 16 << 10
	// MaximumDiagnosticBytes bounds aggregate retained diagnostic turns.
	MaximumDiagnosticBytes = 16 << 10
	// MaximumReplayTurns bounds backend replacement context.
	MaximumReplayTurns = 32
	// MaximumReplayBytes bounds backend replacement context.
	MaximumReplayBytes = 128 << 10
	// MaximumArchivedConversations bounds archives retained beside one active
	// conversation.
	MaximumArchivedConversations = 9
)

// ScopeKind distinguishes an existing project from a parent or proposed
// application target.
type ScopeKind string

const (
	// ScopeProject identifies one compatible Hatmax project root.
	ScopeProject ScopeKind = "project"
	// ScopePreProject identifies a parent directory or proposed application
	// target before bootstrap completes.
	ScopePreProject ScopeKind = "pre_project"
)

// Scope identifies a conversation without persisting its filesystem path.
type Scope struct {
	Key  string    `json:"key"`
	Kind ScopeKind `json:"kind"`
}

// BookContract identifies the deterministic and interpretation contracts with
// which a conversation is compatible.
type BookContract struct {
	BookVersion        int `json:"book_version"`
	InterpreterVersion int `json:"interpreter_version"`
}

// BackendIdentity contains only non-secret backend classification.
type BackendIdentity struct {
	Adapter string `json:"adapter"`
	Version string `json:"version,omitempty"`
	Model   string `json:"model,omitempty"`
}

// Status identifies the durable conversation lifecycle.
type Status string

const (
	// StatusActive is the default resumable conversation for a scope.
	StatusActive Status = "active"
	// StatusReset identifies a conversation superseded by an explicit reset.
	StatusReset Status = "reset"
	// StatusIncompatible preserves state that cannot be loaded by current
	// contracts.
	StatusIncompatible Status = "incompatible"
	// StatusArchived identifies a compatible conversation not resumed by
	// default.
	StatusArchived Status = "archived"
)

// Role identifies one visible speaker.
type Role string

const (
	// RoleUser is a user-authored turn.
	RoleUser Role = "user"
	// RoleHatmax is a Hatmax-authored turn.
	RoleHatmax Role = "hatmax"
)

// TurnKind classifies retained user-visible content.
type TurnKind string

const (
	// TurnDialogue is ordinary conversation.
	TurnDialogue TurnKind = "dialogue"
	// TurnGoal records a user-visible product goal.
	TurnGoal TurnKind = "goal"
	// TurnClarification records a focused question or answer.
	TurnClarification TurnKind = "clarification"
	// TurnDecision records an explicit product decision.
	TurnDecision TurnKind = "decision"
	// TurnResult records an operation outcome.
	TurnResult TurnKind = "result"
	// TurnDiagnostic records a user-visible failure detail.
	TurnDiagnostic TurnKind = "diagnostic"
)

// Turn is one bounded user-visible conversation entry.
type Turn struct {
	ID          uint64    `json:"id"`
	Role        Role      `json:"role"`
	Kind        TurnKind  `json:"kind"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	OperationID string    `json:"operation_id,omitempty"`
}

// OperationStatus identifies one proposed Hatmax mutation lifecycle.
type OperationStatus string

const (
	// OperationCandidate has been recognized as possible Hatmax work.
	OperationCandidate OperationStatus = "candidate"
	// OperationClarifying is collecting required product decisions.
	OperationClarifying OperationStatus = "clarifying"
	// OperationPlanned has a sealed plan but no durable approval.
	OperationPlanned OperationStatus = "planned"
	// OperationCancelled was explicitly abandoned.
	OperationCancelled OperationStatus = "cancelled"
	// OperationStale requires reinterpretation or replanning.
	OperationStale OperationStatus = "stale"
	// OperationFailed retains one bounded failure summary.
	OperationFailed OperationStatus = "failed"
	// OperationCompleted retains one bounded successful result.
	OperationCompleted OperationStatus = "completed"
)

// ProposedOperation is historical and resumable context, never current
// execution authority. Approval is deliberately absent.
type ProposedOperation struct {
	ID                string          `json:"id"`
	ConversationID    string          `json:"conversation_id"`
	Status            OperationStatus `json:"status"`
	RequestSummary    string          `json:"request_summary"`
	IntentContract    int             `json:"intent_contract,omitempty"`
	Intent            *intent.Intent  `json:"intent,omitempty"`
	PlanDigest        string          `json:"plan_digest,omitempty"`
	SourceFingerprint string          `json:"source_fingerprint,omitempty"`
	ResultSummary     string          `json:"result_summary,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// Conversation is one versioned, bounded durable snapshot.
type Conversation struct {
	SchemaVersion   int                 `json:"schema_version"`
	ID              string              `json:"id"`
	Scope           Scope               `json:"scope"`
	BookContract    BookContract        `json:"book_contract"`
	BackendIdentity BackendIdentity     `json:"backend_identity"`
	BackendThreadID string              `json:"backend_thread_id,omitempty"`
	Status          Status              `json:"status"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	NextTurnID      uint64              `json:"next_turn_id"`
	Turns           []Turn              `json:"turns"`
	Operations      []ProposedOperation `json:"operations"`
}

// Summary is the bounded conversation metadata returned by list operations.
type Summary struct {
	ID        string    `json:"id"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OpenRequest identifies the exact scope and compatible contracts to resume.
type OpenRequest struct {
	Scope           Scope
	BookContract    BookContract
	BackendIdentity BackendIdentity
}

// Store opens one lock-owning mutable scope session and supports read-only
// listing without exposing other scope paths or content.
type Store interface {
	Open(context.Context, OpenRequest) (Session, error)
	List(context.Context, Scope) ([]Summary, error)
}

// Session owns the advisory scope lock until Close. Replace persists one
// complete already-validated snapshot atomically.
type Session interface {
	Current() Conversation
	Replace(context.Context, Conversation) error
	Reset(context.Context, string, time.Time) (Conversation, error)
	Rebind(context.Context, Scope, time.Time) error
	Close() error
}

// Error is one stable conversation model or persistence failure.
type Error struct {
	Code    string
	Field   string
	Message string
}

func (e Error) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	return fmt.Sprintf("%s at %s: %s", e.Code, e.Field, e.Message)
}

func modelError(code, field, format string, arguments ...any) error {
	return Error{Code: code, Field: field, Message: fmt.Sprintf(format, arguments...)}
}
