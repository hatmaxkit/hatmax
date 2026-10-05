//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

// Captured events exercise real adapters; callbacks acquire the real subject lock.
type securityCapture struct {
	mu         sync.Mutex
	events     []core.SecurityEvent
	db         *sql.DB
	probeError error
	probes     int
	after      func(context.Context, core.SecurityEvent) error
}

func (c *securityCapture) Observe(ctx context.Context, event core.SecurityEvent) error {
	c.mu.Lock()
	c.events = append(c.events, event)
	c.mu.Unlock()

	if event.Subject != "" {
		tx, err := c.db.BeginTx(ctx, nil)
		if err == nil {
			_, err = tx.ExecContext(ctx, "SET LOCAL lock_timeout='25ms'")
			if err == nil {
				var id string

				err = tx.QueryRowContext(ctx, "SELECT id FROM users WHERE id=$1 FOR UPDATE", event.Subject).Scan(&id)
			}

			_ = tx.Rollback()
		}

		c.mu.Lock()
		if err != nil {
			c.probeError = err
		} else {
			c.probes++
		}
		c.mu.Unlock()

		if err != nil {
			return err
		}
	}

	if c.after != nil {
		return c.after(ctx, event)
	}

	return ctx.Err()
}

func (c *securityCapture) take(t *testing.T, operation core.SecurityOperation, outcome core.SecurityOutcome, proof core.ProofMethod) core.SecurityEvent {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.probeError != nil {
		t.Fatalf("observer could not acquire released subject lock: %v", c.probeError)
	}

	if len(c.events) != 1 {
		t.Fatalf("terminal event count: %d", len(c.events))
	}

	e := c.events[0]
	c.events = nil

	err := e.Check()
	if err != nil {
		t.Fatal(err)
	}

	if e.Operation != operation || e.Outcome != outcome || e.Proof != proof {
		t.Fatalf("event classification: %+v", e)
	}

	return e
}

func securityFixture(t *testing.T) (*sql.DB, *Queries, *config.Config, *core.Service, *core.SecurityObservations, *securityCapture) {
	t.Helper()
	db, q, cfg := credentialDatabase(t)
	capture := &securityCapture{db: db}

	observations, err := core.NewSecurityObservations(capture, cfg.SecurityObservation)
	if err != nil {
		t.Fatal(err)
	}

	base, err := core.NewService(q, cfg, NewPasswordChecker(), credentialAdmissionForTest(t, q, cfg.CredentialAdmission), observations, log.NewTestLogger("error"))
	if err != nil {
		t.Fatal(err)
	}

	return db, q, cfg, base, observations, capture
}

// Real PostgreSQL, credentials, signed UV WebAuthn and OTP/backup proofs establish
// event provenance, released locks, neutral denials and unchanged authority.
func TestSecurityObservationTransactions(t *testing.T) {
	t.Run("password sessions recovery", testSecurityPassword)
	t.Run("verified factor outcomes", testSecurityFactors)
	t.Run("initial fallback proof", testSecurityInitialFallback)
	t.Run("delivery and ambiguous commit", testSecurityFailure)
	t.Run("saturated committed mutation", testSecuritySaturation)
}

func testSecurityPassword(t *testing.T) {
	db, q, _, base, _, capture := securityFixture(t)
	password := "a distinct safe password"

	_, err := base.Signin(t.Context(), "raw@example.com", strings.Repeat("private-password\n", 300), PasswordRequirement())
	if err == nil {
		t.Fatal("oversized password admitted")
	}

	malformed := capture.take(t, core.SecuritySignin, core.SecurityInvalidProof, 0)
	if malformed.Subject != "" || malformed.Record != "" {
		t.Fatal("untrusted identity became reference")
	}

	user, err := base.Signup(t.Context(), "session@example.com", password)
	if err != nil {
		t.Fatal(err)
	}

	e := capture.take(t, core.SecurityRegistration, core.SecurityCommitted, 0)
	if e.Subject != user.ID || e.Record != user.ID {
		t.Fatal("registration reference is not committed user")
	}

	_, err = base.Signup(t.Context(), "session@example.com", password)
	if !errors.Is(err, core.ErrEmailTaken) {
		t.Fatal("duplicate authority changed")
	}

	capture.take(t, core.SecurityRegistration, core.SecurityPolicyRejected, 0)

	_, err = base.Signin(t.Context(), "unknown@example.com", password, PasswordRequirement())
	if !errors.Is(err, core.ErrUserNotFound) {
		t.Fatal(err)
	}

	capture.take(t, core.SecuritySignin, core.SecurityMissing, 0)

	_, err = base.Signin(t.Context(), "session@example.com", "wrong password", PasswordRequirement())
	if !errors.Is(err, core.ErrInvalidPassword) {
		t.Fatal(err)
	}

	capture.take(t, core.SecuritySignin, core.SecurityInvalidProof, 0)

	err = q.UpdateUserActive(t.Context(), user.ID, false, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	_, err = base.Signin(t.Context(), "session@example.com", password, PasswordRequirement())
	if !errors.Is(err, core.ErrUserInactive) {
		t.Fatal(err)
	}

	capture.take(t, core.SecuritySignin, core.SecurityInactive, 0)

	err = q.UpdateUserActive(t.Context(), user.ID, true, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	partial, err := base.Signin(t.Context(), "session@example.com", password, fallbackRequirement())
	if err != nil || partial.Issued != nil || partial.Outcome != core.AuthenticationPendingEnrollment {
		t.Fatal("partial proof authorized access")
	}

	capture.take(t, core.SecuritySignin, core.SecurityPendingEnrollment, core.PasswordProof)

	actor, err := testSignin(base, t.Context(), "session@example.com", password)
	if err != nil {
		t.Fatal(err)
	}

	e = capture.take(t, core.SecuritySignin, core.SecurityAuthenticated, core.PasswordProof)
	if e.Record != actor.ID || e.Subject != actor.UserID {
		t.Fatal("session references are not committed result")
	}

	_, err = base.ValidateSession(t.Context(), actor.Token, PasswordRequirement(), core.NoActivity)
	if err != nil {
		t.Fatal(err)
	}

	_, err = base.ListSessions(t.Context(), actor.Token, PasswordRequirement(), "")
	if err != nil {
		t.Fatal(err)
	}

	if len(capture.events) != 0 {
		t.Fatal("lookup or listing emitted mutation event")
	}

	result, err := base.Reauthenticate(t.Context(), actor.Token, password, PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}

	next, ok := result.CompletedSession()
	if !ok {
		t.Fatal("rotation did not complete")
	}

	capture.take(t, core.SecurityReauthentication, core.SecurityAuthenticated, core.PasswordProof)

	_, err = base.RevokeSessions(t.Context(), next.Token, PasswordRequirement(), core.SessionSelection{Scope: core.SessionOthers})
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecuritySessionRevocation, core.SecurityCommitted, 0)

	err = base.Signout(t.Context(), next.Token)
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecuritySignout, core.SecurityCommitted, 0)
	recovery := recoveryForChange(t, base, q, config.RecoveryConfig{})

	issue, err := recovery.RequestMailboxVerification(t.Context(), "session@example.com")
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityMailboxIssue, core.SecurityPending, 0)

	verified, err := recovery.ConfirmMailboxVerification(t.Context(), issue.Token.Bearer())
	if err != nil || verified.Subject != user.ID {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityMailboxVerification, core.SecurityCommitted, 0)

	_, err = recovery.ConfirmMailboxVerification(t.Context(), issue.Token.Bearer())
	if !errors.Is(err, core.ErrRecoveryUnavailable) {
		t.Fatal("replay succeeded")
	}

	capture.take(t, core.SecurityMailboxVerification, core.SecurityUnavailableState, 0)

	reset, err := recovery.RequestPasswordReset(t.Context(), "session@example.com")
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityPasswordResetIssue, core.SecurityPending, 0)

	resetResult, err := recovery.ResetPassword(t.Context(), reset.Token.Bearer(), changedPassword)
	if err != nil || resetResult.Subject != user.ID {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityPasswordReset, core.SecurityCommitted, 0)

	actor, err = testSignin(base, t.Context(), "session@example.com", changedPassword)
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecuritySignin, core.SecurityAuthenticated, core.PasswordProof)

	_, err = recovery.ChangePassword(t.Context(), actor.Token, "another distinct complete password", passwordChangePolicy())
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityPasswordChange, core.SecurityCommitted, core.PasswordProof)

	if mailboxCount(t, db, "SELECT count(*) FROM sessions") != 0 || mailboxCount(t, db, "SELECT count(*) FROM recovery_notices") != 3 {
		t.Fatal("security event changed invalidation or durable notification authority")
	}

	if capture.probes < 10 {
		t.Fatal("insufficient actual released-lock observations")
	}
}

func testSecurityFactors(t *testing.T) {
	db, q, _, base, _, capture := securityFixture(t)

	_, err := base.Signup(t.Context(), "session@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityRegistration, core.SecurityCommitted, 0)

	enrollment, err := core.NewAuthenticatorService(base, q, factorConfig())
	if err != nil {
		t.Fatal(err)
	}

	challenge, err := enrollment.BeginWebAuthnEnrollment(t.Context(), "session@example.com", "a distinct safe password", enrollmentRequirement())
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityEnrollmentBegin, core.SecurityPending, core.PasswordProof)
	body, key, id := enrollmentKeyResponse(t, challenge)

	factor, err := enrollment.FinishWebAuthnEnrollment(t.Context(), challenge.Token, body, enrollmentRequirement())
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityEnrollmentFinish, core.SecurityCommitted, 0)

	webauthn, err := core.NewWebAuthnService(base, q, factorConfig())
	if err != nil {
		t.Fatal(err)
	}

	fixture := webAuthnFixture{db: db, q: q, base: base, svc: webauthn, key: key, id: id, factor: factor}
	assertion := beginAssertion(t, fixture)
	capture.take(t, core.SecurityWebAuthnBegin, core.SecurityPending, 0)

	_, err = webauthn.FinishWebAuthn(t.Context(), assertion.Token, []byte("private-password\nraw@example.com"), webAuthnRequirement())
	if err == nil {
		t.Fatal("malformed assertion authorized")
	}

	bad := capture.take(t, core.SecurityWebAuthnFinish, core.SecurityInvalidProof, 0)

	encoded, err := json.Marshal(bad)
	if err != nil || strings.Contains(string(encoded), "private-password") || strings.Contains(string(encoded), "raw@example.com") {
		t.Fatal("malformed assertion leaked credential data")
	}

	actor := finishAssertion(t, fixture, assertion, 1)
	capture.take(t, core.SecurityWebAuthnFinish, core.SecurityAuthenticated, core.WebAuthnProof)

	_, err = webauthn.FinishWebAuthn(t.Context(), assertion.Token, signedAssertion(t, fixture, assertion, 2, 5, nil), webAuthnRequirement())
	if err == nil {
		t.Fatal("replayed assertion succeeded")
	}

	capture.take(t, core.SecurityWebAuthnFinish, core.SecurityUnavailableState, 0)

	step, err := webauthn.BeginWebAuthnStepUp(t.Context(), actor.Token, webAuthnRequirement())
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityWebAuthnStepUp, core.SecurityPending, core.WebAuthnProof)
	actor = finishAssertion(t, fixture, step, 2)
	capture.take(t, core.SecurityWebAuthnFinish, core.SecurityAuthenticated, core.WebAuthnProof)
	fallback := fallbackService(t, base, q, config.FallbackConfig{})

	management, err := core.NewFactorService(base, q, enrollment, fallback)
	if err != nil {
		t.Fatal(err)
	}

	policy := factorPolicy()

	begin, err := management.BeginWebAuthnChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityWebAuthnChangeBegin, core.SecurityPending, core.WebAuthnProof)
	changeBody, _, _ := enrollmentKeyResponse(t, begin)

	changed, err := management.FinishWebAuthnChange(t.Context(), actor.Token, begin.Token, changeBody, policy)
	if err != nil || changed.Issued == nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityWebAuthnChangeFinish, core.SecurityCommitted, core.WebAuthnProof)

	secondFactor := changed.Factor
	actor = changed.Issued

	totpSetup, err := management.BeginTOTPChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityTOTPChangeBegin, core.SecurityPending, core.WebAuthnProof)

	parsed, err := url.Parse(totpSetup.URL)
	if err != nil {
		t.Fatal(err)
	}

	secret := parsed.Query().Get("secret")

	code, err := previousTOTPCode(t, secret)
	if err != nil {
		t.Fatal(err)
	}

	changed, err = management.FinishTOTPChange(t.Context(), actor.Token, totpSetup.Token, code, policy)
	if err != nil || changed.Issued == nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityTOTPChangeFinish, core.SecurityCommitted, core.WebAuthnProof)

	actor = changed.Issued

	codes, rotated, err := fallback.IssueBackupCodes(t.Context(), actor.Token, webAuthnRequirement())
	if err != nil || len(codes) == 0 || rotated == nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityBackupIssue, core.SecurityCommitted, core.WebAuthnProof)

	actor = rotated

	pending, err := fallback.BeginFallbackAuthentication(t.Context(), "session@example.com", "a distinct safe password", core.FallbackBackup, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityFallbackBegin, core.SecurityPending, core.PasswordProof)

	backup, err := fallback.FinishFallback(t.Context(), pending.Token, codes[0], fallbackRequirement())
	if err != nil || backup == nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityFallbackFinish, core.SecurityAuthenticated, core.PasswordBackupProof)

	pending, err = fallback.BeginFallbackStepUp(t.Context(), backup.Token, "a distinct safe password", core.FallbackTOTP, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityFallbackStepUp, core.SecurityPending, core.PasswordProof)

	totpFixture := fallbackFixture{db: db, q: q, base: base, svc: fallback, secret: secret}

	otp, err := fallback.FinishFallback(t.Context(), pending.Token, fallbackCode(t, totpFixture), fallbackRequirement())
	if err != nil || otp == nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityFallbackFinish, core.SecurityAuthenticated, core.PasswordTOTPProof)
	// Remove the second passkey while retaining the actor's original proof constituent.
	removed, err := management.Remove(t.Context(), actor.Token, core.FactorSelection{Kind: core.FactorWebAuthn, ID: secondFactor.ID, Revision: secondFactor.Revision}, policy)
	if err != nil || removed == nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityFactorRemoval, core.SecurityCommitted, core.WebAuthnProof)

	if capture.probes < 15 {
		t.Fatal("factor observations did not acquire released locks")
	}
}

func testSecurityInitialFallback(t *testing.T) {
	db, q, _, base, _, capture := securityFixture(t)

	_, err := base.Signup(t.Context(), "session@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityRegistration, core.SecurityCommitted, 0)
	fallback := fallbackService(t, base, q, config.FallbackConfig{})

	setup, err := fallback.BeginTOTPSetup(t.Context(), "session@example.com", "a distinct safe password", fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityTOTPSetupBegin, core.SecurityPending, core.PasswordProof)

	parsed, err := url.Parse(setup.URL)
	if err != nil {
		t.Fatal(err)
	}

	secret := parsed.Query().Get("secret")

	code, err := previousTOTPCode(t, secret)
	if err != nil {
		t.Fatal(err)
	}

	err = fallback.ConfirmTOTPSetup(t.Context(), setup.Token, code, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityTOTPSetupFinish, core.SecurityCommitted, 0)

	partial, err := base.Signin(t.Context(), "session@example.com", "a distinct safe password", fallbackRequirement())
	if err != nil || partial.Issued != nil || partial.Outcome != core.AuthenticationPendingEnrollment {
		t.Fatal("password proof satisfied MFA")
	}

	capture.take(t, core.SecuritySignin, core.SecurityPendingEnrollment, core.PasswordProof)

	p, err := fallback.BeginFallbackAuthentication(t.Context(), "session@example.com", "a distinct safe password", core.FallbackTOTP, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityFallbackBegin, core.SecurityPending, core.PasswordProof)

	f := fallbackFixture{db: db, q: q, base: base, svc: fallback, secret: secret}

	issued, err := fallback.FinishFallback(t.Context(), p.Token, fallbackCode(t, f), fallbackRequirement())
	if err != nil || issued.Proof.Method != core.PasswordTOTPProof {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityFallbackFinish, core.SecurityAuthenticated, core.PasswordTOTPProof)
}

type lostSecurityQueries struct{ *Queries }

func (q lostSecurityQueries) CreateSession(ctx context.Context, state core.CredentialState, record core.SessionRecord, required core.AccessRequirement, limit int) (*core.Session, error) {
	_, err := q.Queries.CreateSession(ctx, state, record, required, limit)
	if err != nil {
		return nil, err
	}

	return nil, context.DeadlineExceeded
}

func testSecurityFailure(t *testing.T) {
	db, q, cfg, base, observations, capture := securityFixture(t)
	capture.after = func(context.Context, core.SecurityEvent) error {
		return errors.New("private observer password bearer mailbox detail")
	}

	user, err := base.Signup(t.Context(), "session@example.com", "a distinct safe password")
	if err != nil || user == nil {
		t.Fatal("observer error changed committed signup")
	}

	capture.take(t, core.SecurityRegistration, core.SecurityCommitted, 0)

	if observations.Diagnostics().Operating != 1 || mailboxCount(t, db, "SELECT count(*) FROM users") != 1 {
		t.Fatal("failed callback changed storage")
	}

	capture.after = func(ctx context.Context, _ core.SecurityEvent) error {
		<-ctx.Done()

		return nil
	}

	actor, err := testSignin(base, t.Context(), "session@example.com", "a distinct safe password")
	if err != nil || actor == nil {
		t.Fatal("late callback changed session issuance")
	}

	capture.take(t, core.SecuritySignin, core.SecurityAuthenticated, core.PasswordProof)

	if observations.Diagnostics().Deadline != 1 || observations.Diagnostics().Delivered != 0 {
		t.Fatal("late success counted as delivered")
	}

	_, err = base.ValidateSession(t.Context(), actor.Token, PasswordRequirement(), core.NoActivity)
	if err != nil {
		t.Fatal("committed session lost authority after observation timeout")
	}

	capture.after = nil

	lost, err := core.NewService(lostSecurityQueries{q}, cfg, NewPasswordChecker(), credentialAdmissionForTest(t, q, cfg.CredentialAdmission), observations, log.NewTestLogger("error"))
	if err != nil {
		t.Fatal(err)
	}

	unknown, err := lost.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
	if !errors.Is(err, context.DeadlineExceeded) || unknown != nil {
		t.Fatal("lost commit result was retried or authorized")
	}

	event := capture.take(t, core.SecuritySignin, core.SecurityOperatingUnknown, core.PasswordProof)
	if event.Record != "" || mailboxCount(t, db, "SELECT count(*) FROM sessions") != 2 {
		t.Fatal("ambiguous commit was called completed or rolled back")
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}

	for _, secret := range []string{actor.Token, "session@example.com", "a distinct safe password", "private observer"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("security event leaked private material")
		}
	}
	// Re-entry performs an independent real mutation after probe transactions end.
	capture.after = func(ctx context.Context, event core.SecurityEvent) error {
		if event.Operation == core.SecuritySignin && event.Outcome == core.SecurityMissing {
			return base.Signout(ctx, actor.Token)
		}

		return ctx.Err()
	}

	_, err = base.Signin(t.Context(), "unknown@example.com", "a distinct safe password", PasswordRequirement())
	if !errors.Is(err, core.ErrUserNotFound) {
		t.Fatal(err)
	}

	capture.mu.Lock()
	reentered := append([]core.SecurityEvent(nil), capture.events...)
	capture.events = nil
	capture.mu.Unlock()

	if len(reentered) != 2 || reentered[0].Operation != core.SecuritySignin || reentered[0].Outcome != core.SecurityMissing || reentered[1].Operation != core.SecuritySignout || reentered[1].Outcome != core.SecurityCommitted {
		t.Fatal("independent reentrant entrypoint lost or duplicated events")
	}

	if observations.Diagnostics().Delivered != 3 {
		t.Fatal("callback re-entry held locks or waited for capacity")
	}

	_, err = base.ValidateSession(t.Context(), actor.Token, PasswordRequirement(), core.NoActivity)
	if !errors.Is(err, core.ErrSessionNotFound) {
		t.Fatal("reentrant signout did not commit")
	}
}

// Real session deletion commits while another callback owns the only observation slot.
func testSecuritySaturation(t *testing.T) {
	_, q, cfg, base, _, capture := securityFixture(t)

	_, err := base.Signup(t.Context(), "session@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecurityRegistration, core.SecurityCommitted, 0)

	actor, err := testSignin(base, t.Context(), "session@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	capture.take(t, core.SecuritySignin, core.SecurityAuthenticated, core.PasswordProof)

	one, err := core.NewSecurityObservations(capture, config.SecurityObservationConfig{Concurrency: 1})
	if err != nil {
		t.Fatal(err)
	}

	bounded, err := core.NewService(q, cfg, NewPasswordChecker(), credentialAdmissionForTest(t, q, cfg.CredentialAdmission), one, log.NewTestLogger("error"))
	if err != nil {
		t.Fatal(err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	capture.after = func(ctx context.Context, _ core.SecurityEvent) error {
		close(entered)

		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	done := make(chan error, 1)

	go func() {
		_, err := testSignin(bounded, t.Context(), "session@example.com", "a distinct safe password")
		done <- err
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("actual callback not reached")
	}

	err = bounded.Signout(t.Context(), actor.Token)

	close(release)

	if err != nil {
		t.Fatal("saturation prevented committed signout")
	}

	err = <-done
	if err != nil {
		t.Fatal("held observer prevented committed sign-in")
	}

	capture.take(t, core.SecuritySignin, core.SecurityAuthenticated, core.PasswordProof)

	if one.Diagnostics().Saturated != 1 || one.Diagnostics().Delivered != 1 {
		t.Fatal("saturated callback waited or repeated mutation")
	}

	_, err = base.ValidateSession(t.Context(), actor.Token, PasswordRequirement(), core.NoActivity)
	if !errors.Is(err, core.ErrSessionNotFound) {
		t.Fatal("saturated signout retained authority")
	}
}
