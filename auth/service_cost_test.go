// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

// Signup must persist a hash at the configured work factor, and sign-in must
// compare hashes created at different costs without changing session behavior.
func TestSignupCost(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost int
	}{
		{name: "minimum", cost: bcrypt.MinCost},
		{name: "nondefault", cost: 6},
		{name: "config default", cost: config.New().Auth.BCryptCost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.New()
			cfg.Auth.BCryptCost = tc.cost
			queries := newMockQueries()
			service := NewService(queries, cfg, log.NewTestLogger("error"))

			user, err := service.Signup(context.Background(), "ledger@example.test", "correct-password")
			if err != nil {
				t.Fatal(err)
			}

			cost, err := bcrypt.Cost([]byte(user.PasswordHash))
			if err != nil || cost != tc.cost {
				t.Fatalf("hash cost = %d, %v; want %d", cost, err, tc.cost)
			}

			if queries.users[user.ID].PasswordHash != user.PasswordHash || !model.ComparePassword(user.PasswordHash, "correct-password") {
				t.Fatal("persisted hash does not match the signup password")
			}

			// Sign-in uses the cost embedded in an existing hash, not the
			// configured work factor for future signups.
			cfg.Auth.BCryptCost = bcrypt.MinCost

			session, err := service.Signin(context.Background(), user.Email, "correct-password")
			if err != nil || session == nil || session.UserID != user.ID {
				t.Fatalf("sign-in failed: %v", err)
			}

			_, err = service.Signin(context.Background(), user.Email, "wrong-password")
			if !errors.Is(err, ErrInvalidPassword) {
				t.Fatalf("incorrect password error = %v", err)
			}
		})
	}
}

// Invalid costs must remain validation errors and must not silently fall back
// to bcrypt.DefaultCost or persist a user; bcrypt length errors stay wrapped.
func TestSignupHashFailure(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cost     int
		password string
		want     error
	}{
		{name: "zero", cost: 0, password: "valid-password", want: bcrypt.InvalidCostError(0)},
		{name: "below minimum", cost: bcrypt.MinCost - 1, password: "valid-password", want: bcrypt.InvalidCostError(bcrypt.MinCost - 1)},
		{name: "above maximum", cost: bcrypt.MaxCost + 1, password: "valid-password", want: bcrypt.InvalidCostError(bcrypt.MaxCost + 1)},
		{name: "password too long", cost: bcrypt.MinCost, password: strings.Repeat("x", 73), want: bcrypt.ErrPasswordTooLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.New()

			cfg.Auth.BCryptCost = tc.cost
			if tc.cost < bcrypt.MinCost || tc.cost > bcrypt.MaxCost {
				if cfg.Validate() == nil {
					t.Fatal("invalid configured cost passed validation")
				}
			}

			queries := newMockQueries()
			service := NewService(queries, cfg, log.NewTestLogger("error"))

			user, err := service.Signup(context.Background(), "ledger@example.test", tc.password)
			if user != nil || !errors.Is(err, tc.want) || !strings.Contains(err.Error(), "cannot hash password") {
				t.Fatalf("signup returned user=%v, err=%v; want wrapped %v", user, err, tc.want)
			}

			if len(queries.users) != 0 {
				t.Fatal("hash failure persisted a user")
			}
		})
	}
}
