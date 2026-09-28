package eval

import "encoding/json"

// InterpretationOutputSchema returns the versioned JSON Schema supplied to an
// interpreter backend. The schema admits no plan, path, command, dependency,
// edit, or approval fields.
func InterpretationOutputSchema() ([]byte, error) {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"schema_version", "kind", "intent", "clarifications", "diagnostics"},
		"properties": map[string]any{
			"schema_version": map[string]any{"type": "integer", "enum": []int{CurrentInterpretationSchemaVersion}},
			"kind":           enumSchema("intent", "clarification_required", "unsupported"),
			"intent":         nullableSchema(map[string]any{"$ref": "#/$defs/intent"}),
			"clarifications": nullableSchema(map[string]any{
				"type":  "array",
				"items": map[string]any{"$ref": "#/$defs/clarification"},
			}),
			"diagnostics": nullableSchema(map[string]any{
				"type":  "array",
				"items": map[string]any{"$ref": "#/$defs/diagnostic"},
			}),
		},
		"$defs": interpretationDefinitions(),
	}

	return json.Marshal(schema)
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
				"schema_version":      map[string]any{"type": "integer", "enum": []int{1}},
				"operation":           enumSchema("create_feature", "add_field", "add_validation"),
				"project_fingerprint": map[string]any{"type": "string"},
				"hatmax_version":      nonemptyStringSchema(),
				"book_version":        map[string]any{"type": "integer"},
				"archetype":           nonemptyStringSchema(),
				"feature":             nonemptyStringSchema(),
				"domain":              map[string]any{"$ref": "#/$defs/domain"},
				"capabilities": map[string]any{
					"type":  "array",
					"items": nonemptyStringSchema(),
				},
				"documentation": enumSchema("not_requested", "document_existing_behavior", "document_planned_change"),
				"exceptions": map[string]any{
					"type":  "array",
					"items": map[string]any{"$ref": "#/$defs/exception"},
				},
			},
		},
		"domain": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required": []string{
				"entity", "route", "label", "ownership", "fields", "field",
				"validation", "rules",
			},
			"properties": map[string]any{
				"entity":    nullableSchema(nonemptyStringSchema()),
				"route":     nullableSchema(nonemptyStringSchema()),
				"label":     nullableSchema(nonemptyStringSchema()),
				"ownership": nullableSchema(nonemptyStringSchema()),
				"fields": nullableSchema(map[string]any{
					"type":  "array",
					"items": map[string]any{"$ref": "#/$defs/field"},
				}),
				"field":      nullableSchema(map[string]any{"$ref": "#/$defs/field"}),
				"validation": nullableSchema(map[string]any{"$ref": "#/$defs/validation"}),
				"rules": nullableSchema(map[string]any{
					"type":  "array",
					"items": map[string]any{"$ref": "#/$defs/business_rule"},
				}),
			},
		},
		"field": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"name", "type", "label", "required"},
			"properties": map[string]any{
				"name":     nonemptyStringSchema(),
				"type":     nonemptyStringSchema(),
				"label":    map[string]any{"type": []string{"string", "null"}},
				"required": map[string]any{"type": "boolean"},
			},
		},
		"validation": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"field", "kind", "value", "message", "scope"},
			"properties": map[string]any{
				"field":   nonemptyStringSchema(),
				"kind":    nonemptyStringSchema(),
				"value":   map[string]any{"type": []string{"string", "null"}},
				"message": map[string]any{"type": []string{"string", "null"}},
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
				"code":    map[string]any{"type": "string"},
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
	return map[string]any{"type": "string"}
}

func boundedStringSchema() map[string]any {
	return map[string]any{"type": "string"}
}

func nullableSchema(value map[string]any) map[string]any {
	return map[string]any{"anyOf": []any{value, map[string]any{"type": "null"}}}
}
