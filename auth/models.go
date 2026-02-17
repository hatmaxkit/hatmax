package auth

import "time"

// User represents an authenticated user in the system.
type User struct {
	ID             string
	Email          string
	PasswordHash   string
	Roles          []string
	Active         bool
	TOTPSecret     string
	TOTPEnabled    bool
	TOTPVerifiedAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
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

// NeedsTOTPSetup returns true if the user has not set up TOTP yet.
func (u *User) NeedsTOTPSetup() bool {
	return u.TOTPSecret == ""
}

// InTOTPGracePeriod returns true if the user is within the grace period.
func (u *User) InTOTPGracePeriod(days int) bool {
	if days <= 0 {
		return false
	}
	gracePeriodEnd := u.CreatedAt.AddDate(0, 0, days)
	return time.Now().Before(gracePeriodEnd)
}

// Session represents a user session.
type Session struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}
