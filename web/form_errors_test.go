package web

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"hatmax.adrianpk.com/validation"
)

func TestNewFormErrors(t *testing.T) {
	errs := NewFormErrors("Review the following errors.")

	if errs.General != "Review the following errors." {
		t.Fatalf("General = %q, want review summary", errs.General)
	}

	if errs.Fields == nil {
		t.Fatal("Fields is nil")
	}

	if !errs.Any() {
		t.Fatal("Any() = false, want true")
	}
}

func TestFormErrorsAddAndReadFieldMessages(t *testing.T) {
	var errs FormErrors

	errs.Add("email", "is invalid")
	errs.Add("email", "is already in use")
	errs.Add("phone", "must contain 7 to 15 digits")

	wantEmail := []string{"is invalid", "is already in use"}
	if got := errs.For("email"); !reflect.DeepEqual(got, wantEmail) {
		t.Fatalf("For(email) = %v, want %v", got, wantEmail)
	}

	if got := errs.First("email"); got != "is invalid" {
		t.Errorf("First(email) = %q, want %q", got, "is invalid")
	}

	if !errs.Has("phone") {
		t.Error("Has(phone) = false, want true")
	}

	if errs.Has("name") {
		t.Error("Has(name) = true, want false")
	}

	if !errs.Any() {
		t.Error("Any() = false, want true")
	}
}

func TestFormErrorsAddGeneralMessage(t *testing.T) {
	errs := NewFormErrors("Review the following errors.")

	errs.Add("", "The form could not be submitted.")
	errs.Add("email", "")

	if errs.General != "The form could not be submitted." {
		t.Fatalf("General = %q, want submitted message", errs.General)
	}

	if errs.Has("email") {
		t.Fatal("Has(email) = true after adding an empty message")
	}
}

func TestFormErrorsFromWrappedValidationErrors(t *testing.T) {
	validationErrors := validation.ValidationErrors{
		{Field: "email", Message: "must be a valid email address"},
		{Field: "email", Message: "is already in use"},
		{Field: "phone", Message: "must contain 7 to 15 digits"},
	}
	err := fmt.Errorf("create contact: %w", validationErrors)

	formErrors := FormErrorsFrom(err, "Review the highlighted fields.")

	if formErrors.General != "Review the highlighted fields." {
		t.Errorf("General = %q, want highlighted-fields summary", formErrors.General)
	}

	wantEmail := []string{"must be a valid email address", "is already in use"}
	if got := formErrors.For("email"); !reflect.DeepEqual(got, wantEmail) {
		t.Errorf("For(email) = %v, want %v", got, wantEmail)
	}

	if got := formErrors.First("phone"); got != "must contain 7 to 15 digits" {
		t.Errorf("First(phone) = %q, want phone message", got)
	}
}

func TestFormErrorsFromPointerValidationErrors(t *testing.T) {
	validationErrors := validation.ValidationErrors{
		{Field: "name", Message: "is required"},
	}

	formErrors := FormErrorsFrom(&validationErrors, "Review the highlighted fields.")

	if got := formErrors.First("name"); got != "is required" {
		t.Fatalf("First(name) = %q, want required message", got)
	}
}

func TestFormErrorsFromGeneralValidationError(t *testing.T) {
	validationErrors := validation.ValidationErrors{
		{Message: "At least one contact method is required."},
	}

	formErrors := FormErrorsFrom(validationErrors, "Review the following errors.")

	if formErrors.General != "At least one contact method is required." {
		t.Fatalf("General = %q, want contact-method message", formErrors.General)
	}
}

func TestFormErrorsFromDoesNotExposeUnexpectedError(t *testing.T) {
	err := errors.New("database password rejected")

	formErrors := FormErrorsFrom(err, "The form could not be submitted.")

	if formErrors.General != "The form could not be submitted." {
		t.Errorf("General = %q, want safe fallback", formErrors.General)
	}

	if len(formErrors.Fields) != 0 {
		t.Errorf("Fields = %v, want no field errors", formErrors.Fields)
	}
}

func TestEmptyFormErrors(t *testing.T) {
	var errs FormErrors

	if errs.Any() {
		t.Fatal("Any() = true, want false")
	}

	if got := errs.First("email"); got != "" {
		t.Fatalf("First(email) = %q, want empty string", got)
	}

	if got := errs.For("email"); got != nil {
		t.Fatalf("For(email) = %v, want nil", got)
	}
}
