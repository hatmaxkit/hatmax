// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrBackupCode = errors.New("invalid backup code")

// BackupSecret is immediate issuance material. Do not persist or log Secret/Wire.
type BackupSecret struct{ ID, Secret, Wire string }

func NewBackupSecret() (BackupSecret, error) {
	var secret [16]byte

	_, err := rand.Read(secret[:])
	if err != nil {
		return BackupSecret{}, err
	}

	id := uuid.NewString()
	value := base64.RawURLEncoding.EncodeToString(secret[:])

	return BackupSecret{ID: id, Secret: value, Wire: "backup1." + id + "." + value}, nil
}

// ParseBackupCode rejects normalization, aliases, padding and excessive input
// before subject/identifier lookup or any KDF.
func ParseBackupCode(code string) (BackupSecret, error) {
	if len(code) != 67 || code[:8] != "backup1." || code[44] != '.' {
		return BackupSecret{}, ErrBackupCode
	}

	id := code[8:44]

	parsed, err := uuid.Parse(id)
	if err != nil || parsed.String() != id {
		return BackupSecret{}, ErrBackupCode
	}

	secret := code[45:]

	raw, err := base64.RawURLEncoding.Strict().DecodeString(secret)
	if err != nil || len(raw) != 16 || base64.RawURLEncoding.EncodeToString(raw) != secret {
		return BackupSecret{}, ErrBackupCode
	}

	return BackupSecret{ID: id, Secret: secret, Wire: code}, nil
}

// BackupVerifierInput binds the PHC to its purpose, subject and nonsecret ID.
func BackupVerifierInput(subject string, code BackupSecret) (string, error) {
	if len(subject) == 0 || len(subject) > 128 || strings.ContainsRune(subject, 0) {
		return "", ErrBackupCode
	}

	parsed, err := ParseBackupCode(code.Wire)
	if err != nil || parsed.ID != code.ID || parsed.Secret != code.Secret {
		return "", ErrBackupCode
	}

	return "hatmax/backup/v1\x00" + subject + "\x00" + code.ID + "\x00" + code.Secret, nil
}
