package todo

import (
	"time"

	"github.com/google/uuid"
)

// Item is a child of an aggregate root.
type Item struct {
	ID        uuid.UUID `json:"id"`
	Text      string    `json:"text"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// NewItem creates a new Item with a generated ID.
func NewItem() *Item {
	return &Item{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// GetID returns the ID of the Item.
func (c *Item) GetID() uuid.UUID {
	return c.ID
}

// SetID sets the ID of the Item.
func (c *Item) SetID(id uuid.UUID) {
	c.ID = id
}

// EnsureID ensures the child model has a valid ID.
func (c *Item) EnsureID() {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
}

// BeforeCreate sets creation timestamps.
func (c *Item) BeforeCreate() {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
}

// BeforeUpdate sets update timestamps.
func (c *Item) BeforeUpdate() {
	c.UpdatedAt = time.Now()
}
