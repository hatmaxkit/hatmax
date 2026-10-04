//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
)

type factorBoundary struct {
	*Queries
	before func(core.FactorChangePending, core.FactorChangeRecord, core.SessionRecord)
	deny   bool
}

func (q factorBoundary) CompleteFactorChange(ctx context.Context, p core.FactorChangePending, r core.FactorChangeRecord, s core.SessionRecord, policy core.FactorPolicy, settings config.EnrollmentSettings) (*core.Factor, *core.Session, error) {
	if q.before != nil {
		q.before(p, r, s)
	}

	if q.deny {
		return nil, nil, core.ErrFactorChange
	}

	return q.Queries.CompleteFactorChange(ctx, p, r, s, policy, settings)
}

// Negative authority tests use real verification before the production commit,
// real locks and SQL fault injection, never caller-asserted proof fixtures.
func TestFactorAuthority(t *testing.T) {
	t.Run("owned selections", testFactorOwnership)
	t.Run("current state conflicts", testFactorConflicts)
	t.Run("mutation races", testFactorRaces)
	t.Run("transaction rollback", testFactorRollback)
	t.Run("post-lock age", testFactorLockAge)
	t.Run("oldest composite age", testFactorCompositeAge)
	t.Run("post-write age", testFactorWriteAge)
	t.Run("shared bounds", testFactorBounds)
	t.Run("usable factor profile", testFactorUsability)
}
func testFactorOwnership(t *testing.T) {
	f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
	svc := factorService(t, f, f.q)
	policy := factorPolicy()
	actor := finishAssertion(t, f, beginAssertion(t, f), 1)

	_, err := f.base.Signup(t.Context(), "foreign@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	begin, err := f.svc.BeginWebAuthnEnrollment(t.Context(), "foreign@example.com", "a distinct safe password", enrollmentRequirement())
	if err != nil {
		t.Fatal(err)
	}

	body, _, _ := enrollmentKeyResponseFlags(t, begin, 0x45)

	foreign, err := f.svc.FinishWebAuthnEnrollment(t.Context(), begin.Token, body, enrollmentRequirement())
	if err != nil {
		t.Fatal(err)
	}

	target := factorSelection(foreign)
	for _, selection := range []core.FactorSelection{target, {Kind: core.FactorWebAuthn, ID: f.factor.ID, Revision: 2}, {Kind: core.FactorTOTP, ID: f.factor.ID, Revision: 1}} {
		_, err = svc.BeginWebAuthnChange(t.Context(), actor.Token, selection, policy)
		if err == nil {
			t.Fatal("foreign/stale target admitted")
		}

		_, err = svc.Remove(t.Context(), actor.Token, selection, policy)
		if err == nil {
			t.Fatal("foreign/stale target removed")
		}
	}

	factors, err := svc.List(t.Context(), actor.Token, policy)
	if err != nil || len(factors) != 1 || factors[0].ID != f.factor.ID {
		t.Fatal("safe listing crossed owner")
	}

	password, err := f.base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}

	issued, ok := password.CompletedSession()
	if !ok {
		t.Fatal("missing actual password fixture")
	}

	lower := core.FactorPolicy{Management: fallbackRequirement(), Access: core.RequireMFA}

	_, err = svc.List(t.Context(), issued.Token, lower)
	if err == nil {
		t.Fatal("password listed established factors")
	}

	_, err = svc.BeginWebAuthnChange(t.Context(), issued.Token, core.FactorSelection{}, lower)
	if err == nil {
		t.Fatal("password authorized additional enrollment")
	}
}
func testFactorConflicts(t *testing.T) {
	tests := []struct {
		name   string
		change func(*testing.T, webAuthnFixture, core.FactorChangePending)
	}{
		{"actor generation", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "UPDATE sessions SET generation=generation+1 WHERE id=$1", p.Actor.ID)
		}},
		{"actor revoked", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "DELETE FROM sessions WHERE id=$1", p.Actor.ID)
		}},
		{"account version", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "UPDATE users SET auth_version=auth_version+1 WHERE id=$1", p.State.UserID)
		}},
		{"account inactive", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "UPDATE users SET active=false WHERE id=$1", p.State.UserID)
		}},
		{"actor factor revision", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "UPDATE authenticators SET revision=revision+1 WHERE id=$1", p.Actor.Proof.FactorID)
		}},
		{"target revision", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "UPDATE authenticators SET revision=revision+1 WHERE id=$1", p.Target.ID)
		}},
		{"target removed", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "DELETE FROM authenticators WHERE id=$1", p.Target.ID)
		}},
		{"lease revision", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "UPDATE auth_pending SET revision=revision+1 WHERE digest=$1", p.Digest[:])
		}},
		{"lease expired", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "UPDATE auth_pending SET lease_until=clock_timestamp() WHERE digest=$1", p.Digest[:])
		}},
		{"actor policy", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "UPDATE sessions SET policy_revision='changed-v2' WHERE id=$1", p.Actor.ID)
		}},
		{"actor proof", func(t *testing.T, f webAuthnFixture, p core.FactorChangePending) {
			sessionSQL(t, f.db, "UPDATE sessions SET proof_verified_at=proof_verified_at-interval '1 microsecond',created_at=created_at-interval '1 microsecond' WHERE id=$1", p.Actor.ID)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			plain := factorService(t, f, f.q)
			policy := factorPolicy()
			actor := finishAssertion(t, f, beginAssertion(t, f), 1)
			added, _ := addPasskey(t, f, plain, actor, core.FactorSelection{}, policy)
			actor = added.Issued
			svc := factorService(t, f, factorBoundary{Queries: f.q, before: func(p core.FactorChangePending, _ core.FactorChangeRecord, _ core.SessionRecord) {
				test.change(t, f, p)
			}})

			begin, err := svc.BeginWebAuthnChange(t.Context(), actor.Token, added.Factor.FactorSelection, policy)
			if err != nil {
				t.Fatal(err)
			}

			body, _, _ := enrollmentKeyResponseFlags(t, begin, 0x45)

			result, err := svc.FinishWebAuthnChange(t.Context(), actor.Token, begin.Token, body, policy)
			if err == nil || result != nil {
				t.Fatal("stale verified operation committed")
			}

			var attempts int

			err = f.db.QueryRowContext(t.Context(), "SELECT attempts FROM auth_pending WHERE purpose=7").Scan(&attempts)
			if err != nil || attempts != 1 {
				t.Fatalf("reservation was refunded: %d %v", attempts, err)
			}
		})
	}

	t.Run("captured policy and purpose", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		svc := factorService(t, f, f.q)
		policy := factorPolicy()
		actor := finishAssertion(t, f, beginAssertion(t, f), 1)

		begin, err := svc.BeginWebAuthnChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
		if err != nil {
			t.Fatal(err)
		}

		body, _, _ := enrollmentKeyResponseFlags(t, begin, 0x45)
		changed := policy
		changed.Access = core.RequireMFA

		result, err := svc.FinishWebAuthnChange(t.Context(), actor.Token, begin.Token, body, changed)
		if err == nil || result != nil {
			t.Fatal("captured policy weakened")
		}

		_, err = f.svc.FinishWebAuthnEnrollment(t.Context(), begin.Token, body, enrollmentRequirement())
		if err == nil {
			t.Fatal("change continuation completed initial enrollment")
		}

		_, err = svc.FinishTOTPChange(t.Context(), actor.Token, begin.Token, "123456", policy)
		if err == nil {
			t.Fatal("cross-kind change completed")
		}

		second := finishAssertion(t, f, beginAssertion(t, f), 2)

		_, err = svc.FinishWebAuthnChange(t.Context(), second.Token, begin.Token, body, policy)
		if err == nil {
			t.Fatal("different actor bearer completed change")
		}
	})
}
func testFactorRaces(t *testing.T) {
	tests := []struct {
		name string
		same bool
	}{{"same pending", true}, {"competing enrollment", false}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			svc := factorService(t, f, f.q)
			policy := factorPolicy()
			actor := finishAssertion(t, f, beginAssertion(t, f), 1)

			first, err := svc.BeginWebAuthnChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
			if err != nil {
				t.Fatal(err)
			}

			second := first
			if !test.same {
				second, err = svc.BeginWebAuthnChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
				if err != nil {
					t.Fatal(err)
				}
			}

			firstBody, _, _ := enrollmentKeyResponseFlags(t, first, 0x45)

			secondBody := firstBody
			if !test.same {
				secondBody, _, _ = enrollmentKeyResponseFlags(t, second, 0x45)
			}

			start := make(chan struct{})
			success := make(chan bool, 2)

			var workers sync.WaitGroup
			for _, candidate := range []struct {
				token string
				body  []byte
			}{{first.Token, firstBody}, {second.Token, secondBody}} {
				workers.Add(1)
				go func() {
					defer workers.Done()

					<-start

					result, err := svc.FinishWebAuthnChange(t.Context(), actor.Token, candidate.token, candidate.body, policy)
					success <- err == nil && result != nil && result.Issued != nil
				}()
			}

			close(start)
			workers.Wait()
			close(success)

			wins := 0

			for ok := range success {
				if ok {
					wins++
				}
			}

			if wins != 1 {
				t.Fatalf("completion winners: %d", wins)
			}

			var factors, sessions, pending int

			err = f.db.QueryRowContext(t.Context(), "SELECT (SELECT count(*) FROM authenticators),(SELECT count(*) FROM sessions),(SELECT count(*) FROM auth_pending)").Scan(&factors, &sessions, &pending)
			if err != nil || factors != 2 || sessions != 1 || pending != 0 {
				t.Fatalf("race partial state: %d/%d/%d %v", factors, sessions, pending, err)
			}
		})
	}

	t.Run("last factor under removal race", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		svc := factorService(t, f, f.q)
		policy := factorPolicy()
		actor := finishAssertion(t, f, beginAssertion(t, f), 1)
		added, next := addPasskey(t, f, svc, actor, core.FactorSelection{}, policy)
		other := finishAssertion(t, next, beginAssertion(t, next), 1)
		start := make(chan struct{})
		done := make(chan bool, 2)

		for _, candidate := range []struct {
			token  string
			target core.FactorSelection
		}{{added.Issued.Token, added.Factor.FactorSelection}, {other.Token, factorSelection(f.factor)}} {
			go func() {
				<-start

				result, err := svc.Remove(t.Context(), candidate.token, candidate.target, policy)
				done <- err == nil && result != nil
			}()
		}

		close(start)

		wins := 0

		for range 2 {
			if <-done {
				wins++
			}
		}

		var count int

		err := f.db.QueryRowContext(t.Context(), "SELECT count(*) FROM authenticators").Scan(&count)
		if err != nil || wins != 1 || count != 1 {
			t.Fatalf("last factor race: wins=%d count=%d %v", wins, count, err)
		}
	})
}
func testFactorRollback(t *testing.T) {
	tests := []struct {
		name    string
		trigger string
		remove  bool
	}{
		{"activation", `CREATE TRIGGER reject_change BEFORE INSERT ON authenticators FOR EACH ROW EXECUTE FUNCTION reject_change()`, false},
		{"version", `CREATE TRIGGER reject_change BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION reject_change()`, false},
		{"rotation", `CREATE TRIGGER reject_change BEFORE INSERT ON sessions FOR EACH ROW EXECUTE FUNCTION reject_change()`, false},
		{"removal rotation", `CREATE TRIGGER reject_change BEFORE INSERT ON sessions FOR EACH ROW EXECUTE FUNCTION reject_change()`, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			svc := factorService(t, f, f.q)
			policy := factorPolicy()
			actor := finishAssertion(t, f, beginAssertion(t, f), 1)
			added, _ := addPasskey(t, f, svc, actor, core.FactorSelection{}, policy)
			actor = added.Issued

			begin, err := svc.BeginWebAuthnChange(t.Context(), actor.Token, added.Factor.FactorSelection, policy)
			if err != nil {
				t.Fatal(err)
			}

			body, _, _ := enrollmentKeyResponseFlags(t, begin, 0x45)
			other := finishAssertion(t, f, beginAssertion(t, f), 2)
			sessionSQL(t, f.db, `CREATE FUNCTION reject_change() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced factor change failure'; END $$`)
			sessionSQL(t, f.db, test.trigger)

			var result *core.FactorChangeResult
			if test.remove {
				result, err = svc.Remove(t.Context(), actor.Token, added.Factor.FactorSelection, policy)
			} else {
				result, err = svc.FinishWebAuthnChange(t.Context(), actor.Token, begin.Token, body, policy)
			}

			if err == nil || result != nil {
				t.Fatal("failed transaction returned success")
			}

			var (
				version                              int64
				factors, sessions, pending, attempts int
			)

			err = f.db.QueryRowContext(t.Context(), "SELECT auth_version,(SELECT count(*) FROM authenticators),(SELECT count(*) FROM sessions),(SELECT count(*) FROM auth_pending),(SELECT attempts FROM auth_pending WHERE purpose=7) FROM users").Scan(&version, &factors, &sessions, &pending, &attempts)

			expectedAttempts := 1
			if test.remove {
				expectedAttempts = 0
			}

			if err != nil || version != actor.AuthVersion || factors != 2 || sessions != 2 || pending != 1 || attempts != expectedAttempts {
				t.Fatalf("partial rollback: %d %d/%d/%d/%d %v", version, factors, sessions, pending, attempts, err)
			}

			for _, token := range []string{actor.Token, other.Token} {
				_, err = f.base.ValidateSession(t.Context(), token, webAuthnRequirement(), core.NoActivity)
				if err != nil {
					t.Fatal("rollback revoked original actor")
				}
			}
		})
	}
}
func testFactorLockAge(t *testing.T) {
	f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
	plain := factorService(t, f, f.q)
	policy := factorPolicy()
	policy.Management.MaxAge = time.Second
	actor := finishAssertion(t, f, beginAssertion(t, f), 1)

	var (
		pending     core.FactorChangePending
		record      core.FactorChangeRecord
		replacement core.SessionRecord
	)

	capture := factorService(t, f, factorBoundary{Queries: f.q, deny: true, before: func(p core.FactorChangePending, r core.FactorChangeRecord, s core.SessionRecord) {
		pending = p
		record = r
		replacement = s
	}})

	begin, err := plain.BeginWebAuthnChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
	if err != nil {
		t.Fatal(err)
	}

	body, _, _ := enrollmentKeyResponseFlags(t, begin, 0x45)

	result, err := capture.FinishWebAuthnChange(t.Context(), actor.Token, begin.Token, body, policy)
	if err == nil || result != nil || pending.Revision < 1 {
		t.Fatal("actual verification capture failed")
	}

	sessionSQL(t, f.db, "UPDATE auth_pending SET lease_until=$2 WHERE digest=$1", pending.Digest[:], pending.LeaseUntil)

	policy = pending.Policy

	settings, err := factorConfig().EnrollmentSettings()
	if err != nil {
		t.Fatal(err)
	}

	holder, err := f.db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback()

	var id string

	err = holder.QueryRowContext(t.Context(), "SELECT id FROM authenticators FOR UPDATE").Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		factor, session, err := f.q.CompleteFactorChange(ctx, pending, record, replacement, policy, settings)
		if factor != nil || session != nil {
			done <- errors.New("expired proof returned success")

			return
		}

		done <- err
	}()

	waitSessionLock(t, ctx, f.db, holder)
	waitProofExpiry(t, ctx, f.db, actor.Proof.VerifiedAt.Add(time.Second))

	err = holder.Rollback()
	if err != nil {
		t.Fatal(err)
	}

	err = <-done
	if !errors.Is(err, core.ErrSessionProofExpired) {
		t.Fatalf("post-lock age: %v", err)
	}

	var count int

	err = f.db.QueryRowContext(t.Context(), "SELECT count(*) FROM authenticators").Scan(&count)
	if err != nil || count != 1 {
		t.Fatal("expired proof activated factor")
	}
}
func testFactorWriteAge(t *testing.T) {
	for _, remove := range []bool{false, true} {
		name := "activation"
		if remove {
			name = "removal"
		}

		t.Run(name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			svc := factorService(t, f, f.q)
			policy := factorPolicy()
			actor := finishAssertion(t, f, beginAssertion(t, f), 1)
			added, _ := addPasskey(t, f, svc, actor, core.FactorSelection{}, policy)
			actor = added.Issued
			policy.Management.MaxAge = time.Second

			begin, err := svc.BeginWebAuthnChange(t.Context(), actor.Token, added.Factor.FactorSelection, policy)
			if err != nil {
				t.Fatal(err)
			}

			body, _, _ := enrollmentKeyResponseFlags(t, begin, 0x45)
			sessionSQL(t, f.db, `CREATE FUNCTION delay_change() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(1.1); RETURN NEW; END $$`)
			sessionSQL(t, f.db, `CREATE TRIGGER delay_change AFTER INSERT ON sessions FOR EACH ROW EXECUTE FUNCTION delay_change()`)

			var result *core.FactorChangeResult
			if remove {
				result, err = svc.Remove(t.Context(), actor.Token, added.Factor.FactorSelection, policy)
			} else {
				result, err = svc.FinishWebAuthnChange(t.Context(), actor.Token, begin.Token, body, policy)
			}

			if err == nil || result != nil {
				t.Fatal("expired authority survived writes")
			}

			var (
				version int64
				count   int
			)

			err = f.db.QueryRowContext(t.Context(), "SELECT auth_version,(SELECT count(*) FROM authenticators) FROM users").Scan(&version, &count)
			if err != nil || version != actor.AuthVersion || count != 2 {
				t.Fatal("expiry did not roll back mutation")
			}
		})
	}
}
func testFactorBounds(t *testing.T) {
	t.Run("shared retained setups", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{MaxAuthenticators: 2})
		svc := factorService(t, f, f.q)
		policy := factorPolicy()
		actor := finishAssertion(t, f, beginAssertion(t, f), 1)

		begin, err := svc.BeginWebAuthnChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
		if err != nil {
			t.Fatal(err)
		}

		_, err = svc.BeginTOTPChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
		if !errors.Is(err, core.ErrEnrollmentCapacity) {
			t.Fatalf("setup bound not shared: %v", err)
		}

		sessionSQL(t, f.db, "UPDATE auth_pending SET created_at=created_at-interval '6 minutes',expires_at=expires_at-interval '6 minutes'")

		_, err = svc.BeginTOTPChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
		if err != nil {
			t.Fatalf("bounded expired reclamation failed: %v", err)
		}

		_, err = f.svc.FinishWebAuthnEnrollment(t.Context(), begin.Token, []byte(`{}`), enrollmentRequirement())
		if err == nil {
			t.Fatal("restricted token conferred setup authority")
		}
	})
	t.Run("durable subject budget", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{SubjectAttempts: 2})
		svc := factorService(t, f, f.q)
		policy := factorPolicy()
		actor := finishAssertion(t, f, beginAssertion(t, f), 1)
		// Actual enrollment/assertion already charged the same subject budget. A new
		// management continuation cannot reset exhausted admission.
		begin, err := svc.BeginWebAuthnChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
		if err != nil {
			t.Fatal(err)
		}

		body, _, _ := enrollmentKeyResponseFlags(t, begin, 0x45)

		result, err := svc.FinishWebAuthnChange(t.Context(), actor.Token, begin.Token, body, policy)
		if !errors.Is(err, core.ErrEnrollmentAttempts) || result != nil {
			t.Fatalf("management reset budget: %v", err)
		}
	})
	t.Run("key unavailable", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		svc := factorService(t, f, f.q)
		policy := factorPolicy()
		actor := finishAssertion(t, f, beginAssertion(t, f), 1)

		begin, err := svc.BeginTOTPChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
		if err != nil {
			t.Fatal(err)
		}

		parsed, err := url.Parse(begin.URL)
		if err != nil {
			t.Fatal(err)
		}

		code, err := previousTOTPCode(t, parsed.Query().Get("secret"))
		if err != nil {
			t.Fatal(err)
		}

		fallback, err := core.NewFallbackService(f.base, f.q, config.FallbackConfig{Issuer: "Ticked test"}, core.SeedKeys{Active: "other", Keys: map[string][]byte{"other": make([]byte, 32)}})
		if err != nil {
			t.Fatal(err)
		}

		changed, err := core.NewFactorService(f.base, f.q, f.svc.AuthenticatorService, fallback)
		if err != nil {
			t.Fatal(err)
		}

		result, err := changed.FinishTOTPChange(t.Context(), actor.Token, begin.Token, code, policy)
		if err == nil || result != nil {
			t.Fatal("unavailable key confirmed setup")
		}
	})
}

// Current TOTP verification cannot renew the password constituent that actually
// began this MFA proof. The locked management transaction uses the oldest time.
func testFactorCompositeAge(t *testing.T) {
	f := newFallbackFixture(t, config.FallbackConfig{})

	enrollment, err := core.NewAuthenticatorService(f.base, f.q, factorConfig())
	if err != nil {
		t.Fatal(err)
	}

	policy := core.FactorPolicy{Management: fallbackRequirement(), Access: core.RequireMFA}
	policy.Management.MaxAge = time.Second
	authentication := beginFallback(t, f, core.FallbackTOTP)

	time.Sleep(100 * time.Millisecond)

	actor, err := f.svc.FinishFallback(t.Context(), authentication.Token, fallbackCode(t, f), fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	var (
		pending     core.FactorChangePending
		record      core.FactorChangeRecord
		replacement core.SessionRecord
	)

	boundary := factorBoundary{Queries: f.q, deny: true, before: func(p core.FactorChangePending, r core.FactorChangeRecord, s core.SessionRecord) {
		pending = p
		record = r
		replacement = s
	}}

	svc, err := core.NewFactorService(f.base, boundary, enrollment, f.svc)
	if err != nil {
		t.Fatal(err)
	}

	begin, err := svc.BeginWebAuthnChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
	if err != nil {
		t.Fatal(err)
	}

	body, _, _ := enrollmentKeyResponseFlags(t, begin, 0x45)

	result, err := svc.FinishWebAuthnChange(t.Context(), actor.Token, begin.Token, body, policy)
	if err == nil || result != nil || pending.Revision < 1 {
		t.Fatal("actual composite capture failed")
	}

	sessionSQL(t, f.db, "UPDATE auth_pending SET lease_until=$2 WHERE digest=$1", pending.Digest[:], pending.LeaseUntil)

	policy = pending.Policy

	settings, err := factorConfig().EnrollmentSettings()
	if err != nil {
		t.Fatal(err)
	}

	holder, err := f.db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback()

	var id string

	err = holder.QueryRowContext(t.Context(), "SELECT id FROM totp_authenticators FOR UPDATE").Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		factor, session, err := f.q.CompleteFactorChange(ctx, pending, record, replacement, policy, settings)
		if factor != nil || session != nil {
			done <- errors.New("expired composite returned success")

			return
		}

		done <- err
	}()

	waitSessionLock(t, ctx, f.db, holder)
	waitProofExpiry(t, ctx, f.db, actor.Proof.VerifiedAt.Add(time.Second))

	var now time.Time

	err = f.db.QueryRowContext(ctx, "SELECT clock_timestamp()").Scan(&now)
	if err != nil || !now.Before(actor.Proof.FactorAt.Add(time.Second)) {
		t.Fatal("fixture lost its newer factor freshness")
	}

	err = holder.Rollback()
	if err != nil {
		t.Fatal(err)
	}

	err = <-done
	if !errors.Is(err, core.ErrSessionProofExpired) {
		t.Fatalf("oldest constituent renewed: %v", err)
	}
}

// Usability includes the configured RP and actual available TOTP key identities,
// not just an enrolled row of a generally sufficient authenticator class.
func testFactorUsability(t *testing.T) {
	t.Run("other RP", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		svc := factorService(t, f, f.q)
		policy := factorPolicy()
		actor := finishAssertion(t, f, beginAssertion(t, f), 1)
		added, _ := addPasskey(t, f, svc, actor, core.FactorSelection{}, policy)
		actor = added.Issued

		sessionSQL(t, f.db, "INSERT INTO webauthn_subjects(user_id,rp_id,handle) SELECT user_id,'other.example.com',handle FROM webauthn_subjects")
		sessionSQL(t, f.db, "UPDATE authenticators SET rp_id='other.example.com' WHERE id=$1", added.Factor.ID)

		factors, err := svc.List(t.Context(), actor.Token, policy)
		if err != nil || len(factors) != 1 || factors[0].ID != f.factor.ID {
			t.Fatal("listing mixed RP scopes")
		}

		result, err := svc.Remove(t.Context(), actor.Token, factorSelection(f.factor), policy)
		if err == nil || result != nil {
			t.Fatal("other RP counted as usable primary")
		}
	})
	t.Run("available TOTP keys", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		svc := factorService(t, f, f.q)
		policy := factorPolicy()
		policy.Access = core.RequireMFA
		actor := finishAssertion(t, f, beginAssertion(t, f), 1)

		setup, err := svc.BeginTOTPChange(t.Context(), actor.Token, core.FactorSelection{}, policy)
		if err != nil {
			t.Fatal(err)
		}

		parsed, err := url.Parse(setup.URL)
		if err != nil {
			t.Fatal(err)
		}

		code, err := previousTOTPCode(t, parsed.Query().Get("secret"))
		if err != nil {
			t.Fatal(err)
		}

		added, err := svc.FinishTOTPChange(t.Context(), actor.Token, setup.Token, code, policy)
		if err != nil {
			t.Fatal(err)
		}

		actor = added.Issued

		for _, name := range []string{"disabled", "retired key"} {
			t.Run(name, func(t *testing.T) {
				var fallback *core.FallbackService
				if name == "retired key" {
					fallback, err = core.NewFallbackService(f.base, f.q, config.FallbackConfig{Issuer: "Ticked test"}, core.SeedKeys{Active: "other", Keys: map[string][]byte{"other": make([]byte, 32)}})
					if err != nil {
						t.Fatal(err)
					}
				}

				unsupported, err := core.NewFactorService(f.base, f.q, f.svc.AuthenticatorService, fallback)
				if err != nil {
					t.Fatal(err)
				}

				result, err := unsupported.Remove(t.Context(), actor.Token, factorSelection(f.factor), policy)
				if err == nil || result != nil {
					t.Fatal("unavailable TOTP counted as usable primary")
				}
			})
		}
		// The real configured fallback is a usable remaining MFA factor. Removing the
		// WebAuthn constituent still revokes the actor instead of fabricating TOTP proof.
		removed, err := svc.Remove(t.Context(), actor.Token, factorSelection(f.factor), policy)
		if err != nil || removed.Issued != nil {
			t.Fatalf("usable MFA profile: %v", err)
		}
	})
}
