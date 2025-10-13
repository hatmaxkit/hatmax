package todo

import (
	"context"

	"github.com/adrianpk/hatmax/pkg/lib/hm"
	"github.com/google/uuid"
)

// ValidateCreateTag validates a Tag for creation.
func ValidateCreateTag(ctx context.Context, model Tag) []hm.ValidationError {
	var errors []hm.ValidationError
	// Validations for field name
	if model.Name == "" {
		errors = append(errors, hm.ValidationError{Field: "name", Code: "required", Message: "name is required"})
	}

	return errors
}

// ValidateUpdateTag validates a Tag for update.
func ValidateUpdateTag(ctx context.Context, id uuid.UUID, model Tag) []hm.ValidationError {
	var errors []hm.ValidationError

	if !hm.IsRequiredUUID(id) {
		errors = append(errors, hm.ValidationError{
			Field:   "id",
			Code:    "required",
			Message: "ID is required for update",
		})
	}

	errors = append(errors, ValidateCreateTag(ctx, model)...)

	return errors
}

// ValidateDeleteTag validates a Tag for deletion.
func ValidateDeleteTag(ctx context.Context, id uuid.UUID) []hm.ValidationError {
	var errors []hm.ValidationError

	if !hm.IsRequiredUUID(id) {
		errors = append(errors, hm.ValidationError{
			Field:   "id",
			Code:    "required",
			Message: "ID is required for deletion",
		})
	}

	return errors
}
