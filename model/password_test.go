package model

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "simple password",
			password: "password123",
		},
		{
			name:     "complex password",
			password: "C0mpl3x!P@ssw0rd",
		},
		{
			name:     "long password within bcrypt limit",
			password: strings.Repeat("a", 70),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if hash == "" {
				t.Error("expected hash to be generated")
			}

			if hash == tt.password {
				t.Error("expected hash to be different from password")
			}
		})
	}
}

func TestComparePassword(t *testing.T) {
	password := "password123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("cannot hash password: %v", err)
	}

	tests := []struct {
		name     string
		hash     string
		password string
		expected bool
	}{
		{
			name:     "correct password",
			hash:     hash,
			password: password,
			expected: true,
		},
		{
			name:     "incorrect password",
			hash:     hash,
			password: "wrongpassword",
			expected: false,
		},
		{
			name:     "empty password",
			hash:     hash,
			password: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComparePassword(tt.hash, tt.password)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "short password",
			length: 8,
		},
		{
			name:   "medium password",
			length: 16,
		},
		{
			name:   "long password",
			length: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := GenerateRandomPassword(tt.length)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(password) != tt.length {
				t.Errorf("expected length %d, got %d", tt.length, len(password))
			}
		})
	}
}

func TestGenerateRandomPasswordUniqueness(t *testing.T) {
	pwd1, err := GenerateRandomPassword(16)
	if err != nil {
		t.Fatalf("cannot generate password: %v", err)
	}

	pwd2, err := GenerateRandomPassword(16)
	if err != nil {
		t.Fatalf("cannot generate password: %v", err)
	}

	if pwd1 == pwd2 {
		t.Error("expected unique passwords")
	}
}
