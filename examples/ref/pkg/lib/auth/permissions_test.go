package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEvaluatePermissions(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	roleID := uuid.New()
	
	adminRole := Role{
		ID:          roleID,
		Name:        "admin",
		Permissions: []string{"users:read", "users:write", "orders:read", "orders:write"},
	}
	
	roles := []Role{adminRole}

	tests := []struct {
		name       string
		grants     []Grant
		permission string
		scope      Scope
		expected   bool
	}{
		{
			name: "direct permission grant matches",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "users:read",
					Scope:     Scope{Type: "team", ID: "123"},
				},
			},
			permission: "users:read",
			scope:      Scope{Type: "team", ID: "123"},
			expected:   true,
		},
		{
			name: "direct permission grant wrong permission",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "users:read",
					Scope:     Scope{Type: "team", ID: "123"},
				},
			},
			permission: "users:write",
			scope:      Scope{Type: "team", ID: "123"},
			expected:   false,
		},
		{
			name: "role grant with matching permission",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypeRole,
					Value:     roleID.String(),
					Scope:     Scope{Type: "team", ID: "123"},
				},
			},
			permission: "users:read",
			scope:      Scope{Type: "team", ID: "123"},
			expected:   true,
		},
		{
			name: "role grant without matching permission",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypeRole,
					Value:     roleID.String(),
					Scope:     Scope{Type: "team", ID: "123"},
				},
			},
			permission: "system:admin",
			scope:      Scope{Type: "team", ID: "123"},
			expected:   false,
		},
		{
			name: "global scope matches any request scope",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "users:read",
					Scope:     Scope{Type: "global", ID: ""},
				},
			},
			permission: "users:read",
			scope:      Scope{Type: "team", ID: "456"},
			expected:   true,
		},
		{
			name: "wrong scope does not match",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "users:read",
					Scope:     Scope{Type: "team", ID: "123"},
				},
			},
			permission: "users:read",
			scope:      Scope{Type: "team", ID: "456"},
			expected:   false,
		},
		{
			name: "expired grant does not match",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "users:read",
					Scope:     Scope{Type: "team", ID: "123"},
					ExpiresAt: &[]time.Time{now.Add(-time.Hour)}[0],
				},
			},
			permission: "users:read",
			scope:      Scope{Type: "team", ID: "123"},
			expected:   false,
		},
		{
			name: "future expiration matches",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "users:read",
					Scope:     Scope{Type: "team", ID: "123"},
					ExpiresAt: &[]time.Time{now.Add(time.Hour)}[0],
				},
			},
			permission: "users:read",
			scope:      Scope{Type: "team", ID: "123"},
			expected:   true,
		},
		{
			name: "no expiration matches",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "users:read",
					Scope:     Scope{Type: "team", ID: "123"},
					ExpiresAt: nil,
				},
			},
			permission: "users:read",
			scope:      Scope{Type: "team", ID: "123"},
			expected:   true,
		},
		{
			name:       "no grants",
			grants:     []Grant{},
			permission: "users:read",
			scope:      Scope{Type: "team", ID: "123"},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluatePermissions(tt.grants, roles, tt.permission, tt.scope, now)
			if result != tt.expected {
				t.Errorf("EvaluatePermissions() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestScopeMatches(t *testing.T) {
	tests := []struct {
		name         string
		grantScope   Scope
		requestScope Scope
		expected     bool
	}{
		{
			name:         "global scope matches any request",
			grantScope:   Scope{Type: "global", ID: ""},
			requestScope: Scope{Type: "team", ID: "123"},
			expected:     true,
		},
		{
			name:         "global scope matches global request",
			grantScope:   Scope{Type: "global", ID: ""},
			requestScope: Scope{Type: "global", ID: ""},
			expected:     true,
		},
		{
			name:         "exact scope match",
			grantScope:   Scope{Type: "team", ID: "123"},
			requestScope: Scope{Type: "team", ID: "123"},
			expected:     true,
		},
		{
			name:         "same type different ID",
			grantScope:   Scope{Type: "team", ID: "123"},
			requestScope: Scope{Type: "team", ID: "456"},
			expected:     false,
		},
		{
			name:         "different type same ID",
			grantScope:   Scope{Type: "team", ID: "123"},
			requestScope: Scope{Type: "organization", ID: "123"},
			expected:     false,
		},
		{
			name:         "different type and ID",
			grantScope:   Scope{Type: "team", ID: "123"},
			requestScope: Scope{Type: "organization", ID: "456"},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScopeMatches(tt.grantScope, tt.requestScope)
			if result != tt.expected {
				t.Errorf("ScopeMatches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetRolePermissions(t *testing.T) {
	roleID1 := uuid.New()
	roleID2 := uuid.New()
	
	roles := []Role{
		{
			ID:          roleID1,
			Name:        "admin",
			Permissions: []string{"users:read", "users:write"},
		},
		{
			ID:          roleID2,
			Name:        "viewer",
			Permissions: []string{"users:read"},
		},
	}

	tests := []struct {
		name     string
		roleID   string
		expected []string
	}{
		{
			name:     "existing role returns permissions",
			roleID:   roleID1.String(),
			expected: []string{"users:read", "users:write"},
		},
		{
			name:     "different existing role",
			roleID:   roleID2.String(),
			expected: []string{"users:read"},
		},
		{
			name:     "non-existing role returns nil",
			roleID:   uuid.New().String(),
			expected: nil,
		},
		{
			name:     "empty role ID returns nil",
			roleID:   "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRolePermissions(roles, tt.roleID)
			if !equalStringSlices(result, tt.expected) {
				t.Errorf("GetRolePermissions() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterValidGrants(t *testing.T) {
	now := time.Now()
	userID := uuid.New()

	tests := []struct {
		name     string
		grants   []Grant
		expected int
	}{
		{
			name: "all valid grants",
			grants: []Grant{
				{ID: uuid.New(), UserID: userID, ExpiresAt: nil},
				{ID: uuid.New(), UserID: userID, ExpiresAt: &[]time.Time{now.Add(time.Hour)}[0]},
			},
			expected: 2,
		},
		{
			name: "mixed valid and expired grants",
			grants: []Grant{
				{ID: uuid.New(), UserID: userID, ExpiresAt: nil},
				{ID: uuid.New(), UserID: userID, ExpiresAt: &[]time.Time{now.Add(-time.Hour)}[0]},
				{ID: uuid.New(), UserID: userID, ExpiresAt: &[]time.Time{now.Add(time.Hour)}[0]},
			},
			expected: 2,
		},
		{
			name: "all expired grants",
			grants: []Grant{
				{ID: uuid.New(), UserID: userID, ExpiresAt: &[]time.Time{now.Add(-time.Hour)}[0]},
				{ID: uuid.New(), UserID: userID, ExpiresAt: &[]time.Time{now.Add(-time.Minute)}[0]},
			},
			expected: 0,
		},
		{
			name:     "empty grants",
			grants:   []Grant{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterValidGrants(tt.grants, now)
			if len(result) != tt.expected {
				t.Errorf("FilterValidGrants() returned %d grants, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestHasAnyPermission(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	roleID := uuid.New()
	
	adminRole := Role{
		ID:          roleID,
		Name:        "admin",
		Permissions: []string{"users:read", "users:write"},
	}
	
	roles := []Role{adminRole}

	tests := []struct {
		name        string
		grants      []Grant
		permissions []string
		scope       Scope
		expected    bool
	}{
		{
			name: "has one of multiple permissions",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "users:read",
					Scope:     Scope{Type: "team", ID: "123"},
				},
			},
			permissions: []string{"users:read", "users:write", "orders:read"},
			scope:       Scope{Type: "team", ID: "123"},
			expected:    true,
		},
		{
			name: "has none of multiple permissions",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "orders:read",
					Scope:     Scope{Type: "team", ID: "123"},
				},
			},
			permissions: []string{"users:read", "users:write"},
			scope:       Scope{Type: "team", ID: "123"},
			expected:    false,
		},
		{
			name:        "empty permissions list",
			grants:      []Grant{},
			permissions: []string{},
			scope:       Scope{Type: "team", ID: "123"},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasAnyPermission(tt.grants, roles, tt.permissions, tt.scope, now)
			if result != tt.expected {
				t.Errorf("HasAnyPermission() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasAllPermissions(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	roleID := uuid.New()
	
	adminRole := Role{
		ID:          roleID,
		Name:        "admin",
		Permissions: []string{"users:read", "users:write", "orders:read"},
	}
	
	roles := []Role{adminRole}

	tests := []struct {
		name        string
		grants      []Grant
		permissions []string
		scope       Scope
		expected    bool
	}{
		{
			name: "has all permissions via role",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypeRole,
					Value:     roleID.String(),
					Scope:     Scope{Type: "team", ID: "123"},
				},
			},
			permissions: []string{"users:read", "users:write"},
			scope:       Scope{Type: "team", ID: "123"},
			expected:    true,
		},
		{
			name: "missing one permission",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "users:read",
					Scope:     Scope{Type: "team", ID: "123"},
				},
			},
			permissions: []string{"users:read", "users:write"},
			scope:       Scope{Type: "team", ID: "123"},
			expected:    false,
		},
		{
			name:        "empty permissions list",
			grants:      []Grant{},
			permissions: []string{},
			scope:       Scope{Type: "team", ID: "123"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasAllPermissions(tt.grants, roles, tt.permissions, tt.scope, now)
			if result != tt.expected {
				t.Errorf("HasAllPermissions() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsGlobalAdmin(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	roleID := uuid.New()
	
	adminRole := Role{
		ID:          roleID,
		Name:        "global_admin",
		Permissions: []string{"system:admin", "users:read", "users:write"},
	}
	
	roles := []Role{adminRole}

	tests := []struct {
		name     string
		grants   []Grant
		expected bool
	}{
		{
			name: "global admin via direct permission",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "system:admin",
					Scope:     Scope{Type: "global", ID: ""},
				},
			},
			expected: true,
		},
		{
			name: "global admin via role",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypeRole,
					Value:     roleID.String(),
					Scope:     Scope{Type: "global", ID: ""},
				},
			},
			expected: true,
		},
		{
			name: "team admin not global admin",
			grants: []Grant{
				{
					ID:        uuid.New(),
					UserID:    userID,
					GrantType: GrantTypePermission,
					Value:     "system:admin",
					Scope:     Scope{Type: "team", ID: "123"},
				},
			},
			expected: false,
		},
		{
			name:     "no grants",
			grants:   []Grant{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGlobalAdmin(tt.grants, roles, now)
			if result != tt.expected {
				t.Errorf("IsGlobalAdmin() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}