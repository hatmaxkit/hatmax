// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package model

import (
	"testing"
)

func TestGenerateRandomPassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "short password",
			length: 8,
		},
		{
			name:   "medium password",
			length: 16,
		},
		{
			name:   "long password",
			length: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := GenerateRandomPassword(tt.length)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(password) != tt.length {
				t.Errorf("expected length %d, got %d", tt.length, len(password))
			}
		})
	}
}

func TestGenerateRandomPasswordUniqueness(t *testing.T) {
	pwd1, err := GenerateRandomPassword(16)
	if err != nil {
		t.Fatalf("cannot generate password: %v", err)
	}

	pwd2, err := GenerateRandomPassword(16)
	if err != nil {
		t.Fatalf("cannot generate password: %v", err)
	}

	if pwd1 == pwd2 {
		t.Error("expected unique passwords")
	}
}
