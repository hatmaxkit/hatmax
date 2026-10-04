// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"bytes"
	"github.com/pquerna/otp/totp"
	"hatmax.adrianpk.com/crypto"
	"hatmax.adrianpk.com/log"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/config"
)

// Ciphertext cannot be moved between subjects, factor identities or key identities.
func TestSeedBinding(t *testing.T) {
	key := make([]byte, 32)

	seeds, err := newSeedCipher(SeedKeys{Active: "key-one", Keys: map[string][]byte{"key-one": key, "key-two": key}})
	if err != nil {
		t.Fatal(err)
	}

	secret := "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"

	id, envelope, err := seeds.seal("subject", "factor", secret)
	if err != nil || id != "key-one" {
		t.Fatal(err)
	}
	// Mutating caller-owned keys after construction cannot change the copied cipher.
	key[0] = 1

	opened, err := seeds.open("subject", "factor", id, envelope)
	if err != nil || opened != secret {
		t.Fatal("key aliasing changed cipher")
	}

	_, second, err := seeds.seal("subject", "factor", secret)
	if err != nil || bytes.Equal(second, envelope) {
		t.Fatal("nonce repeated")
	}

	tests := []struct {
		name, subject, factor, key string
		change                     func([]byte)
	}{
		{name: "foreign subject", subject: "other", factor: "factor", key: id},
		{name: "foreign factor", subject: "subject", factor: "other", key: id},
		{name: "alternate key identity", subject: "subject", factor: "factor", key: "key-two"},
		{name: "missing key", subject: "subject", factor: "factor", key: "unknown"},
		{name: "tampered ciphertext", subject: "subject", factor: "factor", key: id, change: func(b []byte) { b[len(b)-1] ^= 1 }},
		{name: "unknown version", subject: "subject", factor: "factor", key: id, change: func(b []byte) { b[0] = 2 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := append([]byte(nil), envelope...)
			if test.change != nil {
				test.change(b)
			}

			value, err := seeds.open(test.subject, test.factor, test.key, b)
			if err == nil || value != "" {
				t.Fatal("substituted envelope opened")
			}
		})
	}
}

// Construction rejects absent/oversized key rings and any unsupported key identity/size.
func TestSeedConfiguration(t *testing.T) {
	tests := []struct {
		name string
		keys SeedKeys
	}{
		{name: "absent"},
		{name: "unknown active", keys: SeedKeys{Active: "unknown", Keys: map[string][]byte{"one": make([]byte, 32)}}},
		{name: "short", keys: SeedKeys{Active: "one", Keys: map[string][]byte{"one": make([]byte, 31)}}},
		{name: "long", keys: SeedKeys{Active: "one", Keys: map[string][]byte{"one": make([]byte, 33)}}},
		{name: "control identity", keys: SeedKeys{Active: "one\n", Keys: map[string][]byte{"one\n": make([]byte, 32)}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := newSeedCipher(test.keys)
			if err == nil || s != nil {
				t.Fatal("invalid key configuration accepted")
			}
		})
	}
}

// Freshness is the oldest actual constituent, even with a newer factor/activity.
func TestCompositeFreshness(t *testing.T) {
	now := time.Unix(1800000000, 0)

	settings, err := (config.AuthConfig{}).SessionSettings()
	if err != nil {
		t.Fatal(err)
	}

	for _, method := range []ProofMethod{PasswordTOTPProof, PasswordBackupProof} {
		s := Session{ID: "session", UserID: "subject", AuthVersion: 1, PolicyRevision: "policy", Generation: 1, CreatedAt: now.Add(-5 * time.Minute), AuthenticatedAt: now, LastActivityAt: now, ExpiresAt: now.Add(settings.TTL), InactivityTTL: settings.InactivityTTL, Proof: VerifiedProof{Method: method, VerifiedAt: now.Add(-5 * time.Minute), FactorAt: now, FactorID: "factor", FactorRevision: 1}}
		required := AccessRequirement{Proof: RequireMFA, Revision: "policy", MaxAge: 5 * time.Minute}

		err = required.Evaluate(s, now)
		if err != ErrSessionProofExpired {
			t.Fatal("newer factor renewed old password proof")
		}

		s.Proof.VerifiedAt = now.Add(-5*time.Minute + time.Microsecond)

		err = required.Evaluate(s, now)
		if err != nil {
			t.Fatal(err)
		}

		required.Proof = RequirePhishingResistantMFA

		err = required.Evaluate(s, now)
		if err != ErrSessionProof {
			t.Fatal("fallback provided phishing resistance")
		}
	}
}

type unreachableFallback struct{ FallbackQueries }

// Full verifier admission fails synchronously before storage or cryptographic work.
func TestFallbackAdmission(t *testing.T) {
	base := newServiceForTest(t, newMockQueries(), config.New(), log.NewTestLogger("error"))

	svc, err := NewFallbackService(base, unreachableFallback{}, config.FallbackConfig{Issuer: "Example"}, SeedKeys{Active: "one", Keys: map[string][]byte{"one": make([]byte, 32)}})
	if err != nil {
		t.Fatal(err)
	}

	for range cap(svc.slots) {
		svc.slots <- struct{}{}
	}

	required := AccessRequirement{Proof: RequireMFA, Revision: "policy", MaxAge: time.Minute}

	issued, err := svc.FinishFallback(t.Context(), "fallback1."+strings.Repeat("A", 43), "123456", required)
	if err != ErrEnrollmentBusy || issued != nil {
		t.Fatal("full admission reached storage")
	}

	err = svc.ConfirmTOTPSetup(t.Context(), "totpset1."+strings.Repeat("A", 43), "123456", required)
	if err != ErrEnrollmentBusy {
		t.Fatal("setup ignored full admission")
	}

	codes, actor, err := svc.IssueBackupCodes(t.Context(), "unreachable", required)
	if err != ErrEnrollmentBusy || codes != nil || actor != nil {
		t.Fatal("backup generation ignored full admission")
	}
}

// An actual matched previous-period code expires at the next exact step boundary,
// even while pending, lease and password proof remain valid.
func TestTOTPCompletionWindow(t *testing.T) {
	now := time.Unix(1800000029, 0).UTC()
	secret := "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"

	code, err := totp.GenerateCode(secret, now.Add(-30*time.Second))
	if err != nil {
		t.Fatal(err)
	}

	step, err := crypto.MatchTOTPCode(secret, code, now, 1)
	if err != nil {
		t.Fatal(err)
	}

	settings, err := (config.FallbackConfig{Issuer: "Example"}).Settings()
	if err != nil {
		t.Fatal(err)
	}

	required := AccessRequirement{Proof: RequireMFA, Revision: "policy", MaxAge: time.Minute}
	p := FallbackPending{Digest: FallbackDigest{1}, Purpose: FallbackSignin, State: CredentialState{UserID: "subject", Version: 1}, Requirement: required, Material: FallbackMaterial{Method: FallbackTOTP, Skew: 1, Factor: FactorBinding{ID: "factor", Revision: 1}}, PasswordAt: now, CreatedAt: now, ExpiresAt: now.Add(settings.PendingTTL), LeaseUntil: now.Add(settings.Lease), Attempts: 1, Revision: 1}
	r := FallbackReservation{Pending: p, Factor: FallbackFactor{FactorBinding: p.Material.Factor, AcceptedStep: step - 1, ReplayRevision: 1}}
	s := Session{ID: "session", UserID: "subject", AuthVersion: 1, Generation: 1, PolicyRevision: required.Revision, Proof: VerifiedProof{Method: PasswordTOTPProof, VerifiedAt: now, FactorAt: now, FactorID: "factor", FactorRevision: 1}, CreatedAt: now, AuthenticatedAt: now, LastActivityAt: now, ExpiresAt: now.Add(time.Hour), InactivityTTL: 30 * time.Minute}
	c := FallbackCompletion{Step: step, FactorAt: now, Session: SessionRecord{Session: s, Digest: SessionDigest{1}}}

	err = c.Check(r, now, required, settings)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Check(r, now.Add(time.Second), required, settings)
	if err != ErrFallback {
		t.Fatal("step equality did not expire commit")
	}
}
