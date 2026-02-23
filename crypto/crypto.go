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
	ErrInvalidTag        = errors.New("invalid authentication tag")
	ErrDecryptionFailed  = errors.New("decryption failed")
)

func EncryptEmail(plaintext string, key []byte) (ciphertext, iv, tag string, err error) {
	if len(key) != aesKeyLength {
		return "", "", "", ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", "", err
	}

	nonce := make([]byte, gcm.NonceSize())

	err = fillRandom(nonce)
	if err != nil {
		return "", "", "", err
	}

	sealed := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	tagSize := gcm.Overhead()
	if len(sealed) < tagSize {
		return "", "", "", ErrInvalidCiphertext
	}

	ciphertextBytes := sealed[:len(sealed)-tagSize]
	tagBytes := sealed[len(sealed)-tagSize:]

	return base64.StdEncoding.EncodeToString(ciphertextBytes),
		base64.StdEncoding.EncodeToString(nonce),
		base64.StdEncoding.EncodeToString(tagBytes),
		nil
}

func DecryptEmail(ciphertextB64, ivB64, tagB64 string, key []byte) (string, error) {
	if len(key) != aesKeyLength {
		return "", ErrInvalidKey
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	iv, err := base64.StdEncoding.DecodeString(ivB64)
	if err != nil {
		return "", ErrInvalidIV
	}

	tag, err := base64.StdEncoding.DecodeString(tagB64)
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

	if len(iv) != gcm.NonceSize() {
		return "", ErrInvalidIV
	}

	sealed := append(ciphertext, tag...)

	plaintext, err := gcm.Open(nil, iv, sealed, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

func ComputeLookupHash(email string, signingKey []byte) string {
	if len(signingKey) != hmacKeyLength {
		return ""
	}

	h := hmac.New(sha256.New, signingKey)
	h.Write([]byte(email))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
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
