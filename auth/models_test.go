package auth

import "testing"

func TestUserHasRole(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		check string
		want  bool
	}{
		{
			name:  "has role",
			roles: []string{"admin", "user"},
			check: "admin",
			want:  true,
		},
		{
			name:  "does not have role",
			roles: []string{"user"},
			check: "admin",
			want:  false,
		},
		{
			name:  "empty roles",
			roles: []string{},
			check: "admin",
			want:  false,
		},
		{
			name:  "nil roles",
			roles: nil,
			check: "admin",
			want:  false,
		},
		{
			name:  "check empty string",
			roles: []string{"admin", ""},
			check: "",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Roles: tt.roles}
			got := u.HasRole(tt.check)
			if got != tt.want {
				t.Errorf("HasRole(%q) = %v, want %v", tt.check, got, tt.want)
			}
		})
	}
}

func TestUserHasAnyRole(t *testing.T) {
	tests := []struct {
		name   string
		roles  []string
		checks []string
		want   bool
	}{
		{
			name:   "has first role",
			roles:  []string{"admin"},
			checks: []string{"admin", "moderator"},
			want:   true,
		},
		{
			name:   "has second role",
			roles:  []string{"moderator"},
			checks: []string{"admin", "moderator"},
			want:   true,
		},
		{
			name:   "has both roles",
			roles:  []string{"admin", "moderator"},
			checks: []string{"admin", "moderator"},
			want:   true,
		},
		{
			name:   "has none of the roles",
			roles:  []string{"user"},
			checks: []string{"admin", "moderator"},
			want:   false,
		},
		{
			name:   "empty user roles",
			roles:  []string{},
			checks: []string{"admin"},
			want:   false,
		},
		{
			name:   "empty check roles",
			roles:  []string{"admin"},
			checks: []string{},
			want:   false,
		},
		{
			name:   "nil check roles",
			roles:  []string{"admin"},
			checks: nil,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Roles: tt.roles}
			got := u.HasAnyRole(tt.checks...)
			if got != tt.want {
				t.Errorf("HasAnyRole(%v) = %v, want %v", tt.checks, got, tt.want)
			}
		})
	}
}
