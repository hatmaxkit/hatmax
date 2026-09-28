package intent

import (
	"fmt"
	"regexp"
	"strings"
)

var fingerprintPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

// ValidateSchema validates fields whose meaning does not depend on a project
// inventory or Hatmax Book.
func ValidateSchema(value Intent) error {
	if value.SchemaVersion != CurrentSchemaVersion {
		return schemaError(
			"intent_schema_unsupported",
			"schema_version",
			"schema version %d is not supported",
			value.SchemaVersion,
		)
	}

	if !validOperation(value.Operation) {
		return schemaError("intent_operation_invalid", "operation", "unknown operation %q", value.Operation)
	}

	if value.ProjectFingerprint == "" {
		return schemaError("intent_required_field", "project_fingerprint", "project fingerprint is required")
	}

	if !fingerprintPattern.MatchString(value.ProjectFingerprint) {
		return schemaError("intent_fingerprint_invalid", "project_fingerprint", "fingerprint must use sha256:<hex>")
	}

	if strings.TrimSpace(value.HatmaxVersion) == "" {
		return schemaError("intent_required_field", "hatmax_version", "Hatmax version is required")
	}

	if value.BookVersion < 1 {
		return schemaError("intent_book_version_invalid", "book_version", "Book version must be positive")
	}

	if strings.TrimSpace(value.Archetype) == "" {
		return schemaError("intent_required_field", "archetype", "archetype is required")
	}

	if strings.TrimSpace(value.Feature) == "" {
		return schemaError("intent_required_field", "feature", "feature is required")
	}

	if !validDocumentation(value.Documentation) {
		return schemaError("intent_documentation_invalid", "documentation", "unknown documentation intent %q", value.Documentation)
	}

	err := validateDocumentationShape(value)
	if err != nil {
		return err
	}

	err = validateUniqueCapabilities(value.Capabilities)
	if err != nil {
		return err
	}

	return validateOperationShape(value)
}

func validOperation(operation Operation) bool {
	return operation == OperationCreateFeature || operation == OperationAddField || operation == OperationAddValidation || operation == OperationDocumentFeature
}

func validateDocumentationShape(value Intent) error {
	targets := value.DocumentationTargets
	if value.Documentation == DocumentationNotRequested {
		if len(targets) != 0 {
			return schemaError("intent_documentation_targets_unexpected", "documentation_targets", "documentation targets require explicit documentation intent")
		}

		if value.Operation == OperationDocumentFeature {
			return schemaError("intent_documentation_mode_invalid", "documentation", "document_feature requires document_existing_behavior")
		}

		return nil
	}

	if len(targets) == 0 {
		return schemaError("intent_documentation_targets_required", "documentation_targets", "active documentation requires at least one target")
	}

	if len(targets) > MaximumDocumentationTargets {
		return schemaError("intent_documentation_targets_limit", "documentation_targets", "documentation supports at most %d targets", MaximumDocumentationTargets)
	}

	if value.Operation == OperationDocumentFeature && value.Documentation != DocumentationExisting {
		return schemaError("intent_documentation_mode_invalid", "documentation", "document_feature requires document_existing_behavior")
	}

	if value.Operation != OperationDocumentFeature && value.Documentation != DocumentationPlanned {
		return schemaError("intent_documentation_mode_invalid", "documentation", "implementation operations require document_planned_change when documentation is active")
	}

	seen := make(map[string]struct{}, len(targets))
	for index, target := range targets {
		path := fmt.Sprintf("documentation_targets[%d]", index)
		if !validDocumentationQuadrant(target.Quadrant) {
			return schemaError("intent_documentation_quadrant_invalid", path+".quadrant", "unknown Diataxis quadrant %q", target.Quadrant)
		}

		if !featureNamePattern.MatchString(target.Subject) {
			return schemaError("intent_documentation_subject_invalid", path+".subject", "documentation subject %q must use lower snake case", target.Subject)
		}

		if strings.TrimSpace(target.ReaderGoal) == "" {
			return schemaError("intent_required_field", path+".reader_goal", "reader goal is required")
		}

		if len(target.ReaderGoal) > MaximumDocumentationReaderGoalBytes {
			return schemaError("intent_documentation_reader_goal_too_large", path+".reader_goal", "reader goal exceeds %d bytes", MaximumDocumentationReaderGoalBytes)
		}

		identity := string(target.Quadrant) + "\x00" + target.Subject
		if _, exists := seen[identity]; exists {
			return schemaError("intent_documentation_target_duplicate", path, "documentation target %q for %q is duplicated", target.Quadrant, target.Subject)
		}

		seen[identity] = struct{}{}
	}

	return nil
}

func validDocumentationQuadrant(quadrant DocumentationQuadrant) bool {
	return quadrant == DocumentationTutorial || quadrant == DocumentationHowTo || quadrant == DocumentationReference || quadrant == DocumentationExplanation
}

func validDocumentation(documentation Documentation) bool {
	return documentation == DocumentationNotRequested || documentation == DocumentationExisting || documentation == DocumentationPlanned
}

func validateUniqueCapabilities(capabilities []string) error {
	seen := make(map[string]struct{}, len(capabilities))
	for index, capability := range capabilities {
		if strings.TrimSpace(capability) == "" {
			return schemaError("intent_required_field", "capabilities", "capability at index %d is empty", index)
		}

		if _, exists := seen[capability]; exists {
			return schemaError("intent_duplicate_value", "capabilities", "duplicate capability %q", capability)
		}

		seen[capability] = struct{}{}
	}

	return nil
}

func validateOperationShape(value Intent) error {
	switch value.Operation {
	case OperationCreateFeature:
		if value.Domain.Field != nil || value.Domain.Validation != nil {
			return schemaError("intent_domain_shape_invalid", "domain", "create_feature uses fields, not field or validation")
		}
	case OperationAddField:
		if len(value.Domain.Fields) > 0 || value.Domain.Validation != nil {
			return schemaError("intent_domain_shape_invalid", "domain", "add_field uses field, not fields or validation")
		}
	case OperationAddValidation:
		if len(value.Domain.Fields) > 0 || value.Domain.Field != nil {
			return schemaError("intent_domain_shape_invalid", "domain", "add_validation uses validation, not field or fields")
		}
	case OperationDocumentFeature:
		if value.Domain.Entity != "" || value.Domain.Route != "" || value.Domain.Label != "" || value.Domain.Ownership != "" || len(value.Domain.Fields) > 0 || value.Domain.Field != nil || value.Domain.Validation != nil || len(value.Domain.Rules) > 0 {
			return schemaError("intent_domain_shape_invalid", "domain", "document_feature does not accept domain changes")
		}
	}

	return nil
}
