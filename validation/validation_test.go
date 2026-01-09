package validation

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with plus",
			email:   "user+tag@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with dots",
			email:   "first.last@example.com",
			wantErr: nil,
		},
		{
			name:    "email too long",
			email:   strings.Repeat("a", 250) + "@example.com",
			wantErr: ErrEmailTooLong,
		},
		{
			name:    "local part too long",
			email:   strings.Repeat("a", 65) + "@example.com",
			wantErr: ErrEmailLocalTooLong,
		},
		{
			name:    "missing at sign",
			email:   "userexample.com",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "multiple at signs",
			email:   "user@@example.com",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "missing domain",
			email:   "user@",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "missing local part",
			email:   "@example.com",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "missing TLD",
			email:   "user@example",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "TLD too short",
			email:   "user@example.c",
			wantErr: ErrEmailInvalid,
		},
		{
			name:    "invalid characters",
			email:   "user name@example.com",
			wantErr: ErrEmailInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)

			if err != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "valid password",
			password: "SecurePass123!",
			wantErr:  nil,
		},
		{
			name:     "valid with all characters",
			password: "Aa1!Aa1!",
			wantErr:  nil,
		},
		{
			name:     "password too short",
			password: "Aa1!",
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "password too long",
			password: strings.Repeat("Aa1!", 33),
			wantErr:  ErrPasswordTooLong,
		},
		{
			name:     "no uppercase",
			password: "securepass123!",
			wantErr:  ErrPasswordNoUppercase,
		},
		{
			name:     "no lowercase",
			password: "SECUREPASS123!",
			wantErr:  ErrPasswordNoLowercase,
		},
		{
			name:     "no digit",
			password: "SecurePass!",
			wantErr:  ErrPasswordNoDigit,
		},
		{
			name:     "no special character",
			password: "SecurePass123",
			wantErr:  ErrPasswordNoSpecial,
		},
		{
			name:     "only lowercase and digit",
			password: "password123",
			wantErr:  ErrPasswordNoUppercase,
		},
		{
			name:     "only uppercase and digit",
			password: "PASSWORD123",
			wantErr:  ErrPasswordNoLowercase,
		},
		{
			name:     "symbols as special",
			password: "SecurePass123$",
			wantErr:  nil,
		},
		{
			name:     "punctuation as special",
			password: "SecurePass123,",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)

			if err != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  error
	}{
		{
			name:     "valid username",
			username: "user123",
			wantErr:  nil,
		},
		{
			name:     "valid with underscore",
			username: "user_name",
			wantErr:  nil,
		},
		{
			name:     "valid with hyphen",
			username: "user-name",
			wantErr:  nil,
		},
		{
			name:     "valid mixed",
			username: "User_123-name",
			wantErr:  nil,
		},
		{
			name:     "minimum length",
			username: "abc",
			wantErr:  nil,
		},
		{
			name:     "maximum length",
			username: strings.Repeat("a", 32),
			wantErr:  nil,
		},
		{
			name:     "too short",
			username: "ab",
			wantErr:  ErrUsernameTooShort,
		},
		{
			name:     "too long",
			username: strings.Repeat("a", 33),
			wantErr:  ErrUsernameTooLong,
		},
		{
			name:     "invalid characters spaces",
			username: "user name",
			wantErr:  ErrUsernameInvalid,
		},
		{
			name:     "invalid characters dot",
			username: "user.name",
			wantErr:  ErrUsernameInvalid,
		},
		{
			name:     "invalid characters special",
			username: "user@name",
			wantErr:  ErrUsernameInvalid,
		},
		{
			name:     "empty string",
			username: "",
			wantErr:  ErrUsernameTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)

			if err != tt.wantErr {
				t.Errorf("ValidateUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "lowercase already",
			email: "user@example.com",
			want:  "user@example.com",
		},
		{
			name:  "uppercase to lowercase",
			email: "USER@EXAMPLE.COM",
			want:  "user@example.com",
		},
		{
			name:  "mixed case",
			email: "User@Example.Com",
			want:  "user@example.com",
		},
		{
			name:  "with leading spaces",
			email: "  user@example.com",
			want:  "user@example.com",
		},
		{
			name:  "with trailing spaces",
			email: "user@example.com  ",
			want:  "user@example.com",
		},
		{
			name:  "with both spaces",
			email: "  user@example.com  ",
			want:  "user@example.com",
		},
		{
			name:  "uppercase with spaces",
			email: "  USER@EXAMPLE.COM  ",
			want:  "user@example.com",
		},
		{
			name:  "empty string",
			email: "",
			want:  "",
		},
		{
			name:  "only spaces",
			email: "   ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeEmail(tt.email)

			if got != tt.want {
				t.Errorf("NormalizeEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}
