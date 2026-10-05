//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

// A consumed backup stays consumed and the remaining verifier survives the
// complete password mutation; actual fallback verification establishes both.
func testPasswordChangeBackups(t *testing.T) {
	f, codes, _ := backupFixture(t)
	challenge := beginFallback(t, f, core.FallbackBackup)

	actor, err := f.svc.FinishFallback(t.Context(), challenge.Token, codes[0], fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	var before string

	err = f.db.QueryRowContext(t.Context(), "SELECT jsonb_build_object('sets',(SELECT jsonb_agg(to_jsonb(s) ORDER BY id) FROM backup_sets s),'codes',(SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM backup_codes c))::text").Scan(&before)
	if err != nil {
		t.Fatal(err)
	}

	s := recoveryForChange(t, f.base, f.q, config.RecoveryConfig{})
	p := passwordChangePolicy()
	p.Requirement.Proof = core.RequireMFA

	_, err = s.ChangePassword(t.Context(), actor.Token, changedPassword, p)
	if err != nil {
		t.Fatal(err)
	}

	var after string

	err = f.db.QueryRowContext(t.Context(), "SELECT jsonb_build_object('sets',(SELECT jsonb_agg(to_jsonb(s) ORDER BY id) FROM backup_sets s),'codes',(SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM backup_codes c))::text").Scan(&after)
	if err != nil {
		t.Fatal(err)
	}

	if before != after {
		t.Fatal("backup verifier or consumption state changed")
	}

	outcome, err := f.base.Signin(t.Context(), "session@example.com", changedPassword, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := outcome.CompletedSession(); ok {
		t.Fatal("password bypassed current MFA")
	}

	for i, code := range codes {
		challenge, err = f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", changedPassword, core.FallbackBackup, fallbackRequirement())
		if err != nil {
			t.Fatal(err)
		}

		issued, finishErr := f.svc.FinishFallback(t.Context(), challenge.Token, code, fallbackRequirement())
		if i == 0 && (finishErr == nil || issued != nil) {
			t.Fatal("consumed backup revived")
		}

		if i == 1 && (finishErr != nil || issued == nil) {
			t.Fatalf("remaining backup: %v", finishErr)
		}
	}
}

// A factor is changed while the actual final transaction waits on the subject.
// This isolates post-lock factor checks without a credential-version shortcut.
func testPasswordChangeFactorLock(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"revision", "UPDATE authenticators SET revision=revision+1"},
		{"removed", "DELETE FROM authenticators"},
		{"actor generation", "UPDATE sessions SET generation=generation+1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			actor := finishAssertion(t, f, beginAssertion(t, f), 1)
			policy := passwordChangePolicy()
			snapshot, settings := passwordAuthorization(t, f.q, actor, policy)
			encoded := replacementHash(t)

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			holder, err := f.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer holder.Rollback()

			_, err = dal.New(holder).GetUserForAuth(ctx, actor.UserID)
			if err != nil {
				t.Fatal(err)
			}

			result := make(chan error, 1)

			go func() { _, e := f.q.CommitPasswordChange(ctx, *snapshot, encoded, policy, settings); result <- e }()

			waitSessionLock(t, ctx, f.db, holder)

			_, err = holder.ExecContext(ctx, tc.sql)
			if err != nil {
				t.Fatal(err)
			}

			err = holder.Commit()
			if err != nil {
				t.Fatal(err)
			}

			err = <-result
			if err == nil {
				t.Fatal("cached post-lock authority accepted")
			}

			if mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices") != 0 || mailboxCount(t, f.db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 1 {
				t.Fatal("authority failure committed or refunded admission")
			}
		})
	}
}

// All eight requests first commit real durable admission and enter the checker.
// Releasing them together proves the actual shared KDF rejects concurrent work,
// with exactly one complete mutation and no refunded failures.
func testPasswordChangeKDF(t *testing.T) {
	const workers = 8

	db, q, _, actor := sessionFixture(t)
	cfg := config.New()
	cfg.Auth.ArgonMemoryKiB = 19456
	cfg.Auth.ArgonIterations = 2
	cfg.Auth.ArgonParallelism = 1
	cfg.Auth.PasswordMaxConcurrent = 1
	entered := make(chan struct{}, workers)
	release := make(chan struct{})

	base, err := core.NewService(q, cfg, changeChecker{func(ctx context.Context, _ string) (bool, error) {
		entered <- struct{}{}

		select {
		case <-release:
			return false, nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}}, credentialAdmissionForTest(t, q, cfg.CredentialAdmission), log.NewTestLogger("error"))
	if err != nil {
		t.Fatal(err)
	}

	s := recoveryForChange(t, base, q, config.RecoveryConfig{})

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	results := make(chan error, workers)
	for range workers {
		go func() {
			_, e := s.ChangePassword(ctx, actor.Token, changedPassword, passwordChangePolicy())
			results <- e
		}()
	}

	for range workers {
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}

	close(release)

	busy, winners := 0, 0

	for range workers {
		err = <-results
		if err == nil {
			winners++
		}

		if errors.Is(err, model.ErrPasswordVerifierBusy) {
			busy++
		}
	}

	if winners != 1 || busy == 0 || mailboxCount(t, db, "SELECT auth_version FROM users") != 2 || mailboxCount(t, db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != workers || mailboxCount(t, db, "SELECT count(*) FROM recovery_notices WHERE kind=2") != 1 {
		t.Fatal("shared KDF/admission boundary failed")
	}
}
