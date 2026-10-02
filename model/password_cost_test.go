// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package model

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Explicit work factors must be encoded into the hash, with the existing
// comparison behavior and bcrypt's byte-length boundary preserved.
func TestHashCost(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cost     int
		password string
		want     error
	}{
		{name: "minimum", cost: bcrypt.MinCost, password: "correct-password"},
		{name: "nondefault", cost: 6, password: "correct-password"},
		{name: "empty password", cost: bcrypt.MinCost},
		{name: "length boundary", cost: bcrypt.MinCost, password: strings.Repeat("x", 72)},
		{name: "length exceeded", cost: bcrypt.MinCost, password: strings.Repeat("x", 73), want: bcrypt.ErrPasswordTooLong},
		{name: "negative cost", cost: -1, password: "correct-password", want: bcrypt.InvalidCostError(-1)},
		{name: "zero cost", cost: 0, password: "correct-password", want: bcrypt.InvalidCostError(0)},
		{name: "below minimum", cost: bcrypt.MinCost - 1, password: "correct-password", want: bcrypt.InvalidCostError(bcrypt.MinCost - 1)},
		{name: "above maximum", cost: bcrypt.MaxCost + 1, password: "correct-password", want: bcrypt.InvalidCostError(bcrypt.MaxCost + 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := HashPasswordWithCost(tc.password, tc.cost)
			if tc.want != nil {
				if hash != "" || !errors.Is(err, tc.want) {
					t.Fatalf("hash failure = %v; want %v and no hash", err, tc.want)
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			cost, err := bcrypt.Cost([]byte(hash))
			if err != nil || cost != tc.cost {
				t.Fatalf("hash cost = %d, %v; want %d", cost, err, tc.cost)
			}

			if !ComparePassword(hash, tc.password) || ComparePassword(hash, "wrong-password") {
				t.Fatal("password comparison changed")
			}
		})
	}
}

// The original helper must keep its non-variadic function signature and cost,
// independent of the separately configured signup policy.
func TestHashCompatibility(t *testing.T) {
	var hashPassword func(string) (string, error) = HashPassword

	hash, err := hashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}

	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil || cost != bcrypt.DefaultCost || !ComparePassword(hash, "correct-password") {
		t.Fatalf("standalone helper compatibility: cost=%d, err=%v", cost, err)
	}
}
