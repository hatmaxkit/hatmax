// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/config"
)

// Canonical encodings prevent alternative representations of the same bearer;
// redaction and domain separation keep it outside unrelated authority namespaces.
func TestRecoveryToken(t *testing.T) {
	token, err := newRecoveryToken()
	if err != nil {
		t.Fatal(err)
	}

	raw := token.Bearer()

	parsed, err := ParseRecoveryToken(raw)
	if err != nil || parsed != token || len(raw) != 80 {
		t.Fatal("canonical round trip failed")
	}

	for _, tc := range []struct{ name, value string }{
		{"empty", ""}, {"extra", raw + "x"}, {"padded", raw + "="}, {"uppercase ID", strings.ToUpper(raw[:36]) + raw[36:]},
		{"wrong version", raw[:14] + "1" + raw[15:]}, {"wrong variant", raw[:19] + "0" + raw[20:]}, {"invalid secret", raw[:37] + strings.Repeat("!", 43)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseRecoveryToken(tc.value)
			if err == nil {
				t.Fatal("noncanonical bearer accepted")
			}
		})
	}

	digest := RecoveryDigest(token, "subject", VerifyMailbox)
	if digest == RecoveryDigest(token, "other", VerifyMailbox) || digest == RecoveryDigest(token, "subject", ResetPassword) {
		t.Fatal("digest domains overlap")
	}

	encoded, err := json.Marshal(MailboxIssue{Token: token, Target: "private@example.com"})
	if err != nil || strings.Contains(string(encoded), raw) || strings.Contains(string(encoded), "private") {
		t.Fatal("dispatch serialization disclosed secrets")
	}

	for _, format := range []string{"%v", "%+v", "%#v"} {
		if strings.Contains(fmt.Sprintf(format, token), raw[37:]) {
			t.Fatal("format disclosed secret")
		}
	}
}

// Stored state is accepted only for the current finite policy at a strictly
// earlier time than expiry; target, purpose and version are part of that state.
func TestMailboxRecord(t *testing.T) {
	token, err := newRecoveryToken()
	if err != nil {
		t.Fatal(err)
	}

	settings, err := (config.RecoveryConfig{}).RecoverySettings()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	valid := MailboxRecord{ID: token.ID(), State: CredentialState{UserID: "subject", Version: 1}, Purpose: VerifyMailbox, Target: "user@example.com", PolicyRevision: "v1", Digest: RecoveryDigest(token, "subject", VerifyMailbox), CreatedAt: now, ExpiresAt: now.Add(settings.VerificationTTL), AttemptLimit: settings.TokenAttempts, Revision: 1}
	if valid.Check(now, "v1", settings) != nil || !token.Matches(valid) {
		t.Fatal("valid record rejected")
	}

	tests := []struct {
		name   string
		change func(*MailboxRecord)
		clock  time.Time
	}{
		{"expiry equality", func(*MailboxRecord) {}, valid.ExpiresAt},
		{"purpose", func(p *MailboxRecord) { p.Purpose = ResetPassword }, now},
		{"changed policy", func(p *MailboxRecord) { p.PolicyRevision = "v2" }, now},
		{"invalid version", func(p *MailboxRecord) { p.State.Version = 0 }, now},
		{"consumed", func(p *MailboxRecord) { p.ConsumedAt = now }, now},
		{"revoked", func(p *MailboxRecord) { p.RevokedAt = now }, now},
		{"future creation", func(p *MailboxRecord) { p.CreatedAt = now.Add(time.Minute) }, now},
		{"wrong TTL", func(p *MailboxRecord) { p.ExpiresAt = p.ExpiresAt.Add(time.Second) }, now},
		{"negative attempts", func(p *MailboxRecord) { p.Attempts = -1 }, now},
		{"policy limit", func(p *MailboxRecord) { p.AttemptLimit++ }, now},
		{"target injection", func(p *MailboxRecord) { p.Target = "a\r\nb" }, now},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := valid
			tc.change(&p)

			if p.Check(tc.clock, "v1", settings) == nil {
				t.Fatal("invalid state accepted")
			}
		})
	}

	for _, change := range []func(*MailboxRecord){func(p *MailboxRecord) { p.State.UserID = "other" }, func(p *MailboxRecord) { p.Purpose = ResetPassword }, func(p *MailboxRecord) { p.ID = "foreign" }} {
		p := valid
		change(&p)

		if token.Matches(p) {
			t.Fatal("foreign token authority accepted")
		}
	}

	encoded, err := json.Marshal(valid)
	if err != nil || strings.Contains(string(encoded), "Digest") {
		t.Fatal("snapshot serialization exposed digest")
	}
}

// Untrusted input fuzzing is bounded and performs neither KDF nor database work.
func FuzzRecoveryToken(f *testing.F) {
	f.Add("")
	f.Add("invalid")

	token, err := newRecoveryToken()
	if err != nil {
		f.Fatal(err)
	}

	f.Add(token.Bearer())
	f.Fuzz(func(t *testing.T, input string) {
		parsed, err := ParseRecoveryToken(input)
		if err == nil && parsed.Bearer() != input {
			t.Fatal("parser accepted an alternative encoding")
		}
	})
}
