// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package book

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadRejectsUnreadableSources(t *testing.T) {
	_, err := Load(nil)
	if err == nil || !strings.Contains(err.Error(), "source is nil") {
		t.Fatalf("Load(nil) error = %v, want nil source error", err)
	}

	_, err = Load(fstest.MapFS{})
	if err == nil || !strings.Contains(err.Error(), "read manifest.yaml") {
		t.Fatalf("Load(empty FS) error = %v, want missing manifest error", err)
	}
}

func TestLoadUsesStrictSingleDocumentYAML(t *testing.T) {
	tests := []struct {
		name   string
		append string
		want   string
	}{
		{
			name:   "unknown field",
			append: "unknown_field: true\n",
			want:   "field unknown_field not found",
		},
		{
			name:   "multiple documents",
			append: "---\n{}\n",
			want:   "multiple YAML documents",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := defaultBookFS(t)
			source[manifestPath].Data = append(source[manifestPath].Data, []byte(test.append)...)

			_, err := Load(source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Load() error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestLoadRejectsInvalidManifestContracts(t *testing.T) {
	tests := []struct {
		name     string
		oldValue string
		newValue string
		wantCode string
	}{
		{
			name:     "unsupported schema",
			oldValue: "schema_version: 1",
			newValue: "schema_version: 2",
			wantCode: "book_schema_unsupported",
		},
		{
			name:     "nonpositive Book version",
			oldValue: "book_version: 1",
			newValue: "book_version: 0",
			wantCode: "book_invalid_version",
		},
		{
			name:     "malformed Hatmax version",
			oldValue: "minimum: 0.4.0",
			newValue: "minimum: 0.4",
			wantCode: "book_invalid_version",
		},
		{
			name:     "empty Hatmax version range",
			oldValue: "maximum_exclusive: 0.6.0",
			newValue: "maximum_exclusive: 0.4.0",
			wantCode: "book_invalid_version",
		},
		{
			name:     "duplicate manifest path",
			oldValue: "  - capabilities/runtime-validation.yaml",
			newValue: "  - capabilities/runtime-validation.yaml\n  - capabilities/runtime-validation.yaml",
			wantCode: "book_duplicate_path",
		},
		{
			name:     "invalid manifest path",
			oldValue: "capabilities/runtime-validation.yaml",
			newValue: "rules/runtime-validation.yaml",
			wantCode: "book_invalid_path",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := defaultBookFS(t)
			replaceBookText(t, source, manifestPath, test.oldValue, test.newValue)

			_, err := Load(source)
			requireValidationCode(t, err, test.wantCode)
		})
	}
}
