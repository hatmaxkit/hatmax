// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import "testing"

func TestValidateAcceptsCompleteManifest(t *testing.T) {
	err := Validate(validTestManifest())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsInvalidManifest(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
		code   string
	}{
		{name: "schema", mutate: func(value *Manifest) { value.SchemaVersion++ }, code: "execution_schema_unsupported"},
		{name: "plan digest", mutate: func(value *Manifest) { value.PlanDigest = "invalid" }, code: "execution_identity_invalid"},
		{name: "intent", mutate: func(value *Manifest) { value.Intent = "remove_feature" }, code: "execution_intent_invalid"},
		{name: "feature", mutate: func(value *Manifest) { value.Feature = "Invoice" }, code: "execution_feature_invalid"},
		{name: "surface", mutate: func(value *Manifest) { value.Edits[0].Surface = "wiring" }, code: "execution_surface_undeclared"},
		{name: "escaping target", mutate: func(value *Manifest) { value.Edits[0].Target = "../model.go" }, code: "execution_target_invalid"},
		{name: "overlapping target", mutate: func(value *Manifest) { value.Edits[1].Target = value.Edits[0].Target }, code: "execution_target_overlap"},
		{name: "forward dependency", mutate: func(value *Manifest) { value.Edits[0].DependsOn = []string{value.Edits[1].ID} }, code: "execution_dependency_order_invalid"},
		{name: "missing precondition", mutate: func(value *Manifest) { value.Edits[0].Preconditions = nil }, code: "execution_required_field"},
		{name: "invalid slot", mutate: func(value *Manifest) { value.Edits[0].Slots[0].Source = "" }, code: "execution_slot_invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validTestManifest()
			test.mutate(&value)

			requireExecutionCode(t, Validate(value), test.code)
		})
	}
}
