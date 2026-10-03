// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package model

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateRandomPassword generates a random password of the specified length.
func GenerateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("cannot generate random password: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
