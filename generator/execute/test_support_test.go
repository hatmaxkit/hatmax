package execute

import (
	"errors"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
)

const testDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

func validTestManifest() Manifest {
	return Manifest{
		SchemaVersion:      CurrentSchemaVersion,
		PlanDigest:         testDigest,
		ProjectFingerprint: testDigest,
		Intent:             intent.OperationCreateFeature,
		Feature:            "invoice",
		AllowedSurfaces:    []string{"model", "tests"},
		Edits: []Edit{
			{
				ID:        "feature.model",
				Operation: "archetype.server_rendered_crud.feature_package",
				Owner:     plan.Owner{Kind: plan.OwnerArchetype, ID: "server_rendered_crud"},
				Kind:      EditCreateFile,
				Surface:   "model",
				Target:    "internal/feat/invoice/model.go",
				Recipe:    "server_rendered_crud.model",
				Rules:     []string{"hatmax.feature.cohesive_package"},
				Preconditions: []Condition{
					{Kind: ConditionPathAbsent},
				},
				Postconditions: []Condition{
					{Kind: ConditionPathPresent},
					{Kind: ConditionGoParses},
				},
				Slots: []ImplementationSlot{
					{Name: "domain_fields", Source: "plan.domain.fields"},
				},
			},
			{
				ID:        "feature.model_tests",
				Operation: "archetype.server_rendered_crud.feature_tests",
				Owner:     plan.Owner{Kind: plan.OwnerArchetype, ID: "server_rendered_crud"},
				Kind:      EditCreateFile,
				Surface:   "tests",
				Target:    "internal/feat/invoice/model_test.go",
				Recipe:    "server_rendered_crud.model_tests",
				Rules:     []string{"hatmax.testing.required_boundaries"},
				DependsOn: []string{"feature.model"},
				Preconditions: []Condition{
					{Kind: ConditionPathAbsent},
				},
				Postconditions: []Condition{
					{Kind: ConditionPathPresent},
					{Kind: ConditionGoParses},
				},
			},
		},
		Commands: []Command{},
	}
}

func requireExecutionCode(t *testing.T, err error, code string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want execution code %q", code)
	}

	var executionErr Error
	if !errors.As(err, &executionErr) {
		t.Fatalf("error = %v, want execute.Error", err)
	}

	if executionErr.Code != code {
		t.Fatalf("execution code = %q, want %q", executionErr.Code, code)
	}
}
