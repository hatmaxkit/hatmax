package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptEmail(t *testing.T) {
	validKey := make([]byte, aesKeyLength)
	for i := range validKey {
		validKey[i] = byte(i)
	}

	tests := []struct {
		name      string
		plaintext string
		key       []byte
		wantErr   error
	}{
		{
			name:      "valid encryption",
			plaintext: "test@example.com",
			key:       validKey,
			wantErr:   nil,
		},
		{
			name:      "empty plaintext",
			plaintext: "",
			key:       validKey,
			wantErr:   nil,
		},
		{
			name:      "long email",
			plaintext: "very.long.email.address.with.many.parts@subdomain.example.com",
			key:       validKey,
			wantErr:   nil,
		},
		{
			name:      "invalid key length short",
			plaintext: "test@example.com",
			key:       make([]byte, 16),
			wantErr:   ErrInvalidKey,
		},
		{
			name:      "invalid key length long",
			plaintext: "test@example.com",
			key:       make([]byte, 64),
			wantErr:   ErrInvalidKey,
		},
		{
			name:      "nil key",
			plaintext: "test@example.com",
			key:       nil,
			wantErr:   ErrInvalidKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, iv, tag, err := EncryptEmail(tt.plaintext, tt.key)

			if err != tt.wantErr {
				t.Errorf("EncryptEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil {
				if iv == "" {
					t.Error("EncryptEmail() returned empty iv")
				}
				if tag == "" {
					t.Error("EncryptEmail() returned empty tag")
				}
			}
		})
	}
}

func TestDecryptEmail(t *testing.T) {
	validKey := make([]byte, aesKeyLength)
	for i := range validKey {
		validKey[i] = byte(i)
	}

	ciphertext, iv, tag, _ := EncryptEmail("test@example.com", validKey)

	tests := []struct {
		name       string
		ciphertext string
		iv         string
		tag        string
		key        []byte
		want       string
		wantErr    error
	}{
		{
			name:       "valid decryption",
			ciphertext: ciphertext,
			iv:         iv,
			tag:        tag,
			key:        validKey,
			want:       "test@example.com",
			wantErr:    nil,
		},
		{
			name:       "invalid key length",
			ciphertext: ciphertext,
			iv:         iv,
			tag:        tag,
			key:        make([]byte, 16),
			want:       "",
			wantErr:    ErrInvalidKey,
		},
		{
			name:       "invalid ciphertext base64",
			ciphertext: "not-valid-base64!!!",
			iv:         iv,
			tag:        tag,
			key:        validKey,
			want:       "",
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "invalid iv base64",
			ciphertext: ciphertext,
			iv:         "not-valid-base64!!!",
			tag:        tag,
			key:        validKey,
			want:       "",
			wantErr:    ErrInvalidIV,
		},
		{
			name:       "invalid tag base64",
			ciphertext: ciphertext,
			iv:         iv,
			tag:        "not-valid-base64!!!",
			key:        validKey,
			want:       "",
			wantErr:    ErrInvalidTag,
		},
		{
			name:       "wrong iv length",
			ciphertext: ciphertext,
			iv:         "AAAA",
			tag:        tag,
			key:        validKey,
			want:       "",
			wantErr:    ErrInvalidIV,
		},
		{
			name:       "tampered ciphertext",
			ciphertext: "dGFtcGVyZWQ=",
			iv:         iv,
			tag:        tag,
			key:        validKey,
			want:       "",
			wantErr:    ErrDecryptionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecryptEmail(tt.ciphertext, tt.iv, tt.tag, tt.key)

			if err != tt.wantErr {
				t.Errorf("DecryptEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("DecryptEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	validKey := make([]byte, aesKeyLength)
	for i := range validKey {
		validKey[i] = byte(i)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple email",
			plaintext: "user@example.com",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "unicode email",
			plaintext: "用户@例子.公司",
		},
		{
			name:      "long email",
			plaintext: "very.long.email.address.with.subdomain@mail.company.example.org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, iv, tag, err := EncryptEmail(tt.plaintext, validKey)
			if err != nil {
				t.Fatalf("EncryptEmail() error = %v", err)
			}

			got, err := DecryptEmail(ciphertext, iv, tag, validKey)
			if err != nil {
				t.Fatalf("DecryptEmail() error = %v", err)
			}

			if got != tt.plaintext {
				t.Errorf("Roundtrip failed: got %v, want %v", got, tt.plaintext)
			}
		})
	}
}

func TestComputeLookupHash(t *testing.T) {
	validKey := make([]byte, hmacKeyLength)
	for i := range validKey {
		validKey[i] = byte(i)
	}

	tests := []struct {
		name       string
		email      string
		signingKey []byte
		wantEmpty  bool
	}{
		{
			name:       "valid hash",
			email:      "test@example.com",
			signingKey: validKey,
			wantEmpty:  false,
		},
		{
			name:       "empty email",
			email:      "",
			signingKey: validKey,
			wantEmpty:  false,
		},
		{
			name:       "invalid key length short",
			email:      "test@example.com",
			signingKey: make([]byte, 16),
			wantEmpty:  true,
		},
		{
			name:       "invalid key length long",
			email:      "test@example.com",
			signingKey: make([]byte, 64),
			wantEmpty:  true,
		},
		{
			name:       "nil key",
			email:      "test@example.com",
			signingKey: nil,
			wantEmpty:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeLookupHash(tt.email, tt.signingKey)

			if tt.wantEmpty && got != "" {
				t.Errorf("ComputeLookupHash() = %v, want empty", got)
			}

			if !tt.wantEmpty && got == "" {
				t.Error("ComputeLookupHash() returned empty, want non-empty")
			}
		})
	}
}

func TestComputeLookupHashDeterministic(t *testing.T) {
	validKey := make([]byte, hmacKeyLength)
	for i := range validKey {
		validKey[i] = byte(i)
	}

	email := "test@example.com"
	hash1 := ComputeLookupHash(email, validKey)
	hash2 := ComputeLookupHash(email, validKey)

	if hash1 != hash2 {
		t.Errorf("ComputeLookupHash() not deterministic: %v != %v", hash1, hash2)
	}

	differentEmail := "other@example.com"
	hash3 := ComputeLookupHash(differentEmail, validKey)

	if hash1 == hash3 {
		t.Error("ComputeLookupHash() same hash for different emails")
	}
}

func TestHashPassword(t *testing.T) {
	validSalt := make([]byte, saltLength)
	for i := range validSalt {
		validSalt[i] = byte(i)
	}

	tests := []struct {
		name     string
		password string
		salt     []byte
		wantNil  bool
	}{
		{
			name:     "valid password",
			password: "securePassword123",
			salt:     validSalt,
			wantNil:  false,
		},
		{
			name:     "empty password",
			password: "",
			salt:     validSalt,
			wantNil:  false,
		},
		{
			name:     "unicode password",
			password: "密码🔐secure",
			salt:     validSalt,
			wantNil:  false,
		},
		{
			name:     "invalid salt length short",
			password: "password",
			salt:     make([]byte, 16),
			wantNil:  true,
		},
		{
			name:     "invalid salt length long",
			password: "password",
			salt:     make([]byte, 64),
			wantNil:  true,
		},
		{
			name:     "nil salt",
			password: "password",
			salt:     nil,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashPassword(tt.password, tt.salt)

			if tt.wantNil && got != nil {
				t.Errorf("HashPassword() = %v, want nil", got)
			}

			if !tt.wantNil && got == nil {
				t.Error("HashPassword() returned nil, want non-nil")
			}

			if !tt.wantNil && len(got) != argonKeyLength {
				t.Errorf("HashPassword() length = %d, want %d", len(got), argonKeyLength)
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	validSalt := make([]byte, saltLength)
	for i := range validSalt {
		validSalt[i] = byte(i)
	}

	password := "securePassword123"
	hash := HashPassword(password, validSalt)

	tests := []struct {
		name     string
		password string
		hash     []byte
		salt     []byte
		want     bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			salt:     validSalt,
			want:     true,
		},
		{
			name:     "wrong password",
			password: "wrongPassword",
			hash:     hash,
			salt:     validSalt,
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			salt:     validSalt,
			want:     false,
		},
		{
			name:     "invalid salt length",
			password: password,
			hash:     hash,
			salt:     make([]byte, 16),
			want:     false,
		},
		{
			name:     "nil salt",
			password: password,
			hash:     hash,
			salt:     nil,
			want:     false,
		},
		{
			name:     "tampered hash",
			password: password,
			hash:     append([]byte{0xFF}, hash[1:]...),
			salt:     validSalt,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyPassword(tt.password, tt.hash, tt.salt)

			if got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}

	if len(salt1) != saltLength {
		t.Errorf("GenerateSalt() length = %d, want %d", len(salt1), saltLength)
	}

	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}

	if bytes.Equal(salt1, salt2) {
		t.Error("GenerateSalt() not random, got same salt twice")
	}
}

func TestGenerateSecureToken(t *testing.T) {
	tests := []struct {
		name      string
		length    int
		minLength int
	}{
		{
			name:      "default minimum length",
			length:    0,
			minLength: 32,
		},
		{
			name:      "small length uses minimum",
			length:    16,
			minLength: 32,
		},
		{
			name:      "exact minimum",
			length:    32,
			minLength: 32,
		},
		{
			name:      "larger length",
			length:    64,
			minLength: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateSecureToken(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecureToken() error = %v", err)
			}

			if token == "" {
				t.Error("GenerateSecureToken() returned empty token")
			}
		})
	}
}

func TestGenerateSecureTokenUniqueness(t *testing.T) {
	token1, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("GenerateSecureToken() error = %v", err)
	}

	token2, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("GenerateSecureToken() error = %v", err)
	}

	if token1 == token2 {
		t.Error("GenerateSecureToken() not random, got same token twice")
	}
}
