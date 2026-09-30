// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package intent

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/project"
)

func TestDecodeApplicationIntentNormalizesIdentityAndFeatures(t *testing.T) {
	value := loadIntentFixture(t, "valid", "create-application.yaml")

	if value.SchemaVersion != ApplicationSchemaVersion || value.Operation != OperationCreateApplication {
		t.Fatalf("intent identity = schema %d operation %q", value.SchemaVersion, value.Operation)
	}

	if value.Application == nil || value.Application.ProjectSlug != "real-estate" {
		t.Fatalf("Application = %#v, want real-estate identity", value.Application)
	}

	if value.Target == nil || value.Target.Directory != "real-estate" {
		t.Fatalf("Target = %#v, want real-estate directory", value.Target)
	}

	if len(value.InitialFeatures) != 1 || value.InitialFeatures[0].Feature != "invoice" {
		t.Fatalf("InitialFeatures = %#v, want normalized invoice", value.InitialFeatures)
	}

	field := value.InitialFeatures[0].Domain.Fields[0]
	if field.Name != "number" || field.Label != "Number" {
		t.Errorf("initial field = %#v, want normalized number", field)
	}
}

func TestValidateAdmitsApplicationIntent(t *testing.T) {
	value := loadIntentFixture(t, "valid", "create-application.yaml")
	contextValue := applicationValidationContext(t, "real-estate")
	value.SourceFingerprint = contextValue.Fingerprint.Value

	result := Validate(value, contextValue)
	if !result.Admitted() {
		t.Fatalf("Validate() status = %q, diagnostics = %#v, clarifications = %#v", result.Status, result.Diagnostics, result.Clarifications)
	}

	if result.Intent.Application.ModulePath != "example.com/alex/real-estate" {
		t.Errorf("ModulePath = %q, want explicit module path", result.Intent.Application.ModulePath)
	}

	if len(result.Selection.Capabilities) != 1 || result.Selection.Capabilities[0].ID != "postgres_persistence" {
		t.Errorf("Selection.Capabilities = %#v, want canonical Postgres capability", result.Selection.Capabilities)
	}
}

func TestValidateApplicationDerivesModuleFromRemoteEvidence(t *testing.T) {
	value := loadIntentFixture(t, "valid", "create-application.yaml")
	value.Application.ModulePath = ""

	contextValue := applicationValidationContext(t, "real-estate")
	contextValue.Target.RemoteModulePath = "forge.example/alex/real-estate"
	value.SourceFingerprint = contextValue.Fingerprint.Value

	result := Validate(value, contextValue)
	if !result.Admitted() {
		t.Fatalf("Validate() status = %q, diagnostics = %#v, clarifications = %#v", result.Status, result.Diagnostics, result.Clarifications)
	}

	if result.Intent.Application.ModulePath != "forge.example/alex/real-estate" {
		t.Errorf("ModulePath = %q, want derived remote module", result.Intent.Application.ModulePath)
	}
}

func TestValidateApplicationRequestsOnlyRequiredIdentity(t *testing.T) {
	value := loadIntentFixture(t, "valid", "create-application.yaml")
	value.Application.DisplayName = ""
	value.Application.ProjectSlug = ""
	value.Application.ModulePath = ""
	value.Target.Directory = ""

	contextValue := applicationValidationContext(t, "pending")
	value.SourceFingerprint = contextValue.Fingerprint.Value

	result := Validate(value, contextValue)
	if result.Status != StatusClarificationRequired {
		t.Fatalf("Validate() status = %q, diagnostics = %#v", result.Status, result.Diagnostics)
	}

	for _, field := range []string{"application.display_name", "application.module_path"} {
		if !hasClarification(result, field) {
			t.Errorf("Clarifications = %#v, want %q", result.Clarifications, field)
		}
	}
}

func TestValidateApplicationRejectsTargetState(t *testing.T) {
	value := loadIntentFixture(t, "valid", "create-application.yaml")
	contextValue := applicationValidationContext(t, "real-estate")
	contextValue.Target.Admission = project.TargetIncompatible
	value.SourceFingerprint = contextValue.Fingerprint.Value

	result := Validate(value, contextValue)
	if result.Status != StatusIncompatible || !hasDiagnostic(result, "HMGEN-TARGET-INCOMPATIBLE") {
		t.Fatalf("Validate() = status %q diagnostics %#v, want incompatible target", result.Status, result.Diagnostics)
	}
}

func TestApplicationSchemaIsVersionBound(t *testing.T) {
	application := loadIntentFixture(t, "valid", "create-application.yaml")
	application.SchemaVersion = CurrentSchemaVersion
	application.ProjectFingerprint = application.SourceFingerprint
	application.SourceFingerprint = ""
	requireSchemaCode(t, ValidateSchema(application), "intent_operation_invalid")

	feature := loadIntentFixture(t, "valid", "create-feature.yaml")
	feature.SchemaVersion = ApplicationSchemaVersion
	feature.SourceFingerprint = feature.ProjectFingerprint

	feature.ProjectFingerprint = ""

	err := ValidateSchema(feature)
	if err != nil {
		t.Errorf("ValidateSchema(schema 3 feature) error = %v", err)
	}
}

func TestApplicationSchemaBoundsInitialFeatures(t *testing.T) {
	value := loadIntentFixture(t, "valid", "create-application.yaml")
	value.InitialFeatures = make([]InitialFeature, MaximumInitialFeatures+1)

	requireSchemaCode(t, ValidateSchema(value), "intent_initial_features_limit")
}

func applicationValidationContext(t *testing.T, directory string) ValidationContext {
	t.Helper()

	selectedBook := loadApplicationBook(t)
	parent := t.TempDir()
	targetPath := filepath.Join(parent, directory)

	target, err := project.InspectTarget(context.Background(), project.TargetRequest{
		Parent: parent,
		Target: targetPath,
		PlannedPaths: []string{
			"go.mod",
			"main.go",
			"internal/application/application.go",
		},
		Book: selectedBook,
	})
	if err != nil {
		t.Fatalf("project.InspectTarget() error = %v", err)
	}

	fingerprint, err := target.Fingerprint(selectedBook.Manifest().BookVersion)
	if err != nil {
		t.Fatalf("Target.Fingerprint() error = %v", err)
	}

	return ValidationContext{Target: &target, Fingerprint: fingerprint, Book: selectedBook}
}

func loadApplicationBook(t *testing.T) *book.Book {
	t.Helper()

	source := fstest.MapFS{
		"manifest.yaml": &fstest.MapFile{Data: []byte(`schema_version: 1
book_version: 2
hatmax:
  minimum: 0.5.0
  maximum_exclusive: 0.6.0
capabilities:
  - capabilities/postgres-persistence.yaml
archetypes:
  - archetypes/server-rendered-hatmax-application.yaml
rules:
  - rules/application-canonical.yaml
`)},
		"capabilities/postgres-persistence.yaml": &fstest.MapFile{Data: []byte(`id: postgres_persistence
intent: Persist application data in PostgreSQL.
surfaces:
  - database
obligations:
  - id: database
    surfaces:
      - database
    rules:
      - hatmax.application.canonical
rules:
  - hatmax.application.canonical
`)},
		"archetypes/server-rendered-hatmax-application.yaml": &fstest.MapFile{Data: []byte(`id: server_rendered_hatmax_application
operations:
  - id: create_application
    required_capabilities:
      - postgres_persistence
    obligations:
      - application_foundation
required_capabilities:
  - postgres_persistence
surfaces:
  - application
  - database
obligations:
  - id: application_foundation
    surfaces:
      - application
    rules:
      - hatmax.application.canonical
implementation_slots:
  - product identity
rules:
  - hatmax.application.canonical
`)},
		"rules/application-canonical.yaml": &fstest.MapFile{Data: []byte(`id: hatmax.application.canonical
level: required
title: Canonical application
rationale: Keep application creation deterministic.
applies_when:
  - the application archetype is selected
requires:
  - canonical application structure
diagnostics:
  - HMGEN-APPLICATION-CANONICAL
capabilities:
  - postgres_persistence
archetypes:
  - server_rendered_hatmax_application
`)},
	}

	loaded, err := book.Load(source)
	if err != nil {
		t.Fatalf("book.Load() error = %v\n%s", err, formatBookFixture(source))
	}

	return loaded
}

func formatBookFixture(source fstest.MapFS) string {
	paths := make([]string, 0, len(source))
	for path := range source {
		paths = append(paths, path)
	}

	return fmt.Sprintf("book fixture paths: %s", strings.Join(paths, ", "))
}
