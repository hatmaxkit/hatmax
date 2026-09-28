package intent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeYAMLAcceptsSupportedOperations(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		operation Operation
	}{
		{name: "create feature", fixture: "create-feature.yaml", operation: OperationCreateFeature},
		{name: "add field", fixture: "add-field.yaml", operation: OperationAddField},
		{name: "add validation", fixture: "add-validation.yaml", operation: OperationAddValidation},
		{name: "document feature", fixture: "document-feature.yaml", operation: OperationDocumentFeature},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := loadIntentFixture(t, "valid", test.fixture)

			if value.Operation != test.operation {
				t.Errorf("Operation = %q, want %q", value.Operation, test.operation)
			}

			if value.ProjectFingerprint != testFingerprint {
				t.Errorf("ProjectFingerprint = %q, want fixture fingerprint", value.ProjectFingerprint)
			}

			if value.Documentation == "" || value.Capabilities == nil || value.DocumentationTargets == nil || value.Exceptions == nil {
				t.Errorf("DecodeYAML() did not normalize intent: %#v", value)
			}

			if value.Domain.Fields == nil || value.Domain.Rules == nil {
				t.Errorf("DecodeYAML() did not normalize domain slices: %#v", value.Domain)
			}
		})
	}
}

func TestDecodeYAMLDefaultsDocumentationToNotRequested(t *testing.T) {
	value := loadIntentFixture(t, "valid", "add-field.yaml")

	if value.Documentation != DocumentationNotRequested {
		t.Errorf("Documentation = %q, want %q", value.Documentation, DocumentationNotRequested)
	}
}

func TestDecodeYAMLCanonicalizesHatmaxVersion(t *testing.T) {
	value := loadIntentFixture(t, "valid", "create-feature.yaml")

	if value.HatmaxVersion != "v0.4.0" {
		t.Errorf("HatmaxVersion = %q, want v0.4.0", value.HatmaxVersion)
	}
}

func TestDecodeYAMLRejectsInvalidFixtures(t *testing.T) {
	tests := []struct {
		fixture string
		code    string
	}{
		{fixture: "duplicate-capability.yaml", code: "intent_duplicate_value"},
		{fixture: "fingerprint.yaml", code: "intent_fingerprint_invalid"},
		{fixture: "domain-shape.yaml", code: "intent_domain_shape_invalid"},
	}

	for _, test := range tests {
		t.Run(test.fixture, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", "invalid", test.fixture))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			_, err = DecodeYAML(data)
			requireSchemaCode(t, err, test.code)
		})
	}
}

func TestDecodeYAMLRejectsUnknownFields(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "invalid", "unknown-field.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	_, err = DecodeYAML(data)
	if err == nil || !strings.Contains(err.Error(), "field architecture not found") {
		t.Fatalf("DecodeYAML() error = %v, want unknown architecture field", err)
	}
}

func TestDecodeYAMLRejectsMultipleDocuments(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "valid", "add-field.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	_, err = DecodeYAML(append(data, []byte("\n---\nschema_version: 1\n")...))
	if err == nil || !strings.Contains(err.Error(), "multiple YAML documents") {
		t.Fatalf("DecodeYAML() error = %v, want multiple-document rejection", err)
	}
}

func TestValidateSchemaRejectsInvalidEnvelopeFields(t *testing.T) {
	base := loadIntentFixture(t, "valid", "add-field.yaml")
	tests := []struct {
		name   string
		mutate func(*Intent)
		code   string
	}{
		{name: "schema version", mutate: func(value *Intent) { value.SchemaVersion = CurrentSchemaVersion + 1 }, code: "intent_schema_unsupported"},
		{name: "operation", mutate: func(value *Intent) { value.Operation = "remove_feature" }, code: "intent_operation_invalid"},
		{name: "project fingerprint", mutate: func(value *Intent) { value.ProjectFingerprint = "" }, code: "intent_required_field"},
		{name: "Hatmax version", mutate: func(value *Intent) { value.HatmaxVersion = " " }, code: "intent_required_field"},
		{name: "Book version", mutate: func(value *Intent) { value.BookVersion = 0 }, code: "intent_book_version_invalid"},
		{name: "archetype", mutate: func(value *Intent) { value.Archetype = "" }, code: "intent_required_field"},
		{name: "feature", mutate: func(value *Intent) { value.Feature = "" }, code: "intent_required_field"},
		{name: "documentation", mutate: func(value *Intent) { value.Documentation = "always" }, code: "intent_documentation_invalid"},
		{name: "empty capability", mutate: func(value *Intent) { value.Capabilities = append(value.Capabilities, " ") }, code: "intent_required_field"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := base
			value.Capabilities = append([]string(nil), base.Capabilities...)
			test.mutate(&value)

			requireSchemaCode(t, ValidateSchema(value), test.code)
		})
	}
}

func TestValidateSchemaEnforcesDocumentationShape(t *testing.T) {
	base := loadIntentFixture(t, "valid", "document-feature.yaml")
	tests := []struct {
		name   string
		mutate func(*Intent)
		code   string
	}{
		{name: "active targets missing", mutate: func(value *Intent) { value.DocumentationTargets = nil }, code: "intent_documentation_targets_required"},
		{name: "too many targets", mutate: func(value *Intent) {
			value.DocumentationTargets = append(value.DocumentationTargets, DocumentationTarget{Quadrant: DocumentationTutorial, Subject: "property_tutorial", ReaderGoal: "Learn another property workflow."}, DocumentationTarget{Quadrant: DocumentationHowTo, Subject: "property_editing", ReaderGoal: "Update a property."}, DocumentationTarget{Quadrant: DocumentationExplanation, Subject: "property_ownership", ReaderGoal: "Understand property ownership."}, DocumentationTarget{Quadrant: DocumentationReference, Subject: "property_routes", ReaderGoal: "Find property routes."})
		}, code: "intent_documentation_targets_limit"},
		{name: "invalid quadrant", mutate: func(value *Intent) { value.DocumentationTargets[0].Quadrant = "guide" }, code: "intent_documentation_quadrant_invalid"},
		{name: "invalid subject", mutate: func(value *Intent) { value.DocumentationTargets[0].Subject = "Property Guide" }, code: "intent_documentation_subject_invalid"},
		{name: "empty reader goal", mutate: func(value *Intent) { value.DocumentationTargets[0].ReaderGoal = " " }, code: "intent_required_field"},
		{name: "large reader goal", mutate: func(value *Intent) {
			value.DocumentationTargets[0].ReaderGoal = strings.Repeat("a", MaximumDocumentationReaderGoalBytes+1)
		}, code: "intent_documentation_reader_goal_too_large"},
		{name: "duplicate target", mutate: func(value *Intent) {
			value.DocumentationTargets = append(value.DocumentationTargets, value.DocumentationTargets[0])
		}, code: "intent_documentation_target_duplicate"},
		{name: "document planned behavior", mutate: func(value *Intent) { value.Documentation = DocumentationPlanned }, code: "intent_documentation_mode_invalid"},
		{name: "document feature domain", mutate: func(value *Intent) { value.Domain.Label = "Properties" }, code: "intent_domain_shape_invalid"},
		{name: "inactive targets", mutate: func(value *Intent) {
			value.Operation = OperationAddField
			value.Domain.Field = &Field{Name: "summary", Type: "text"}
			value.Documentation = DocumentationNotRequested
		}, code: "intent_documentation_targets_unexpected"},
		{name: "implementation existing behavior", mutate: func(value *Intent) {
			value.Operation = OperationAddField
			value.Domain.Field = &Field{Name: "summary", Type: "text"}
		}, code: "intent_documentation_mode_invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := base
			value.DocumentationTargets = append([]DocumentationTarget{}, base.DocumentationTargets...)
			test.mutate(&value)

			requireSchemaCode(t, ValidateSchema(value), test.code)
		})
	}
}

func TestSchemaErrorIncludesPath(t *testing.T) {
	err := ValidateSchema(Intent{})

	var schemaErr SchemaError
	if !errors.As(err, &schemaErr) {
		t.Fatalf("ValidateSchema() error = %v, want SchemaError", err)
	}

	if !strings.Contains(err.Error(), "at schema_version") {
		t.Errorf("SchemaError.Error() = %q, want field path", err.Error())
	}
}
