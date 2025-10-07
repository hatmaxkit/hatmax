package todo

import (
	"time"

	"github.com/adrianpk/hatmax/pkg/lib/hm"
	"github.com/google/uuid"
)

// Item represents a Item.
//
// hatmax:model
type Item struct {
	id        uuid.UUID `json:"id"`
	Text      string    `json:"text"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy uuid.UUID `json:"created_by"`
	UpdatedBy uuid.UUID `json:"updated_by"`
}

// ID returns the ID of the Item.
func (m *Item) ID() uuid.UUID {
	return m.id
}

func (m *Item) SetID(id uuid.UUID) {
	m.id = id
}

func (m *Item) EnsureID() {
	if m.id == uuid.Nil {
		m.id = hm.GenerateNewID()
	}
}

// BeforeCreate sets the initial timestamps and createdBy for the model.
func (m *Item) BeforeCreate() {
	hm.SetAuditFieldsBeforeCreate(&m.CreatedAt, &m.UpdatedAt, &m.CreatedBy, &m.UpdatedBy)
	if m.id == uuid.Nil {
		m.id = hm.GenerateNewID()
	}
}

// BeforeUpdate updates the UpdatedAt timestamp and UpdatedBy for the model.
func (m *Item) BeforeUpdate() {
	hm.SetAuditFieldsBeforeUpdate(&m.UpdatedAt, &m.UpdatedBy)
}
