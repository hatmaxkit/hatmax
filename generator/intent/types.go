// Package intent defines and validates typed Hatmax generator intents.
package intent

import "fmt"

// CurrentSchemaVersion is the intent schema understood by this package.
const CurrentSchemaVersion = 1

// Operation is one admitted generator operation.
type Operation string

const (
	// OperationCreateFeature creates one complete canonical feature.
	OperationCreateFeature Operation = "create_feature"
	// OperationAddField adds one field across every required representation.
	OperationAddField Operation = "add_field"
	// OperationAddValidation adds one validation at every owning boundary.
	OperationAddValidation Operation = "add_validation"
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
	SchemaVersion      int           `json:"schema_version" yaml:"schema_version"`
	Operation          Operation     `json:"operation" yaml:"operation"`
	ProjectFingerprint string        `json:"project_fingerprint" yaml:"project_fingerprint"`
	HatmaxVersion      string        `json:"hatmax_version" yaml:"hatmax_version"`
	BookVersion        int           `json:"book_version" yaml:"book_version"`
	Archetype          string        `json:"archetype" yaml:"archetype"`
	Feature            string        `json:"feature" yaml:"feature"`
	Domain             Domain        `json:"domain" yaml:"domain"`
	Capabilities       []string      `json:"capabilities" yaml:"capabilities"`
	Documentation      Documentation `json:"documentation,omitempty" yaml:"documentation,omitempty"`
	Exceptions         []Exception   `json:"exceptions,omitempty" yaml:"exceptions,omitempty"`
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
