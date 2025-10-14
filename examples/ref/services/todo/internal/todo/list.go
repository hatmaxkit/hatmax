package todo

import (
	"time"

	"github.com/adrianpk/hatmax-ref/pkg/lib/core"
	"github.com/google/uuid"
)

// List is the aggregate root for the List domain.
type List struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by"`
	Items       []Item    `json:"items"`
	Tags        []Tag     `json:"tags"`
}

// GetID returns the ID of the List (implements Identifiable interface).
func (a *List) GetID() uuid.UUID {
	return a.ID
}

// ResourceType returns the resource type for URL generation.
func (a *List) ResourceType() string {
	return "list"
}

// SetID sets the ID of the List.
func (a *List) SetID(id uuid.UUID) {
	a.ID = id
}

// NewList creates a new List with a generated ID and initial version.
func NewList() *List {
	return &List{
		ID: core.GenerateNewID(),
	}
}

// EnsureID ensures the aggregate root has a valid ID.
func (a *List) EnsureID() {
	if a.ID == uuid.Nil {
		a.ID = core.GenerateNewID()
	}
}

// BeforeCreate sets creation timestamps and version.
func (a *List) BeforeCreate() {
	a.EnsureID()
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
}

// BeforeUpdate sets update timestamps and increments version.
func (a *List) BeforeUpdate() {
	a.UpdatedAt = time.Now()
}
