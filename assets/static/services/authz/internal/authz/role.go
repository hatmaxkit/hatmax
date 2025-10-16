package authz

import (
	"time"

	"github.com/google/uuid"
	authpkg "github.com/username/repo/pkg/lib/auth"
)

// Role represents a role aggregate in the authorization domain
type Role struct {
	ID          uuid.UUID
	Name        string
	Permissions []string // Permission codes
	Status      authpkg.UserStatus
	CreatedAt   time.Time
	CreatedBy   string
	UpdatedAt   time.Time
	UpdatedBy   string
}

// NewRole creates a new Role
func NewRole() *Role {
	return &Role{
		Status: authpkg.UserStatusActive,
	}
}

// EnsureID ensures the role has an ID
func (r *Role) EnsureID() {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
}

// BeforeCreate sets up the role before creation
func (r *Role) BeforeCreate() {
	r.EnsureID()
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.Status == "" {
		r.Status = authpkg.UserStatusActive
	}
}

// BeforeUpdate sets up the role before update
func (r *Role) BeforeUpdate() {
	r.UpdatedAt = time.Now()
}

// IsActive returns true if the role is active
func (r *Role) IsActive() bool {
	return r.Status == authpkg.UserStatusActive
}

// HasPermission checks if role has a specific permission
func (r *Role) HasPermission(permission string) bool {
	for _, p := range r.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}