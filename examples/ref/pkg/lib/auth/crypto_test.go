package auth

import (
	"bytes"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic lowercase",
			input:    "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "uppercase domain",
			input:    "user@EXAMPLE.COM",
			expected: "user@example.com",
		},
		{
			name:     "mixed case",
			input:    "User@Example.COM",
			expected: "user@example.com",
		},
		{
			name:     "with leading spaces",
			input:    "  user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "with trailing spaces",
			input:    "user@example.com  ",
			expected: "user@example.com",
		},
		{
			name:     "with leading and trailing spaces",
			input:    "  user@example.com  ",
			expected: "user@example.com",
		},
		{
			name:     "complex email with subdomain",
			input:    "  Test.User@mail.EXAMPLE.org  ",
			expected: "test.user@mail.example.org",
		},
		{
			name:     "email with plus addressing",
			input:    "USER+tag@EXAMPLE.com",
			expected: "user+tag@example.com",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeEmail(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestComputeLookupHash(t *testing.T) {
	key := []byte("test-lookup-key-32-bytes-long!!!")

	tests := []struct {
		name  string
		email string
		key   []byte
	}{
		{
			name:  "basic email",
			email: "user@example.com",
			key:   key,
		},
		{
			name:  "complex email",
			email: "test.user+tag@subdomain.example.org",
			key:   key,
		},
		{
			name:  "empty email",
			email: "",
			key:   key,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := ComputeLookupHash(tt.email, tt.key)
			hash2 := ComputeLookupHash(tt.email, tt.key)

			if !bytes.Equal(hash1, hash2) {
				t.Error("ComputeLookupHash should be deterministic")
			}

			if len(hash1) != 32 {
				t.Errorf("ComputeLookupHash returned %d bytes, want 32 (SHA256)", len(hash1))
			}

			if bytes.Equal(hash1, make([]byte, 32)) {
				t.Error("ComputeLookupHash returned all zeros")
			}
		})
	}
}

func TestComputeLookupHashDifferentInputs(t *testing.T) {
	key := []byte("test-lookup-key-32-bytes-long!!!")

	hash1 := ComputeLookupHash("user1@example.com", key)
	hash2 := ComputeLookupHash("user2@example.com", key)

	if bytes.Equal(hash1, hash2) {
		t.Error("Different emails should produce different hashes")
	}
}

func TestComputeLookupHashDifferentKeys(t *testing.T) {
	email := "user@example.com"
	key1 := []byte("key1-32-bytes-long-for-testing!!!")
	key2 := []byte("key2-32-bytes-long-for-testing!!!")

	hash1 := ComputeLookupHash(email, key1)
	hash2 := ComputeLookupHash(email, key2)

	if bytes.Equal(hash1, hash2) {
		t.Error("Same email with different keys should produce different hashes")
	}
}

func TestGeneratePasswordSalt(t *testing.T) {
	salt1 := GeneratePasswordSalt()
	salt2 := GeneratePasswordSalt()

	if len(salt1) != 32 {
		t.Errorf("GeneratePasswordSalt returned %d bytes, want 32", len(salt1))
	}

	if len(salt2) != 32 {
		t.Errorf("GeneratePasswordSalt returned %d bytes, want 32", len(salt2))
	}

	if bytes.Equal(salt1, salt2) {
		t.Error("GeneratePasswordSalt should generate different salts")
	}

	if bytes.Equal(salt1, make([]byte, 32)) {
		t.Error("GeneratePasswordSalt returned all zeros")
	}
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		salt     []byte
	}{
		{
			name:     "basic password",
			password: "password123",
			salt:     []byte("salt-32-bytes-long-for-testing!!"),
		},
		{
			name:     "complex password",
			password: "Complex!Password@123",
			salt:     []byte("another-salt-32-bytes-long-test!"),
		},
		{
			name:     "unicode password",
			password: "пароль123",
			salt:     []byte("unicode-salt-32-bytes-long-test!"),
		},
		{
			name:     "empty password",
			password: "",
			salt:     []byte("empty-pwd-salt-32-bytes-long-te!"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := HashPassword([]byte(tt.password), tt.salt)
			hash2 := HashPassword([]byte(tt.password), tt.salt)

			if !bytes.Equal(hash1, hash2) {
				t.Error("HashPassword should be deterministic")
			}

			if len(hash1) != 32 {
				t.Errorf("HashPassword returned %d bytes, want 32", len(hash1))
			}

			if bytes.Equal(hash1, make([]byte, 32)) {
				t.Error("HashPassword returned all zeros")
			}
		})
	}
}

func TestHashPasswordDifferentInputs(t *testing.T) {
	salt := []byte("same-salt-32-bytes-long-testing!")

	hash1 := HashPassword([]byte("password1"), salt)
	hash2 := HashPassword([]byte("password2"), salt)

	if bytes.Equal(hash1, hash2) {
		t.Error("Different passwords should produce different hashes")
	}
}

func TestHashPasswordDifferentSalts(t *testing.T) {
	password := []byte("same-password")
	salt1 := []byte("salt1-32-bytes-long-for-testing!")
	salt2 := []byte("salt2-32-bytes-long-for-testing!")

	hash1 := HashPassword(password, salt1)
	hash2 := HashPassword(password, salt2)

	if bytes.Equal(hash1, hash2) {
		t.Error("Same password with different salts should produce different hashes")
	}
}

func TestVerifyPasswordHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		salt     []byte
	}{
		{
			name:     "basic password",
			password: "password123",
			salt:     []byte("test-salt-32-bytes-long-testing!"),
		},
		{
			name:     "complex password",
			password: "Complex!Password@123",
			salt:     []byte("complex-salt-32-bytes-long-test!"),
		},
		{
			name:     "unicode password",
			password: "пароль123",
			salt:     []byte("unicode-salt-32-bytes-long-test!"),
		},
		{
			name:     "empty password",
			password: "",
			salt:     []byte("empty-pwd-salt-32-bytes-long-te!"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashPassword([]byte(tt.password), tt.salt)

			if !VerifyPasswordHash([]byte(tt.password), hash, tt.salt) {
				t.Error("VerifyPasswordHash should return true for correct password")
			}

			if VerifyPasswordHash([]byte("wrong-password"), hash, tt.salt) {
				t.Error("VerifyPasswordHash should return false for wrong password")
			}
		})
	}
}

func TestVerifyPasswordHashWrongSalt(t *testing.T) {
	password := []byte("test-password")
	salt1 := []byte("salt1-32-bytes-long-for-testing!")
	salt2 := []byte("salt2-32-bytes-long-for-testing!")

	hash := HashPassword(password, salt1)

	if VerifyPasswordHash(password, hash, salt2) {
		t.Error("VerifyPasswordHash should return false for wrong salt")
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{name: "16 bytes", length: 16},
		{name: "32 bytes", length: 32},
		{name: "64 bytes", length: 64},
		{name: "1 byte", length: 1},
		{name: "0 bytes", length: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes1 := GenerateRandomBytes(tt.length)
			bytes2 := GenerateRandomBytes(tt.length)

			if len(bytes1) != tt.length {
				t.Errorf("GenerateRandomBytes(%d) returned %d bytes, want %d", tt.length, len(bytes1), tt.length)
			}

			if len(bytes2) != tt.length {
				t.Errorf("GenerateRandomBytes(%d) returned %d bytes, want %d", tt.length, len(bytes2), tt.length)
			}

			if tt.length > 0 && bytes.Equal(bytes1, bytes2) {
				t.Error("GenerateRandomBytes should generate different bytes")
			}

			if tt.length > 0 && bytes.Equal(bytes1, make([]byte, tt.length)) {
				t.Error("GenerateRandomBytes should not return all zeros")
			}
		})
	}
}

func BenchmarkNormalizeEmail(b *testing.B) {
	email := "  Test.User@EXAMPLE.COM  "
	for i := 0; i < b.N; i++ {
		NormalizeEmail(email)
	}
}

func BenchmarkComputeLookupHash(b *testing.B) {
	key := []byte("test-lookup-key-32-bytes-long!!!")
	email := "user@example.com"
	for i := 0; i < b.N; i++ {
		ComputeLookupHash(email, key)
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := []byte("test-password")
	salt := []byte("test-salt-32-bytes-long-testing!")
	for i := 0; i < b.N; i++ {
		HashPassword(password, salt)
	}
}

func BenchmarkVerifyPasswordHash(b *testing.B) {
	password := []byte("test-password")
	salt := []byte("test-salt-32-bytes-long-testing!")
	hash := HashPassword(password, salt)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyPasswordHash(password, hash, salt)
	}
}
