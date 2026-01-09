package auth

import "time"

// User represents an authenticated user in the system.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Roles        []string
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// HasRole checks if the user has the specified role.
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the specified roles.
func (u *User) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if u.HasRole(role) {
			return true
		}
	}
	return false
}

// Session represents a user session.
type Session struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}
