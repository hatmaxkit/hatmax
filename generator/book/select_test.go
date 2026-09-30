// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package book

import "testing"

func TestSelectExpandsPrerequisitesInManifestOrder(t *testing.T) {
	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	selection, err := loaded.Select("server_rendered_crud", []string{"htmx_form"})
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}

	wantCapabilities := []string{"postgres_persistence", "runtime_validation", "htmx_form"}
	if len(selection.Capabilities) != len(wantCapabilities) {
		t.Fatalf("len(Selection.Capabilities) = %d, want %d", len(selection.Capabilities), len(wantCapabilities))
	}

	for index, want := range wantCapabilities {
		if selection.Capabilities[index].ID != want {
			t.Errorf("Selection.Capabilities[%d].ID = %q, want %q", index, selection.Capabilities[index].ID, want)
		}
	}

	bookRules := loaded.Rules()

	wantRules := make([]string, 0, len(bookRules))
	for _, rule := range bookRules {
		wantRules = append(wantRules, rule.ID)
	}

	if len(selection.Rules) != len(wantRules) {
		t.Fatalf("len(Selection.Rules) = %d, want %d", len(selection.Rules), len(wantRules))
	}

	for index, want := range wantRules {
		if selection.Rules[index].ID != want {
			t.Errorf("Selection.Rules[%d].ID = %q, want %q", index, selection.Rules[index].ID, want)
		}
	}
}

func TestSelectRejectsUnsupportedChoices(t *testing.T) {
	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	_, err = loaded.Select("missing", nil)
	requireValidationCode(t, err, "book_missing_reference")

	_, err = loaded.Select("server_rendered_crud", []string{"missing"})
	requireValidationCode(t, err, "book_capability_not_allowed")
}

func TestSelectRejectsCapabilityConflicts(t *testing.T) {
	source := defaultBookFS(t)
	replaceBookText(
		t,
		source,
		"capabilities/runtime-validation.yaml",
		"intent: Validate user and domain data through Hatmax validation primitives and durable invariants.\n",
		"intent: Validate user and domain data through Hatmax validation primitives and durable invariants.\nincompatible_with:\n  - htmx_form\n",
	)

	loaded, err := Load(source)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	_, err = loaded.Select("server_rendered_crud", []string{"htmx_form"})
	requireValidationCode(t, err, "book_capability_conflict")
}

func TestBookAccessorsReturnDetachedValues(t *testing.T) {
	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	manifest := loaded.Manifest()
	manifest.Capabilities[0] = "changed"

	capabilities := loaded.Capabilities()
	if capabilities[0].ID != "postgres_persistence" || capabilities[2].ID != "htmx_form" {
		t.Fatalf("Capabilities() did not preserve manifest order: %v", []string{capabilities[0].ID, capabilities[1].ID, capabilities[2].ID})
	}

	capabilities[0].Obligations[0].Rules[0] = "changed"

	archetypes := loaded.Archetypes()
	if archetypes[0].ID != "server_rendered_crud" {
		t.Fatalf("Archetypes()[0].ID = %q, want server_rendered_crud", archetypes[0].ID)
	}

	archetypes[0].Operations[0].Obligations[0] = "changed"

	capability, _ := loaded.Capability("postgres_persistence")
	capability.Obligations[0].Rules[0] = "changed"

	archetype, _ := loaded.Archetype("server_rendered_crud")
	archetype.Operations[0].Obligations[0] = "changed"

	rule, _ := loaded.Rule("hatmax.validation.layered")
	rule.Diagnostics[0] = "changed"

	selection, selectErr := loaded.Select("server_rendered_crud", []string{"runtime_validation"})
	if selectErr != nil {
		t.Fatalf("Select() error = %v", selectErr)
	}

	selection.Capabilities[0].Obligations[0].Rules[0] = "changed"
	selection.Rules[0].Diagnostics[0] = "changed"

	freshManifest := loaded.Manifest()
	freshCapability, _ := loaded.Capability("postgres_persistence")
	freshArchetype, _ := loaded.Archetype("server_rendered_crud")
	freshRule, _ := loaded.Rule("hatmax.validation.layered")

	if freshManifest.Capabilities[0] != "capabilities/postgres-persistence.yaml" {
		t.Error("Manifest() exposed mutable Book state")
	}

	if freshCapability.Obligations[0].Rules[0] != "hatmax.persistence.postgres_sqlc" {
		t.Error("capability accessors exposed mutable Book state")
	}

	if freshArchetype.Operations[0].Obligations[0] != "feature_package" {
		t.Error("archetype accessors exposed mutable Book state")
	}

	if freshRule.Diagnostics[0] != "HMGEN-VALIDATION-OWNER" {
		t.Error("Rule() exposed mutable Book state")
	}

	freshSelection, selectErr := loaded.Select("server_rendered_crud", []string{"runtime_validation"})
	if selectErr != nil {
		t.Fatalf("second Select() error = %v", selectErr)
	}

	if freshSelection.Capabilities[0].Obligations[0].Rules[0] == "changed" {
		t.Error("Select() exposed mutable capability state")
	}

	if freshSelection.Rules[0].Diagnostics[0] == "changed" {
		t.Error("Select() exposed mutable rule state")
	}
}
