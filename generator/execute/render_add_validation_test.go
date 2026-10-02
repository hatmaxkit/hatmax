// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"context"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

func TestRenderAddValidationLayersDurableRule(t *testing.T) {
	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddValidation, intent.Domain{
		Validation: &intent.ValidationRule{
			Field: "number", Kind: "min_length", Value: "3", Message: "Enter at least three characters.", Scope: intent.ValidationDurable,
		},
	}, []string{"postgres_persistence", "runtime_validation"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := RenderAddValidation(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderAddValidation() error = %v", err)
	}

	repeated, err := RenderAddValidation(value, manifest, inventory)
	if err != nil {
		t.Fatalf("second RenderAddValidation() error = %v", err)
	}

	assertDeterministicMutations(t, mutations, repeated)
	assertCanonicalRenderedGo(t, manifest, mutations)

	if len(mutations) != 6 {
		t.Fatalf("rendered mutations = %d, want 6", len(mutations))
	}

	assertRenderedContains(t, mutations, "add_validation.migration", "CHECK (char_length(number) >= 3)", "DROP CONSTRAINT invoices_number_min_length")
	assertRenderedContains(t, mutations, "add_validation.model", "len(value.Number) < 3", "Enter at least three characters.")
	assertRenderedContains(t, mutations, "add_validation.form_template", `minlength="3"`)
	assertRenderedContains(t, mutations, "add_validation.model_tests", "TestInvoiceRejectsNumberValidation", `strings.Repeat("x", 3-1)`)
	assertRenderedContains(t, mutations, "add_validation.handler_tests", "TestHandlerMapsValidationErrors", "validation.NewSingleError")

	workspace, err := OpenWorkspace(context.Background(), manifest, value, inventory)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	for _, mutation := range mutations {
		_, err = workspace.Stage(mutation)
		if err != nil {
			t.Fatalf("Stage(%q) error = %v", mutation.EditID, err)
		}
	}

	_, err = workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
}

func TestRenderAddValidationKeepsClientOnlyRuleOutOfDomainAndSchema(t *testing.T) {
	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddValidation, intent.Domain{
		Validation: &intent.ValidationRule{
			Field: "number", Kind: "pattern", Value: "^[A-Z]+$", Scope: intent.ValidationClientOnly,
		},
	}, []string{"postgres_persistence", "runtime_validation"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := RenderAddValidation(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderAddValidation() error = %v", err)
	}

	if len(mutations) != 4 {
		t.Fatalf("rendered mutations = %d, want 4", len(mutations))
	}

	assertRenderedContains(t, mutations, "add_validation.form_template", `pattern="^[A-Z]+$"`)

	for _, mutation := range mutations {
		if mutation.EditID == "add_validation.model" || mutation.EditID == "add_validation.migration" {
			t.Errorf("client-only validation rendered durable mutation %q", mutation.EditID)
		}
	}
}

func TestRenderAddValidationRejectsMissingField(t *testing.T) {
	root := generatedInvoiceProject(t)
	value, inventory, selectedBook := executionPlanForFeature(t, root, "invoice", intent.OperationAddValidation, intent.Domain{
		Validation: &intent.ValidationRule{
			Field: "missing", Kind: "required", Scope: intent.ValidationDurable,
		},
	}, []string{"postgres_persistence", "runtime_validation"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	_, err = RenderAddValidation(value, manifest, inventory)
	requireExecutionCode(t, err, "execution_semantic_conflict")
}
