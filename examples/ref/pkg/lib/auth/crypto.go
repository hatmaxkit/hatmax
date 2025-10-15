package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"golang.org/x/crypto/argon2"
)

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ComputeLookupHash(email string, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(email))
	return h.Sum(nil)
}

func GeneratePasswordSalt() []byte {
	salt := make([]byte, 32)
	rand.Read(salt)
	return salt
}

func HashPassword(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, 1, 64*1024, 4, 32)
}

func VerifyPasswordHash(password, hash, salt []byte) bool {
	derived := argon2.IDKey(password, salt, 1, 64*1024, 4, 32)
	return subtle.ConstantTimeCompare(derived, hash) == 1
}

func GenerateRandomBytes(length int) []byte {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return bytes
}
