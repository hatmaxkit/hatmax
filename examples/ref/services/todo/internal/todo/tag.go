package todo

import (
	"time"

	"github.com/google/uuid"
)

// Tag is a child of an aggregate root.
type Tag struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// NewTag creates a new Tag with a generated ID.
func NewTag() *Tag {
	return &Tag{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// GetID returns the ID of the Tag.
func (c *Tag) GetID() uuid.UUID {
	return c.ID
}

// SetID sets the ID of the Tag.
func (c *Tag) SetID(id uuid.UUID) {
	c.ID = id
}

// EnsureID ensures the child model has a valid ID.
func (c *Tag) EnsureID() {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
}

// BeforeCreate sets creation timestamps.
func (c *Tag) BeforeCreate() {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
}

// BeforeUpdate sets update timestamps.
func (c *Tag) BeforeUpdate() {
	c.UpdatedAt = time.Now()
}
