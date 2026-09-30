// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package intent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/project"
)

const testFingerprint = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

func loadIntentFixture(t *testing.T, path ...string) Intent {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(append([]string{"testdata"}, path...)...))
	if err != nil {
		t.Fatalf("read intent fixture: %v", err)
	}

	value, err := DecodeYAML(data)
	if err != nil {
		t.Fatalf("DecodeYAML() error = %v", err)
	}

	return value
}

func validationContext(t *testing.T) ValidationContext {
	t.Helper()

	selectedBook, err := book.LoadDefault()
	if err != nil {
		t.Fatalf("book.LoadDefault() error = %v", err)
	}

	return ValidationContext{
		Inventory: project.Inventory{
			Module: project.Module{
				Hatmax: project.HatmaxModule{
					Version: "v0.4.0",
					Source:  project.HatmaxSourceModule,
				},
			},
			Layout: project.Layout{
				Features: []string{"internal/feat/property"},
			},
		},
		Fingerprint: project.Fingerprint{
			Value:       testFingerprint,
			BookVersion: 1,
		},
		Book: selectedBook,
	}
}

func requireSchemaCode(t *testing.T, err error, code string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want schema code %q", code)
	}

	var schemaErr SchemaError
	if !errors.As(err, &schemaErr) {
		t.Fatalf("error = %v, want SchemaError", err)
	}

	if schemaErr.Code != code {
		t.Fatalf("schema code = %q, want %q", schemaErr.Code, code)
	}
}

func hasDiagnostic(result Result, code string) bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}

	return false
}

func hasClarification(result Result, field string) bool {
	for _, clarification := range result.Clarifications {
		if clarification.Field == field {
			return true
		}
	}

	return false
}
