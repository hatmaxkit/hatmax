// Package plan defines deterministic, inspectable Hatmax generator plans.
package plan

import (
	"fmt"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

// CurrentSchemaVersion is the plan schema understood by this package.
const CurrentSchemaVersion = 5

// ApplicationSchemaVersion adds target-bound application and composite unit
// planning without changing delivered existing-project plans.
const ApplicationSchemaVersion = 6

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
	// PreconditionSourceFingerprint binds a scaffold plan to target state.
	PreconditionSourceFingerprint PreconditionKind = "source_fingerprint"
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
	ID         string       `json:"id" yaml:"id"`
	Owner      Owner        `json:"owner" yaml:"owner"`
	Obligation string       `json:"obligation" yaml:"obligation"`
	Surfaces   []string     `json:"surfaces" yaml:"surfaces"`
	Rules      []string     `json:"rules" yaml:"rules"`
	DependsOn  []string     `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Files      []FileEffect `json:"files,omitempty" yaml:"files,omitempty"`
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

// FileEffectKind identifies one exact target-path mutation.
type FileEffectKind string

const (
	// FileEffectCreate creates a path that must be absent.
	FileEffectCreate FileEffectKind = "create"
	// FileEffectUpdate changes a path produced or admitted earlier.
	FileEffectUpdate FileEffectKind = "update"
)

// FileEffect records one exact target-relative path mutation.
type FileEffect struct {
	Path   string         `json:"path" yaml:"path"`
	Effect FileEffectKind `json:"effect" yaml:"effect"`
}

// AllowedEffects bounds the project surfaces and external dependencies that a
// later executor may affect.
type AllowedEffects struct {
	Surfaces     []string           `json:"surfaces" yaml:"surfaces"`
	Dependencies []DependencyEffect `json:"dependencies" yaml:"dependencies"`
	Files        []FileEffect       `json:"files,omitempty" yaml:"files,omitempty"`
}

// ApplicationTarget binds a scaffold to the inspected child target and the
// existing files that execution must preserve.
type ApplicationTarget struct {
	Base      string                  `json:"base" yaml:"base"`
	Directory string                  `json:"directory" yaml:"directory"`
	Parent    string                  `json:"parent" yaml:"parent"`
	Path      string                  `json:"path" yaml:"path"`
	Admission project.TargetAdmission `json:"admission" yaml:"admission"`
	Preserved []project.TargetEntry   `json:"preserved" yaml:"preserved"`
}

// Unit is one visible, ordered operation in an application bootstrap.
type Unit struct {
	ID               string           `json:"id" yaml:"id"`
	Intent           intent.Operation `json:"intent" yaml:"intent"`
	Archetype        string           `json:"archetype" yaml:"archetype"`
	Feature          string           `json:"feature,omitempty" yaml:"feature,omitempty"`
	Domain           intent.Domain    `json:"domain,omitempty" yaml:"domain,omitempty"`
	Capabilities     []string         `json:"capabilities" yaml:"capabilities"`
	AffectedSurfaces []string         `json:"affected_surfaces" yaml:"affected_surfaces"`
	Operations       []string         `json:"operations" yaml:"operations"`
	DependsOn        []string         `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Effects          []FileEffect     `json:"effects" yaml:"effects"`
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

// DocumentationSnapshot binds one planned Markdown target to inspected state.
type DocumentationSnapshot struct {
	Path         string                            `json:"path" yaml:"path"`
	Exists       bool                              `json:"exists" yaml:"exists"`
	Digest       string                            `json:"digest,omitempty" yaml:"digest,omitempty"`
	ManagedState project.DocumentationManagedState `json:"managed_state,omitempty" yaml:"managed_state,omitempty"`
}

// DocumentationLinkEffect is one canonical navigation link required after
// execution.
type DocumentationLinkEffect struct {
	Title    string `json:"title" yaml:"title"`
	Target   string `json:"target" yaml:"target"`
	Relative string `json:"relative" yaml:"relative"`
}

// DocumentationTargetEffect records one exact Diataxis document derived from
// typed reader intent.
type DocumentationTargetEffect struct {
	Quadrant   intent.DocumentationQuadrant `json:"quadrant" yaml:"quadrant"`
	Subject    string                       `json:"subject" yaml:"subject"`
	ReaderGoal string                       `json:"reader_goal" yaml:"reader_goal"`
	Slug       string                       `json:"slug" yaml:"slug"`
	Title      string                       `json:"title" yaml:"title"`
	Path       string                       `json:"path" yaml:"path"`
	Snapshot   DocumentationSnapshot        `json:"snapshot" yaml:"snapshot"`
}

// DocumentationIndexEffect records one managed navigation section and its
// required links.
type DocumentationIndexEffect struct {
	Kind          string                       `json:"kind" yaml:"kind"`
	Quadrant      intent.DocumentationQuadrant `json:"quadrant,omitempty" yaml:"quadrant,omitempty"`
	Title         string                       `json:"title" yaml:"title"`
	Path          string                       `json:"path" yaml:"path"`
	RequiredLinks []DocumentationLinkEffect    `json:"required_links" yaml:"required_links"`
	Snapshot      DocumentationSnapshot        `json:"snapshot" yaml:"snapshot"`
}

// DocumentationCommand records one inspected repository documentation gate.
type DocumentationCommand struct {
	Name   string              `json:"name" yaml:"name"`
	Args   []string            `json:"args" yaml:"args"`
	Source string              `json:"source" yaml:"source"`
	Kind   project.CommandKind `json:"kind" yaml:"kind"`
}

// DocumentationPlan is the exact bounded documentation surface authorized by
// a sealed plan.
type DocumentationPlan struct {
	Targets            []DocumentationTargetEffect `json:"targets" yaml:"targets"`
	Indexes            []DocumentationIndexEffect  `json:"indexes" yaml:"indexes"`
	ValidationCommands []DocumentationCommand      `json:"validation_commands" yaml:"validation_commands"`
}

// Plan is the complete deterministic planning artifact. It remains ephemeral
// and does not authorize project mutation by itself.
type Plan struct {
	SchemaVersion         int                          `json:"schema_version" yaml:"schema_version"`
	Intent                intent.Operation             `json:"intent" yaml:"intent"`
	Archetype             string                       `json:"archetype" yaml:"archetype"`
	Feature               string                       `json:"feature" yaml:"feature"`
	Domain                intent.Domain                `json:"domain" yaml:"domain"`
	Capabilities          []string                     `json:"capabilities" yaml:"capabilities"`
	AffectedSurfaces      []string                     `json:"affected_surfaces" yaml:"affected_surfaces"`
	Documentation         intent.Documentation         `json:"documentation" yaml:"documentation"`
	DocumentationTargets  []intent.DocumentationTarget `json:"documentation_targets" yaml:"documentation_targets"`
	DocumentationEvidence *project.FeatureEvidence     `json:"documentation_evidence,omitempty" yaml:"documentation_evidence,omitempty"`
	DocumentationPlan     *DocumentationPlan           `json:"documentation_plan,omitempty" yaml:"documentation_plan,omitempty"`
	HatmaxVersion         string                       `json:"hatmax_version" yaml:"hatmax_version"`
	BookVersion           int                          `json:"book_version" yaml:"book_version"`
	ProjectFingerprint    string                       `json:"project_fingerprint,omitempty" yaml:"project_fingerprint,omitempty"`
	SourceFingerprint     string                       `json:"source_fingerprint,omitempty" yaml:"source_fingerprint,omitempty"`
	Application           *intent.ApplicationIdentity  `json:"application,omitempty" yaml:"application,omitempty"`
	Target                *ApplicationTarget           `json:"target,omitempty" yaml:"target,omitempty"`
	Units                 []Unit                       `json:"units,omitempty" yaml:"units,omitempty"`
	FingerprintInputs     FingerprintInputs            `json:"fingerprint_inputs" yaml:"fingerprint_inputs"`
	Rules                 []RuleRef                    `json:"rules" yaml:"rules"`
	Operations            []Operation                  `json:"operations" yaml:"operations"`
	Preconditions         []Precondition               `json:"preconditions" yaml:"preconditions"`
	ExpectedObservations  []project.Observation        `json:"expected_observations" yaml:"expected_observations"`
	AllowedEffects        AllowedEffects               `json:"allowed_effects" yaml:"allowed_effects"`
	Validation            []ValidationObligation       `json:"validation" yaml:"validation"`
	Exceptions            []intent.Exception           `json:"exceptions" yaml:"exceptions"`
	Digest                string                       `json:"digest,omitempty" yaml:"digest,omitempty"`
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
