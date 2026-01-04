package model

import "testing"

func TestHasRole(t *testing.T) {
	tests := []struct {
		name     string
		roles    []string
		checkFor string
		expected bool
	}{
		{
			name:     "user has role",
			roles:    []string{"user", "admin"},
			checkFor: "admin",
			expected: true,
		},
		{
			name:     "user does not have role",
			roles:    []string{"user"},
			checkFor: "admin",
			expected: false,
		},
		{
			name:     "superadmin has any role",
			roles:    []string{"superadmin"},
			checkFor: "admin",
			expected: true,
		},
		{
			name:     "superadmin with other roles",
			roles:    []string{"superadmin", "user"},
			checkFor: "nonexistent",
			expected: true,
		},
		{
			name:     "empty roles",
			roles:    []string{},
			checkFor: "admin",
			expected: false,
		},
		{
			name:     "check for empty role",
			roles:    []string{"user"},
			checkFor: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasRole(tt.roles, tt.checkFor)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
