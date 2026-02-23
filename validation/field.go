package validation

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// FieldValidator provides fluent validation for a string field.
type FieldValidator struct {
	field  string
	value  string
	errors ValidationErrors
}

// Field creates a new field validator for a string value.
func Field(field, value string) *FieldValidator {
	return &FieldValidator{field: field, value: value}
}

// Required validates the field is not empty.
func (f *FieldValidator) Required() *FieldValidator {
	if strings.TrimSpace(f.value) == "" {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "Required",
			Message: "is required",
		})
	}

	return f
}

// MinLength validates minimum string length.
func (f *FieldValidator) MinLength(min int) *FieldValidator {
	if f.value != "" && len(f.value) < min {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "MinLength",
			Message: fmt.Sprintf("must be at least %d characters", min),
			Params:  map[string]any{"min": min},
		})
	}

	return f
}

// MaxLength validates maximum string length.
func (f *FieldValidator) MaxLength(max int) *FieldValidator {
	if len(f.value) > max {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "MaxLength",
			Message: fmt.Sprintf("must be at most %d characters", max),
			Params:  map[string]any{"max": max},
		})
	}

	return f
}

// Email validates email format.
func (f *FieldValidator) Email() *FieldValidator {
	err := ValidateEmailField(f.field, f.value)
	if !err.IsEmpty() {
		f.errors = append(f.errors, err)
	}

	return f
}

// Phone validates phone format.
func (f *FieldValidator) Phone() *FieldValidator {
	err := ValidatePhoneField(f.field, f.value)
	if !err.IsEmpty() {
		f.errors = append(f.errors, err)
	}

	return f
}

// NoHTML validates no HTML tags present.
func (f *FieldValidator) NoHTML() *FieldValidator {
	err := NoHTML(f.field, f.value)
	if !err.IsEmpty() {
		f.errors = append(f.errors, err)
	}

	return f
}

// OneOf validates value is in allowed list.
func (f *FieldValidator) OneOf(allowed []string) *FieldValidator {
	if f.value == "" {
		return f
	}

	for _, a := range allowed {
		if f.value == a {
			return f
		}
	}

	f.errors = append(f.errors, ValidationError{
		Field:   f.field,
		Rule:    "OneOf",
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")),
		Params:  map[string]any{"allowed": allowed},
	})

	return f
}

// OneOfWithLabels validates value is in allowed list, showing labels in error message.
// The values slice contains the actual valid values to match against.
// The labels slice contains human-readable labels for the error message.
// Both slices must have the same length.
func (f *FieldValidator) OneOfWithLabels(values, labels []string) *FieldValidator {
	if f.value == "" {
		return f
	}

	for _, v := range values {
		if f.value == v {
			return f
		}
	}
	// Use labels for error message if provided, otherwise fall back to values
	display := labels
	if len(labels) == 0 || len(labels) != len(values) {
		display = values
	}

	f.errors = append(f.errors, ValidationError{
		Field:   f.field,
		Rule:    "OneOf",
		Message: fmt.Sprintf("must be one of: %s", strings.Join(display, ", ")),
		Params:  map[string]any{"allowed": values, "labels": labels},
	})

	return f
}

// Alphanumeric validates the value contains only letters and digits.
func (f *FieldValidator) Alphanumeric() *FieldValidator {
	if f.value == "" {
		return f
	}

	for _, r := range f.value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			f.errors = append(f.errors, ValidationError{
				Field:   f.field,
				Rule:    "Alphanumeric",
				Message: "must contain only letters and numbers",
			})

			break
		}
	}

	return f
}

// Errors returns accumulated errors.
func (f *FieldValidator) Errors() ValidationErrors {
	return f.errors
}

// --- UUID Field Validator ---

// UUIDFieldValidator provides fluent validation for a UUID field.
type UUIDFieldValidator struct {
	field  string
	value  uuid.UUID
	errors ValidationErrors
}

// UUIDField creates a new field validator for a UUID value.
func UUIDField(field string, value uuid.UUID) *UUIDFieldValidator {
	return &UUIDFieldValidator{field: field, value: value}
}

// Required validates the UUID is not nil.
func (f *UUIDFieldValidator) Required() *UUIDFieldValidator {
	if f.value == uuid.Nil {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "Required",
			Message: "is required",
		})
	}

	return f
}

// Errors returns accumulated errors.
func (f *UUIDFieldValidator) Errors() ValidationErrors {
	return f.errors
}

// --- Int Field Validator ---

// IntFieldValidator provides fluent validation for an int field.
type IntFieldValidator struct {
	field  string
	value  int
	errors ValidationErrors
}

// IntField creates a new field validator for an int value.
func IntField(field string, value int) *IntFieldValidator {
	return &IntFieldValidator{field: field, value: value}
}

// Min validates the integer is at least the minimum value.
func (f *IntFieldValidator) Min(min int) *IntFieldValidator {
	if f.value < min {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "Min",
			Message: fmt.Sprintf("must be at least %d", min),
			Params:  map[string]any{"min": min},
		})
	}

	return f
}

// Max validates the integer does not exceed the maximum value.
func (f *IntFieldValidator) Max(max int) *IntFieldValidator {
	if f.value > max {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "Max",
			Message: fmt.Sprintf("must be at most %d", max),
			Params:  map[string]any{"max": max},
		})
	}

	return f
}

// InRange validates the integer is within the specified range.
func (f *IntFieldValidator) InRange(min, max int) *IntFieldValidator {
	if f.value < min || f.value > max {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "InRange",
			Message: fmt.Sprintf("must be between %d and %d", min, max),
			Params:  map[string]any{"min": min, "max": max},
		})
	}

	return f
}

// Positive validates the integer is greater than zero.
func (f *IntFieldValidator) Positive() *IntFieldValidator {
	if f.value <= 0 {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "Positive",
			Message: "must be greater than zero",
		})
	}

	return f
}

// NonNegative validates the integer is zero or greater.
func (f *IntFieldValidator) NonNegative() *IntFieldValidator {
	if f.value < 0 {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "NonNegative",
			Message: "must be zero or greater",
		})
	}

	return f
}

// Errors returns accumulated errors.
func (f *IntFieldValidator) Errors() ValidationErrors {
	return f.errors
}

// --- Float Field Validator ---

// FloatFieldValidator provides fluent validation for a float64 field.
type FloatFieldValidator struct {
	field  string
	value  float64
	errors ValidationErrors
}

// FloatField creates a new field validator for a float64 value.
func FloatField(field string, value float64) *FloatFieldValidator {
	return &FloatFieldValidator{field: field, value: value}
}

// Min validates the float is at least the minimum value.
func (f *FloatFieldValidator) Min(min float64) *FloatFieldValidator {
	if f.value < min {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "Min",
			Message: fmt.Sprintf("must be at least %.2f", min),
			Params:  map[string]any{"min": min},
		})
	}

	return f
}

// Max validates the float does not exceed the maximum value.
func (f *FloatFieldValidator) Max(max float64) *FloatFieldValidator {
	if f.value > max {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "Max",
			Message: fmt.Sprintf("must be at most %.2f", max),
			Params:  map[string]any{"max": max},
		})
	}

	return f
}

// Positive validates the float is greater than zero.
func (f *FloatFieldValidator) Positive() *FloatFieldValidator {
	if f.value <= 0 {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "Positive",
			Message: "must be greater than zero",
		})
	}

	return f
}

// NonNegative validates the float is zero or greater.
func (f *FloatFieldValidator) NonNegative() *FloatFieldValidator {
	if f.value < 0 {
		f.errors = append(f.errors, ValidationError{
			Field:   f.field,
			Rule:    "NonNegative",
			Message: "must be zero or greater",
		})
	}

	return f
}

// Errors returns accumulated errors.
func (f *FloatFieldValidator) Errors() ValidationErrors {
	return f.errors
}

// --- Aggregate Validator ---

// FieldErrors is an interface for validators that return errors.
type FieldErrors interface {
	Errors() ValidationErrors
}

// ValidateAll collects errors from multiple field validators.
func ValidateAll(validators ...FieldErrors) ValidationErrors {
	var errs ValidationErrors

	for _, v := range validators {
		if v != nil {
			errs.Merge(v.Errors())
		}
	}

	return errs
}
