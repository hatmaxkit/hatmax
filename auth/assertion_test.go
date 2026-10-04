// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// The strict replay policy accepts counterless devices but never resets counters.
func TestAssertionCounter(t *testing.T) {
	tests := []struct {
		name      string
		old, next uint32
		valid     bool
	}{
		{"counterless", 0, 0, true}, {"first count", 0, 1, true}, {"increasing", 3, 4, true},
		{"equal", 3, 3, false}, {"lower", 3, 2, false}, {"reset", 3, 0, false}, {"overflow", ^uint32(0), 0, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if ValidAssertionCounter(test.old, test.next) != test.valid {
				t.Fatal("incorrect counter admission")
			}
		})
	}
}

// Purpose namespaces and canonical encoding prevent ordinary session lookup.
func TestAssertionTokens(t *testing.T) {
	for _, purpose := range []AssertionPurpose{AssertionSignin, AssertionStepUp} {
		token := assertionPrefix(purpose) + strings.Repeat("A", 43)

		digest, got, err := parseAssertionToken(token)
		if err != nil || digest == (AssertionDigest{}) || got != purpose {
			t.Fatalf("token: %v", err)
		}

		_, err = ParseSessionToken(token)
		if !errors.Is(err, ErrSessionToken) {
			t.Fatal("pending resolved as session")
		}

		_, err = parseEnrollmentToken(token)
		if !errors.Is(err, ErrEnrollment) {
			t.Fatal("assertion resolved as enrollment")
		}
	}

	a, _, _ := parseAssertionToken("assert1." + strings.Repeat("A", 43))

	b, _, _ := parseAssertionToken("stepup1." + strings.Repeat("A", 43))
	if a == b {
		t.Fatal("purpose digests collide")
	}

	for _, token := range []string{"", strings.Repeat("A", 51), "assert1." + strings.Repeat("A", 42) + "B", "enroll1." + strings.Repeat("A", 43)} {
		_, _, err := parseAssertionToken(token)
		if err == nil {
			t.Fatal("noncanonical token admitted")
		}
	}
}

// Closed proof shape and fixed time establish policy/freshness, not verification.
// Actual signature-to-session evidence lives in TestWebAuthnTransactions.
func TestWebAuthnProof(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	session := Session{ID: "session", UserID: "user", AuthVersion: 1, Generation: 1, PolicyRevision: "v1", CreatedAt: now.Add(-time.Minute), AuthenticatedAt: now.Add(-time.Minute), LastActivityAt: now, ExpiresAt: now.Add(time.Hour), InactivityTTL: time.Hour, Proof: VerifiedProof{Method: WebAuthnProof, VerifiedAt: now.Add(-time.Minute), FactorID: "factor", FactorRevision: 1}}

	tests := []struct {
		name     string
		required RequiredProof
		age      time.Duration
		change   func(*Session)
		want     error
	}{
		{name: "password minimum", required: RequirePassword},
		{name: "MFA", required: RequireMFA},
		{name: "phishing resistance", required: RequirePhishingResistantMFA},
		{name: "freshness equality", required: RequireMFA, age: time.Minute, want: ErrSessionProofExpired},
		{name: "missing factor", required: RequireMFA, change: func(s *Session) { s.Proof.FactorID = "" }, want: ErrSessionRecord},
		{name: "missing revision", required: RequireMFA, change: func(s *Session) { s.Proof.FactorRevision = 0 }, want: ErrSessionRecord},
		{name: "password with factor", required: RequirePassword, change: func(s *Session) { s.Proof.Method = PasswordProof }, want: ErrSessionRecord},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stored := session
			if test.change != nil {
				test.change(&stored)
			}

			err := (AccessRequirement{Proof: test.required, Revision: "v1", MaxAge: test.age}).Evaluate(stored, now)
			if !errors.Is(err, test.want) {
				t.Fatalf("proof: %v", err)
			}
		})
	}
}
