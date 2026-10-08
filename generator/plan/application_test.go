// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package plan

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

func TestExpandApplicationPlansScaffoldAndInitialFeatures(t *testing.T) {
	admission, contextValue := admittedApplicationExpansion(t)

	result, err := Expand(admission, contextValue)
	if err != nil {
		t.Fatalf("Expand() error = %v", err)
	}

	if result.SchemaVersion != ApplicationSchemaVersion || result.SourceFingerprint != contextValue.Fingerprint.Value {
		t.Errorf("application plan identity = schema %d fingerprint %q", result.SchemaVersion, result.SourceFingerprint)
	}

	if result.Application == nil || result.Application.ProjectSlug != "real-estate" || result.Target == nil || result.Target.Path != contextValue.Target.Target {
		t.Errorf("application target = application %#v target %#v", result.Application, result.Target)
	}

	if len(result.Units) != 3 || result.Units[0].ID != "application" || result.Units[1].ID != "feature.catalog" || result.Units[2].ID != "feature.invoice" {
		t.Fatalf("Units = %#v, want application, catalog, and invoice", result.Units)
	}

	requireFileEffect(t, result.Units[0].Effects, "main.go", FileEffectCreate)
	requireNoFileEffect(t, result.Units[0].Effects, "assets/migration/postgres/0001-catalog.sql")
	requireFileEffect(t, result.Units[1].Effects, "assets/migration/postgres/0001-catalog.sql", FileEffectCreate)
	requireFileEffect(t, result.Units[0].Effects, "sqlc.yaml", FileEffectCreate)
	requireFileEffect(t, result.Units[1].Effects, "sqlc.yaml", FileEffectUpdate)
	requireFileEffect(t, result.Units[2].Effects, "assets/migration/postgres/0002-invoice.sql", FileEffectCreate)
	requireFileEffect(t, result.Units[2].Effects, "sqlc.yaml", FileEffectUpdate)
	requireNoFileEffect(t, result.AllowedEffects.Files, "README.md")

	err = Validate(result)
	if err != nil {
		t.Errorf("expanded application plan is invalid: %v", err)
	}
}

func TestApplicationExpansionIsDeterministic(t *testing.T) {
	admission, contextValue := admittedApplicationExpansion(t)

	first, err := Expand(admission, contextValue)
	if err != nil {
		t.Fatalf("Expand(first) error = %v", err)
	}

	second, err := Expand(admission, contextValue)
	if err != nil {
		t.Fatalf("Expand(second) error = %v", err)
	}

	firstYAML, err := MarshalYAML(first)
	if err != nil {
		t.Fatalf("MarshalYAML(first) error = %v", err)
	}

	secondYAML, err := MarshalYAML(second)
	if err != nil {
		t.Fatalf("MarshalYAML(second) error = %v", err)
	}

	if first.Digest != second.Digest || !bytes.Equal(firstYAML, secondYAML) {
		t.Error("equivalent application inputs produced different sealed plans")
	}
}

func TestApplicationPlanDigestCoversIdentityAndUnits(t *testing.T) {
	admission, contextValue := admittedApplicationExpansion(t)

	result, err := Expand(admission, contextValue)
	if err != nil {
		t.Fatalf("Expand() error = %v", err)
	}

	result.Application.DisplayName = "Changed"
	requirePlanCode(t, VerifyDigest(result), "plan_digest_mismatch")

	result, err = Expand(admission, contextValue)
	if err != nil {
		t.Fatalf("second Expand() error = %v", err)
	}

	result.Units[1].Domain.Label = "Changed catalog"
	requirePlanCode(t, VerifyDigest(result), "plan_digest_mismatch")
}

func TestExpandApplicationRequiresExactTargetPathInventory(t *testing.T) {
	admission, contextValue := admittedApplicationExpansion(t)
	contextValue.Target.PlannedPaths = contextValue.Target.PlannedPaths[1:]

	fingerprint, err := contextValue.Target.Fingerprint(admission.Intent.BookVersion)
	if err != nil {
		t.Fatalf("Target.Fingerprint() error = %v", err)
	}

	contextValue.Fingerprint = fingerprint
	admission.Intent.SourceFingerprint = fingerprint.Value

	_, err = Expand(admission, contextValue)
	requirePlanCode(t, err, "plan_target_inventory_incomplete")
}

func TestApplicationPlanDetectsTargetDrift(t *testing.T) {
	admission, contextValue := admittedApplicationExpansion(t)

	result, err := Expand(admission, contextValue)
	if err != nil {
		t.Fatalf("Expand() error = %v", err)
	}

	err = os.MkdirAll(contextValue.Target.Target, 0o755)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	err = os.WriteFile(filepath.Join(contextValue.Target.Target, "note.txt"), []byte("changed\n"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	current, err := project.InspectTarget(context.Background(), project.TargetRequest{
		Parent: contextValue.Target.Parent, Target: contextValue.Target.Target,
		PlannedPaths: contextValue.Target.PlannedPaths, Book: contextValue.Book,
	})
	if err != nil {
		t.Fatalf("InspectTarget() error = %v", err)
	}

	currentFingerprint, err := current.Fingerprint(result.BookVersion)
	if err != nil {
		t.Fatalf("current Fingerprint() error = %v", err)
	}

	freshness, err := CheckFingerprint(result, currentFingerprint)
	if err != nil {
		t.Fatalf("CheckFingerprint() error = %v", err)
	}

	if !freshness.Stale || len(freshness.Diagnostics) != 1 || freshness.Diagnostics[0].Field != "source_fingerprint" {
		t.Errorf("Freshness = %#v, want source target drift", freshness)
	}
}

func TestExpandApplicationCarriesPreservedRepositoryFiles(t *testing.T) {
	admission, contextValue := admittedApplicationExpansionWithTarget(t, func(target string) {
		err := os.MkdirAll(target, 0o755)
		if err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		command := exec.Command("git", "init", "--quiet", target)

		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git init error = %v: %s", err, output)
		}

		err = os.WriteFile(filepath.Join(target, "NOTICE"), []byte("preserve\n"), 0o600)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	})

	result, err := Expand(admission, contextValue)
	if err != nil {
		t.Fatalf("Expand() error = %v", err)
	}

	if result.Target.Admission != project.TargetPreservable || !hasPreservedTargetPath(result.Target.Preserved, "NOTICE") {
		t.Errorf("Target = %#v, want preserved NOTICE", result.Target)
	}

	requireNoFileEffect(t, result.AllowedEffects.Files, "NOTICE")
}

func admittedApplicationExpansion(t *testing.T) (intent.Result, ExpansionContext) {
	t.Helper()

	return admittedApplicationExpansionWithTarget(t, nil)
}

func admittedApplicationExpansionWithTarget(
	t *testing.T,
	prepare func(string),
) (intent.Result, ExpansionContext) {
	t.Helper()

	selectedBook, err := book.LoadRelease(2)
	if err != nil {
		t.Fatalf("book.LoadRelease(2) error = %v", err)
	}

	value := intent.Intent{
		SchemaVersion: intent.ApplicationSchemaVersion,
		Operation:     intent.OperationCreateApplication,
		HatmaxVersion: "0.5.0", BookVersion: 2,
		Archetype:    "server_rendered_hatmax_application",
		Capabilities: []string{}, Documentation: intent.DocumentationNotRequested,
		Application: &intent.ApplicationIdentity{
			DisplayName: "Real Estate", ProjectSlug: "real-estate",
			ModulePath: "example.com/alex/real-estate",
		},
		Target: &intent.ApplicationTarget{Base: "session_directory", Directory: "real-estate"},
		InitialFeatures: []intent.InitialFeature{
			{Feature: "catalog", Domain: applicationFeatureDomain("Catalog", "/catalogs")},
			{Feature: "invoice", Domain: applicationFeatureDomain("Invoice", "/invoices")},
		},
	}

	plannedPaths, err := ApplicationTargetPaths(value, selectedBook)
	if err != nil {
		t.Fatalf("ApplicationTargetPaths() error = %v", err)
	}

	parent := t.TempDir()

	targetPath := filepath.Join(parent, "real-estate")
	if prepare != nil {
		prepare(targetPath)
	}

	target, err := project.InspectTarget(context.Background(), project.TargetRequest{
		Parent: parent, Target: targetPath,
		PlannedPaths: plannedPaths, Book: selectedBook,
	})
	if err != nil {
		t.Fatalf("project.InspectTarget() error = %v", err)
	}

	fingerprint, err := target.Fingerprint(selectedBook.Manifest().BookVersion)
	if err != nil {
		t.Fatalf("Target.Fingerprint() error = %v", err)
	}

	value.SourceFingerprint = fingerprint.Value

	admission := intent.Validate(value, intent.ValidationContext{
		Target: &target, Fingerprint: fingerprint, Book: selectedBook,
	})
	if !admission.Admitted() {
		t.Fatalf("intent.Validate() status = %q diagnostics %#v clarifications %#v", admission.Status, admission.Diagnostics, admission.Clarifications)
	}

	return admission, ExpansionContext{Book: selectedBook, Target: &target, Fingerprint: fingerprint}
}

func hasPreservedTargetPath(values []project.TargetEntry, expected string) bool {
	for _, value := range values {
		if value.Path == expected {
			return true
		}
	}

	return false
}

func applicationFeatureDomain(entity, route string) intent.Domain {
	return intent.Domain{
		Entity: entity, Route: route, Label: entity + "s",
		Fields: []intent.Field{{Name: "name", Label: "Name", Type: "string", Required: true}},
	}
}

func requireFileEffect(t *testing.T, values []FileEffect, effectPath string, effect FileEffectKind) {
	t.Helper()

	for _, value := range values {
		if value.Path == effectPath && value.Effect == effect {
			return
		}
	}

	t.Errorf("file effects = %#v, want %s %s", values, effect, effectPath)
}

func requireNoFileEffect(t *testing.T, values []FileEffect, effectPath string) {
	t.Helper()

	for _, value := range values {
		if value.Path == effectPath {
			t.Errorf("file effects unexpectedly contain %q", effectPath)

			return
		}
	}
}
