package auth

import (
	"testing"
	"time"
)

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

func TestUserNeedsTOTPSetup(t *testing.T) {
	tests := []struct {
		name       string
		totpSecret string
		want       bool
	}{
		{
			name:       "no secret needs setup",
			totpSecret: "",
			want:       true,
		},
		{
			name:       "has secret no setup needed",
			totpSecret: "JBSWY3DPEHPK3PXP",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{TOTPSecret: tt.totpSecret}

			got := u.NeedsTOTPSetup()
			if got != tt.want {
				t.Errorf("NeedsTOTPSetup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserInTOTPGracePeriod(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		createdAt time.Time
		graceDays int
		want      bool
	}{
		{
			name:      "within grace period",
			createdAt: now.AddDate(0, 0, -3),
			graceDays: 7,
			want:      true,
		},
		{
			name:      "outside grace period",
			createdAt: now.AddDate(0, 0, -10),
			graceDays: 7,
			want:      false,
		},
		{
			name:      "exactly at grace period end",
			createdAt: now.AddDate(0, 0, -7),
			graceDays: 7,
			want:      false,
		},
		{
			name:      "zero grace days",
			createdAt: now,
			graceDays: 0,
			want:      false,
		},
		{
			name:      "negative grace days",
			createdAt: now,
			graceDays: -1,
			want:      false,
		},
		{
			name:      "created today",
			createdAt: now,
			graceDays: 7,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{CreatedAt: tt.createdAt}

			got := u.InTOTPGracePeriod(tt.graceDays)
			if got != tt.want {
				t.Errorf("InTOTPGracePeriod(%d) = %v, want %v", tt.graceDays, got, tt.want)
			}
		})
	}
}
