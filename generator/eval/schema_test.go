package eval

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

func TestInterpretationOutputSchemaIsClosedAndVersioned(t *testing.T) {
	data, err := InterpretationOutputSchema()
	if err != nil {
		t.Fatalf("InterpretationOutputSchema() error = %v", err)
	}

	var schema map[string]any

	err = json.Unmarshal(data, &schema)
	if err != nil {
		t.Fatalf("decode output schema: %v", err)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok || len(properties) != 5 {
		t.Fatalf("Output schema properties = %#v, want five", schema["properties"])
	}

	assertClosedSchemaObjects(t, schema, "schema")
	assertStrictRequiredProperties(t, schema, "schema")

	for _, forbidden := range []string{`"plan"`, `"path"`, `"command"`, `"dependency"`, `"edit"`, `"approval"`} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Errorf("Output schema contains forbidden field %s", forbidden)
		}
	}
}

func assertStrictRequiredProperties(t *testing.T, value any, path string) {
	t.Helper()

	switch typed := value.(type) {
	case map[string]any:
		if properties, hasProperties := typed["properties"].(map[string]any); hasProperties {
			required, ok := typed["required"].([]any)
			if !ok || len(required) != len(properties) {
				t.Errorf("%s requires %#v for properties %#v", path, typed["required"], properties)
			} else {
				requiredSet := make(map[string]struct{}, len(required))
				for _, name := range required {
					requiredSet[name.(string)] = struct{}{}
				}

				for name := range properties {
					if _, exists := requiredSet[name]; !exists {
						t.Errorf("%s property %q is not required", path, name)
					}
				}
			}
		}

		for key, child := range typed {
			assertStrictRequiredProperties(t, child, path+"."+key)
		}
	case []any:
		for index, child := range typed {
			assertStrictRequiredProperties(t, child, path+"["+strconv.Itoa(index)+"]")
		}
	}
}

func TestDecodeInterpretationAcceptsEachResultVariant(t *testing.T) {
	value := validOutputIntent()
	tests := []struct {
		name  string
		value Interpretation
	}{
		{name: "intent", value: Interpretation{SchemaVersion: CurrentInterpretationSchemaVersion, Kind: InterpretationIntent, Intent: &value}},
		{name: "clarification", value: Interpretation{SchemaVersion: CurrentInterpretationSchemaVersion, Kind: InterpretationClarification, Clarifications: []intent.Clarification{{Field: "domain.validation.kind", Question: "Which validation?"}}}},
		{name: "unsupported", value: Interpretation{SchemaVersion: CurrentInterpretationSchemaVersion, Kind: InterpretationUnsupported, Diagnostics: []intent.Diagnostic{{Code: "HMGEN-REQUEST-UNSUPPORTED", Field: "prompt", Message: "outside the Book"}}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.value)
			if err != nil {
				t.Fatalf("marshal interpretation: %v", err)
			}

			result, err := DecodeInterpretation(data)
			if err != nil {
				t.Fatalf("DecodeInterpretation() error = %v", err)
			}

			if result.SchemaVersion != CurrentInterpretationSchemaVersion || result.Kind != test.value.Kind {
				t.Errorf("DecodeInterpretation() = %#v, want kind %q", result, test.value.Kind)
			}
		})
	}
}

func TestDecodeInterpretationRejectsInvalidModelOutput(t *testing.T) {
	validIntent := validOutputIntent()

	intentData, err := json.Marshal(validIntent)
	if err != nil {
		t.Fatalf("marshal intent: %v", err)
	}

	tests := []struct {
		name string
		data []byte
		code string
	}{
		{name: "empty", data: nil, code: "evaluation_output_invalid"},
		{name: "oversized", data: bytes.Repeat([]byte(" "), MaximumInterpretationBytes+1), code: "evaluation_output_too_large"},
		{name: "unknown top-level field", data: []byte(`{"schema_version":2,"kind":"unsupported","diagnostics":[{"code":"HMGEN-REQUEST-UNSUPPORTED","field":"prompt","message":"unsupported"}],"plan":{}}`), code: "evaluation_output_invalid"},
		{name: "unknown intent field", data: []byte(`{"schema_version":2,"kind":"intent","intent":{"schema_version":2,"operation":"create_feature","project_fingerprint":"sha256:0000000000000000000000000000000000000000000000000000000000000000","hatmax_version":"v0.4.0","book_version":1,"archetype":"server_rendered_crud","feature":"invoice","domain":{"fields":[]},"capabilities":[],"documentation":"not_requested","documentation_targets":[],"exceptions":[],"path":"main.go"}}`), code: "evaluation_output_invalid"},
		{name: "multiple values", data: []byte(`{"schema_version":2,"kind":"unsupported","diagnostics":[{"code":"HMGEN-REQUEST-UNSUPPORTED","field":"prompt","message":"unsupported"}]} {}`), code: "evaluation_output_invalid"},
		{name: "unsupported schema", data: []byte(`{"schema_version":3,"kind":"unsupported","diagnostics":[{"code":"HMGEN-REQUEST-UNSUPPORTED","field":"prompt","message":"unsupported"}]}`), code: "evaluation_output_schema_unsupported"},
		{name: "contradictory variant", data: []byte(`{"schema_version":2,"kind":"intent","intent":` + string(intentData) + `,"diagnostics":[{"code":"HMGEN-REQUEST-UNSUPPORTED","field":"prompt","message":"unsupported"}]}`), code: "evaluation_result_invalid"},
		{name: "free-form result", data: []byte(`{"schema_version":2,"kind":"free_form"}`), code: "evaluation_kind_invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeInterpretation(test.data)
			requireEvaluationCode(t, err, test.code)
		})
	}
}

func assertClosedSchemaObjects(t *testing.T, value any, path string) {
	t.Helper()

	switch typed := value.(type) {
	case map[string]any:
		if _, hasProperties := typed["properties"]; hasProperties && typed["additionalProperties"] != false {
			t.Errorf("%s has properties without additionalProperties false", path)
		}

		for key, child := range typed {
			assertClosedSchemaObjects(t, child, path+"."+key)
		}
	case []any:
		for index, child := range typed {
			assertClosedSchemaObjects(t, child, path+"["+strconv.Itoa(index)+"]")
		}
	}
}

func validOutputIntent() intent.Intent {
	return intent.Intent{
		SchemaVersion:      intent.CurrentSchemaVersion,
		Operation:          intent.OperationCreateFeature,
		ProjectFingerprint: "sha256:" + strings.Repeat("0", 64),
		HatmaxVersion:      "v0.4.0",
		BookVersion:        1,
		Archetype:          "server_rendered_crud",
		Feature:            "invoice",
		Domain: intent.Domain{
			Entity: "Invoice",
			Fields: []intent.Field{{Name: "name", Type: "string", Required: true}},
		},
		Capabilities:         []string{"postgres_persistence", "htmx_form", "runtime_validation"},
		Documentation:        intent.DocumentationNotRequested,
		DocumentationTargets: []intent.DocumentationTarget{},
		Exceptions:           []intent.Exception{},
	}
}
