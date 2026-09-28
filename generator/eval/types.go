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

// Request contains one natural-language goal and its bounded Hatmax context.
type Request struct {
	Prompt  string         `json:"prompt" yaml:"prompt"`
	Project ProjectContext `json:"project" yaml:"project"`
	Book    BookContext    `json:"book" yaml:"book"`
}

// Interpretation is the only provider result admitted by the evaluation
// boundary. It cannot contain plans, edits, commands, or file paths.
type Interpretation struct {
	Kind           InterpretationKind     `json:"kind" yaml:"kind"`
	Intent         *intent.Intent         `json:"intent,omitempty" yaml:"intent,omitempty"`
	Clarifications []intent.Clarification `json:"clarifications,omitempty" yaml:"clarifications,omitempty"`
	Diagnostics    []intent.Diagnostic    `json:"diagnostics,omitempty" yaml:"diagnostics,omitempty"`
}

// Interpreter converts one bounded natural-language request into a structured
// result. Provider adapters implement this interface outside the kernel.
type Interpreter interface {
	Interpret(context.Context, Request) (Interpretation, error)
}

// Context binds evaluation to one inspected project, fingerprint, and Book.
type Context struct {
	Inventory   project.Inventory
	Fingerprint project.Fingerprint
	Book        *book.Book
}

// Result records the deterministic outcome after interpreter output is
// validated by the kernel.
type Result struct {
	Status         intent.Status
	Intent         *intent.Intent
	Plan           *plan.Plan
	Diagnostics    []intent.Diagnostic
	Clarifications []intent.Clarification
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
