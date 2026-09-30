// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

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
	if value.SchemaVersion != CurrentSchemaVersion && value.SchemaVersion != ApplicationSchemaVersion {
		return schemaError(
			"intent_schema_unsupported",
			"schema_version",
			"schema version %d is not supported",
			value.SchemaVersion,
		)
	}

	if !validOperation(value.SchemaVersion, value.Operation) {
		return schemaError("intent_operation_invalid", "operation", "unknown operation %q", value.Operation)
	}

	err := validateFingerprintShape(value)
	if err != nil {
		return err
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

	if value.Operation != OperationCreateApplication && strings.TrimSpace(value.Feature) == "" {
		return schemaError("intent_required_field", "feature", "feature is required")
	}

	if !validDocumentation(value.Documentation) {
		return schemaError("intent_documentation_invalid", "documentation", "unknown documentation intent %q", value.Documentation)
	}

	err = validateDocumentationShape(value)
	if err != nil {
		return err
	}

	err = validateUniqueCapabilities(value.Capabilities)
	if err != nil {
		return err
	}

	err = validateOperationShape(value)
	if err != nil {
		return err
	}

	return validateApplicationShape(value)
}

func validOperation(schemaVersion int, operation Operation) bool {
	if operation == OperationCreateApplication {
		return schemaVersion == ApplicationSchemaVersion
	}

	return operation == OperationCreateFeature || operation == OperationAddField || operation == OperationAddValidation || operation == OperationDocumentFeature
}

func validateFingerprintShape(value Intent) error {
	if value.SchemaVersion == ApplicationSchemaVersion {
		if value.ProjectFingerprint != "" {
			return schemaError("intent_fingerprint_unexpected", "project_fingerprint", "schema version 3 uses source_fingerprint")
		}

		if value.SourceFingerprint == "" {
			return schemaError("intent_required_field", "source_fingerprint", "source fingerprint is required")
		}

		if !fingerprintPattern.MatchString(value.SourceFingerprint) {
			return schemaError("intent_fingerprint_invalid", "source_fingerprint", "fingerprint must use sha256:<hex>")
		}

		return nil
	}

	if value.SourceFingerprint != "" {
		return schemaError("intent_fingerprint_unexpected", "source_fingerprint", "schema version 2 uses project_fingerprint")
	}

	if value.ProjectFingerprint == "" {
		return schemaError("intent_required_field", "project_fingerprint", "project fingerprint is required")
	}

	if !fingerprintPattern.MatchString(value.ProjectFingerprint) {
		return schemaError("intent_fingerprint_invalid", "project_fingerprint", "fingerprint must use sha256:<hex>")
	}

	return nil
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
	case OperationCreateApplication:
		if value.Feature != "" || !domainEmpty(value.Domain) {
			return schemaError("intent_domain_shape_invalid", "domain", "create_application uses initial_features, not feature or domain")
		}
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

func validateApplicationShape(value Intent) error {
	if value.Operation != OperationCreateApplication {
		if value.Application != nil || value.Target != nil || len(value.InitialFeatures) > 0 {
			return schemaError("intent_application_shape_invalid", "application", "application fields require create_application")
		}

		return nil
	}

	if value.Application == nil {
		return schemaError("intent_required_field", "application", "application identity is required")
	}

	if value.Target == nil {
		return schemaError("intent_required_field", "target", "application target is required")
	}

	if len(value.InitialFeatures) > MaximumInitialFeatures {
		return schemaError("intent_initial_features_limit", "initial_features", "application supports at most %d initial features", MaximumInitialFeatures)
	}

	if len(value.Application.Description) > MaximumApplicationTextBytes {
		return schemaError("intent_application_text_too_large", "application.description", "description exceeds %d bytes", MaximumApplicationTextBytes)
	}

	if len(value.Application.Niche) > MaximumApplicationTextBytes {
		return schemaError("intent_application_text_too_large", "application.niche", "niche exceeds %d bytes", MaximumApplicationTextBytes)
	}

	return nil
}

func domainEmpty(value Domain) bool {
	return value.Entity == "" && value.Route == "" && value.Label == "" && value.Ownership == "" &&
		len(value.Fields) == 0 && value.Field == nil && value.Validation == nil && len(value.Rules) == 0
}
