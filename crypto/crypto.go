package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime      = 1
	argonMemory    = 64 * 1024
	argonThreads   = 4
	argonKeyLength = 32
	saltLength     = 32
	aesKeyLength   = 32
	hmacKeyLength  = 32
)

var (
	ErrInvalidKey        = errors.New("invalid encryption key length")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrInvalidIV         = errors.New("invalid initialization vector")
	ErrInvalidNonce      = errors.New("invalid nonce")
	ErrInvalidTag        = errors.New("invalid authentication tag")
	ErrDecryptionFailed  = errors.New("decryption failed")
)

// EncryptedString is an authenticated encrypted string representation suitable
// for separate ciphertext, nonce, and tag persistence columns.
type EncryptedString struct {
	Ciphertext string
	Nonce      string
	Tag        string
}

// EncryptString encrypts plaintext with AES-256-GCM and authenticates
// associatedData without encrypting it. The same associatedData must be passed
// to DecryptString.
func EncryptString(plaintext string, key, associatedData []byte) (EncryptedString, error) {
	if len(key) != aesKeyLength {
		return EncryptedString{}, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return EncryptedString{}, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return EncryptedString{}, err
	}

	nonce := make([]byte, gcm.NonceSize())

	err = fillRandom(nonce)
	if err != nil {
		return EncryptedString{}, err
	}

	sealed := gcm.Seal(nil, nonce, []byte(plaintext), associatedData)

	tagSize := gcm.Overhead()
	if len(sealed) < tagSize {
		return EncryptedString{}, ErrInvalidCiphertext
	}

	return EncryptedString{
		Ciphertext: base64.StdEncoding.EncodeToString(sealed[:len(sealed)-tagSize]),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Tag:        base64.StdEncoding.EncodeToString(sealed[len(sealed)-tagSize:]),
	}, nil
}

// DecryptString authenticates associatedData and decrypts value with
// AES-256-GCM. associatedData must be identical to the value passed to
// EncryptString.
func DecryptString(value EncryptedString, key, associatedData []byte) (string, error) {
	if len(key) != aesKeyLength {
		return "", ErrInvalidKey
	}

	ciphertext, err := base64.StdEncoding.DecodeString(value.Ciphertext)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	nonce, err := base64.StdEncoding.DecodeString(value.Nonce)
	if err != nil {
		return "", ErrInvalidNonce
	}

	tag, err := base64.StdEncoding.DecodeString(value.Tag)
	if err != nil {
		return "", ErrInvalidTag
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(nonce) != gcm.NonceSize() {
		return "", ErrInvalidNonce
	}

	plaintext, err := gcm.Open(nil, nonce, append(ciphertext, tag...), associatedData)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// DeriveLookupHash derives a deterministic HMAC-SHA-256 lookup hash for the
// exact bytes in value. Callers own any required normalization before deriving
// and storing a lookup hash.
func DeriveLookupHash(value string, key []byte) (string, error) {
	if len(key) != hmacKeyLength {
		return "", ErrInvalidKey
	}

	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(value))

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

// EncryptEmail encrypts plaintext with AES-256-GCM.
//
// Deprecated: use EncryptString.
func EncryptEmail(plaintext string, key []byte) (ciphertext, iv, tag string, err error) {
	value, err := EncryptString(plaintext, key, nil)
	if err != nil {
		return "", "", "", err
	}

	return value.Ciphertext, value.Nonce, value.Tag, nil
}

// DecryptEmail authenticates and decrypts encrypted email data.
//
// Deprecated: use DecryptString.
func DecryptEmail(ciphertextB64, ivB64, tagB64 string, key []byte) (string, error) {
	plaintext, err := DecryptString(EncryptedString{
		Ciphertext: ciphertextB64,
		Nonce:      ivB64,
		Tag:        tagB64,
	}, key, nil)
	if errors.Is(err, ErrInvalidNonce) {
		return "", ErrInvalidIV
	}

	return plaintext, err
}

// ComputeLookupHash derives a deterministic lookup hash for value.
//
// Deprecated: use DeriveLookupHash, which reports an invalid key explicitly.
func ComputeLookupHash(value string, signingKey []byte) string {
	hash, err := DeriveLookupHash(value, signingKey)
	if err != nil {
		return ""
	}

	return hash
}

func HashPassword(password string, salt []byte) []byte {
	if len(salt) != saltLength {
		return nil
	}

	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLength)
}

func VerifyPassword(password string, hash, salt []byte) bool {
	if len(salt) != saltLength {
		return false
	}

	computedHash := HashPassword(password, salt)
	if computedHash == nil {
		return false
	}

	return subtle.ConstantTimeCompare(hash, computedHash) == 1
}

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLength)

	err := fillRandom(salt)
	if err != nil {
		return nil, err
	}

	return salt, nil
}

func GenerateSecureToken(length int) (string, error) {
	if length < 32 {
		length = 32
	}

	bytes := make([]byte, length)

	err := fillRandom(bytes)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

func fillRandom(dst []byte) error {
	_, err := io.ReadFull(rand.Reader, dst)

	return err
}
