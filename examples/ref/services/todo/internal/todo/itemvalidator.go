package todo

import (
	"context"

	"github.com/adrianpk/hatmax/pkg/lib/hm"
	"github.com/google/uuid"
)

// ValidateCreateItem validates a Item for creation.
func ValidateCreateItem(ctx context.Context, model Item) []hm.ValidationError {
	var errors []hm.ValidationError
	// Validations for field text
	if model.Text == "" {
		errors = append(errors, hm.ValidationError{Field: "text", Code: "required", Message: "text is required"})
	}

	return errors
}

// ValidateUpdateItem validates a Item for update.
func ValidateUpdateItem(ctx context.Context, id uuid.UUID, model Item) []hm.ValidationError {
	var errors []hm.ValidationError

	if !hm.IsRequiredUUID(id) {
		errors = append(errors, hm.ValidationError{
			Field:   "id",
			Code:    "required",
			Message: "ID is required for update",
		})
	}

	errors = append(errors, ValidateCreateItem(ctx, model)...)

	return errors
}

// ValidateDeleteItem validates a Item for deletion.
func ValidateDeleteItem(ctx context.Context, id uuid.UUID) []hm.ValidationError {
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
