package eval

import "encoding/json"

// InterpretationOutputSchema returns the versioned JSON Schema supplied to an
// interpreter backend. The schema admits no plan, path, command, dependency,
// edit, or approval fields.
func InterpretationOutputSchema() ([]byte, error) {
	schema := map[string]any{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title":   "Hatmax interpreter result v1",
		"oneOf": []any{
			interpretationVariant("intent", "intent", map[string]any{"$ref": "#/definitions/intent"}),
			interpretationVariant("clarification_required", "clarifications", map[string]any{
				"type":     "array",
				"minItems": 1,
				"maxItems": MaximumClarificationExchanges,
				"items":    map[string]any{"$ref": "#/definitions/clarification"},
			}),
			interpretationVariant("unsupported", "diagnostics", map[string]any{
				"type":     "array",
				"minItems": 1,
				"maxItems": MaximumInterpretationDiagnostics,
				"items":    map[string]any{"$ref": "#/definitions/diagnostic"},
			}),
		},
		"definitions": interpretationDefinitions(),
	}

	return json.Marshal(schema)
}

func interpretationVariant(kind, payload string, payloadSchema map[string]any) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"schema_version", "kind", payload},
		"properties": map[string]any{
			"schema_version": map[string]any{"const": CurrentInterpretationSchemaVersion},
			"kind":           map[string]any{"const": kind},
			payload:          payloadSchema,
		},
	}
}

func interpretationDefinitions() map[string]any {
	return map[string]any{
		"intent": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required": []string{
				"schema_version", "operation", "project_fingerprint",
				"hatmax_version", "book_version", "archetype", "feature",
				"domain", "capabilities", "documentation", "exceptions",
			},
			"properties": map[string]any{
				"schema_version":      map[string]any{"const": 1},
				"operation":           enumSchema("create_feature", "add_field", "add_validation"),
				"project_fingerprint": map[string]any{"type": "string", "pattern": "^sha256:[a-f0-9]{64}$"},
				"hatmax_version":      nonemptyStringSchema(),
				"book_version":        map[string]any{"type": "integer", "minimum": 1},
				"archetype":           nonemptyStringSchema(),
				"feature":             nonemptyStringSchema(),
				"domain":              map[string]any{"$ref": "#/definitions/domain"},
				"capabilities": map[string]any{
					"type":        "array",
					"uniqueItems": true,
					"items":       nonemptyStringSchema(),
				},
				"documentation": enumSchema("not_requested", "document_existing_behavior", "document_planned_change"),
				"exceptions": map[string]any{
					"type":  "array",
					"items": map[string]any{"$ref": "#/definitions/exception"},
				},
			},
		},
		"domain": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"entity":    nonemptyStringSchema(),
				"route":     nonemptyStringSchema(),
				"label":     nonemptyStringSchema(),
				"ownership": nonemptyStringSchema(),
				"fields": map[string]any{
					"type":  "array",
					"items": map[string]any{"$ref": "#/definitions/field"},
				},
				"field":      map[string]any{"$ref": "#/definitions/field"},
				"validation": map[string]any{"$ref": "#/definitions/validation"},
				"rules": map[string]any{
					"type":  "array",
					"items": map[string]any{"$ref": "#/definitions/business_rule"},
				},
			},
		},
		"field": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"name", "type", "required"},
			"properties": map[string]any{
				"name":     nonemptyStringSchema(),
				"type":     nonemptyStringSchema(),
				"label":    map[string]any{"type": "string"},
				"required": map[string]any{"type": "boolean"},
			},
		},
		"validation": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"field", "kind", "scope"},
			"properties": map[string]any{
				"field":   nonemptyStringSchema(),
				"kind":    nonemptyStringSchema(),
				"value":   map[string]any{"type": "string"},
				"message": map[string]any{"type": "string"},
				"scope":   enumSchema("durable", "client_only"),
			},
		},
		"business_rule": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"name", "description", "owner"},
			"properties": map[string]any{
				"name":        nonemptyStringSchema(),
				"description": nonemptyStringSchema(),
				"owner":       nonemptyStringSchema(),
			},
		},
		"exception": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"rule", "reason", "scope", "approved"},
			"properties": map[string]any{
				"rule":     nonemptyStringSchema(),
				"reason":   nonemptyStringSchema(),
				"scope":    nonemptyStringSchema(),
				"approved": map[string]any{"type": "boolean"},
			},
		},
		"clarification": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"field", "question"},
			"properties": map[string]any{
				"field":    boundedStringSchema(),
				"question": boundedStringSchema(),
			},
		},
		"diagnostic": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"code", "field", "message"},
			"properties": map[string]any{
				"code":    map[string]any{"type": "string", "pattern": "^[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)+$"},
				"field":   boundedStringSchema(),
				"message": boundedStringSchema(),
			},
		},
	}
}

func enumSchema(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}

func nonemptyStringSchema() map[string]any {
	return map[string]any{"type": "string", "minLength": 1}
}

func boundedStringSchema() map[string]any {
	return map[string]any{"type": "string", "minLength": 1, "maxLength": MaximumClarificationTextBytes}
}
