package hm

import (
	"time"

	"github.com/google/uuid"
)

// Identifiable ensures that a model has a unique identifier.
type Identifiable interface {
	ID() uuid.UUID
}

// Lifecycle hooks for models.
type Lifecycle interface {
	BeforeCreate()
	BeforeUpdate()
}

// GenerateNewID generates a new UUID.
func GenerateNewID() uuid.UUID {
	return uuid.New()
}

// SetAuditFieldsBeforeCreate sets the initial timestamps and createdBy/updatedBy for a model.
// It expects pointers to the model's audit fields.
func SetAuditFieldsBeforeCreate(
	createdAt, updatedAt *time.Time,
	createdBy, updatedBy *uuid.UUID,
) {
	now := time.Now().UTC()
	*createdAt = now
	*updatedAt = now
	// TODO: Extract CreatedBy from context and set *createdBy
	// TODO: Extract UpdatedBy from context and set *updatedBy
}

// SetAuditFieldsBeforeUpdate updates the UpdatedAt timestamp and UpdatedBy for a model.
// It expects pointers to the model's audit fields.
func SetAuditFieldsBeforeUpdate(
	updatedAt *time.Time,
	updatedBy *uuid.UUID,
) {
	*updatedAt = time.Now().UTC()
	// TODO: Extract UpdatedBy from context
	// TODO: Extract UpdatedBy from context and set *updatedBy
}
