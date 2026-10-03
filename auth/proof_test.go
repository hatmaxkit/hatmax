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

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

// Requirements are explicit bounded server policy, with no unknown-profile fallback.
func TestProofRequirement(t *testing.T) {
	tests := []struct {
		name        string
		requirement AccessRequirement
		valid       bool
	}{
		{"password", AccessRequirement{Proof: RequirePassword, Revision: "v1"}, true},
		{"MFA", AccessRequirement{Proof: RequireMFA, Revision: "v1", MaxAge: time.Second}, true},
		{"phishing resistant", AccessRequirement{Proof: RequirePhishingResistantMFA, Revision: "v1", MaxAge: time.Hour}, true},
		{"revision maximum", AccessRequirement{Proof: RequirePassword, Revision: strings.Repeat("a", 128)}, true},
		{"missing profile", AccessRequirement{Revision: "v1"}, false},
		{"unknown profile", AccessRequirement{Proof: RequiredProof(255), Revision: "v1"}, false},
		{"missing revision", AccessRequirement{Proof: RequirePassword}, false},
		{"revision overflow", AccessRequirement{Proof: RequirePassword, Revision: strings.Repeat("a", 129)}, false},
		{"control revision", AccessRequirement{Proof: RequirePassword, Revision: "v1\n"}, false},
		{"non ASCII revision", AccessRequirement{Proof: RequirePassword, Revision: "v1é"}, false},
		{"negative freshness", AccessRequirement{Proof: RequirePassword, Revision: "v1", MaxAge: -time.Second}, false},
		{"short freshness", AccessRequirement{Proof: RequirePassword, Revision: "v1", MaxAge: time.Second - time.Nanosecond}, false},
		{"long freshness", AccessRequirement{Proof: RequirePassword, Revision: "v1", MaxAge: time.Hour + time.Second}, false},
		{"precision", AccessRequirement{Proof: RequirePassword, Revision: "v1", MaxAge: time.Second + time.Nanosecond}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.requirement.Check(time.Hour)
			if (err == nil) != test.valid {
				t.Fatalf("requirement: %v", err)
			}
		})
	}
}

// Fixed time pins freshness equality and corrupt/future facts without creating
// a hypothetical MFA receipt. Activity cannot renew the actual password time.
func TestProofEvaluation(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	session := Session{ID: "session", UserID: "user", AuthVersion: 1, PolicyRevision: "v1", Generation: 1,
		CreatedAt: now.Add(-10 * time.Minute), AuthenticatedAt: now.Add(-9 * time.Minute), LastActivityAt: now,
		ExpiresAt: now.Add(time.Hour), InactivityTTL: 30 * time.Minute, Proof: VerifiedProof{Method: PasswordProof, VerifiedAt: now.Add(-9 * time.Minute)}}

	tests := []struct {
		name     string
		required AccessRequirement
		change   func(*Session)
		want     error
	}{
		{name: "supported", required: AccessRequirement{Proof: RequirePassword, Revision: "v1"}},
		{name: "fresh before equality", required: AccessRequirement{Proof: RequirePassword, Revision: "v1", MaxAge: 9*time.Minute + time.Microsecond}},
		{name: "freshness equality", required: AccessRequirement{Proof: RequirePassword, Revision: "v1", MaxAge: 9 * time.Minute}, want: ErrSessionProofExpired},
		{name: "activity cannot renew proof", required: AccessRequirement{Proof: RequirePassword, Revision: "v1", MaxAge: 5 * time.Minute}, want: ErrSessionProofExpired},
		{name: "current revision", required: AccessRequirement{Proof: RequirePassword, Revision: "v2"}, want: ErrSessionPolicy},
		{name: "MFA unmet", required: AccessRequirement{Proof: RequireMFA, Revision: "v1"}, want: ErrSessionProof},
		{name: "phishing resistance unmet", required: AccessRequirement{Proof: RequirePhishingResistantMFA, Revision: "v1"}, want: ErrSessionProof},
		{name: "unknown method", required: AccessRequirement{Proof: RequirePassword, Revision: "v1"}, change: func(s *Session) { s.Proof.Method = ProofMethod(255) }, want: ErrSessionRecord},
		{name: "missing time", required: AccessRequirement{Proof: RequirePassword, Revision: "v1"}, change: func(s *Session) { s.Proof.VerifiedAt = time.Time{} }, want: ErrSessionRecord},
		{name: "future proof", required: AccessRequirement{Proof: RequirePassword, Revision: "v1"}, change: func(s *Session) { s.Proof.VerifiedAt = now.Add(time.Second) }, want: ErrSessionRecord},
		{name: "proof before creation", required: AccessRequirement{Proof: RequirePassword, Revision: "v1"}, change: func(s *Session) { s.Proof.VerifiedAt = s.CreatedAt.Add(-time.Second) }, want: ErrSessionRecord},
		{name: "bad stored revision", required: AccessRequirement{Proof: RequirePassword, Revision: "v1"}, change: func(s *Session) { s.PolicyRevision = "" }, want: ErrSessionRecord},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stored := session
			if test.change != nil {
				test.change(&stored)
			}

			err := test.required.Evaluate(stored, now)
			if !errors.Is(err, test.want) {
				t.Fatalf("evaluation: %v", err)
			}
		})
	}
}

// Actual password verification produces only password facts. Enrollment/roles
// explain unmet requirements but cannot issue stronger access or continuation.
func TestAuthenticationOutcomes(t *testing.T) {
	tests := []struct {
		name     string
		proof    RequiredProof
		enrolled bool
		want     AuthenticationOutcome
	}{
		{"password", RequirePassword, false, AuthenticationCompleted},
		{"enrollment required", RequireMFA, false, AuthenticationPendingEnrollment},
		{"proof required", RequireMFA, true, AuthenticationPendingProof},
		{"phishing unavailable", RequirePhishingResistantMFA, false, AuthenticationDenied},
		{"enrollment not phishing proof", RequirePhishingResistantMFA, true, AuthenticationDenied},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := newMockQueries()
			svc := newServiceForTest(t, q, config.New(), log.NewTestLogger("error"))

			user, err := svc.Signup(t.Context(), "outcome@example.com", "a distinct valid password")
			if err != nil {
				t.Fatal(err)
			}

			q.users[user.ID].TOTPEnabled = test.enrolled
			q.users[user.ID].Roles = []string{"administrator"}
			future := time.Now().Add(time.Hour)
			q.users[user.ID].TOTPVerifiedAt = &future
			generated := 0
			svc.sessionToken = func() (string, SessionDigest, error) {
				generated++

				return newSessionToken()
			}
			required := AccessRequirement{Proof: test.proof, Revision: "v1"}

			result, err := svc.Signin(t.Context(), user.Email, "a distinct valid password", required)
			if err != nil || result == nil || result.Outcome != test.want {
				t.Fatalf("outcome: %v", err)
			}

			issued, completed := result.CompletedSession()
			if test.want != AuthenticationCompleted {
				if completed || issued != nil || result.Issued != nil || result.Reason != AuthenticationMethodUnavailable || generated != 0 || len(q.sessions) != 0 {
					t.Fatal("unmet requirement created authority")
				}

				_, err = svc.ValidateSession(t.Context(), "", required, NoActivity)
				if !errors.Is(err, ErrSessionToken) {
					t.Fatal("pending result could enter full-session lookup")
				}

				return
			}

			if !completed || generated != 1 || len(q.sessions) != 1 || issued.Proof.Method != PasswordProof || issued.PolicyRevision != "v1" {
				t.Fatal("password completion metadata")
			}

			_, err = svc.ValidateSession(t.Context(), issued.Token, AccessRequirement{Proof: RequireMFA, Revision: "v1"}, RelevantActivity)
			if !errors.Is(err, ErrSessionProof) {
				t.Fatal("weak session gained stronger authority")
			}
		})
	}
}

// A malformed trusted policy cannot begin credential work or produce a result.
func TestInvalidOperationPolicy(t *testing.T) {
	svc := newServiceForTest(t, newMockQueries(), config.New(), log.NewTestLogger("error"))

	result, err := svc.Signin(t.Context(), "missing@example.com", "invalid", AccessRequirement{})
	if result != nil || !errors.Is(err, ErrAccessRequirement) {
		t.Fatal("invalid sign-in requirement reached credentials")
	}

	validated, err := svc.ValidateSession(t.Context(), "invalid", AccessRequirement{}, NoActivity)
	if validated != nil || !errors.Is(err, ErrAccessRequirement) {
		t.Fatal("invalid validation requirement reached bearer parsing")
	}
}
