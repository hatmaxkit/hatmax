package crypto

import (
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestGenerateTOTPKey(t *testing.T) {
	tests := []struct {
		name    string
		issuer  string
		account string
		wantErr bool
	}{
		{
			name:    "valid issuer and account",
			issuer:  "MyApp",
			account: "user@example.com",
			wantErr: false,
		},
		{
			name:    "empty issuer",
			issuer:  "",
			account: "user@example.com",
			wantErr: true,
		},
		{
			name:    "empty account",
			issuer:  "MyApp",
			account: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateTOTPKey(tt.issuer, tt.account)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTOTPKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if key == nil {
					t.Error("expected non-nil key")
					return
				}
				if key.Secret() == "" {
					t.Error("expected non-empty secret")
				}
			}
		})
	}
}

func TestValidateTOTPCode(t *testing.T) {
	key, err := GenerateTOTPKey("TestApp", "test@example.com")
	if err != nil {
		t.Fatalf("cannot generate test key: %v", err)
	}

	validCode, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("cannot generate valid code: %v", err)
	}

	tests := []struct {
		name   string
		secret string
		code   string
		want   bool
	}{
		{
			name:   "valid code",
			secret: key.Secret(),
			code:   validCode,
			want:   true,
		},
		{
			name:   "invalid code",
			secret: key.Secret(),
			code:   "000000",
			want:   false,
		},
		{
			name:   "empty code",
			secret: key.Secret(),
			code:   "",
			want:   false,
		},
		{
			name:   "wrong secret",
			secret: "INVALIDSECRET",
			code:   validCode,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateTOTPCode(tt.secret, tt.code); got != tt.want {
				t.Errorf("ValidateTOTPCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateTOTPCodeWithSkew(t *testing.T) {
	key, err := GenerateTOTPKey("TestApp", "test@example.com")
	if err != nil {
		t.Fatalf("cannot generate test key: %v", err)
	}

	validCode, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("cannot generate valid code: %v", err)
	}

	tests := []struct {
		name   string
		secret string
		code   string
		skew   uint
		want   bool
	}{
		{
			name:   "valid code with zero skew",
			secret: key.Secret(),
			code:   validCode,
			skew:   0,
			want:   true,
		},
		{
			name:   "valid code with skew",
			secret: key.Secret(),
			code:   validCode,
			skew:   1,
			want:   true,
		},
		{
			name:   "invalid code",
			secret: key.Secret(),
			code:   "000000",
			skew:   1,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateTOTPCodeWithSkew(tt.secret, tt.code, tt.skew); got != tt.want {
				t.Errorf("ValidateTOTPCodeWithSkew() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateQRCodePNG(t *testing.T) {
	key, err := GenerateTOTPKey("TestApp", "test@example.com")
	if err != nil {
		t.Fatalf("cannot generate test key: %v", err)
	}

	tests := []struct {
		name    string
		key     interface{ Image(int, int) (interface{ Bounds() interface{} }, error) }
		size    int
		wantErr bool
	}{
		{
			name:    "valid QR code generation",
			size:    200,
			wantErr: false,
		},
		{
			name:    "small size",
			size:    50,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateQRCodePNG(key, tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateQRCodePNG() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) == 0 {
					t.Error("expected non-empty PNG data")
				}
				if !strings.HasPrefix(string(got[:8]), "\x89PNG") {
					t.Error("expected PNG header")
				}
			}
		})
	}
}

func TestGenerateBackupCodes(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		wantCount int
		wantErr   bool
	}{
		{
			name:      "default count",
			count:     0,
			wantCount: 8,
			wantErr:   false,
		},
		{
			name:      "negative count defaults to 8",
			count:     -1,
			wantCount: 8,
			wantErr:   false,
		},
		{
			name:      "custom count",
			count:     5,
			wantCount: 5,
			wantErr:   false,
		},
		{
			name:      "single code",
			count:     1,
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plain, hashed, err := GenerateBackupCodes(tt.count)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateBackupCodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(plain) != tt.wantCount {
					t.Errorf("expected %d plain codes, got %d", tt.wantCount, len(plain))
				}
				if len(hashed) != tt.wantCount {
					t.Errorf("expected %d hashed codes, got %d", tt.wantCount, len(hashed))
				}
				for i, code := range plain {
					if len(code) != backupCodeLength {
						t.Errorf("code %d: expected length %d, got %d", i, backupCodeLength, len(code))
					}
					if code != strings.ToUpper(code) {
						t.Errorf("code %d: expected uppercase, got %s", i, code)
					}
				}
			}
		})
	}
}

func TestGenerateBackupCodesUniqueness(t *testing.T) {
	plain, _, err := GenerateBackupCodes(10)
	if err != nil {
		t.Fatalf("cannot generate backup codes: %v", err)
	}

	seen := make(map[string]bool)
	for _, code := range plain {
		if seen[code] {
			t.Errorf("duplicate code found: %s", code)
		}
		seen[code] = true
	}
}

func TestVerifyBackupCode(t *testing.T) {
	plain, hashed, err := GenerateBackupCodes(3)
	if err != nil {
		t.Fatalf("cannot generate backup codes: %v", err)
	}

	tests := []struct {
		name      string
		code      string
		hashed    []string
		wantValid bool
		wantIndex int
	}{
		{
			name:      "valid first code",
			code:      plain[0],
			hashed:    hashed,
			wantValid: true,
			wantIndex: 0,
		},
		{
			name:      "valid last code",
			code:      plain[2],
			hashed:    hashed,
			wantValid: true,
			wantIndex: 2,
		},
		{
			name:      "valid code with lowercase",
			code:      strings.ToLower(plain[1]),
			hashed:    hashed,
			wantValid: true,
			wantIndex: 1,
		},
		{
			name:      "valid code with spaces",
			code:      "  " + plain[0] + "  ",
			hashed:    hashed,
			wantValid: true,
			wantIndex: 0,
		},
		{
			name:      "invalid code",
			code:      "INVALID1",
			hashed:    hashed,
			wantValid: false,
			wantIndex: -1,
		},
		{
			name:      "empty code",
			code:      "",
			hashed:    hashed,
			wantValid: false,
			wantIndex: -1,
		},
		{
			name:      "empty hashed list",
			code:      plain[0],
			hashed:    []string{},
			wantValid: false,
			wantIndex: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, index := VerifyBackupCode(tt.code, tt.hashed)
			if valid != tt.wantValid {
				t.Errorf("VerifyBackupCode() valid = %v, want %v", valid, tt.wantValid)
			}
			if index != tt.wantIndex {
				t.Errorf("VerifyBackupCode() index = %v, want %v", index, tt.wantIndex)
			}
		})
	}
}

func TestHashBackupCode(t *testing.T) {
	code1 := "TESTCODE"
	code2 := "TESTCODE"
	code3 := "DIFFCODE"

	hash1 := hashBackupCode(code1)
	hash2 := hashBackupCode(code2)
	hash3 := hashBackupCode(code3)

	if hash1 != hash2 {
		t.Error("same input should produce same hash")
	}

	if hash1 == hash3 {
		t.Error("different inputs should produce different hashes")
	}

	if hash1 == "" {
		t.Error("hash should not be empty")
	}
}
