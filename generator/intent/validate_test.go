package intent

import (
	"testing"

	"hatmax.adrianpk.com/generator/project"
)

func TestValidateAdmitsSupportedIntents(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
	}{
		{name: "create feature", fixture: "create-feature.yaml"},
		{name: "add field", fixture: "add-field.yaml"},
		{name: "add validation", fixture: "add-validation.yaml"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Validate(loadIntentFixture(t, "valid", test.fixture), validationContext(t))

			if !result.Admitted() {
				t.Fatalf("Validate() status = %q, diagnostics = %#v, clarifications = %#v", result.Status, result.Diagnostics, result.Clarifications)
			}

			if len(result.Diagnostics) != 0 || len(result.Clarifications) != 0 {
				t.Errorf("admitted result has diagnostics or clarifications: %#v", result)
			}
		})
	}
}

func TestValidateSelectsBookContextInCanonicalOrder(t *testing.T) {
	result := Validate(loadIntentFixture(t, "valid", "create-feature.yaml"), validationContext(t))
	if !result.Admitted() {
		t.Fatalf("Validate() status = %q, diagnostics = %#v", result.Status, result.Diagnostics)
	}

	wantCapabilities := []string{"postgres_persistence", "runtime_validation", "htmx_form"}
	if len(result.Selection.Capabilities) != len(wantCapabilities) {
		t.Fatalf("len(Selection.Capabilities) = %d, want %d", len(result.Selection.Capabilities), len(wantCapabilities))
	}

	for index, want := range wantCapabilities {
		if result.Selection.Capabilities[index].ID != want {
			t.Errorf("Selection.Capabilities[%d].ID = %q, want %q", index, result.Selection.Capabilities[index].ID, want)
		}
	}

	if len(result.Selection.Rules) != len(validationContext(t).Book.Rules()) {
		t.Errorf("len(Selection.Rules) = %d, want all applicable Book rules", len(result.Selection.Rules))
	}
}

func TestValidateRequestsOnlyMissingProductDecisions(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		wantFields []string
	}{
		{
			name:       "create feature",
			fixture:    "create-feature.yaml",
			wantFields: []string{"domain.entity", "domain.fields", "domain.route"},
		},
		{
			name:       "add field",
			fixture:    "add-field.yaml",
			wantFields: []string{"domain.field"},
		},
		{
			name:       "add validation",
			fixture:    "add-validation.yaml",
			wantFields: []string{"domain.validation.scope", "domain.validation.value"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Validate(loadIntentFixture(t, "ambiguous", test.fixture), validationContext(t))
			if result.Status != StatusClarificationRequired {
				t.Fatalf("Validate() status = %q, want %q; diagnostics = %#v", result.Status, StatusClarificationRequired, result.Diagnostics)
			}

			if len(result.Clarifications) != len(test.wantFields) {
				t.Fatalf("len(Clarifications) = %d, want %d: %#v", len(result.Clarifications), len(test.wantFields), result.Clarifications)
			}

			for _, field := range test.wantFields {
				if !hasClarification(result, field) {
					t.Errorf("Clarifications does not contain %q: %#v", field, result.Clarifications)
				}
			}
		})
	}
}

func TestValidateClassifiesFixtureFailures(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		fixture   string
		status    Status
		code      string
	}{
		{name: "unsupported capability", directory: "unsupported", fixture: "capability.yaml", status: StatusCapabilityUnsupported, code: "HMGEN-CAPABILITY-UNSUPPORTED"},
		{name: "missing prerequisite", directory: "incompatible", fixture: "missing-prerequisite.yaml", status: StatusIncompatible, code: "HMGEN-CAPABILITY-PREREQUISITE"},
		{name: "existing feature", directory: "incompatible", fixture: "existing-feature.yaml", status: StatusIncompatible, code: "HMGEN-FEATURE-EXISTS"},
		{name: "client-only unique", directory: "incompatible", fixture: "client-only-unique.yaml", status: StatusIncompatible, code: "HMGEN-VALIDATION-DURABLE"},
		{name: "approved exception", directory: "incompatible", fixture: "exception.yaml", status: StatusExceptionRequired, code: "HMGEN-EXCEPTION-EXECUTION-UNSUPPORTED"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Validate(loadIntentFixture(t, test.directory, test.fixture), validationContext(t))
			if result.Status != test.status {
				t.Fatalf("Validate() status = %q, want %q; diagnostics = %#v", result.Status, test.status, result.Diagnostics)
			}

			if !hasDiagnostic(result, test.code) {
				t.Errorf("Diagnostics does not contain %q: %#v", test.code, result.Diagnostics)
			}
		})
	}
}

func TestValidateRejectsContextDrift(t *testing.T) {
	base := loadIntentFixture(t, "valid", "add-field.yaml")
	tests := []struct {
		name   string
		mutate func(*Intent, *ValidationContext)
		code   string
		status Status
	}{
		{name: "missing Book", mutate: func(_ *Intent, context *ValidationContext) { context.Book = nil }, code: "HMGEN-BOOK-MISSING", status: StatusIncompatible},
		{name: "intent Book version", mutate: func(value *Intent, _ *ValidationContext) { value.BookVersion = 2 }, code: "HMGEN-BOOK-VERSION", status: StatusIncompatible},
		{name: "fingerprint Book version", mutate: func(_ *Intent, context *ValidationContext) { context.Fingerprint.BookVersion = 2 }, code: "HMGEN-BOOK-VERSION", status: StatusIncompatible},
		{name: "project fingerprint", mutate: func(value *Intent, _ *ValidationContext) {
			value.ProjectFingerprint = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
		}, code: "HMGEN-PROJECT-FINGERPRINT", status: StatusIncompatible},
		{name: "Hatmax mismatch", mutate: func(value *Intent, _ *ValidationContext) { value.HatmaxVersion = "0.4.1" }, code: "HMGEN-HATMAX-VERSION-MISMATCH", status: StatusIncompatible},
		{name: "Hatmax incompatible", mutate: func(value *Intent, context *ValidationContext) {
			value.HatmaxVersion = "0.3.0"
			context.Inventory.Module.Hatmax.Version = "v0.3.0"
		}, code: "HMGEN-BOOK-INCOMPATIBLE", status: StatusIncompatible},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := base
			context := validationContext(t)
			test.mutate(&value, &context)

			result := Validate(value, context)
			if result.Status != test.status || !hasDiagnostic(result, test.code) {
				t.Errorf("Validate() = status %q, diagnostics %#v; want %q with %q", result.Status, result.Diagnostics, test.status, test.code)
			}
		})
	}
}

func TestValidateRejectsInvalidDomainDecisions(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		mutate  func(*Intent)
		code    string
	}{
		{
			name:    "missing feature",
			fixture: "add-field.yaml",
			mutate:  func(value *Intent) { value.Feature = "invoice" },
			code:    "HMGEN-FEATURE-MISSING",
		},
		{
			name:    "feature name",
			fixture: "create-feature.yaml",
			mutate:  func(value *Intent) { value.Feature = "InvoiceItem" },
			code:    "HMGEN-FEATURE-NAME",
		},
		{
			name:    "route",
			fixture: "create-feature.yaml",
			mutate:  func(value *Intent) { value.Domain.Route = "invoices" },
			code:    "HMGEN-ROUTE-NAME",
		},
		{
			name:    "field type",
			fixture: "add-field.yaml",
			mutate:  func(value *Intent) { value.Domain.Field.Type = "money" },
			code:    "HMGEN-FIELD-TYPE",
		},
		{
			name:    "required field capability",
			fixture: "add-field.yaml",
			mutate:  func(value *Intent) { value.Domain.Field.Required = true },
			code:    "HMGEN-CAPABILITY-REQUIRED",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := loadIntentFixture(t, "valid", test.fixture)
			test.mutate(&value)

			result := Validate(value, validationContext(t))
			if result.Status != StatusIncompatible || !hasDiagnostic(result, test.code) {
				t.Errorf("Validate() = status %q, diagnostics %#v; want incompatible with %q", result.Status, result.Diagnostics, test.code)
			}
		})
	}
}

func TestValidateRejectsInvalidExceptions(t *testing.T) {
	value := loadIntentFixture(t, "valid", "add-field.yaml")
	value.Exceptions = []Exception{{Rule: "missing", Approved: false}}

	result := Validate(value, validationContext(t))
	if result.Status != StatusExceptionRequired {
		t.Fatalf("Validate() status = %q, want %q", result.Status, StatusExceptionRequired)
	}

	for _, code := range []string{
		"HMGEN-EXCEPTION-RULE",
		"HMGEN-EXCEPTION-REASON",
		"HMGEN-EXCEPTION-SCOPE",
		"HMGEN-EXCEPTION-APPROVAL",
	} {
		if !hasDiagnostic(result, code) {
			t.Errorf("Diagnostics does not contain %q: %#v", code, result.Diagnostics)
		}
	}
}

func TestValidateReportsSchemaErrors(t *testing.T) {
	value := loadIntentFixture(t, "valid", "add-field.yaml")
	value.Capabilities = append(value.Capabilities, value.Capabilities[0])

	result := Validate(value, validationContext(t))
	if result.Status != StatusInvalid || !hasDiagnostic(result, "HMGEN-INTENT-DUPLICATE-VALUE") {
		t.Errorf("Validate() = status %q, diagnostics %#v; want invalid duplicate capability", result.Status, result.Diagnostics)
	}
}

func TestValidateRejectsMissingVersionedHatmaxDependency(t *testing.T) {
	value := loadIntentFixture(t, "valid", "add-field.yaml")
	context := validationContext(t)
	context.Inventory.Module.Hatmax = project.HatmaxModule{}

	result := Validate(value, context)
	if result.Status != StatusIncompatible || !hasDiagnostic(result, "HMGEN-HATMAX-MISSING") {
		t.Errorf("Validate() = status %q, diagnostics %#v; want missing Hatmax dependency", result.Status, result.Diagnostics)
	}
}
