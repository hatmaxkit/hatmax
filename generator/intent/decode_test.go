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

			if value.Documentation == "" || value.Capabilities == nil || value.Exceptions == nil {
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
		{name: "schema version", mutate: func(value *Intent) { value.SchemaVersion = 2 }, code: "intent_schema_unsupported"},
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
