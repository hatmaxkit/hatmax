// Package plan defines deterministic, inspectable Hatmax generator plans.
package plan

import (
	"fmt"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

// CurrentSchemaVersion is the plan schema understood by this package.
const CurrentSchemaVersion = 2

// OwnerKind identifies the Book entry that owns one logical operation.
type OwnerKind string

const (
	// OwnerArchetype identifies an archetype-owned obligation.
	OwnerArchetype OwnerKind = "archetype"
	// OwnerCapability identifies a capability-owned obligation.
	OwnerCapability OwnerKind = "capability"
)

// PreconditionKind identifies one condition that must still hold before a plan
// can be consumed.
type PreconditionKind string

const (
	// PreconditionProjectFingerprint binds the plan to relevant project state.
	PreconditionProjectFingerprint PreconditionKind = "project_fingerprint"
	// PreconditionHatmaxVersion binds the plan to the selected Hatmax release.
	PreconditionHatmaxVersion PreconditionKind = "hatmax_version"
	// PreconditionBookVersion binds the plan to the selected Book release.
	PreconditionBookVersion PreconditionKind = "book_version"
)

// RuleRef records one selected Book rule without copying its prose.
type RuleRef struct {
	ID    string     `json:"id" yaml:"id"`
	Level book.Level `json:"level" yaml:"level"`
}

// Owner attributes one logical operation to a structured Book entry.
type Owner struct {
	Kind OwnerKind `json:"kind" yaml:"kind"`
	ID   string    `json:"id" yaml:"id"`
}

// Operation is one ordered logical unit of work derived from a Book
// obligation. It does not contain executable source or shell text.
type Operation struct {
	ID         string   `json:"id" yaml:"id"`
	Owner      Owner    `json:"owner" yaml:"owner"`
	Obligation string   `json:"obligation" yaml:"obligation"`
	Surfaces   []string `json:"surfaces" yaml:"surfaces"`
	Rules      []string `json:"rules" yaml:"rules"`
	DependsOn  []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
}

// Precondition records an expected planning input in user-inspectable form.
type Precondition struct {
	ID       string           `json:"id" yaml:"id"`
	Kind     PreconditionKind `json:"kind" yaml:"kind"`
	Expected string           `json:"expected" yaml:"expected"`
}

// DependencyEffect records one Book-admitted dependency effect and its owner.
type DependencyEffect struct {
	Capability string `json:"capability" yaml:"capability"`
	Kind       string `json:"kind" yaml:"kind"`
	Module     string `json:"module" yaml:"module"`
	Purpose    string `json:"purpose" yaml:"purpose"`
}

// AllowedEffects bounds the project surfaces and external dependencies that a
// later executor may affect.
type AllowedEffects struct {
	Surfaces     []string           `json:"surfaces" yaml:"surfaces"`
	Dependencies []DependencyEffect `json:"dependencies" yaml:"dependencies"`
}

// FingerprintInputs records the bounded project observations selected when the
// plan fingerprint was computed so execution can reproduce it exactly.
type FingerprintInputs struct {
	SelectedPaths        []string `json:"selected_paths" yaml:"selected_paths"`
	SelectedDependencies []string `json:"selected_dependencies" yaml:"selected_dependencies"`
	PlannedSurfaces      []string `json:"planned_surfaces" yaml:"planned_surfaces"`
}

// ValidationObligation describes one selected Book rule that conformance must
// evaluate after execution.
type ValidationObligation struct {
	Rule        string     `json:"rule" yaml:"rule"`
	Level       book.Level `json:"level" yaml:"level"`
	Diagnostics []string   `json:"diagnostics" yaml:"diagnostics"`
	Surfaces    []string   `json:"surfaces,omitempty" yaml:"surfaces,omitempty"`
}

// Plan is the complete deterministic planning artifact. It remains ephemeral
// and does not authorize project mutation by itself.
type Plan struct {
	SchemaVersion        int                    `json:"schema_version" yaml:"schema_version"`
	Intent               intent.Operation       `json:"intent" yaml:"intent"`
	Archetype            string                 `json:"archetype" yaml:"archetype"`
	Feature              string                 `json:"feature" yaml:"feature"`
	Domain               intent.Domain          `json:"domain" yaml:"domain"`
	Capabilities         []string               `json:"capabilities" yaml:"capabilities"`
	AffectedSurfaces     []string               `json:"affected_surfaces" yaml:"affected_surfaces"`
	Documentation        intent.Documentation   `json:"documentation" yaml:"documentation"`
	HatmaxVersion        string                 `json:"hatmax_version" yaml:"hatmax_version"`
	BookVersion          int                    `json:"book_version" yaml:"book_version"`
	ProjectFingerprint   string                 `json:"project_fingerprint" yaml:"project_fingerprint"`
	FingerprintInputs    FingerprintInputs      `json:"fingerprint_inputs" yaml:"fingerprint_inputs"`
	Rules                []RuleRef              `json:"rules" yaml:"rules"`
	Operations           []Operation            `json:"operations" yaml:"operations"`
	Preconditions        []Precondition         `json:"preconditions" yaml:"preconditions"`
	ExpectedObservations []project.Observation  `json:"expected_observations" yaml:"expected_observations"`
	AllowedEffects       AllowedEffects         `json:"allowed_effects" yaml:"allowed_effects"`
	Validation           []ValidationObligation `json:"validation" yaml:"validation"`
	Exceptions           []intent.Exception     `json:"exceptions" yaml:"exceptions"`
	Digest               string                 `json:"digest,omitempty" yaml:"digest,omitempty"`
}

// Error describes an invalid plan contract.
type Error struct {
	Code    string
	Path    string
	Message string
}

func (e Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	return fmt.Sprintf("%s at %s: %s", e.Code, e.Path, e.Message)
}

func planError(code, path, format string, arguments ...any) error {
	return Error{
		Code:    code,
		Path:    path,
		Message: fmt.Sprintf(format, arguments...),
	}
}
