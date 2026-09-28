package plan

import (
	"errors"
	"testing"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

const testFingerprint = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

func validTestPlan() Plan {
	featurePackage := "archetype.server_rendered_crud.feature_package"

	return Plan{
		SchemaVersion:      CurrentSchemaVersion,
		Intent:             intent.OperationCreateFeature,
		Archetype:          "server_rendered_crud",
		Feature:            "invoice",
		Capabilities:       []string{"postgres_persistence"},
		AffectedSurfaces:   []string{"model", "tests"},
		Documentation:      intent.DocumentationNotRequested,
		HatmaxVersion:      "v0.4.0",
		BookVersion:        1,
		ProjectFingerprint: testFingerprint,
		Rules: []RuleRef{
			{ID: "hatmax.feature.cohesive_package", Level: book.LevelRequired},
			{ID: "hatmax.testing.required_boundaries", Level: book.LevelRequired},
		},
		Operations: []Operation{
			{
				ID:         featurePackage,
				Owner:      Owner{Kind: OwnerArchetype, ID: "server_rendered_crud"},
				Obligation: "feature_package",
				Surfaces:   []string{"model"},
				Rules:      []string{"hatmax.feature.cohesive_package"},
			},
			{
				ID:         "archetype.server_rendered_crud.feature_tests",
				Owner:      Owner{Kind: OwnerArchetype, ID: "server_rendered_crud"},
				Obligation: "feature_tests",
				Surfaces:   []string{"tests"},
				Rules:      []string{"hatmax.testing.required_boundaries"},
				DependsOn:  []string{featurePackage},
			},
		},
		Preconditions: []Precondition{
			{ID: "project_fingerprint", Kind: PreconditionProjectFingerprint, Expected: testFingerprint},
			{ID: "hatmax_version", Kind: PreconditionHatmaxVersion, Expected: "v0.4.0"},
			{ID: "book_version", Kind: PreconditionBookVersion, Expected: "1"},
		},
		ExpectedObservations: []project.Observation{},
		AllowedEffects: AllowedEffects{
			Surfaces:     []string{"model", "tests"},
			Dependencies: []DependencyEffect{},
		},
		Validation: []ValidationObligation{
			{
				Rule:        "hatmax.feature.cohesive_package",
				Level:       book.LevelRequired,
				Diagnostics: []string{"HMGEN-FEATURE-PACKAGE"},
				Surfaces:    []string{"model"},
			},
			{
				Rule:        "hatmax.testing.required_boundaries",
				Level:       book.LevelRequired,
				Diagnostics: []string{"HMGEN-TEST-BOUNDARY"},
				Surfaces:    []string{"tests"},
			},
		},
		Exceptions: []intent.Exception{},
	}
}

func requirePlanCode(t *testing.T, err error, code string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want plan code %q", code)
	}

	var planErr Error
	if !errors.As(err, &planErr) {
		t.Fatalf("error = %v, want plan.Error", err)
	}

	if planErr.Code != code {
		t.Fatalf("plan code = %q, want %q", planErr.Code, code)
	}
}
