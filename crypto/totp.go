package crypto

import (
	"bytes"
	"crypto/subtle"
	"encoding/base32"
	"fmt"
	"image/png"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/argon2"
)

const (
	backupCodeLength = 8
	backupCodeBytes  = 6
)

// GenerateTOTPKey creates a new TOTP key for the given issuer and account.
func GenerateTOTPKey(issuer, account string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
}

// ValidateTOTPCode validates a TOTP code against the secret.
func ValidateTOTPCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// ValidateTOTPCodeWithSkew validates a TOTP code allowing for time skew.
// skew is the number of periods before/after to allow (e.g., 1 allows +-30 seconds).
func ValidateTOTPCodeWithSkew(secret, code string, skew uint) bool {
	valid, _ := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      skew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})

	return valid
}

// GenerateQRCodePNG generates a PNG image of the QR code for the TOTP key.
func GenerateQRCodePNG(key *otp.Key, size int) ([]byte, error) {
	img, err := key.Image(size, size)
	if err != nil {
		return nil, fmt.Errorf("cannot generate QR code image: %w", err)
	}

	var buf bytes.Buffer

	encodeErr := png.Encode(&buf, img)
	if encodeErr != nil {
		return nil, fmt.Errorf("cannot encode QR code as PNG: %w", encodeErr)
	}

	return buf.Bytes(), nil
}

// GenerateBackupCodes generates a set of one-time backup codes.
// Returns plain text codes for display and hashed codes for storage.
func GenerateBackupCodes(count int) (plain, hashed []string, err error) {
	if count <= 0 {
		count = 8
	}

	plain = make([]string, count)
	hashed = make([]string, count)

	for i := 0; i < count; i++ {
		codeBytes := make([]byte, backupCodeBytes)

		readErr := fillRandom(codeBytes)
		if readErr != nil {
			return nil, nil, fmt.Errorf("cannot generate backup code: %w", readErr)
		}

		code := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(codeBytes))
		if len(code) > backupCodeLength {
			code = code[:backupCodeLength]
		}

		plain[i] = code
		hashed[i] = hashBackupCode(code)
	}

	return plain, hashed, nil
}

// VerifyBackupCode checks if a code matches any of the hashed backup codes.
// Returns true if valid and the index of the matched code, -1 otherwise.
func VerifyBackupCode(code string, hashedCodes []string) (bool, int) {
	code = strings.ToUpper(strings.TrimSpace(code))
	codeHash := hashBackupCode(code)

	for i, h := range hashedCodes {
		if subtle.ConstantTimeCompare([]byte(codeHash), []byte(h)) == 1 {
			return true, i
		}
	}

	return false, -1
}

func hashBackupCode(code string) string {
	salt := []byte("hatmax-backup-code-salt")
	hash := argon2.IDKey([]byte(code), salt, argonTime, argonMemory, argonThreads, argonKeyLength)

	return base32.StdEncoding.EncodeToString(hash)
}
