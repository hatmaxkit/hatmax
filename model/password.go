// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package model

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt.DefaultCost.
func HashPassword(password string) (string, error) {
	return HashPasswordWithCost(password, bcrypt.DefaultCost)
}

// HashPasswordWithCost hashes a password at the explicit bcrypt cost.
// Costs outside bcrypt.MinCost through bcrypt.MaxCost return an error without
// falling back to bcrypt.DefaultCost. Bcrypt's 72-byte password limit applies.
func HashPasswordWithCost(password string, cost int) (string, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return "", fmt.Errorf("cannot hash password: %w", bcrypt.InvalidCostError(cost))
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("cannot hash password: %w", err)
	}

	return string(hash), nil
}

// ComparePassword compares a password with its hash.
func ComparePassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}

// GenerateRandomPassword generates a random password of the specified length.
func GenerateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("cannot generate random password: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
