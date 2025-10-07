package hm

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// ValidationError represents a validation error for a specific field.
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ValidationErrors is a slice of ValidationError.
type ValidationErrors []ValidationError

// Error implements the error interface for ValidationErrors.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	return "validation failed"
}

// HasErrors returns true if there are any validation errors.
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// Validator defines the interface for a model validator.
type Validator interface {
	Validate(ctx context.Context, model interface{}) ValidationErrors
}

// NoopValidator is a validator that performs no validation.
type NoopValidator struct{}

// Validate always returns nil for NoopValidator.
func (nv *NoopValidator) Validate(ctx context.Context, model interface{}) ValidationErrors {
	return nil
}

// FakeValidator is a test validator that can return predefined errors.
type FakeValidator struct {
	Errors ValidationErrors
}

// Validate returns the predefined errors for FakeValidator.
func (fv *FakeValidator) Validate(ctx context.Context, model interface{}) ValidationErrors {
	return fv.Errors
}

// NilUUID represents an empty UUID.
var NilUUID = uuid.Nil

// IsRequired checks if a string value is required (not empty or whitespace).
func IsRequired(value string) bool {
	return strings.TrimSpace(value) != ""
}

// IsRequiredUUID checks if a UUID value is required (not NilUUID).
func IsRequiredUUID(value uuid.UUID) bool {
	return value != NilUUID
}

// MinLength checks if a string value meets the minimum length requirement.
func MinLength(value string, min int) bool {
	return len(value) >= min
}

// MaxLength checks if a string value meets the maximum length requirement.
func MaxLength(value string, max int) bool {
	return len(value) <= max
}

// IsEmail checks if a string value is a valid email address.
func IsEmail(value string) bool {
	if value == "" {
		return true // Empty string is considered valid if not required
	}
	return regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).MatchString(value)
}

// MinValueInt checks if an int value meets the minimum value requirement.
func MinValueInt(value, min int) bool {
	return value >= min
}

// MaxValueInt checks if an int value meets the maximum value requirement.
func MaxValueInt(value, max int) bool {
	return value <= max
}
