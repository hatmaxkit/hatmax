package todo

import (
	"time"

	"github.com/google/uuid"
	"github.com/adrianpk/hatmax/pkg/lib/hm"
)

// Tag represents a Tag.
//
// hatmax:model
type Tag struct {
	id        uuid.UUID `json:"id"`
	Name string `json:"name"`
	Color string `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy uuid.UUID `json:"created_by"`
	UpdatedBy uuid.UUID `json:"updated_by"`
}

// ID returns the ID of the Tag.
func (m *Tag) ID() uuid.UUID {
	return m.id
}

func (m *Tag) SetID(id uuid.UUID) {
	m.id = id
}

func (m *Tag) EnsureID() {
	if m.id == uuid.Nil {
		m.id = hm.GenerateNewID()
	}
}


// BeforeCreate sets the initial timestamps and createdBy for the model.
func (m *Tag) BeforeCreate() {
	hm.SetAuditFieldsBeforeCreate(&m.CreatedAt, &m.UpdatedAt, &m.CreatedBy, &m.UpdatedBy)
	if m.id == uuid.Nil {
		m.id = hm.GenerateNewID()
	}
}

// BeforeUpdate updates the UpdatedAt timestamp and UpdatedBy for the model.
func (m *Tag) BeforeUpdate() {
	hm.SetAuditFieldsBeforeUpdate(&m.UpdatedAt, &m.UpdatedBy)
}
