//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
)

func webAuthnRequirement() core.AccessRequirement {
	r := PasswordRequirement()
	r.Proof = core.RequirePhishingResistantMFA
	r.MaxAge = 5 * time.Minute

	return r
}

type webAuthnFixture struct {
	db     *sql.DB
	q      *Queries
	base   *core.Service
	svc    *core.WebAuthnService
	key    *ecdsa.PrivateKey
	id     []byte
	factor *core.Authenticator
}

func newWebAuthnFixture(t *testing.T, options config.AuthenticatorConfig) webAuthnFixture {
	t.Helper()

	return newWebAuthnFixtureFlags(t, options, 0x45)
}

func newWebAuthnFixtureFlags(t *testing.T, options config.AuthenticatorConfig, flags byte) webAuthnFixture {
	t.Helper()
	db, q, base, enrollment, challenge := enrollmentFixture(t, options)
	body, key, id := enrollmentKeyResponseFlags(t, challenge, flags)

	factor, err := enrollment.FinishWebAuthnEnrollment(t.Context(), challenge.Token, body, enrollmentRequirement())
	if err != nil {
		t.Fatal(err)
	}

	options.RPID = "example.com"
	options.RPName = "Example"
	options.Origins = []string{"https://example.com"}

	svc, err := core.NewWebAuthnService(base, q, options)
	if err != nil {
		t.Fatal(err)
	}

	return webAuthnFixture{db: db, q: q, base: base, svc: svc, key: key, id: id, factor: factor}
}

// signedAssertion uses a real ES256 key enrolled by the production verifier.
// Mutations are made before signing to test semantic checks independently.
func signedAssertion(t *testing.T, f webAuthnFixture, challenge *core.AssertionChallenge, counter uint32, flags byte, change func(map[string]any, []byte)) []byte {
	t.Helper()

	rp := sha256.Sum256([]byte("example.com"))
	data := append(rp[:], flags, 0, 0, 0, 0)
	binary.BigEndian.PutUint32(data[33:], counter)

	client := map[string]any{"type": "webauthn.get", "challenge": base64.RawURLEncoding.EncodeToString(challenge.Options.Response.Challenge), "origin": "https://example.com", "crossOrigin": false}
	if change != nil {
		change(client, data)
	}

	encoded, err := json.Marshal(client)
	if err != nil {
		t.Fatal(err)
	}

	clientHash := sha256.Sum256(encoded)
	signed := sha256.Sum256(append(append([]byte(nil), data...), clientHash[:]...))

	signature, err := ecdsa.SignASN1(rand.Reader, f.key, signed[:])
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]any{"id": base64.RawURLEncoding.EncodeToString(f.id), "rawId": base64.RawURLEncoding.EncodeToString(f.id), "type": "public-key", "response": map[string]any{"clientDataJSON": base64.RawURLEncoding.EncodeToString(encoded), "authenticatorData": base64.RawURLEncoding.EncodeToString(data), "signature": base64.RawURLEncoding.EncodeToString(signature)}, "clientExtensionResults": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}

	return body
}
func beginAssertion(t *testing.T, f webAuthnFixture) *core.AssertionChallenge {
	t.Helper()

	challenge, err := f.svc.BeginWebAuthnAuthentication(t.Context(), "session@example.com", webAuthnRequirement())
	if err != nil {
		t.Fatal(err)
	}

	return challenge
}
func finishAssertion(t *testing.T, f webAuthnFixture, challenge *core.AssertionChallenge, counter uint32) *core.IssuedSession {
	t.Helper()

	issued, err := f.svc.FinishWebAuthn(t.Context(), challenge.Token, signedAssertion(t, f, challenge, counter, 5, nil), webAuthnRequirement())
	if err != nil || issued == nil {
		t.Fatalf("assertion completion: %v", err)
	}

	return issued
}

// Real signatures and PostgreSQL establish completed proof and bound rotation.
func TestWebAuthnTransactions(t *testing.T) {
	t.Run("protocol and replay", testWebAuthnProtocol)
	t.Run("transaction conflicts", testWebAuthnConflicts)
	t.Run("factor ownership", testWebAuthnOwnership)
	t.Run("current backup state", testWebAuthnBackupConflict)
	t.Run("completion races", testWebAuthnRaces)
	t.Run("rollback", testWebAuthnRollback)
	t.Run("lock-time bounds", testWebAuthnLockTime)
	t.Run("post-write expiry", testWebAuthnWriteWait)
	t.Run("admission budgets", testWebAuthnBudgets)
	t.Run("session admission", testWebAuthnCapacity)
	t.Run("signed authentication", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		challenge := beginAssertion(t, f)

		issued := finishAssertion(t, f, challenge, 1)
		if issued.Proof.Method != core.WebAuthnProof || issued.Proof.FactorID != f.factor.ID || issued.Proof.FactorRevision != 1 {
			t.Fatal("missing actual constituent")
		}

		for _, method := range []core.RequiredProof{core.RequirePassword, core.RequireMFA, core.RequirePhishingResistantMFA} {
			requirement := webAuthnRequirement()
			requirement.Proof = method

			current, err := f.base.ValidateSession(t.Context(), issued.Token, requirement, core.NoActivity)
			if err != nil || current == nil {
				t.Fatalf("proof %d: %v", method, err)
			}
		}

		replay, err := f.svc.FinishWebAuthn(t.Context(), challenge.Token, signedAssertion(t, f, challenge, 2, 5, nil), webAuthnRequirement())
		if err == nil || replay != nil {
			t.Fatal("pending replay issued access")
		}

		finishAssertion(t, f, beginAssertion(t, f), 2)

		_, err = f.base.ValidateSession(t.Context(), issued.Token, webAuthnRequirement(), core.NoActivity)
		if err != nil {
			t.Fatalf("routine counter update invalidated prior proof: %v", err)
		}
	})
	t.Run("password actor step-up", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})

		result, err := f.base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
		if err != nil {
			t.Fatal(err)
		}

		old, _ := result.CompletedSession()

		_, err = f.base.ValidateSession(t.Context(), old.Token, webAuthnRequirement(), core.NoActivity)
		if err == nil {
			t.Fatal("password satisfies stronger route")
		}

		challenge, err := f.svc.BeginWebAuthnStepUp(t.Context(), old.Token, webAuthnRequirement())
		if err != nil {
			t.Fatal(err)
		}

		issued := finishAssertion(t, f, challenge, 1)
		if issued.ID != old.ID || issued.Generation != old.Generation+1 || issued.Token == old.Token || !issued.CreatedAt.Equal(old.CreatedAt) {
			t.Fatal("step-up did not rotate actor")
		}

		_, err = f.base.ValidateSession(t.Context(), old.Token, PasswordRequirement(), core.NoActivity)
		if err == nil {
			t.Fatal("old actor bearer survived")
		}

		_, err = f.base.ValidateSession(t.Context(), issued.Token, webAuthnRequirement(), core.NoActivity)
		if err != nil {
			t.Fatal(err)
		}

		weaker, err := f.base.Reauthenticate(t.Context(), issued.Token, "a distinct safe password", PasswordRequirement())
		if err != nil {
			t.Fatal(err)
		}

		pwd, _ := weaker.CompletedSession()
		if pwd.Proof.Method != core.PasswordProof || pwd.Proof.FactorID != "" {
			t.Fatal("password refreshed WebAuthn proof")
		}

		_, err = f.base.ValidateSession(t.Context(), pwd.Token, webAuthnRequirement(), core.NoActivity)
		if err == nil {
			t.Fatal("password reauth issued stronger access")
		}
	})
}

func testWebAuthnProtocol(t *testing.T) {
	tests := []struct {
		name   string
		flags  byte
		change func(map[string]any, []byte)
		wire   func(map[string]any)
	}{
		{name: "wrong challenge", flags: 5, change: func(c map[string]any, _ []byte) { c["challenge"] = "other" }},
		{name: "wrong RP", flags: 5, change: func(_ map[string]any, a []byte) { a[0] ^= 1 }},
		{name: "wrong origin", flags: 5, change: func(c map[string]any, _ []byte) { c["origin"] = "https://other.example.com" }},
		{name: "origin spelling", flags: 5, change: func(c map[string]any, _ []byte) { c["origin"] = "https://example.com/" }},
		{name: "cross origin", flags: 5, change: func(c map[string]any, _ []byte) { c["crossOrigin"] = true }},
		{name: "top origin", flags: 5, change: func(c map[string]any, _ []byte) { c["topOrigin"] = "https://example.com" }},
		{name: "wrong ceremony", flags: 5, change: func(c map[string]any, _ []byte) { c["type"] = "webauthn.create" }},
		{name: "missing presence", flags: 4},
		{name: "missing verification", flags: 1},
		{name: "eligibility changed", flags: 13},
		{name: "invalid backup state", flags: 21},
		{name: "attested data", flags: 69},
		{name: "extensions", flags: 133},
		{name: "wrong credential", flags: 5, wire: func(w map[string]any) { w["id"] = "b3RoZXI"; w["rawId"] = "b3RoZXI" }},
		{name: "wrong handle", flags: 5, wire: func(w map[string]any) {
			w["response"].(map[string]any)["userHandle"] = base64.RawURLEncoding.EncodeToString(make([]byte, 32))
		}},
		{name: "bad signature", flags: 5, wire: func(w map[string]any) { w["response"].(map[string]any)["signature"] = "AA" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			challenge := beginAssertion(t, f)

			body := signedAssertion(t, f, challenge, 1, test.flags, test.change)
			if test.wire != nil {
				var wire map[string]any

				err := json.Unmarshal(body, &wire)
				if err != nil {
					t.Fatal(err)
				}

				test.wire(wire)

				body, err = json.Marshal(wire)
				if err != nil {
					t.Fatal(err)
				}
			}

			issued, err := f.svc.FinishWebAuthn(t.Context(), challenge.Token, body, webAuthnRequirement())
			if err == nil || issued != nil {
				t.Fatal("invalid assertion issued access")
			}

			var attempts, pending, sessions, counter int

			err = f.db.QueryRowContext(t.Context(), "SELECT (SELECT attempts FROM auth_pending),(SELECT count(*) FROM auth_pending),(SELECT count(*) FROM sessions),sign_count FROM authenticators").Scan(&attempts, &pending, &sessions, &counter)
			if err != nil || attempts != 1 || pending != 1 || sessions != 0 || counter != 0 {
				t.Fatalf("failed proof changed completion state: %d %d %d %d %v", attempts, pending, sessions, counter, err)
			}
		})
	}

	t.Run("strict counters", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		finishAssertion(t, f, beginAssertion(t, f), 0)
		finishAssertion(t, f, beginAssertion(t, f), 0)
		finishAssertion(t, f, beginAssertion(t, f), 3)

		for _, counter := range []uint32{3, 2, 0} {
			ch := beginAssertion(t, f)

			issued, err := f.svc.FinishWebAuthn(t.Context(), ch.Token, signedAssertion(t, f, ch, counter, 5, nil), webAuthnRequirement())
			if err == nil || issued != nil {
				t.Fatalf("counter %d admitted", counter)
			}
		}

		finishAssertion(t, f, beginAssertion(t, f), 4)
	})
	t.Run("backup state updates", func(t *testing.T) {
		f := newWebAuthnFixtureFlags(t, config.AuthenticatorConfig{}, 0x4d)
		for i, flags := range []byte{29, 13} {
			ch := beginAssertion(t, f)

			issued, err := f.svc.FinishWebAuthn(t.Context(), ch.Token, signedAssertion(t, f, ch, uint32(i+1), flags, nil), webAuthnRequirement())
			if err != nil || issued == nil {
				t.Fatalf("backup flags: %v", err)
			}

			var be, bs bool

			err = f.db.QueryRowContext(t.Context(), "SELECT backup_eligible,backup_state FROM authenticators").Scan(&be, &bs)
			if err != nil || !be || bs != (flags&16 != 0) {
				t.Fatal("backup flags not updated atomically")
			}
		}
	})
}

type assertionBoundary struct {
	*Queries
	before func(core.AssertionReservation, core.AssertionCompletion)
	deny   bool
}

func (q assertionBoundary) CompleteAssertion(ctx context.Context, reserved core.AssertionReservation, c core.AssertionCompletion, r core.AccessRequirement, s config.EnrollmentSettings, limit int) (*core.Session, error) {
	if q.before != nil {
		q.before(reserved, c)
	}

	if q.deny {
		return nil, core.ErrWebAuthn
	}

	return q.Queries.CompleteAssertion(ctx, reserved, c, r, s, limit)
}
func assertionService(t *testing.T, f webAuthnFixture, queries core.WebAuthnQueries) *core.WebAuthnService {
	t.Helper()

	svc, err := core.NewWebAuthnService(f.base, queries, config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}})
	if err != nil {
		t.Fatal(err)
	}

	return svc
}

// Mutations happen after actual signature verification, before final SQL checks.
func testWebAuthnConflicts(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"factor removed", "DELETE FROM authenticators"},
		{"factor revision", "UPDATE authenticators SET revision=revision+1"},
		{"replay revision", "UPDATE authenticators SET replay_revision=replay_revision+1"},
		{"counter changed", "UPDATE authenticators SET sign_count=sign_count+1"},
		{"eligibility changed", "UPDATE authenticators SET backup_eligible=true"},
		{"account version", "UPDATE users SET auth_version=auth_version+1"},
		{"account inactive", "UPDATE users SET active=false"},
		{"current policy", "UPDATE auth_pending SET policy_revision='changed-v2'"},
		{"lease revision", "UPDATE auth_pending SET revision=revision+1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			ch := beginAssertion(t, f)
			svc := assertionService(t, f, assertionBoundary{Queries: f.q, before: func(_ core.AssertionReservation, _ core.AssertionCompletion) { sessionSQL(t, f.db, test.query) }})

			issued, err := svc.FinishWebAuthn(t.Context(), ch.Token, signedAssertion(t, f, ch, 1, 5, nil), webAuthnRequirement())
			if err == nil || issued != nil {
				t.Fatal("stale verified assertion issued access")
			}

			var count, attempts int

			err = f.db.QueryRowContext(t.Context(), "SELECT (SELECT count(*) FROM sessions),attempts FROM auth_pending").Scan(&count, &attempts)
			if err != nil || count != 0 || attempts != 1 {
				t.Fatalf("conflict completed or refunded: %v", err)
			}
		})
	}

	t.Run("actor generation changed", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})

		pwd, err := f.base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
		if err != nil {
			t.Fatal(err)
		}

		old, _ := pwd.CompletedSession()

		ch, err := f.svc.BeginWebAuthnStepUp(t.Context(), old.Token, webAuthnRequirement())
		if err != nil {
			t.Fatal(err)
		}

		svc := assertionService(t, f, assertionBoundary{Queries: f.q, before: func(_ core.AssertionReservation, _ core.AssertionCompletion) {
			_, err := f.base.Reauthenticate(t.Context(), old.Token, "a distinct safe password", PasswordRequirement())
			if err != nil {
				t.Fatal(err)
			}
		}})

		issued, err := svc.FinishWebAuthn(t.Context(), ch.Token, signedAssertion(t, f, ch, 1, 5, nil), webAuthnRequirement())
		if err == nil || issued != nil {
			t.Fatal("stale actor generation rotated")
		}
	})
	t.Run("current session factor", func(t *testing.T) {
		for _, query := range []string{"UPDATE authenticators SET revision=revision+1", "DELETE FROM authenticators"} {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			issued := finishAssertion(t, f, beginAssertion(t, f), 1)
			sessionSQL(t, f.db, query)

			_, err := f.base.ValidateSession(t.Context(), issued.Token, webAuthnRequirement(), core.RelevantActivity)
			if !errors.Is(err, core.ErrSessionProof) {
				t.Fatalf("stale factor proof: %v", err)
			}

			_, err = f.base.ListSessions(t.Context(), issued.Token, webAuthnRequirement(), "")
			if !errors.Is(err, core.ErrSessionProof) {
				t.Fatalf("control retained factor proof: %v", err)
			}
		}
	})
}

func testWebAuthnRaces(t *testing.T) {
	t.Run("one pending winner", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		ch := beginAssertion(t, f)
		body := signedAssertion(t, f, ch, 1, 5, nil)
		start := make(chan struct{})
		results := make(chan bool, 2)

		var workers sync.WaitGroup
		for range 2 {
			workers.Go(func() {
				<-start

				issued, err := f.svc.FinishWebAuthn(t.Context(), ch.Token, body, webAuthnRequirement())
				results <- err == nil && issued != nil
			})
		}

		close(start)
		workers.Wait()
		close(results)

		successes := 0

		for success := range results {
			if success {
				successes++
			}
		}

		if successes != 1 {
			t.Fatalf("pending winners: %d", successes)
		}
	})

	for _, stepUp := range []bool{false, true} {
		name := "counterless snapshot winner"
		if stepUp {
			name = "one rotation winner"
		}

		t.Run(name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			challenges := []*core.AssertionChallenge{nil, nil}

			if stepUp {
				pwd, err := f.base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
				if err != nil {
					t.Fatal(err)
				}

				old, _ := pwd.CompletedSession()
				for i := range challenges {
					challenges[i], err = f.svc.BeginWebAuthnStepUp(t.Context(), old.Token, webAuthnRequirement())
					if err != nil {
						t.Fatal(err)
					}
				}
			} else {
				for i := range challenges {
					challenges[i] = beginAssertion(t, f)
				}
			}

			ready := make(chan struct{}, 2)
			release := make(chan struct{})
			svc := assertionService(t, f, assertionBoundary{Queries: f.q, before: func(_ core.AssertionReservation, _ core.AssertionCompletion) { ready <- struct{}{}; <-release }})
			results := make(chan bool, 2)

			bodies := [][]byte{signedAssertion(t, f, challenges[0], 0, 5, nil), signedAssertion(t, f, challenges[1], 0, 5, nil)}
			for i := range challenges {
				go func() {
					issued, err := svc.FinishWebAuthn(t.Context(), challenges[i].Token, bodies[i], webAuthnRequirement())
					results <- err == nil && issued != nil
				}()
			}

			for range 2 {
				select {
				case <-ready:
				case <-time.After(5 * time.Second):
					t.Fatal("verifier did not reach barrier")
				}
			}

			close(release)

			successes := 0

			for range 2 {
				if <-results {
					successes++
				}
			}

			if successes != 1 {
				t.Fatalf("completion winners: %d", successes)
			}

			var pending, sessions, replay int

			err := f.db.QueryRowContext(t.Context(), "SELECT (SELECT count(*) FROM auth_pending),(SELECT count(*) FROM sessions),replay_revision FROM authenticators").Scan(&pending, &sessions, &replay)
			if err != nil || pending != 1 || sessions != 1 || replay != 2 {
				t.Fatalf("inconsistent winner state: %d %d %d %v", pending, sessions, replay, err)
			}
		})
	}
}

func testWebAuthnRollback(t *testing.T) {
	for _, stepUp := range []bool{false, true} {
		name := "insert rollback"
		if stepUp {
			name = "rotation rollback"
		}

		t.Run(name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})

			var old *core.IssuedSession

			ch := beginAssertion(t, f)
			event := "INSERT"

			if stepUp {
				pwd, err := f.base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
				if err != nil {
					t.Fatal(err)
				}

				old, _ = pwd.CompletedSession()

				ch, err = f.svc.BeginWebAuthnStepUp(t.Context(), old.Token, webAuthnRequirement())
				if err != nil {
					t.Fatal(err)
				}

				event = "UPDATE"
			}

			sessionSQL(t, f.db, `CREATE FUNCTION reject_assertion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced completion failure'; END $$;
CREATE CONSTRAINT TRIGGER reject_assertion AFTER `+event+` ON sessions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_assertion();`)
			body := signedAssertion(t, f, ch, 1, 5, nil)

			issued, err := f.svc.FinishWebAuthn(t.Context(), ch.Token, body, webAuthnRequirement())
			if err == nil || issued != nil {
				t.Fatal("failed commit issued bearer")
			}

			var counter, replay, attempts int

			err = f.db.QueryRowContext(t.Context(), "SELECT sign_count,replay_revision,(SELECT attempts FROM auth_pending WHERE purpose=$1) FROM authenticators", map[bool]int{false: 2, true: 3}[stepUp]).Scan(&counter, &replay, &attempts)
			if err != nil || counter != 0 || replay != 1 || attempts != 1 {
				t.Fatalf("partial rollback: %d %d %d %v", counter, replay, attempts, err)
			}

			if old != nil {
				_, err = f.base.ValidateSession(t.Context(), old.Token, PasswordRequirement(), core.NoActivity)
				if err != nil {
					t.Fatalf("rollback lost actor: %v", err)
				}
			}

			sessionSQL(t, f.db, "DROP TRIGGER reject_assertion ON sessions")

			issued, err = f.svc.FinishWebAuthn(t.Context(), ch.Token, body, webAuthnRequirement())
			if err != nil || issued == nil {
				t.Fatalf("reverification after rollback: %v", err)
			}
		})
	}
}

// Capture a command only after the production signature verifier has succeeded.
// The denied service response issues nothing; storage tests retain that bounded
// verified command to exercise trusted time after deliberate PostgreSQL waits.
func captureAssertion(t *testing.T, f webAuthnFixture, stepUp bool) (core.AssertionReservation, core.AssertionCompletion) {
	t.Helper()

	var (
		reserved   core.AssertionReservation
		completion core.AssertionCompletion
	)

	svc := assertionService(t, f, assertionBoundary{Queries: f.q, deny: true, before: func(r core.AssertionReservation, c core.AssertionCompletion) { reserved = r; completion = c }})

	ch := beginAssertion(t, f)
	if stepUp {
		result, err := f.base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
		if err != nil {
			t.Fatal(err)
		}

		old, _ := result.CompletedSession()

		ch, err = f.svc.BeginWebAuthnStepUp(t.Context(), old.Token, webAuthnRequirement())
		if err != nil {
			t.Fatal(err)
		}
	}

	issued, err := svc.FinishWebAuthn(t.Context(), ch.Token, signedAssertion(t, f, ch, 1, 5, nil), webAuthnRequirement())
	if err == nil || issued != nil || reserved.Pending.Revision != 1 {
		t.Fatalf("signature command capture: %v", err)
	}
	// Service error released its lease. Restore only this owned fixture lease;
	// no attempt is refunded and no new verifier result is fabricated.
	sessionSQL(t, f.db, "UPDATE auth_pending SET lease_until=$2 WHERE digest=$1", reserved.Pending.Digest[:], reserved.Pending.LeaseUntil)

	return reserved, completion
}

func testWebAuthnLockTime(t *testing.T) {
	for _, name := range []string{"pending expiry", "lease expiry", "proof expiry", "actor expiry", "canceled wait"} {
		t.Run(name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			reserved, c := captureAssertion(t, f, name == "actor expiry")

			options := config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}}
			if name == "proof expiry" {
				options.RecentProofAge = "1s"
			}

			settings, err := options.EnrollmentSettings()
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
			defer cancel()

			now := time.Now().UTC().Truncate(time.Microsecond)
			deadline := now.Add(350 * time.Millisecond)

			switch name {
			case "pending expiry":
				reserved.Pending.CreatedAt = deadline.Add(-settings.PendingTTL)
				reserved.Pending.ExpiresAt = deadline

				var ceremony map[string]any

				err = json.Unmarshal(reserved.Pending.Ceremony, &ceremony)
				if err != nil {
					t.Fatal(err)
				}

				ceremony["expires"] = deadline

				reserved.Pending.Ceremony, err = json.Marshal(ceremony)
				if err != nil {
					t.Fatal(err)
				}

				sessionSQL(t, f.db, "UPDATE auth_pending SET created_at=$2,expires_at=$3,ceremony_data=$4 WHERE digest=$1", reserved.Pending.Digest[:], reserved.Pending.CreatedAt, deadline, reserved.Pending.Ceremony)
			case "lease expiry":
				reserved.Pending.LeaseUntil = deadline
				sessionSQL(t, f.db, "UPDATE auth_pending SET lease_until=$2 WHERE digest=$1", reserved.Pending.Digest[:], deadline)
			case "proof expiry":
				deadline = c.Session.Proof.VerifiedAt.Add(time.Second)
			case "actor expiry":
				old := deadline.Add(-time.Minute)
				c.Session.CreatedAt = old
				sessionSQL(t, f.db, "UPDATE sessions SET created_at=$2,proof_verified_at=$2,authenticated_at=$2,last_activity_at=$2,expires_at=$3,inactivity_us=60000000 WHERE id=$1", reserved.Pending.ActorID, old, deadline)
			}

			actor := reserved.Actor
			if actor != nil {
				row, readErr := dal.New(f.db).GetSessionByDigest(t.Context(), reserved.Pending.ActorDigest[:])
				if readErr != nil {
					t.Fatal(readErr)
				}

				actor, readErr = toAuthSession(row)
				if readErr != nil {
					t.Fatal(readErr)
				}
			}

			err = c.Check(reserved.Pending, actor, time.Now().UTC(), webAuthnRequirement(), settings)
			if err != nil {
				t.Fatalf("fixture invalid before lock wait: %v", err)
			}

			holder, err := f.db.BeginTx(t.Context(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer holder.Rollback()

			var id string

			err = holder.QueryRowContext(ctx, "SELECT id FROM authenticators WHERE id=$1 FOR UPDATE", f.factor.ID).Scan(&id)
			if err != nil {
				t.Fatal(err)
			}

			type outcome struct {
				session *core.Session
				err     error
			}

			result := make(chan outcome, 1)

			go func() {
				session, err := f.q.CompleteAssertion(ctx, reserved, c, webAuthnRequirement(), settings, 10)
				result <- outcome{session: session, err: err}
			}()

			waitSessionLock(t, ctx, f.db, holder)

			if name == "canceled wait" {
				cancel()
			} else {
				waitProofExpiry(t, ctx, f.db, deadline)
			}

			err = holder.Rollback()
			if err != nil {
				t.Fatal(err)
			}

			completed := <-result
			if completed.err == nil || completed.session != nil {
				t.Fatal("post-lock bound did not reject")
			}

			var counter int

			err = f.db.QueryRowContext(t.Context(), "SELECT sign_count FROM authenticators").Scan(&counter)
			if err != nil || counter != 0 {
				t.Fatalf("rejection advanced replay: %v", err)
			}
		})
	}
}

func testWebAuthnBudgets(t *testing.T) {
	t.Run("shared subject budget", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{SubjectAttempts: 2})
		ch := beginAssertion(t, f)

		issued, err := f.svc.FinishWebAuthn(t.Context(), ch.Token, []byte("{"), webAuthnRequirement())
		if err == nil || issued != nil {
			t.Fatal("invalid body issued access")
		}

		other := beginAssertion(t, f)

		issued, err = f.svc.FinishWebAuthn(t.Context(), other.Token, signedAssertion(t, f, other, 1, 5, nil), webAuthnRequirement())
		if !errors.Is(err, core.ErrEnrollmentAttempts) || issued != nil {
			t.Fatalf("reissue bypassed shared enrollment/assertion budget: %v", err)
		}

		var attempts int

		err = f.db.QueryRowContext(t.Context(), "SELECT attempts FROM auth_factor_budgets").Scan(&attempts)
		if err != nil || attempts != 2 {
			t.Fatalf("budget: %d %v", attempts, err)
		}
	})
	t.Run("finite pending retention", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{MaxPending: 2})
		beginAssertion(t, f)
		beginAssertion(t, f)

		_, err := f.svc.BeginWebAuthnAuthentication(t.Context(), "session@example.com", webAuthnRequirement())
		if !errors.Is(err, core.ErrEnrollmentCapacity) {
			t.Fatalf("retention cap: %v", err)
		}
	})
	t.Run("pending attempt limit", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{PendingAttempts: 2})

		ch := beginAssertion(t, f)
		for range 2 {
			issued, err := f.svc.FinishWebAuthn(t.Context(), ch.Token, []byte("{"), webAuthnRequirement())
			if err == nil || issued != nil {
				t.Fatal("malformed assertion issued access")
			}
		}

		issued, err := f.svc.FinishWebAuthn(t.Context(), ch.Token, signedAssertion(t, f, ch, 1, 5, nil), webAuthnRequirement())
		if !errors.Is(err, core.ErrEnrollmentAttempts) || issued != nil {
			t.Fatalf("pending attempt cap: %v", err)
		}
	})
	t.Run("purpose isolation", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		ch := beginAssertion(t, f)

		_, err := f.base.ValidateSession(t.Context(), ch.Token, PasswordRequirement(), core.NoActivity)
		if !errors.Is(err, core.ErrSessionToken) {
			t.Fatalf("assertion entered session lookup: %v", err)
		}

		_, err = f.svc.FinishWebAuthnEnrollment(t.Context(), ch.Token, []byte("{}"), enrollmentRequirement())
		if !errors.Is(err, core.ErrEnrollment) {
			t.Fatalf("assertion entered enrollment: %v", err)
		}

		other := "stepup1." + ch.Token[8:]

		issued, err := f.svc.FinishWebAuthn(t.Context(), other, signedAssertion(t, f, ch, 1, 5, nil), webAuthnRequirement())
		if err == nil || issued != nil {
			t.Fatal("purpose substitution issued access")
		}
	})
	t.Run("bounded admission before protocol", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		ch := beginAssertion(t, f)
		ready := make(chan struct{}, 2)
		release := make(chan struct{})
		svc := assertionService(t, f, assertionBoundary{Queries: f.q, before: func(_ core.AssertionReservation, _ core.AssertionCompletion) { ready <- struct{}{}; <-release }})
		challenges := []*core.AssertionChallenge{ch, beginAssertion(t, f)}
		results := make(chan error, 2)

		for _, challenge := range challenges {
			body := signedAssertion(t, f, challenge, 0, 5, nil)
			go func() {
				_, err := svc.FinishWebAuthn(t.Context(), challenge.Token, body, webAuthnRequirement())
				results <- err
			}()
		}

		for range 2 {
			select {
			case <-ready:
			case <-time.After(5 * time.Second):
				t.Fatal("protocol slots did not fill")
			}
		}

		third := beginAssertion(t, f)

		issued, err := svc.FinishWebAuthn(t.Context(), third.Token, signedAssertion(t, f, third, 0, 5, nil), webAuthnRequirement())
		if !errors.Is(err, core.ErrEnrollmentBusy) || issued != nil {
			t.Fatalf("queue admitted: %v", err)
		}

		close(release)

		for range 2 {
			<-results
		}

		var attempts int

		err = f.db.QueryRowContext(t.Context(), "SELECT attempts FROM auth_pending WHERE digest=(SELECT digest FROM auth_pending ORDER BY created_at DESC LIMIT 1)").Scan(&attempts)
		if err != nil || attempts != 0 {
			t.Fatalf("busy request spent attempt: %v", err)
		}
	})
}

// The final clock check also guards waits inside session insertion, after replay
// and pending writes; expiration must roll all of those writes back together.
func testWebAuthnWriteWait(t *testing.T) {
	f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
	reserved, c := captureAssertion(t, f, false)
	options := config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}, RecentProofAge: "1s"}

	settings, err := options.EnrollmentSettings()
	if err != nil {
		t.Fatal(err)
	}

	sessionSQL(t, f.db, `CREATE TABLE assertion_guard(id integer PRIMARY KEY); INSERT INTO assertion_guard VALUES(1);
CREATE FUNCTION pause_assertion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM id FROM assertion_guard WHERE id=1 FOR UPDATE; RETURN NEW; END $$;
CREATE TRIGGER pause_assertion BEFORE INSERT ON sessions FOR EACH ROW EXECUTE FUNCTION pause_assertion();`)

	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()

	holder, err := f.db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback()

	_, err = holder.ExecContext(ctx, "SELECT id FROM assertion_guard WHERE id=1 FOR UPDATE")
	if err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		session *core.Session
		err     error
	}

	result := make(chan outcome, 1)

	go func() {
		session, err := f.q.CompleteAssertion(ctx, reserved, c, webAuthnRequirement(), settings, 10)
		result <- outcome{session: session, err: err}
	}()

	waitSessionLock(t, ctx, f.db, holder)
	waitProofExpiry(t, ctx, f.db, c.Session.Proof.VerifiedAt.Add(time.Second))

	err = holder.Rollback()
	if err != nil {
		t.Fatal(err)
	}

	completed := <-result
	if !errors.Is(completed.err, core.ErrSessionProofExpired) || completed.session != nil {
		t.Fatalf("post-write proof expiry: %v", completed.err)
	}

	var counter, replay, pending, sessions int

	err = f.db.QueryRowContext(t.Context(), "SELECT sign_count,replay_revision,(SELECT count(*) FROM auth_pending),(SELECT count(*) FROM sessions) FROM authenticators").Scan(&counter, &replay, &pending, &sessions)
	if err != nil || counter != 0 || replay != 1 || pending != 1 || sessions != 0 {
		t.Fatalf("expiry retained partial completion: %d %d %d %d %v", counter, replay, pending, sessions, err)
	}
}

func testWebAuthnCapacity(t *testing.T) {
	f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
	reserved, c := captureAssertion(t, f, false)

	_, err := f.base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}

	settings, err := (config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}}).EnrollmentSettings()
	if err != nil {
		t.Fatal(err)
	}

	issued, err := f.q.CompleteAssertion(t.Context(), reserved, c, webAuthnRequirement(), settings, 1)
	if !errors.Is(err, core.ErrSessionCapacity) || issued != nil {
		t.Fatalf("session admission: %v", err)
	}

	var counter, pending, attempts, sessions int

	err = f.db.QueryRowContext(t.Context(), "SELECT sign_count,(SELECT count(*) FROM auth_pending),(SELECT attempts FROM auth_pending),(SELECT count(*) FROM sessions) FROM authenticators").Scan(&counter, &pending, &attempts, &sessions)
	if err != nil || counter != 0 || pending != 1 || attempts != 1 || sessions != 1 {
		t.Fatalf("capacity consumed completion state: %v", err)
	}
}

func testWebAuthnOwnership(t *testing.T) {
	f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
	ch := beginAssertion(t, f)

	foreign, err := f.base.Signup(t.Context(), "foreign@example.com", "another distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	sessionSQL(t, f.db, "INSERT INTO webauthn_subjects(user_id,rp_id,handle) VALUES ($1,'example.com',$2)", foreign.ID, make([]byte, 32))
	svc := assertionService(t, f, assertionBoundary{Queries: f.q, before: func(_ core.AssertionReservation, _ core.AssertionCompletion) {
		sessionSQL(t, f.db, "UPDATE authenticators SET user_id=$1", foreign.ID)
	}})

	issued, err := svc.FinishWebAuthn(t.Context(), ch.Token, signedAssertion(t, f, ch, 1, 5, nil), webAuthnRequirement())
	if err == nil || issued != nil {
		t.Fatal("foreign factor satisfied completion")
	}
}

func testWebAuthnBackupConflict(t *testing.T) {
	f := newWebAuthnFixtureFlags(t, config.AuthenticatorConfig{}, 0x4d)
	ch := beginAssertion(t, f)
	svc := assertionService(t, f, assertionBoundary{Queries: f.q, before: func(_ core.AssertionReservation, _ core.AssertionCompletion) {
		sessionSQL(t, f.db, "UPDATE authenticators SET backup_state=true")
	}})

	issued, err := svc.FinishWebAuthn(t.Context(), ch.Token, signedAssertion(t, f, ch, 1, 13, nil), webAuthnRequirement())
	if err == nil || issued != nil {
		t.Fatal("stale backup state accepted")
	}
}
