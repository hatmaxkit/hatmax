package plan

import (
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

func TestValidateAcceptsCompletePlan(t *testing.T) {
	err := Validate(validTestPlan())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateEnforcesDocumentationEffects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Plan)
	}{
		{
			name: "implicit documentation surface",
			mutate: func(value *Plan) {
				value.AffectedSurfaces = append(value.AffectedSurfaces, "documentation")
				value.AllowedEffects.Surfaces = append(value.AllowedEffects.Surfaces, "documentation")
				value.FingerprintInputs.PlannedSurfaces = append(value.FingerprintInputs.PlannedSurfaces, "documentation")
			},
		},
		{
			name: "active documentation without surface",
			mutate: func(value *Plan) {
				value.Documentation = intent.DocumentationPlanned
				value.DocumentationTargets = []intent.DocumentationTarget{{
					Quadrant:   intent.DocumentationReference,
					Subject:    "invoice",
					ReaderGoal: "Find invoice contracts.",
				}}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validTestPlan()
			test.mutate(&value)

			requirePlanCode(t, Validate(value), "plan_documentation_scope_invalid")
		})
	}
}

func TestValidateRejectsInvalidPlanStructure(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Plan)
		code   string
	}{
		{name: "schema", mutate: func(value *Plan) { value.SchemaVersion = ApplicationSchemaVersion + 1 }, code: "plan_schema_unsupported"},
		{name: "identity", mutate: func(value *Plan) { value.Feature = "" }, code: "plan_required_field"},
		{name: "intent", mutate: func(value *Plan) { value.Intent = "remove_feature" }, code: "plan_intent_invalid"},
		{name: "domain incomplete", mutate: func(value *Plan) { value.Domain.Entity = "" }, code: "plan_domain_incomplete"},
		{name: "domain invalid", mutate: func(value *Plan) { value.Domain.Fields[0].Type = "json" }, code: "plan_domain_invalid"},
		{name: "fingerprint surfaces", mutate: func(value *Plan) { value.FingerprintInputs.PlannedSurfaces = []string{"model"} }, code: "plan_fingerprint_incomplete"},
		{name: "documentation", mutate: func(value *Plan) { value.Documentation = "always" }, code: "plan_documentation_invalid"},
		{name: "versions", mutate: func(value *Plan) { value.BookVersion = 0 }, code: "plan_required_field"},
		{name: "fingerprint", mutate: func(value *Plan) { value.ProjectFingerprint = "invalid" }, code: "plan_fingerprint_invalid"},
		{name: "duplicate capability", mutate: func(value *Plan) { value.Capabilities = append(value.Capabilities, value.Capabilities[0]) }, code: "plan_duplicate_value"},
		{name: "missing rules", mutate: func(value *Plan) { value.Rules = nil }, code: "plan_required_field"},
		{name: "invalid rule", mutate: func(value *Plan) { value.Rules[0].ID = "Invalid Rule" }, code: "plan_rule_invalid"},
		{name: "duplicate operation", mutate: func(value *Plan) { value.Operations[1].ID = value.Operations[0].ID }, code: "plan_duplicate_value"},
		{name: "unknown owner", mutate: func(value *Plan) { value.Operations[0].Owner.Kind = "model" }, code: "plan_owner_invalid"},
		{name: "duplicate operation surface", mutate: func(value *Plan) { value.Operations[0].Surfaces = []string{"model", "model"} }, code: "plan_duplicate_value"},
		{name: "unknown operation surface", mutate: func(value *Plan) { value.Operations[0].Surfaces = []string{"wiring"} }, code: "plan_reference_missing"},
		{name: "unknown operation rule", mutate: func(value *Plan) { value.Operations[0].Rules = []string{"hatmax.missing"} }, code: "plan_reference_missing"},
		{name: "forward operation dependency", mutate: func(value *Plan) { value.Operations[0].DependsOn = []string{value.Operations[1].ID} }, code: "plan_operation_order_invalid"},
		{name: "invalid precondition", mutate: func(value *Plan) { value.Preconditions[0].Kind = "revision" }, code: "plan_precondition_invalid"},
		{name: "precondition mismatch", mutate: func(value *Plan) {
			value.Preconditions[0].Expected = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
		}, code: "plan_precondition_mismatch"},
		{name: "unknown effect surface", mutate: func(value *Plan) { value.AllowedEffects.Surfaces = []string{"wiring"} }, code: "plan_reference_missing"},
		{name: "incomplete effect surfaces", mutate: func(value *Plan) { value.AllowedEffects.Surfaces = []string{"model"} }, code: "plan_effects_incomplete"},
		{name: "invalid dependency", mutate: func(value *Plan) {
			value.AllowedEffects.Dependencies = []DependencyEffect{{Capability: "postgres_persistence"}}
		}, code: "plan_dependency_invalid"},
		{name: "unknown validation rule", mutate: func(value *Plan) { value.Validation[0].Rule = "hatmax.missing" }, code: "plan_reference_missing"},
		{name: "validation level mismatch", mutate: func(value *Plan) { value.Validation[0].Level = "recommended" }, code: "plan_validation_invalid"},
		{name: "duplicate validation", mutate: func(value *Plan) { value.Validation[1].Rule = value.Validation[0].Rule }, code: "plan_duplicate_value"},
		{name: "incomplete validation", mutate: func(value *Plan) { value.Validation = value.Validation[:1] }, code: "plan_validation_incomplete"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validTestPlan()
			test.mutate(&value)

			requirePlanCode(t, Validate(value), test.code)
		})
	}
}
