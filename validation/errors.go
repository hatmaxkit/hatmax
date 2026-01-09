package validation

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ValidationError struct {
	Field   string
	Message string
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
