package model

import "github.com/google/uuid"

// GenerateID generates a new UUID and stores it in the destination pointer.
func GenerateID(dest *string) {
	*dest = uuid.New().String()
}

// NewID generates and returns a new UUID string.
func NewID() string {
	return uuid.New().String()
}

// ParseID parses a string into a UUID.
func ParseID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// NullUUID creates a uuid.NullUUID from a string pointer.
func NullUUID(s *string) uuid.NullUUID {
	if s == nil || *s == "" {
		return uuid.NullUUID{Valid: false}
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}

// FromNullUUID converts uuid.NullUUID to *string.
func FromNullUUID(n uuid.NullUUID) *string {
	if !n.Valid {
		return nil
	}
	s := n.UUID.String()
	return &s
}
