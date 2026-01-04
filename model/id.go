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
