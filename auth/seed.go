// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
)

// SeedKeys is application-owned key configuration. Core copies key bytes into
// immutable AEAD instances. Keep old identities available while records use them.
type SeedKeys struct {
	Active string
	Keys   map[string][]byte
}
type seedCipher struct {
	active string
	keys   map[string]cipher.AEAD
}

func newSeedCipher(keys SeedKeys) (*seedCipher, error) {
	if !boundedID(keys.Active, 64) || len(keys.Keys) < 1 || len(keys.Keys) > 8 {
		return nil, ErrFallback
	}

	result := &seedCipher{active: keys.Active, keys: make(map[string]cipher.AEAD, len(keys.Keys))}
	for id, key := range keys.Keys {
		if !boundedID(id, 64) || len(key) != 32 {
			return nil, ErrFallback
		}

		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}

		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}

		result.keys[id] = aead
	}

	if result.keys[result.active] == nil {
		return nil, ErrFallback
	}

	return result, nil
}
func seedAAD(subject, factor, keyID string) []byte {
	encoded, _ := json.Marshal([]string{"hatmax/totp/seed/v1", subject, factor, keyID})

	return encoded
}
func (s *seedCipher) seal(subject, factor, secret string) (string, []byte, error) {
	aead := s.keys[s.active]
	nonce := make([]byte, aead.NonceSize())

	_, err := rand.Read(nonce)
	if err != nil {
		return "", nil, err
	}

	envelope := append([]byte{1}, nonce...)
	envelope = aead.Seal(envelope, nonce, []byte(secret), seedAAD(subject, factor, s.active))

	return s.active, envelope, nil
}
func (s *seedCipher) open(subject, factor, keyID string, envelope []byte) (string, error) {
	aead := s.keys[keyID]
	if aead == nil || len(envelope) != 61 || envelope[0] != 1 {
		return "", ErrFallback
	}

	plain, err := aead.Open(nil, envelope[1:13], envelope[13:], seedAAD(subject, factor, keyID))
	if err != nil || len(plain) != 32 {
		return "", ErrFallback
	}
	defer clear(plain)

	return string(plain), nil
}
