// Package intent defines and validates typed Hatmax generator intents.
package intent

import "fmt"

// CurrentSchemaVersion is the intent schema understood by this package.
const CurrentSchemaVersion = 2

// ApplicationSchemaVersion adds canonical application creation without
// changing the delivered schema used by existing-project operations.
const ApplicationSchemaVersion = 3

// MaximumInitialFeatures bounds one composite application bootstrap.
const MaximumInitialFeatures = 8

// MaximumApplicationTextBytes bounds optional application enrichment.
const MaximumApplicationTextBytes = 4 << 10

// MaximumDocumentationTargets bounds one documentation request to the four
// canonical Diataxis quadrants.
const MaximumDocumentationTargets = 4

// MaximumDocumentationReaderGoalBytes bounds one user-facing reader goal.
const MaximumDocumentationReaderGoalBytes = 4 << 10

// Operation is one admitted generator operation.
type Operation string

const (
	// OperationCreateApplication creates one canonical Hatmax application.
	OperationCreateApplication Operation = "create_application"
	// OperationCreateFeature creates one complete canonical feature.
	OperationCreateFeature Operation = "create_feature"
	// OperationAddField adds one field across every required representation.
	OperationAddField Operation = "add_field"
	// OperationAddValidation adds one validation at every owning boundary.
	OperationAddValidation Operation = "add_validation"
	// OperationDocumentFeature documents one existing canonical feature.
	OperationDocumentFeature Operation = "document_feature"
)

// Documentation describes explicit documentation intent.
type Documentation string

const (
	// DocumentationNotRequested prevents implicit documentation changes.
	DocumentationNotRequested Documentation = "not_requested"
	// DocumentationExisting documents behavior already present in the project.
	DocumentationExisting Documentation = "document_existing_behavior"
	// DocumentationPlanned documents the admitted planned change.
	DocumentationPlanned Documentation = "document_planned_change"
)

// DocumentationQuadrant identifies one canonical Diataxis reader need.
type DocumentationQuadrant string

const (
	// DocumentationTutorial guides a newcomer through a concrete outcome.
	DocumentationTutorial DocumentationQuadrant = "tutorial"
	// DocumentationHowTo guides an experienced reader through one task.
	DocumentationHowTo DocumentationQuadrant = "how_to"
	// DocumentationReference records exact observed contracts.
	DocumentationReference DocumentationQuadrant = "reference"
	// DocumentationExplanation describes rationale and relationships.
	DocumentationExplanation DocumentationQuadrant = "explanation"
)

// DocumentationTarget records a reader need without prescribing paths,
// headings, prose, or navigation.
type DocumentationTarget struct {
	Quadrant   DocumentationQuadrant `json:"quadrant" yaml:"quadrant"`
	Subject    string                `json:"subject" yaml:"subject"`
	ReaderGoal string                `json:"reader_goal" yaml:"reader_goal"`
}

// ValidationScope identifies the boundary owned by a requested validation.
type ValidationScope string

const (
	// ValidationDurable protects state independently of the client.
	ValidationDurable ValidationScope = "durable"
	// ValidationClientOnly affects interaction without protecting durable state.
	ValidationClientOnly ValidationScope = "client_only"
)

// Intent is the complete ephemeral interpreter output admitted by the schema.
type Intent struct {
	SchemaVersion        int                   `json:"schema_version" yaml:"schema_version"`
	Operation            Operation             `json:"operation" yaml:"operation"`
	ProjectFingerprint   string                `json:"project_fingerprint,omitempty" yaml:"project_fingerprint,omitempty"`
	SourceFingerprint    string                `json:"source_fingerprint,omitempty" yaml:"source_fingerprint,omitempty"`
	HatmaxVersion        string                `json:"hatmax_version" yaml:"hatmax_version"`
	BookVersion          int                   `json:"book_version" yaml:"book_version"`
	Archetype            string                `json:"archetype" yaml:"archetype"`
	Feature              string                `json:"feature" yaml:"feature"`
	Domain               Domain                `json:"domain" yaml:"domain"`
	Capabilities         []string              `json:"capabilities" yaml:"capabilities"`
	Documentation        Documentation         `json:"documentation,omitempty" yaml:"documentation,omitempty"`
	DocumentationTargets []DocumentationTarget `json:"documentation_targets,omitempty" yaml:"documentation_targets,omitempty"`
	Exceptions           []Exception           `json:"exceptions,omitempty" yaml:"exceptions,omitempty"`
	Application          *ApplicationIdentity  `json:"application,omitempty" yaml:"application,omitempty"`
	Target               *ApplicationTarget    `json:"target,omitempty" yaml:"target,omitempty"`
	InitialFeatures      []InitialFeature      `json:"initial_features,omitempty" yaml:"initial_features,omitempty"`
}

// ApplicationIdentity contains product identity without Book-owned file or
// package layout decisions.
type ApplicationIdentity struct {
	DisplayName string `json:"display_name" yaml:"display_name"`
	ProjectSlug string `json:"project_slug,omitempty" yaml:"project_slug,omitempty"`
	ModulePath  string `json:"module_path,omitempty" yaml:"module_path,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Niche       string `json:"niche,omitempty" yaml:"niche,omitempty"`
}

// ApplicationTarget selects the authorized session directory and one child
// directory without exposing scaffold-owned paths.
type ApplicationTarget struct {
	Base      string `json:"base" yaml:"base"`
	Directory string `json:"directory,omitempty" yaml:"directory,omitempty"`
}

// InitialFeature is one product-level feature unit requested with application
// creation. Its architecture and capabilities remain Book-owned.
type InitialFeature struct {
	Feature string `json:"feature" yaml:"feature"`
	Domain  Domain `json:"domain" yaml:"domain"`
}

// Domain contains only application-specific decisions admitted by the
// archetype.
type Domain struct {
	Entity     string          `json:"entity,omitempty" yaml:"entity,omitempty"`
	Route      string          `json:"route,omitempty" yaml:"route,omitempty"`
	Label      string          `json:"label,omitempty" yaml:"label,omitempty"`
	Ownership  string          `json:"ownership,omitempty" yaml:"ownership,omitempty"`
	Fields     []Field         `json:"fields,omitempty" yaml:"fields,omitempty"`
	Field      *Field          `json:"field,omitempty" yaml:"field,omitempty"`
	Validation *ValidationRule `json:"validation,omitempty" yaml:"validation,omitempty"`
	Rules      []BusinessRule  `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// Field describes one domain field without prescribing implementation files
// or storage adapters.
type Field struct {
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Label    string `json:"label,omitempty" yaml:"label,omitempty"`
	Required bool   `json:"required,omitempty" yaml:"required,omitempty"`
}

// ValidationRule describes one requested domain or interaction constraint.
type ValidationRule struct {
	Field   string          `json:"field" yaml:"field"`
	Kind    string          `json:"kind" yaml:"kind"`
	Value   string          `json:"value,omitempty" yaml:"value,omitempty"`
	Message string          `json:"message,omitempty" yaml:"message,omitempty"`
	Scope   ValidationScope `json:"scope" yaml:"scope"`
}

// BusinessRule records one application-specific invariant or workflow rule.
type BusinessRule struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Owner       string `json:"owner" yaml:"owner"`
}

// Exception records one explicitly approved, bounded departure request.
type Exception struct {
	Rule     string `json:"rule" yaml:"rule"`
	Reason   string `json:"reason" yaml:"reason"`
	Scope    string `json:"scope" yaml:"scope"`
	Approved bool   `json:"approved" yaml:"approved"`
}

// SchemaError describes invalid typed intent content.
type SchemaError struct {
	Code    string
	Path    string
	Message string
}

func (e SchemaError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	return fmt.Sprintf("%s at %s: %s", e.Code, e.Path, e.Message)
}

func schemaError(code, path, format string, arguments ...any) error {
	return SchemaError{
		Code:    code,
		Path:    path,
		Message: fmt.Sprintf(format, arguments...),
	}
}
