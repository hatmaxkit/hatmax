package validation

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ValidationError struct {
	Field   string         // Field name (for UI mapping)
	Rule    string         // Rule that was violated (e.g., "Required", "MaxLength")
	Message string         // Human-readable message
	Params  map[string]any // Rule parameters (e.g., {"max": 100})
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

func (e *ValidationErrors) Add(field, message string) {
	*e = append(*e, ValidationError{Field: field, Message: message})
}

func (e *ValidationErrors) AddError(err ValidationError) {
	*e = append(*e, err)
}

func (e *ValidationErrors) Merge(other ValidationErrors) {
	*e = append(*e, other...)
}

func (e ValidationErrors) ForField(field string) []string {
	var messages []string
	for _, err := range e {
		if err.Field == field {
			messages = append(messages, err.Message)
		}
	}
	return messages
}

func (e ValidationErrors) Fields() []string {
	seen := make(map[string]bool)
	var fields []string
	for _, err := range e {
		if err.Field != "" && !seen[err.Field] {
			seen[err.Field] = true
			fields = append(fields, err.Field)
		}
	}
	return fields
}

type Validator interface {
	Validate() ValidationErrors
}

type ValidatorFunc func() ValidationErrors

func (f ValidatorFunc) Validate() ValidationErrors {
	return f()
}

func Combine(validators ...Validator) ValidationErrors {
	var errors ValidationErrors
	for _, validator := range validators {
		if validator != nil {
			errors.Merge(validator.Validate())
		}
	}
	return errors
}

func IsRequired(value string) bool {
	return strings.TrimSpace(value) != ""
}

func IsRequiredUUID(value uuid.UUID) bool {
	return value != uuid.Nil
}

func MinLength(value string, min int) bool {
	return len(value) >= min
}

func MaxLength(value string, max int) bool {
	return len(value) <= max
}

func MinValueInt(value, min int) bool {
	return value >= min
}

func MaxValueInt(value, max int) bool {
	return value <= max
}

func InRange(value, min, max int) bool {
	return value >= min && value <= max
}

func OneOf(value string, allowed []string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

func RequiredString(field, value string) ValidationError {
	if !IsRequired(value) {
		return ValidationError{Field: field, Message: "is required"}
	}
	return ValidationError{}
}

func RequiredUUID(field string, value uuid.UUID) ValidationError {
	if !IsRequiredUUID(value) {
		return ValidationError{Field: field, Message: "is required"}
	}
	return ValidationError{}
}

func StringMinLength(field, value string, min int) ValidationError {
	if !MinLength(value, min) {
		return ValidationError{Field: field, Message: fmt.Sprintf("must be at least %d characters", min)}
	}
	return ValidationError{}
}

func StringMaxLength(field, value string, max int) ValidationError {
	if !MaxLength(value, max) {
		return ValidationError{Field: field, Message: fmt.Sprintf("must be at most %d characters", max)}
	}
	return ValidationError{}
}

func IntMinValue(field string, value, min int) ValidationError {
	if !MinValueInt(value, min) {
		return ValidationError{Field: field, Message: fmt.Sprintf("must be at least %d", min)}
	}
	return ValidationError{}
}

func IntMaxValue(field string, value, max int) ValidationError {
	if !MaxValueInt(value, max) {
		return ValidationError{Field: field, Message: fmt.Sprintf("must be at most %d", max)}
	}
	return ValidationError{}
}

func IntInRange(field string, value, min, max int) ValidationError {
	if !InRange(value, min, max) {
		return ValidationError{Field: field, Message: fmt.Sprintf("must be between %d and %d", min, max)}
	}
	return ValidationError{}
}

func StringOneOf(field, value string, allowed []string) ValidationError {
	if !OneOf(value, allowed) {
		return ValidationError{Field: field, Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", "))}
	}
	return ValidationError{}
}

// ByField returns the first error message for a specific field, or empty string.
func (e ValidationErrors) ByField(field string) string {
	for _, err := range e {
		if err.Field == field {
			return err.Message
		}
	}
	return ""
}

// First returns the first ValidationError, or empty if none.
func (e ValidationErrors) First() ValidationError {
	if len(e) > 0 {
		return e[0]
	}
	return ValidationError{}
}

// AsMap returns errors as a map of field name to slice of messages.
func (e ValidationErrors) AsMap() map[string][]string {
	result := make(map[string][]string)
	for _, err := range e {
		result[err.Field] = append(result[err.Field], err.Message)
	}
	return result
}

// NewSingleError creates a ValidationErrors with a single error.
func NewSingleError(field, message string) ValidationErrors {
	return ValidationErrors{{Field: field, Message: message}}
}

// NewError creates a ValidationErrors with a single general error.
func NewError(message string) ValidationErrors {
	return ValidationErrors{{Message: message}}
}

// IsEmpty checks if a ValidationError is empty (no error).
func (e ValidationError) IsEmpty() bool {
	return e.Field == "" && e.Rule == "" && e.Message == ""
}

// --- Float Predicates ---

// MinValueFloat checks if a float is at least the minimum value.
func MinValueFloat(value, min float64) bool {
	return value >= min
}

// MaxValueFloat checks if a float does not exceed the maximum value.
func MaxValueFloat(value, max float64) bool {
	return value <= max
}

// IsPositive checks if a float is greater than zero.
func IsPositive(value float64) bool {
	return value > 0
}

// IsNonNegative checks if a float is zero or greater.
func IsNonNegative(value float64) bool {
	return value >= 0
}

// FloatInRange checks if a float is within the specified range (inclusive).
func FloatInRange(value, min, max float64) bool {
	return value >= min && value <= max
}

// --- Float Validators ---

// FloatMinValue validates that a float is at least the minimum value.
func FloatMinValue(field string, value, min float64) ValidationError {
	if !MinValueFloat(value, min) {
		return ValidationError{
			Field:   field,
			Rule:    "MinValue",
			Message: fmt.Sprintf("must be at least %.2f", min),
			Params:  map[string]any{"min": min},
		}
	}
	return ValidationError{}
}

// FloatMaxValue validates that a float does not exceed the maximum value.
func FloatMaxValue(field string, value, max float64) ValidationError {
	if !MaxValueFloat(value, max) {
		return ValidationError{
			Field:   field,
			Rule:    "MaxValue",
			Message: fmt.Sprintf("must be at most %.2f", max),
			Params:  map[string]any{"max": max},
		}
	}
	return ValidationError{}
}

// FloatPositive validates that a float is greater than zero.
func FloatPositive(field string, value float64) ValidationError {
	if !IsPositive(value) {
		return ValidationError{
			Field:   field,
			Rule:    "Positive",
			Message: "must be greater than zero",
		}
	}
	return ValidationError{}
}

// FloatNonNegative validates that a float is zero or greater.
func FloatNonNegative(field string, value float64) ValidationError {
	if !IsNonNegative(value) {
		return ValidationError{
			Field:   field,
			Rule:    "NonNegative",
			Message: "must be zero or greater",
		}
	}
	return ValidationError{}
}

// FloatInRangeValidator validates that a float is within the specified range.
func FloatInRangeValidator(field string, value, min, max float64) ValidationError {
	if !FloatInRange(value, min, max) {
		return ValidationError{
			Field:   field,
			Rule:    "InRange",
			Message: fmt.Sprintf("must be between %.2f and %.2f", min, max),
			Params:  map[string]any{"min": min, "max": max},
		}
	}
	return ValidationError{}
}
