//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"github.com/pquerna/otp/totp"
	"reflect"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
)

// Actual retained factors verify normal post-reset authentication. Snapshot
// equality proves reset cannot rewrite replay state, envelopes or backup verifiers.
func testResetMFA(t *testing.T) {
	t.Run("WebAuthn", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		s, issue := resetIssueFor(t, f.q, f.base, config.RecoveryConfig{})
		actor := finishAssertion(t, f, beginAssertion(t, f), 1)

		before, err := dal.New(f.db).LockAuthenticator(t.Context(), f.factor.ID)
		if err != nil {
			t.Fatal(err)
		}

		_, err = s.ResetPassword(t.Context(), issue.Token.Bearer(), changedPassword)
		if err != nil {
			t.Fatal(err)
		}

		after, err := dal.New(f.db).LockAuthenticator(t.Context(), f.factor.ID)
		if err != nil {
			t.Fatal(err)
		}

		if !sameFactor(factorSnapshot(before), factorSnapshot(after)) {
			t.Fatal("WebAuthn/replay state changed")
		}

		_, err = f.base.ValidateSession(t.Context(), actor.Token, webAuthnRequirement(), core.NoActivity)
		if err == nil {
			t.Fatal("MFA session survived")
		}

		outcome, err := f.base.Signin(t.Context(), "session@example.com", changedPassword, webAuthnRequirement())
		if err != nil {
			t.Fatal(err)
		}

		_, ok := outcome.CompletedSession()
		if ok {
			t.Fatal("password bypassed WebAuthn")
		}

		challenge := beginAssertion(t, f)

		issued, err := f.svc.FinishWebAuthn(t.Context(), challenge.Token, signedAssertion(t, f, challenge, 1, 5, nil), webAuthnRequirement())
		if err == nil || issued != nil {
			t.Fatal("reset revived an old counter")
		}

		challenge = beginAssertion(t, f)
		finishAssertion(t, f, challenge, 2)
	})
	t.Run("TOTP", func(t *testing.T) {
		f := newFallbackFixture(t, config.FallbackConfig{})
		s, issue := resetIssueFor(t, f.q, f.base, config.RecoveryConfig{})
		actor := finishTOTP(t, f)

		before, err := dal.New(f.db).LockTOTP(t.Context(), actor.UserID)
		if err != nil {
			t.Fatal(err)
		}

		replayCode, err := totp.GenerateCode(f.secret, time.Unix(before.AcceptedStep*30, 0))
		if err != nil {
			t.Fatal(err)
		}

		_, err = s.ResetPassword(t.Context(), issue.Token.Bearer(), changedPassword)
		if err != nil {
			t.Fatal(err)
		}

		after, err := dal.New(f.db).LockTOTP(t.Context(), before.UserID)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(before, after) {
			t.Fatal("TOTP encryption/replay state changed")
		}

		challenge, err := f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", changedPassword, core.FallbackTOTP, fallbackRequirement())
		if err != nil {
			t.Fatal(err)
		}

		issued, err := f.svc.FinishFallback(t.Context(), challenge.Token, replayCode, fallbackRequirement())
		if err == nil || issued != nil {
			t.Fatal("accepted TOTP step revived")
		}

		challenge, err = f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", changedPassword, core.FallbackTOTP, fallbackRequirement())
		if err != nil {
			t.Fatal(err)
		}

		_, err = f.svc.FinishFallback(t.Context(), challenge.Token, fallbackCode(t, f), fallbackRequirement())
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("backup consumption", func(t *testing.T) {
		f, codes, _ := backupFixture(t)
		s, issue := resetIssueFor(t, f.q, f.base, config.RecoveryConfig{})

		challenge, err := f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", "a distinct safe password", core.FallbackBackup, fallbackRequirement())
		if err != nil {
			t.Fatal(err)
		}

		_, err = f.svc.FinishFallback(t.Context(), challenge.Token, codes[0], fallbackRequirement())
		if err != nil {
			t.Fatal(err)
		}

		snapshot := func() string {
			var data string

			err := f.db.QueryRowContext(t.Context(), "SELECT jsonb_build_object('sets',(SELECT jsonb_agg(to_jsonb(s) ORDER BY id) FROM backup_sets s),'codes',(SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM backup_codes c))::text").Scan(&data)
			if err != nil {
				t.Fatal(err)
			}

			return data
		}
		before := snapshot()

		_, err = s.ResetPassword(t.Context(), issue.Token.Bearer(), changedPassword)
		if err != nil {
			t.Fatal(err)
		}

		if snapshot() != before {
			t.Fatal("backup records changed")
		}

		outcome, err := f.base.Signin(t.Context(), "session@example.com", changedPassword, fallbackRequirement())
		if err != nil {
			t.Fatal(err)
		}

		_, ok := outcome.CompletedSession()
		if ok {
			t.Fatal("password bypassed MFA")
		}

		for i, code := range codes {
			challenge, err = f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", changedPassword, core.FallbackBackup, fallbackRequirement())
			if err != nil {
				t.Fatal(err)
			}

			issued, finishErr := f.svc.FinishFallback(t.Context(), challenge.Token, code, fallbackRequirement())
			if i == 0 && (finishErr == nil || issued != nil) {
				t.Fatal("spent backup revived")
			}

			if i == 1 && (finishErr != nil || issued == nil) {
				t.Fatalf("remaining backup: %v", finishErr)
			}
		}
	})
}
