package todo

import (
	"time"

	"github.com/adrianpk/hatmax/pkg/lib/hm"
	"github.com/google/uuid"
)

// List is the aggregate root for the List domain.
type List struct {
	id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by"`
	Items       []Item    `json:"items"`
}

// ID returns the ID of the List.
func (a *List) ID() uuid.UUID {
	return a.id
}

// SetID sets the ID of the List.
func (a *List) SetID(id uuid.UUID) {
	a.id = id
}

// NewList creates a new List with a generated ID and initial version.
func NewList() *List {
	return &List{
		id: hm.GenerateNewID(),
	}
}

// EnsureID ensures the aggregate root has a valid ID.
func (a *List) EnsureID() {
	if a.id == uuid.Nil {
		a.id = hm.GenerateNewID()
	}
}

// BeforeCreate sets creation timestamps and version.
func (a *List) BeforeCreate() {
	a.EnsureID()
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	// TODO: Set CreatedBy
	// TODO: Set UpdatedBy
}

// BeforeUpdate sets update timestamps and increments version.
func (a *List) BeforeUpdate() {
	a.UpdatedAt = time.Now()
	// TODO: Set UpdatedBy
}
