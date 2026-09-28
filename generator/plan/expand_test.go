package plan

import (
	"bytes"
	"testing"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

func TestExpandInitialOperations(t *testing.T) {
	tests := []struct {
		name             string
		operation        intent.Operation
		capabilities     []string
		wantCapabilities []string
		wantSurfaces     []string
		wantOperations   int
	}{
		{
			name:             "create feature",
			operation:        intent.OperationCreateFeature,
			capabilities:     []string{"htmx_form", "postgres_persistence", "runtime_validation"},
			wantCapabilities: []string{"postgres_persistence", "runtime_validation", "htmx_form"},
			wantSurfaces:     []string{"migration", "model", "store", "service", "handler", "templates", "wiring", "tests"},
			wantOperations:   15,
		},
		{
			name:             "add field",
			operation:        intent.OperationAddField,
			capabilities:     []string{"postgres_persistence"},
			wantCapabilities: []string{"postgres_persistence"},
			wantSurfaces:     []string{"migration", "model", "store", "service", "handler", "templates", "tests"},
			wantOperations:   8,
		},
		{
			name:             "add validation",
			operation:        intent.OperationAddValidation,
			capabilities:     []string{"runtime_validation", "postgres_persistence"},
			wantCapabilities: []string{"postgres_persistence", "runtime_validation"},
			wantSurfaces:     []string{"migration", "model", "store", "handler", "templates", "tests"},
			wantOperations:   10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			admission, context := admittedExpansion(t, test.operation, test.capabilities)

			result, err := Expand(admission, context)
			if err != nil {
				t.Fatalf("Expand() error = %v", err)
			}

			assertStrings(t, "Capabilities", result.Capabilities, test.wantCapabilities)
			assertStrings(t, "AffectedSurfaces", result.AffectedSurfaces, test.wantSurfaces)

			if len(result.Operations) != test.wantOperations {
				t.Errorf("len(Operations) = %d, want %d", len(result.Operations), test.wantOperations)
			}

			if len(result.Rules) != len(context.Book.Rules()) || len(result.Validation) != len(result.Rules) {
				t.Errorf("Rules = %d and Validation = %d, want complete selected Book context", len(result.Rules), len(result.Validation))
			}

			err = Validate(result)
			if err != nil {
				t.Errorf("expanded plan is invalid: %v", err)
			}
		})
	}
}

func TestExpandAttributesOperationsAndEffectsToBookEntries(t *testing.T) {
	admission, context := admittedExpansion(
		t,
		intent.OperationCreateFeature,
		[]string{"postgres_persistence", "runtime_validation", "htmx_form"},
	)

	result, err := Expand(admission, context)
	if err != nil {
		t.Fatalf("Expand() error = %v", err)
	}

	first := result.Operations[0]
	if first.ID != "archetype.server_rendered_crud.feature_package" || first.Owner.Kind != OwnerArchetype {
		t.Errorf("first operation = %#v, want archetype-owned feature package", first)
	}

	queryAdapter := operationByID(result.Operations, "capability.postgres_persistence.query_adapter")
	if queryAdapter == nil {
		t.Fatal("expanded operations do not contain the Postgres query adapter")
	}

	assertStrings(
		t,
		"query adapter dependencies",
		queryAdapter.DependsOn,
		[]string{"capability.postgres_persistence.migration"},
	)

	if len(result.AllowedEffects.Dependencies) != 1 {
		t.Fatalf("len(AllowedEffects.Dependencies) = %d, want 1", len(result.AllowedEffects.Dependencies))
	}

	dependency := result.AllowedEffects.Dependencies[0]
	if dependency.Capability != "postgres_persistence" || dependency.Module != "github.com/sqlc-dev/sqlc/cmd/sqlc" {
		t.Errorf("dependency effect = %#v, want Book-owned SQLC tool", dependency)
	}

	validation := validationByRule(result.Validation, "hatmax.persistence.postgres_sqlc")
	if validation == nil || !containsString(validation.Surfaces, "migration") || !containsString(validation.Surfaces, "store") {
		t.Errorf("persistence validation = %#v, want migration and store attribution", validation)
	}
}

func TestExpandCarriesIntentAndProjectContext(t *testing.T) {
	admission, context := admittedExpansion(t, intent.OperationAddField, []string{"postgres_persistence"})
	admission.Intent.Documentation = intent.DocumentationPlanned

	result, err := Expand(admission, context)
	if err != nil {
		t.Fatalf("Expand() error = %v", err)
	}

	if result.Documentation != intent.DocumentationPlanned || result.ProjectFingerprint != context.Fingerprint.Value {
		t.Errorf("expanded identity = %#v, want admitted intent and fingerprint", result)
	}

	if result.Domain.Field == nil || result.Domain.Field.Name != "notes" {
		t.Errorf("Domain = %#v, want admitted field input", result.Domain)
	}

	if len(result.ExpectedObservations) != 1 || result.ExpectedObservations[0].Key != "module:configuration" {
		t.Errorf("ExpectedObservations = %#v, want detached project observations", result.ExpectedObservations)
	}

	context.Fingerprint.Observations[0].Key = "changed"
	if result.ExpectedObservations[0].Key != "module:configuration" {
		t.Error("Expand() exposed fingerprint observation storage")
	}

	admission.Intent.Domain.Field.Name = "changed"
	if result.Domain.Field.Name != "notes" {
		t.Error("Expand() exposed intent domain storage")
	}
}

func TestExpandIsDeterministic(t *testing.T) {
	admission, context := admittedExpansion(
		t,
		intent.OperationCreateFeature,
		[]string{"htmx_form", "runtime_validation", "postgres_persistence"},
	)

	first, err := Expand(admission, context)
	if err != nil {
		t.Fatalf("Expand() error = %v", err)
	}

	second, err := Expand(admission, context)
	if err != nil {
		t.Fatalf("second Expand() error = %v", err)
	}

	firstYAML, err := MarshalYAML(first)
	if err != nil {
		t.Fatalf("MarshalYAML(first) error = %v", err)
	}

	secondYAML, err := MarshalYAML(second)
	if err != nil {
		t.Fatalf("MarshalYAML(second) error = %v", err)
	}

	if !bytes.Equal(firstYAML, secondYAML) {
		t.Error("equivalent expansion inputs produced different plans")
	}
}

func TestExpandRejectsInvalidAdmissionContext(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*intent.Result, *ExpansionContext)
		code   string
	}{
		{name: "intent not admitted", mutate: func(result *intent.Result, _ *ExpansionContext) { result.Status = intent.StatusIncompatible }, code: "plan_intent_not_admitted"},
		{name: "invalid intent", mutate: func(result *intent.Result, _ *ExpansionContext) { result.Intent.Feature = "" }, code: "plan_intent_invalid"},
		{name: "missing Book", mutate: func(_ *intent.Result, context *ExpansionContext) { context.Book = nil }, code: "plan_book_missing"},
		{name: "Book mismatch", mutate: func(result *intent.Result, _ *ExpansionContext) { result.Intent.BookVersion = 2 }, code: "plan_book_version_mismatch"},
		{name: "fingerprint mismatch", mutate: func(result *intent.Result, _ *ExpansionContext) {
			result.Intent.ProjectFingerprint = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
		}, code: "plan_fingerprint_mismatch"},
		{name: "Hatmax incompatible", mutate: func(result *intent.Result, _ *ExpansionContext) { result.Intent.HatmaxVersion = "0.3.0" }, code: "plan_hatmax_incompatible"},
		{name: "selection invalid", mutate: func(result *intent.Result, _ *ExpansionContext) { result.Intent.Archetype = "missing" }, code: "plan_selection_invalid"},
		{name: "operation capability missing", mutate: func(result *intent.Result, _ *ExpansionContext) {
			result.Intent.Operation = intent.OperationAddValidation
			result.Intent.Domain = testDomain(intent.OperationAddValidation)
		}, code: "plan_capability_missing"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			admission, context := admittedExpansion(t, intent.OperationAddField, []string{"postgres_persistence"})
			test.mutate(&admission, &context)

			_, err := Expand(admission, context)
			requirePlanCode(t, err, test.code)
		})
	}
}

func admittedExpansion(
	t *testing.T,
	operation intent.Operation,
	capabilities []string,
) (intent.Result, ExpansionContext) {
	t.Helper()

	selectedBook, err := book.LoadDefault()
	if err != nil {
		t.Fatalf("book.LoadDefault() error = %v", err)
	}

	selection, err := selectedBook.Select("server_rendered_crud", capabilities)
	if err != nil {
		t.Fatalf("Book.Select() error = %v", err)
	}

	fingerprint := project.Fingerprint{
		Value:                testFingerprint,
		BookVersion:          1,
		SelectedDependencies: []string{"github.com/sqlc-dev/sqlc/cmd/sqlc"},
		PlannedSurfaces:      []string{"migration", "model", "store", "service", "handler", "templates", "wiring", "tests"},
		Observations: []project.Observation{
			{
				Key:    "module:configuration",
				Class:  project.ObservationModule,
				Path:   "go.mod",
				Digest: "fixture-digest",
			},
		},
	}
	value := intent.Intent{
		SchemaVersion:      intent.CurrentSchemaVersion,
		Operation:          operation,
		ProjectFingerprint: testFingerprint,
		HatmaxVersion:      "v0.4.0",
		BookVersion:        1,
		Archetype:          "server_rendered_crud",
		Feature:            "invoice",
		Domain:             testDomain(operation),
		Capabilities:       append([]string{}, capabilities...),
		Documentation:      intent.DocumentationNotRequested,
		Exceptions:         []intent.Exception{},
	}

	return intent.Result{
		Status:    intent.StatusAdmitted,
		Intent:    value,
		Selection: selection,
	}, ExpansionContext{
		Book:        selectedBook,
		Fingerprint: fingerprint,
	}
}

func testDomain(operation intent.Operation) intent.Domain {
	switch operation {
	case intent.OperationCreateFeature:
		return intent.Domain{
			Entity: "Invoice",
			Route:  "/invoices",
			Fields: []intent.Field{{Name: "number", Type: "string"}},
		}
	case intent.OperationAddField:
		return intent.Domain{Field: &intent.Field{Name: "notes", Type: "text"}}
	case intent.OperationAddValidation:
		return intent.Domain{Validation: &intent.ValidationRule{
			Field: "number",
			Kind:  "required",
			Scope: intent.ValidationDurable,
		}}
	default:
		return intent.Domain{}
	}
}

func assertStrings(t *testing.T, name string, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(%s) = %d, want %d: %v", name, len(got), len(want), got)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Errorf("%s[%d] = %q, want %q", name, index, got[index], want[index])
		}
	}
}

func operationByID(operations []Operation, id string) *Operation {
	for index := range operations {
		if operations[index].ID == id {
			return &operations[index]
		}
	}

	return nil
}

func validationByRule(values []ValidationObligation, rule string) *ValidationObligation {
	for index := range values {
		if values[index].Rule == rule {
			return &values[index]
		}
	}

	return nil
}
