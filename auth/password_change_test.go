// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

func changeActor(now time.Time) Session {
	return Session{ID: "actor", UserID: "subject", AuthVersion: 1, Generation: 1, PolicyRevision: "test-v1", Proof: VerifiedProof{Method: PasswordProof, VerifiedAt: now}, AuthenticatedAt: now, CreatedAt: now, LastActivityAt: now, ExpiresAt: now.Add(time.Hour), InactivityTTL: time.Hour}
}

// This table checks factor-dependent policy without claiming storage authority;
// PostgreSQL tests separately establish actual proof and snapshot revalidation.
func TestPasswordChangePolicy(t *testing.T) {
	now := time.Now().UTC()

	for _, tc := range []struct {
		name     string
		allow    bool
		factors  int
		method   ProofMethod
		required RequiredProof
		age      time.Duration
		want     error
	}{
		{"explicit password", true, 0, PasswordProof, RequirePhishingResistantMFA, 0, nil},
		{"password denied", false, 0, PasswordProof, RequirePhishingResistantMFA, 0, ErrSessionProof},
		{"factor blocks password", true, 1, PasswordProof, RequirePhishingResistantMFA, 0, ErrSessionProof},
		{"phishing-resistant", true, 1, WebAuthnProof, RequirePhishingResistantMFA, 0, nil},
		{"fallback denied", true, 1, PasswordTOTPProof, RequirePhishingResistantMFA, 0, ErrSessionProof},
		{"explicit fallback", true, 1, PasswordTOTPProof, RequireMFA, 0, nil},
		{"backup MFA", true, 1, PasswordBackupProof, RequireMFA, 0, nil},
		{"exact proof expiry", true, 0, PasswordProof, RequirePhishingResistantMFA, 5 * time.Minute, ErrSessionProofExpired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := PasswordChangePolicy{Requirement: AccessRequirement{Proof: tc.required, Revision: "test-v1", MaxAge: 5 * time.Minute}, AllowPassword: tc.allow}
			actor := changeActor(now.Add(-tc.age))

			actor.Proof.Method = tc.method
			if tc.method != PasswordProof {
				actor.Proof.FactorID = "factor"
				actor.Proof.FactorRevision = 1
			}

			if tc.method == PasswordTOTPProof || tc.method == PasswordBackupProof {
				actor.Proof.FactorAt = actor.AuthenticatedAt
			}

			snapshot := PasswordChangeAuthorization{Actor: actor, ActorDigest: SessionDigest{1}, Policy: p}
			if tc.factors > 0 {
				snapshot.Factors = []FactorSelection{{Kind: FactorWebAuthn, ID: "factor", Revision: 1}}
			}

			err := snapshot.Check(now, p)
			if !errors.Is(err, tc.want) {
				t.Fatalf("policy: %v", err)
			}
		})
	}
}

type changeStore struct {
	RecoveryQueries
	admitted, committed int
	denied              error
	actor               Session
	base                *Service
}

func (s *changeStore) AuthorizePasswordChange(_ context.Context, d SessionDigest, p PasswordChangePolicy, _ config.RecoverySettings) (*PasswordChangeAuthorization, error) {
	if s.denied != nil {
		return nil, s.denied
	}

	s.admitted++

	return &PasswordChangeAuthorization{Actor: s.actor, ActorDigest: d, Policy: p}, nil
}
func (s *changeStore) CommitPasswordChange(ctx context.Context, a PasswordChangeAuthorization, encoded string, _ PasswordChangePolicy, _ config.RecoverySettings) (*PasswordChanged, error) {
	err := s.base.verifier.Verify(ctx, encoded, "a complete new password")
	if err != nil {
		return nil, err
	}

	s.committed++

	return &PasswordChanged{Subject: a.Actor.UserID, ChangedAt: time.Now().UTC()}, nil
}

// The actual shared verifier checks the complete encoded replacement at commit.
// Fakes establish orchestration only; PostgreSQL tests establish persistence.
func TestPasswordChangeWork(t *testing.T) {
	for _, name := range []string{"success", "storage admission", "oversized", "malformed bearer"} {
		t.Run(name, func(t *testing.T) {
			base := newServiceForTest(t, newMockQueries(), config.New(), log.NewTestLogger("error"))
			store := &changeStore{actor: changeActor(time.Now().UTC()), base: base}

			svc, err := NewRecoveryService(base, store, config.RecoveryConfig{}, "mailbox-v1")
			if err != nil {
				t.Fatal(err)
			}

			token, _, err := newSessionToken()
			if err != nil {
				t.Fatal(err)
			}

			password := "a complete new password"

			var want error

			switch name {
			case "storage admission":
				store.denied = ErrRecoveryAttempts
				want = ErrRecoveryAttempts
			case "oversized":
				password = strings.Repeat("x", 4097)
				want = ErrPasswordTooLong
			case "malformed bearer":
				token = "invalid"
				want = ErrSessionToken
			}

			policy := PasswordChangePolicy{Requirement: AccessRequirement{Revision: "test-v1"}, AllowPassword: true}

			result, err := svc.ChangePassword(t.Context(), token, password, policy)
			if !errors.Is(err, want) {
				t.Fatalf("operation: %v", err)
			}

			if name == "success" {
				if result == nil || store.admitted != 1 || store.committed != 1 {
					t.Fatal("incomplete flow")
				}
			} else if store.committed != 0 {
				t.Fatal("failure committed")
			}

			if (name == "oversized" || name == "malformed bearer" || name == "storage admission") && store.admitted != 0 {
				t.Fatal("early failure admitted work")
			}

			formatted := fmt.Sprintf("%v %#v", store.actor, PasswordChangeAuthorization{ActorDigest: SessionDigest{1}})
			if strings.Contains(formatted, "ActorDigest") {
				t.Fatal("normal formatting exposed digest")
			}
		})
	}
}
