package model

import "time"

// Now returns the current UTC time.
func Now() time.Time {
	return time.Now().UTC()
}

// SetCreated sets created_at and updated_at to current time.
func SetCreated(createdAt, updatedAt *time.Time) {
	now := Now()
	*createdAt = now
	*updatedAt = now
}

// SetUpdated sets updated_at to current time.
func SetUpdated(updatedAt *time.Time) {
	*updatedAt = Now()
}
