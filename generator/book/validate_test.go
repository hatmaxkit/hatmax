// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package book

import (
	"testing"
	"testing/fstest"
)

func TestLoadRejectsInvalidEntryContracts(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		oldValue string
		newValue string
		wantCode string
	}{
		{
			name:     "duplicate capability ID",
			path:     "capabilities/runtime-validation.yaml",
			oldValue: "id: runtime_validation",
			newValue: "id: postgres_persistence",
			wantCode: "book_duplicate_id",
		},
		{
			name:     "invalid stable ID",
			path:     "capabilities/runtime-validation.yaml",
			oldValue: "id: runtime_validation",
			newValue: "id: RuntimeValidation",
			wantCode: "book_invalid_id",
		},
		{
			name:     "missing capability reference",
			path:     "capabilities/htmx-form.yaml",
			oldValue: "  - runtime_validation",
			newValue: "  - missing_validation",
			wantCode: "book_missing_reference",
		},
		{
			name:     "missing rule reference",
			path:     "capabilities/runtime-validation.yaml",
			oldValue: "  - hatmax.dependencies.use_owned_primitive",
			newValue: "  - hatmax.dependencies.missing",
			wantCode: "book_missing_reference",
		},
		{
			name:     "duplicate surface",
			path:     "capabilities/runtime-validation.yaml",
			oldValue: "  - handler\n  - tests",
			newValue: "  - handler\n  - handler",
			wantCode: "book_duplicate_value",
		},
		{
			name:     "incomplete dependency admission",
			path:     "capabilities/postgres-persistence.yaml",
			oldValue: "    purpose: Generate type-safe PostgreSQL query adapters.",
			newValue: "    purpose: ''",
			wantCode: "book_required_field",
		},
		{
			name:     "unknown operation obligation",
			path:     "archetypes/server-rendered-crud.yaml",
			oldValue: "  - id: add_validation\n    required_capabilities:\n      - runtime_validation\n    obligations:\n      - feature_package\n      - feature_handler",
			newValue: "  - id: add_validation\n    required_capabilities:\n      - runtime_validation\n    obligations:\n      - feature_package\n      - missing_boundary",
			wantCode: "book_missing_reference",
		},
		{
			name:     "duplicate operation",
			path:     "archetypes/server-rendered-crud.yaml",
			oldValue: "  - id: add_field",
			newValue: "  - id: create_feature",
			wantCode: "book_duplicate_value",
		},
		{
			name:     "invalid rule level",
			path:     "rules/validation-layered.yaml",
			oldValue: "level: required",
			newValue: "level: mandatory",
			wantCode: "book_invalid_level",
		},
		{
			name:     "invalid diagnostic code",
			path:     "rules/validation-layered.yaml",
			oldValue: "  - HMGEN-VALIDATION-OWNER",
			newValue: "  - invalid-diagnostic",
			wantCode: "book_invalid_diagnostic",
		},
		{
			name:     "missing example",
			path:     "rules/validation-layered.yaml",
			oldValue: "examples/negative/alternate-validation.go.txt",
			newValue: "examples/negative/missing.go.txt",
			wantCode: "book_missing_reference",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := defaultBookFS(t)
			replaceBookText(t, source, test.path, test.oldValue, test.newValue)

			_, err := Load(source)
			requireValidationCode(t, err, test.wantCode)
		})
	}
}

func TestLoadRejectsCapabilityCycles(t *testing.T) {
	source := defaultBookFS(t)
	replaceBookText(
		t,
		source,
		"capabilities/runtime-validation.yaml",
		"intent: Validate user and domain data through Hatmax validation primitives and durable invariants.\n",
		"intent: Validate user and domain data through Hatmax validation primitives and durable invariants.\nrequires:\n  - htmx_form\n",
	)

	_, err := Load(source)
	requireValidationCode(t, err, "book_capability_cycle")
}

func TestLoadRejectsUnreferencedEntries(t *testing.T) {
	source := defaultBookFS(t)
	source["rules/unreferenced.yaml"] = &fstest.MapFile{Data: []byte("id: hatmax.unreferenced\n")}

	_, err := Load(source)
	requireValidationCode(t, err, "book_unreferenced_entry")
}
