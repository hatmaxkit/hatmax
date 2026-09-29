package eval

import (
	"encoding/json"

	"hatmax.adrianpk.com/generator/intent"
)

// InterpretationOutputSchema returns the versioned JSON Schema supplied to an
// interpreter backend. The schema admits no plan, path, command, dependency,
// edit, or approval fields.
func InterpretationOutputSchema() ([]byte, error) {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"schema_version", "kind", "response", "intent", "clarifications", "diagnostics"},
		"properties": map[string]any{
			"schema_version": map[string]any{"type": "integer", "enum": []int{CurrentInterpretationSchemaVersion}},
			"kind":           enumSchema("conversation_response", "intent", "clarification_required", "unsupported"),
			"response":       nullableSchema(map[string]any{"$ref": "#/$defs/conversation_response"}),
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
		"conversation_response": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"content"},
			"properties": map[string]any{
				"content": boundedStringSchema(),
			},
		},
		"intent": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required": []string{
				"schema_version", "operation", "project_fingerprint", "source_fingerprint",
				"hatmax_version", "book_version", "archetype", "feature",
				"domain", "capabilities", "documentation", "documentation_targets",
				"exceptions", "application", "target", "initial_features",
			},
			"properties": map[string]any{
				"schema_version":      map[string]any{"type": "integer", "enum": []int{intent.CurrentSchemaVersion, intent.ApplicationSchemaVersion}},
				"operation":           enumSchema("create_application", "create_feature", "add_field", "add_validation", "document_feature"),
				"project_fingerprint": map[string]any{"type": "string"},
				"source_fingerprint":  map[string]any{"type": "string"},
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
				"documentation_targets": map[string]any{
					"type":     "array",
					"maxItems": intent.MaximumDocumentationTargets,
					"items":    map[string]any{"$ref": "#/$defs/documentation_target"},
				},
				"exceptions": map[string]any{
					"type":  "array",
					"items": map[string]any{"$ref": "#/$defs/exception"},
				},
				"application": nullableSchema(map[string]any{"$ref": "#/$defs/application"}),
				"target":      nullableSchema(map[string]any{"$ref": "#/$defs/application_target"}),
				"initial_features": map[string]any{
					"type":     "array",
					"maxItems": intent.MaximumInitialFeatures,
					"items":    map[string]any{"$ref": "#/$defs/initial_feature"},
				},
			},
		},
		"application": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"display_name", "project_slug", "module_path", "description", "niche"},
			"properties": map[string]any{
				"display_name": nonemptyStringSchema(),
				"project_slug": map[string]any{"type": "string"},
				"module_path":  map[string]any{"type": "string"},
				"description":  map[string]any{"type": "string"},
				"niche":        map[string]any{"type": "string"},
			},
		},
		"application_target": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"base", "directory"},
			"properties": map[string]any{
				"base":      enumSchema("session_directory"),
				"directory": map[string]any{"type": "string"},
			},
		},
		"initial_feature": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"feature", "domain"},
			"properties": map[string]any{
				"feature": nonemptyStringSchema(),
				"domain":  map[string]any{"$ref": "#/$defs/domain"},
			},
		},
		"documentation_target": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"quadrant", "subject", "reader_goal"},
			"properties": map[string]any{
				"quadrant":    enumSchema("tutorial", "how_to", "reference", "explanation"),
				"subject":     nonemptyStringSchema(),
				"reader_goal": boundedStringSchema(),
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
