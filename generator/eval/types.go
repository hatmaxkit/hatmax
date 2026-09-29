// Package eval defines provider-neutral evaluation contracts for the Hatmax
// generator planning kernel.
//
// Deterministic tests use fixture interpreters. Passing those tests proves the
// kernel boundary and expected structured outcomes; it does not validate any
// production model, prompt, provider, or subscription harness.
package eval

import (
	"context"
	"fmt"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

const (
	// CurrentContractVersion is the interactive interpreter contract understood
	// by this package.
	CurrentContractVersion = 2
	// MaximumPromptBytes bounds one natural-language request.
	MaximumPromptBytes = 32 << 10
	// MaximumClarificationExchanges bounds one explicit clarification history.
	MaximumClarificationExchanges = 8
	// MaximumClarificationTextBytes bounds one clarification field, question, or
	// answer.
	MaximumClarificationTextBytes = 4 << 10
	// CurrentInterpretationSchemaVersion is the model-output schema version.
	CurrentInterpretationSchemaVersion = 2
	// MaximumInterpretationBytes bounds one complete model output.
	MaximumInterpretationBytes = 64 << 10
	// MaximumInterpretationDiagnostics bounds one unsupported result.
	MaximumInterpretationDiagnostics = 16
)

// InterpretationKind identifies one structured interpreter response.
type InterpretationKind string

const (
	// InterpretationIntent contains one typed intent for deterministic
	// validation.
	InterpretationIntent InterpretationKind = "intent"
	// InterpretationClarification contains focused product questions.
	InterpretationClarification InterpretationKind = "clarification_required"
	// InterpretationUnsupported rejects a request outside the Hatmax Book.
	InterpretationUnsupported InterpretationKind = "unsupported"
)

// ProjectContext is the bounded project information exposed to an
// interpreter.
type ProjectContext struct {
	Fingerprint      string   `json:"fingerprint" yaml:"fingerprint"`
	HatmaxVersion    string   `json:"hatmax_version" yaml:"hatmax_version"`
	ExistingFeatures []string `json:"existing_features" yaml:"existing_features"`
}

// TargetContext is the bounded pre-project evidence exposed to an
// interpreter. It deliberately omits local paths and target entries.
type TargetContext struct {
	SourceFingerprint string                  `json:"source_fingerprint" yaml:"source_fingerprint"`
	HatmaxVersion     string                  `json:"hatmax_version" yaml:"hatmax_version"`
	Admission         project.TargetAdmission `json:"admission" yaml:"admission"`
	RemoteModulePath  string                  `json:"remote_module_path" yaml:"remote_module_path"`
	Resolved          bool                    `json:"resolved" yaml:"resolved"`
}

// ArchetypeContext describes supported operations and capability choices
// without exposing implementation files.
type ArchetypeContext struct {
	ID                   string   `json:"id" yaml:"id"`
	Operations           []string `json:"operations" yaml:"operations"`
	RequiredCapabilities []string `json:"required_capabilities" yaml:"required_capabilities"`
	OptionalCapabilities []string `json:"optional_capabilities" yaml:"optional_capabilities"`
}

// CapabilityContext describes one Book capability at the interpretation
// boundary.
type CapabilityContext struct {
	ID          string   `json:"id" yaml:"id"`
	Intent      string   `json:"intent" yaml:"intent"`
	Requires    []string `json:"requires" yaml:"requires"`
	Unsupported []string `json:"unsupported" yaml:"unsupported"`
}

// BookContext is the bounded Book projection exposed to an interpreter.
type BookContext struct {
	Version      int                 `json:"version" yaml:"version"`
	Archetypes   []ArchetypeContext  `json:"archetypes" yaml:"archetypes"`
	Capabilities []CapabilityContext `json:"capabilities" yaml:"capabilities"`
}

// ClarificationExchange records one focused question and explicit user answer
// supplied by Hatmax. Backend thread history never replaces this state.
type ClarificationExchange struct {
	Field    string `json:"field" yaml:"field"`
	Question string `json:"question" yaml:"question"`
	Answer   string `json:"answer" yaml:"answer"`
}

// Request contains one natural-language goal and its bounded Hatmax context.
type Request struct {
	ContractVersion int                     `json:"contract_version" yaml:"contract_version"`
	Prompt          string                  `json:"prompt" yaml:"prompt"`
	Clarifications  []ClarificationExchange `json:"clarifications" yaml:"clarifications"`
	Project         ProjectContext          `json:"project" yaml:"project"`
	Target          *TargetContext          `json:"target" yaml:"target"`
	Book            BookContext             `json:"book" yaml:"book"`
}

// Interpretation is the only provider result admitted by the evaluation
// boundary. It cannot contain plans, edits, commands, or file paths.
type Interpretation struct {
	SchemaVersion  int                    `json:"schema_version" yaml:"schema_version"`
	Kind           InterpretationKind     `json:"kind" yaml:"kind"`
	Intent         *intent.Intent         `json:"intent,omitempty" yaml:"intent,omitempty"`
	Clarifications []intent.Clarification `json:"clarifications,omitempty" yaml:"clarifications,omitempty"`
	Diagnostics    []intent.Diagnostic    `json:"diagnostics,omitempty" yaml:"diagnostics,omitempty"`
}

// ModelSelection records how the backend model was chosen.
type ModelSelection string

const (
	// ModelBackendDefault means Hatmax did not request a model override.
	ModelBackendDefault ModelSelection = "backend_default"
	// ModelExplicit means Hatmax supplied an explicit model identifier.
	ModelExplicit ModelSelection = "explicit"
	// ModelNotApplicable is reserved for deterministic fixture interpreters.
	ModelNotApplicable ModelSelection = "not_applicable"
)

// TimingClass is a bounded duration classification that avoids exposing raw
// backend event timing.
type TimingClass string

const (
	// TimingNotMeasured is used by deterministic interpreters without a runtime.
	TimingNotMeasured TimingClass = "not_measured"
	// TimingUnderSecond completed in less than one second.
	TimingUnderSecond TimingClass = "under_1s"
	// TimingUnderTenSeconds completed in less than ten seconds.
	TimingUnderTenSeconds TimingClass = "under_10s"
	// TimingUnderThirtySeconds completed in less than thirty seconds.
	TimingUnderThirtySeconds TimingClass = "under_30s"
	// TimingUnderTwoMinutes completed within the initial turn deadline.
	TimingUnderTwoMinutes TimingClass = "under_2m"
)

// Provenance records bounded, non-secret backend facts for one interpretation.
type Provenance struct {
	Adapter         string         `json:"adapter" yaml:"adapter"`
	ContractVersion int            `json:"contract_version" yaml:"contract_version"`
	BackendVersion  string         `json:"backend_version,omitempty" yaml:"backend_version,omitempty"`
	ProtocolVersion string         `json:"protocol_version,omitempty" yaml:"protocol_version,omitempty"`
	ModelSelection  ModelSelection `json:"model_selection" yaml:"model_selection"`
	EffectiveModel  string         `json:"effective_model,omitempty" yaml:"effective_model,omitempty"`
	RuntimeReused   bool           `json:"runtime_reused" yaml:"runtime_reused"`
	ThreadReused    bool           `json:"thread_reused" yaml:"thread_reused"`
	Timing          TimingClass    `json:"timing" yaml:"timing"`
}

// InterpreterResult combines the only model-controlled value with
// adapter-controlled provenance.
type InterpreterResult struct {
	Interpretation Interpretation
	Provenance     Provenance
}

// Interpreter converts one bounded natural-language request into a structured
// result. Provider adapters implement this interface outside the kernel.
type Interpreter interface {
	Interpret(context.Context, Request) (InterpreterResult, error)
}

// Context binds evaluation to one inspected project, fingerprint, and Book.
type Context struct {
	Inventory             project.Inventory
	Target                *project.TargetInventory
	TargetResolved        bool
	Fingerprint           project.Fingerprint
	Book                  *book.Book
	DocumentationEvidence *project.FeatureEvidence
}

// Result records the deterministic outcome after interpreter output is
// validated by the kernel.
type Result struct {
	Status         intent.Status
	Intent         *intent.Intent
	Plan           *plan.Plan
	Diagnostics    []intent.Diagnostic
	Clarifications []intent.Clarification
	Provenance     Provenance
}

// BackendFailureCode classifies stable interpreter backend failures.
type BackendFailureCode string

const (
	BackendUnavailable            BackendFailureCode = "backend_unavailable"
	BackendAuthenticationRequired BackendFailureCode = "backend_authentication_required"
	BackendIncompatible           BackendFailureCode = "backend_incompatible"
	BackendStartFailed            BackendFailureCode = "backend_start_failed"
	BackendThreadFailed           BackendFailureCode = "backend_thread_failed"
	BackendTimeout                BackendFailureCode = "backend_timeout"
	BackendCancelled              BackendFailureCode = "backend_cancelled"
	BackendOutputInvalid          BackendFailureCode = "backend_output_invalid"
	BackendTurnFailed             BackendFailureCode = "backend_turn_failed"
	BackendProtocolViolation      BackendFailureCode = "backend_protocol_violation"
)

// BackendError reports one bounded interpreter backend failure.
type BackendError struct {
	Code      BackendFailureCode
	Operation string
	Message   string
}

func (e BackendError) Error() string {
	if e.Operation == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	return fmt.Sprintf("%s during %s: %s", e.Code, e.Operation, e.Message)
}

// Error describes an invalid evaluation boundary or interpreter result.
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

func evaluationError(code, field, format string, arguments ...any) error {
	return Error{
		Code:    code,
		Field:   field,
		Message: fmt.Sprintf(format, arguments...),
	}
}
