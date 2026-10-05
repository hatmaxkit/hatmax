//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"database/sql"
	"errors"
	"slices"
	"testing"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
)

type resetFixture struct {
	db       *sql.DB
	q        *Queries
	base     *core.Service
	service  *core.RecoveryService
	issue    *core.MailboxIssue
	actor    *core.IssuedSession
	settings config.RecoverySettings
}

func resetIssueFor(t *testing.T, q *Queries, base *core.Service, options config.RecoveryConfig) (*core.RecoveryService, *core.MailboxIssue) {
	t.Helper()
	verification := recoveryForChange(t, base, q, config.RecoveryConfig{})
	message, err := verification.RequestMailboxVerification(t.Context(), "session@example.com")
	if err != nil {
		t.Fatal(err)
	}
	_, err = verification.ConfirmMailboxVerification(t.Context(), message.Token.Bearer())
	if err != nil {
		t.Fatal(err)
	}
	s := recoveryForChange(t, base, q, options)
	issue, err := s.RequestPasswordReset(t.Context(), "session@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return s, issue
}
func newResetFixture(t *testing.T, options config.RecoveryConfig) resetFixture {
	t.Helper()
	db, q, base, _ := sessionFixture(t)
	s, issue := resetIssueFor(t, q, base, options)
	actor, err := testSignin(base, t.Context(), "session@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}
	settings, err := options.RecoverySettings()
	if err != nil {
		t.Fatal(err)
	}
	return resetFixture{db: db, q: q, base: base, service: s, issue: issue, actor: actor, settings: settings}
}

// Actual verification precedes reset issuance. Real credential/session storage
// establishes one-use replacement, complete invalidation and no new access.
func TestPasswordResetTransactions(t *testing.T) {
	t.Run("complete reset", testPasswordResetComplete)
}
func testPasswordResetComplete(t *testing.T) {
	f := newResetFixture(t, config.RecoveryConfig{})
	enrollment, err := core.NewAuthenticatorService(f.base, f.q, config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	pending, err := enrollment.BeginWebAuthnEnrollment(t.Context(), "session@example.com", "a distinct safe password", enrollmentRequirement())
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.q.GetUserByID(t.Context(), f.actor.UserID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.base.Signup(t.Context(), "other@example.com", "another safe long password")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := testSignin(f.base, t.Context(), "other@example.com", "another safe long password")
	if err != nil {
		t.Fatal(err)
	}
	result, err := f.service.ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
	if err != nil || result == nil {
		t.Fatalf("reset: %v", err)
	}
	after, err := f.q.GetUserByID(t.Context(), before.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.AuthVersion != before.AuthVersion+1 || after.MailboxVerifiedAt == nil || !after.MailboxVerifiedAt.Equal(*before.MailboxVerifiedAt) || after.PasswordHash == before.PasswordHash || !after.Active || !slices.Equal(before.Roles, after.Roles) {
		t.Fatal("invalid reset account effects")
	}
	if mailboxCount(t, f.db, "SELECT count(*) FROM sessions WHERE user_id=$1", before.ID) != 0 ||
		mailboxCount(t, f.db, "SELECT count(*) FROM auth_pending WHERE user_id=$1", before.ID) != 0 ||
		mailboxCount(t, f.db, "SELECT count(*) FROM mailbox_tokens WHERE id=$1 AND consumed_at IS NOT NULL AND revoked_at IS NULL", f.issue.Token.ID()) != 1 ||
		mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices WHERE user_id=$1 AND kind=3", before.ID) != 1 {
		t.Fatal("incomplete reset transaction")
	}
	_, err = enrollment.FinishWebAuthnEnrollment(t.Context(), pending.Token, enrollmentResponse(t, pending), enrollmentRequirement())
	if err == nil {
		t.Fatal("pending enrollment survived")
	}
	_, err = f.service.ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
	if err == nil {
		t.Fatal("reset token replayed")
	}
	_, err = f.base.ValidateSession(t.Context(), f.actor.Token, PasswordRequirement(), core.NoActivity)
	if err == nil {
		t.Fatal("old actor survived")
	}
	_, err = f.base.ValidateSession(t.Context(), foreign.Token, PasswordRequirement(), core.NoActivity)
	if err != nil {
		t.Fatal("foreign session revoked")
	}
	_, err = f.base.Signin(t.Context(), before.Email, "a distinct safe password", PasswordRequirement())
	if !errors.Is(err, core.ErrInvalidPassword) {
		t.Fatalf("old password: %v", err)
	}
	out, err := f.base.Signin(t.Context(), before.Email, changedPassword, PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}
	_, ok := out.CompletedSession()
	if !ok {
		t.Fatal("new credential failed")
	}
}
