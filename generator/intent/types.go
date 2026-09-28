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
	SchemaVersion      int           `yaml:"schema_version"`
	Operation          Operation     `yaml:"operation"`
	ProjectFingerprint string        `yaml:"project_fingerprint"`
	HatmaxVersion      string        `yaml:"hatmax_version"`
	BookVersion        int           `yaml:"book_version"`
	Archetype          string        `yaml:"archetype"`
	Feature            string        `yaml:"feature"`
	Domain             Domain        `yaml:"domain"`
	Capabilities       []string      `yaml:"capabilities"`
	Documentation      Documentation `yaml:"documentation,omitempty"`
	Exceptions         []Exception   `yaml:"exceptions,omitempty"`
}

// Domain contains only application-specific decisions admitted by the
// archetype.
type Domain struct {
	Entity     string          `yaml:"entity,omitempty"`
	Route      string          `yaml:"route,omitempty"`
	Label      string          `yaml:"label,omitempty"`
	Ownership  string          `yaml:"ownership,omitempty"`
	Fields     []Field         `yaml:"fields,omitempty"`
	Field      *Field          `yaml:"field,omitempty"`
	Validation *ValidationRule `yaml:"validation,omitempty"`
	Rules      []BusinessRule  `yaml:"rules,omitempty"`
}

// Field describes one domain field without prescribing implementation files
// or storage adapters.
type Field struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Label    string `yaml:"label,omitempty"`
	Required bool   `yaml:"required,omitempty"`
}

// ValidationRule describes one requested domain or interaction constraint.
type ValidationRule struct {
	Field   string          `yaml:"field"`
	Kind    string          `yaml:"kind"`
	Value   string          `yaml:"value,omitempty"`
	Message string          `yaml:"message,omitempty"`
	Scope   ValidationScope `yaml:"scope"`
}

// BusinessRule records one application-specific invariant or workflow rule.
type BusinessRule struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Owner       string `yaml:"owner"`
}

// Exception records one explicitly approved, bounded departure request.
type Exception struct {
	Rule     string `yaml:"rule"`
	Reason   string `yaml:"reason"`
	Scope    string `yaml:"scope"`
	Approved bool   `yaml:"approved"`
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
