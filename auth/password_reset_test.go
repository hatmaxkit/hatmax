// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"testing"
	"time"

	"hatmax.adrianpk.com/config"
)

// Both purposes have valid distinct lifetimes/digests. A valid wire token still
// needs the intended purpose at its operation boundary; shape alone is no authority.
func TestResetTokenPurpose(t *testing.T) {
	settings, err := (config.RecoveryConfig{}).RecoverySettings()
	if err != nil {
		t.Fatal(err)
	}

	token, err := newRecoveryToken()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	p := MailboxRecord{ID: token.ID(), State: CredentialState{UserID: "subject", Version: 1}, Purpose: ResetPassword, Target: "user@example.com", PolicyRevision: "test-v1", Digest: RecoveryDigest(token, "subject", ResetPassword), CreatedAt: now, ExpiresAt: now.Add(settings.ResetTTL), AttemptLimit: settings.TokenAttempts, Revision: 1}

	err = p.Check(now, "test-v1", settings)
	if err != nil || !token.Matches(p) {
		t.Fatal("valid reset purpose rejected")
	}

	for _, tc := range []struct {
		name   string
		change func(*MailboxRecord)
	}{
		{"verification digest", func(p *MailboxRecord) { p.Digest = RecoveryDigest(token, "subject", VerifyMailbox) }},
		{"verification purpose", func(p *MailboxRecord) { p.Purpose = VerifyMailbox }},
		{"other subject", func(p *MailboxRecord) { p.State.UserID = "foreign" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := p
			tc.change(&changed)

			if token.Matches(changed) {
				t.Fatal("foreign authority accepted")
			}
		})
	}

	for _, tc := range []struct {
		name   string
		change func(*MailboxRecord)
	}{
		{"unknown purpose", func(p *MailboxRecord) { p.Purpose = 3 }},
		{"verification lifetime", func(p *MailboxRecord) { p.ExpiresAt = p.CreatedAt.Add(settings.VerificationTTL) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := p
			tc.change(&changed)

			err := changed.Check(now, "test-v1", settings)
			if err == nil {
				t.Fatal("wrong reset shape accepted")
			}
		})
	}
}
