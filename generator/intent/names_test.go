package intent

import "testing"

func TestValidateNormalizesSemanticNamesDeterministically(t *testing.T) {
	value := loadIntentFixture(t, "valid", "create-feature.yaml")
	value.Feature = "Invoice Item"
	value.Domain.Entity = "invoice item"
	value.Domain.Route = "Invoice Items"
	value.Domain.Label = ""
	value.Domain.Fields = []Field{
		{Name: "Invoice Number", Type: "string", Required: true},
		{Name: "dueOn", Type: "date"},
	}

	result := Validate(value, validationContext(t))
	if !result.Admitted() {
		t.Fatalf("Validate() status = %q, diagnostics = %#v", result.Status, result.Diagnostics)
	}

	if result.Intent.Feature != "invoice_item" {
		t.Errorf("Feature = %q, want invoice_item", result.Intent.Feature)
	}

	if result.Intent.Domain.Entity != "InvoiceItem" {
		t.Errorf("Entity = %q, want InvoiceItem", result.Intent.Domain.Entity)
	}

	if result.Intent.Domain.Route != "/invoice-items" {
		t.Errorf("Route = %q, want /invoice-items", result.Intent.Domain.Route)
	}

	if result.Intent.Domain.Label != "Invoice Items" {
		t.Errorf("Label = %q, want Invoice Items", result.Intent.Domain.Label)
	}

	fields := result.Intent.Domain.Fields
	if fields[0].Name != "invoice_number" || fields[0].Label != "Invoice Number" {
		t.Errorf("first field = %#v, want canonical name and default label", fields[0])
	}

	if fields[1].Name != "due_on" || fields[1].Label != "Due On" {
		t.Errorf("second field = %#v, want canonical name and default label", fields[1])
	}
}

func TestValidateDerivesConventionalCreateFeatureNames(t *testing.T) {
	value := loadIntentFixture(t, "valid", "create-feature.yaml")
	value.Feature = "Category"
	value.Domain.Entity = ""
	value.Domain.Route = ""
	value.Domain.Label = ""

	result := Validate(value, validationContext(t))
	if !result.Admitted() {
		t.Fatalf("Validate() status = %q, diagnostics = %#v", result.Status, result.Diagnostics)
	}

	if result.Intent.Feature != "category" || result.Intent.Domain.Entity != "Category" || result.Intent.Domain.Route != "/categories" || result.Intent.Domain.Label != "Categories" {
		t.Errorf("normalized intent = %#v, want category conventions", result.Intent)
	}
}

func TestValidatePreservesExplicitProductLabels(t *testing.T) {
	value := loadIntentFixture(t, "valid", "create-feature.yaml")
	value.Feature = "Invoice"
	value.Domain.Entity = "invoice"
	value.Domain.Route = "invoices"
	value.Domain.Label = "Accounts receivable"
	value.Domain.Fields[0].Name = "invoice number"
	value.Domain.Fields[0].Label = "Reference code"

	result := Validate(value, validationContext(t))
	if !result.Admitted() {
		t.Fatalf("Validate() status = %q, diagnostics = %#v", result.Status, result.Diagnostics)
	}

	if result.Intent.Domain.Label != "Accounts receivable" {
		t.Errorf("Label = %q, want explicit product label", result.Intent.Domain.Label)
	}

	field := result.Intent.Domain.Fields[0]
	if field.Name != "invoice_number" || field.Label != "Reference code" {
		t.Errorf("field = %#v, want normalized name and explicit label", field)
	}
}

func TestValidateRejectsMateriallyInvalidSemanticNames(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Intent)
		code   string
	}{
		{name: "feature path", mutate: func(value *Intent) { value.Feature = "invoice/item" }, code: "HMGEN-FEATURE-NAME"},
		{name: "route query", mutate: func(value *Intent) { value.Domain.Route = "/invoices?sort=number" }, code: "HMGEN-ROUTE-NAME"},
		{name: "field punctuation", mutate: func(value *Intent) { value.Domain.Fields[0].Name = "invoice.number" }, code: "HMGEN-FIELD-NAME"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := loadIntentFixture(t, "valid", "create-feature.yaml")
			test.mutate(&value)

			result := Validate(value, validationContext(t))
			if result.Status != StatusIncompatible || !hasDiagnostic(result, test.code) {
				t.Errorf("Validate() = status %q, diagnostics %#v; want incompatible with %q", result.Status, result.Diagnostics, test.code)
			}
		})
	}
}
