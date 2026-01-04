package auth

import "time"

// User represents an authenticated user in the system.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Session represents a user session.
type Session struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}
