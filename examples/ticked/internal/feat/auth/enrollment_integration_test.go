//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/fxamacker/cbor/v2"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"sync"
	"testing"
	"time"
)

func enrollmentRequirement() core.AccessRequirement {
	return core.AccessRequirement{Proof: core.RequirePhishingResistantMFA, Revision: "enroll-v1", MaxAge: 5 * time.Minute}
}

func enrollmentFixture(t *testing.T, options config.AuthenticatorConfig) (*sql.DB, *Queries, *core.Service, *core.AuthenticatorService, *core.EnrollmentChallenge) {
	t.Helper()
	db, q, base, _ := sessionFixture(t)
	options.RPID = "example.com"
	options.RPName = "Example"
	options.Origins = []string{"https://example.com"}

	svc, err := core.NewAuthenticatorService(base, q, options)
	if err != nil {
		t.Fatal(err)
	}

	challenge, err := svc.BeginWebAuthnEnrollment(t.Context(), "session@example.com", "a distinct safe password", enrollmentRequirement())
	if err != nil {
		t.Fatal(err)
	}

	return db, q, base, svc, challenge
}

// enrollmentResponse generates a real ES256 credential and an attestation-none
// browser response. It does not bypass the production library verifier.
func enrollmentResponse(t *testing.T, challenge *core.EnrollmentChallenge) []byte {
	t.Helper()
	body, _, _ := enrollmentKeyResponse(t, challenge)

	return body
}

func enrollmentKeyResponse(t *testing.T, challenge *core.EnrollmentChallenge) ([]byte, *ecdsa.PrivateKey, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	cose, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: key.X.FillBytes(make([]byte, 32)), -3: key.Y.FillBytes(make([]byte, 32))})
	if err != nil {
		t.Fatal(err)
	}

	id := make([]byte, 32)

	_, err = rand.Read(id)
	if err != nil {
		t.Fatal(err)
	}

	rp := sha256.Sum256([]byte("example.com"))
	authData := append(rp[:], 0x45, 0, 0, 0, 0)
	authData = append(authData, make([]byte, 16)...)
	authData = append(authData, 0, 32)
	authData = append(authData, id...)
	authData = append(authData, cose...)

	attestation, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": authData})
	if err != nil {
		t.Fatal(err)
	}

	client, err := json.Marshal(map[string]any{"type": "webauthn.create", "challenge": base64.RawURLEncoding.EncodeToString(challenge.Options.Response.Challenge), "origin": "https://example.com", "crossOrigin": false})
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]any{"id": base64.RawURLEncoding.EncodeToString(id), "rawId": base64.RawURLEncoding.EncodeToString(id), "type": "public-key", "response": map[string]any{"clientDataJSON": base64.RawURLEncoding.EncodeToString(client), "attestationObject": base64.RawURLEncoding.EncodeToString(attestation)}, "clientExtensionResults": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}

	return body, key, id
}

// Real registration and PostgreSQL establish atomic activation, invalidation and
// single-use setup without interpreting enrollment as a sign-in proof.
func TestEnrollmentTransactions(t *testing.T) {
	t.Run("transaction bounds", testEnrollmentTransactionBounds)
	t.Run("confirmation invalidates prior authority", func(t *testing.T) {
		db, _, _, svc, challenge := enrollmentFixture(t, config.AuthenticatorConfig{})
		response := enrollmentResponse(t, challenge)

		result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, response, enrollmentRequirement())
		if err != nil || result == nil {
			t.Fatalf("registration: %v", err)
		}

		var factors, sessions, pending, version int

		err = db.QueryRowContext(t.Context(), "SELECT (SELECT count(*) FROM authenticators),(SELECT count(*) FROM sessions),(SELECT count(*) FROM auth_pending),auth_version FROM users WHERE email='session@example.com'").Scan(&factors, &sessions, &pending, &version)
		if err != nil || factors != 1 || sessions != 0 || pending != 0 || version != 2 {
			t.Fatalf("non-atomic registration: %d %d %d %d %v", factors, sessions, pending, version, err)
		}

		replay, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, response, enrollmentRequirement())
		if err == nil || replay != nil {
			t.Fatal("registration replay succeeded")
		}

		setup, err := svc.BeginWebAuthnEnrollment(t.Context(), "session@example.com", "a distinct safe password", enrollmentRequirement())
		if err == nil || setup != nil {
			t.Fatal("password-only established-factor enrollment succeeded")
		}
	})
}

type changingEnrollment struct {
	*Queries
	change func(context.Context, core.EnrollmentPending) error
}

func (q changingEnrollment) ConfirmEnrollment(ctx context.Context, p core.EnrollmentPending, r core.RegistrationRecord, requirement core.AccessRequirement, settings config.EnrollmentSettings) (*core.Authenticator, error) {
	err := q.change(ctx, p)
	if err != nil {
		return nil, err
	}

	return q.Queries.ConfirmEnrollment(ctx, p, r, requirement, settings)
}

func enrollmentDigest(t *testing.T, db *sql.DB, token string) core.EnrollmentDigest {
	t.Helper()

	secret, err := base64.RawURLEncoding.Strict().DecodeString(token[8:])
	if err != nil {
		t.Fatal(err)
	}

	digest := sha256.Sum256(append([]byte("hatmax/pending/enrollment/v1\x00"), secret...))

	return core.EnrollmentDigest(digest)
}

func enrollmentState(t *testing.T, db *sql.DB) (factors, sessions, pending, version int) {
	t.Helper()

	err := db.QueryRowContext(t.Context(), "SELECT (SELECT count(*) FROM authenticators),(SELECT count(*) FROM sessions),(SELECT count(*) FROM auth_pending),auth_version FROM users WHERE email='session@example.com'").Scan(&factors, &sessions, &pending, &version)
	if err != nil {
		t.Fatal(err)
	}

	return factors, sessions, pending, version
}

// Transaction failures, competing setup and actual account mutations cannot
// activate stale protocol results or undo already spent verification attempts.
func testEnrollmentTransactionBounds(t *testing.T) {
	t.Run("competing confirmations", func(t *testing.T) {
		db, _, _, svc, challenge := enrollmentFixture(t, config.AuthenticatorConfig{})
		response := enrollmentResponse(t, challenge)
		start := make(chan struct{})
		results := make(chan error, 2)

		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() {
				<-start

				result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, response, enrollmentRequirement())
				if err == nil && result == nil {
					err = errors.New("empty successful confirmation")
				}

				results <- err
			})
		}

		close(start)
		wg.Wait()
		close(results)

		wins := 0

		for err := range results {
			if err == nil {
				wins++
			}
		}

		factors, sessions, pending, version := enrollmentState(t, db)
		if wins != 1 || factors != 1 || sessions != 0 || pending != 0 || version != 2 {
			t.Fatalf("confirmation winners %d state %d %d %d %d", wins, factors, sessions, pending, version)
		}
	})

	for _, mutation := range []string{"password", "disabled"} {
		t.Run(mutation+" before confirmation", func(t *testing.T) {
			db, q, base, _, challenge := enrollmentFixture(t, config.AuthenticatorConfig{})
			wrapped := changingEnrollment{Queries: q, change: func(ctx context.Context, p core.EnrollmentPending) error {
				if mutation == "disabled" {
					return q.UpdateUserActive(ctx, p.State.UserID, false, time.Now().UTC())
				}

				user, err := q.GetUserByID(ctx, p.State.UserID)
				if err != nil {
					return err
				}

				_, err = q.ReplacePassword(ctx, p.State, user.PasswordHash, time.Now().UTC())

				return err
			}}

			svc, err := core.NewAuthenticatorService(base, wrapped, config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}})
			if err != nil {
				t.Fatal(err)
			}

			result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, enrollmentResponse(t, challenge), enrollmentRequirement())
			if result != nil || !errors.Is(err, core.ErrCredentialChanged) {
				t.Fatalf("stale registration admitted: %v", err)
			}

			factors, _, _, version := enrollmentState(t, db)
			if factors != 0 || version != 2 {
				t.Fatal("mutation did not invalidate setup")
			}

			var attempts int

			err = db.QueryRowContext(t.Context(), "SELECT attempts FROM auth_pending").Scan(&attempts)
			if err != nil || attempts != 1 {
				t.Fatal("account conflict refunded verifier admission")
			}
		})
	}

	t.Run("commit rollback and retry", func(t *testing.T) {
		db, _, _, svc, challenge := enrollmentFixture(t, config.AuthenticatorConfig{})
		response := enrollmentResponse(t, challenge)
		sessionSQL(t, db, `CREATE FUNCTION reject_enrollment() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced confirmation failure'; END $$;
CREATE CONSTRAINT TRIGGER reject_enrollment AFTER INSERT ON authenticators DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_enrollment();`)

		result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, response, enrollmentRequirement())
		if result != nil || err == nil {
			t.Fatal("commit failure returned success")
		}

		factors, sessions, pending, version := enrollmentState(t, db)
		if factors != 0 || sessions != 1 || pending != 1 || version != 1 {
			t.Fatal("confirmation left a partial write")
		}

		var pendingAttempts, subjectAttempts int

		err = db.QueryRowContext(t.Context(), "SELECT p.attempts,b.attempts FROM auth_pending p JOIN auth_factor_budgets b USING(user_id)").Scan(&pendingAttempts, &subjectAttempts)
		if err != nil || pendingAttempts != 1 || subjectAttempts != 1 {
			t.Fatal("commit failure refunded attempts")
		}

		sessionSQL(t, db, "DROP TRIGGER reject_enrollment ON authenticators")

		result, err = svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, response, enrollmentRequirement())
		if err != nil || result == nil {
			t.Fatalf("fresh verification retry failed: %v", err)
		}
	})
	t.Run("subject attempts survive reissue", func(t *testing.T) {
		db, _, _, svc, challenge := enrollmentFixture(t, config.AuthenticatorConfig{SubjectAttempts: 2})
		for range 2 {
			result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, []byte("{broken"), enrollmentRequirement())
			if result != nil || !errors.Is(err, core.ErrEnrollment) {
				t.Fatal("malformed request did not spend an attempt")
			}
		}

		next, err := svc.BeginWebAuthnEnrollment(t.Context(), "session@example.com", "a distinct safe password", enrollmentRequirement())
		if err != nil {
			t.Fatal(err)
		}

		result, err := svc.FinishWebAuthnEnrollment(t.Context(), next.Token, enrollmentResponse(t, next), enrollmentRequirement())
		if result != nil || !errors.Is(err, core.ErrEnrollmentAttempts) {
			t.Fatalf("reissue bypassed budget: %v", err)
		}

		var attempts int

		err = db.QueryRowContext(t.Context(), "SELECT attempts FROM auth_factor_budgets").Scan(&attempts)
		if err != nil || attempts != 2 {
			t.Fatal("subject budget changed on reissue")
		}
	})
	t.Run("pending attempts and lease revision", func(t *testing.T) {
		db, q, _, svc, challenge := enrollmentFixture(t, config.AuthenticatorConfig{PendingAttempts: 2})

		settings, err := (config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}, PendingAttempts: 2}).EnrollmentSettings()
		if err != nil {
			t.Fatal(err)
		}

		digest := enrollmentDigest(t, db, challenge.Token)

		first, err := q.ReserveEnrollment(t.Context(), digest, enrollmentRequirement(), settings)
		if err != nil {
			t.Fatal(err)
		}

		busy, err := q.ReserveEnrollment(t.Context(), digest, enrollmentRequirement(), settings)
		if busy != nil || !errors.Is(err, core.ErrEnrollmentBusy) {
			t.Fatal("simultaneous pending lease admitted")
		}

		sessionSQL(t, db, "UPDATE auth_pending SET lease_until=clock_timestamp()-interval '1 second'")

		second, err := q.ReserveEnrollment(t.Context(), digest, enrollmentRequirement(), settings)
		if err != nil || second.Revision != first.Revision+1 || second.Attempts != 2 {
			t.Fatalf("lease retry: %v", err)
		}

		err = q.ReleaseEnrollment(t.Context(), digest, first.Revision)
		if err != nil {
			t.Fatal(err)
		}

		var retained bool

		err = db.QueryRowContext(t.Context(), "SELECT lease_until IS NOT NULL FROM auth_pending").Scan(&retained)
		if err != nil || !retained {
			t.Fatal("stale release cleared a newer lease")
		}

		err = q.ReleaseEnrollment(t.Context(), digest, second.Revision)
		if err != nil {
			t.Fatal(err)
		}

		result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, enrollmentResponse(t, challenge), enrollmentRequirement())
		if result != nil || !errors.Is(err, core.ErrEnrollmentAttempts) {
			t.Fatalf("pending exhaustion bypassed: %v", err)
		}
	})
	t.Run("retained capacity and expired reclamation", func(t *testing.T) {
		db, q, _, svc, challenge := enrollmentFixture(t, config.AuthenticatorConfig{MaxPending: 2})
		cfg := config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}, MaxPending: 2}

		settings, err := cfg.EnrollmentSettings()
		if err != nil {
			t.Fatal(err)
		}

		digest := enrollmentDigest(t, db, challenge.Token)

		captured, err := q.ReserveEnrollment(t.Context(), digest, enrollmentRequirement(), settings)
		if err != nil {
			t.Fatal(err)
		}

		err = q.ReleaseEnrollment(t.Context(), digest, captured.Revision)
		if err != nil {
			t.Fatal(err)
		}

		captured.Attempts = 0
		captured.Revision = 0
		captured.LeaseUntil = time.Time{}
		results := make(chan error, 8)

		var wg sync.WaitGroup

		for range 8 {
			p := *captured

			_, err = rand.Read(p.Digest[:])
			if err != nil {
				t.Fatal(err)
			}

			wg.Go(func() { results <- q.CreateEnrollment(t.Context(), p, settings) })
		}

		wg.Wait()
		close(results)

		wins := 0

		for err := range results {
			if err == nil {
				wins++
			} else if !errors.Is(err, core.ErrEnrollmentCapacity) {
				t.Fatal(err)
			}
		}

		var retained int

		err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM auth_pending").Scan(&retained)
		if err != nil || retained != 2 || wins != 1 {
			t.Fatal("retained pending cap not serialized")
		}

		sessionSQL(t, db, "WITH n AS (SELECT clock_timestamp() AS ts) UPDATE auth_pending SET password_at=n.ts-interval '5 minutes',created_at=n.ts-interval '5 minutes',expires_at=n.ts FROM n")

		removed, err := q.DeleteExpiredEnrollments(t.Context(), 1)
		if err != nil || removed != 1 {
			t.Fatal("cleanup ignored finite batch")
		}

		next, err := svc.BeginWebAuthnEnrollment(t.Context(), "session@example.com", "a distinct safe password", enrollmentRequirement())
		if err != nil || next == nil {
			t.Fatalf("expired reclamation blocked admission: %v", err)
		}

		err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM auth_pending").Scan(&retained)
		if err != nil || retained != 1 {
			t.Fatal("admission did not reclaim expired subject rows")
		}
	})

	t.Run("RP credential uniqueness across subjects", func(t *testing.T) {
		db, _, base, svc, first := enrollmentFixture(t, config.AuthenticatorConfig{})
		response := enrollmentResponse(t, first)

		result, err := svc.FinishWebAuthnEnrollment(t.Context(), first.Token, response, enrollmentRequirement())
		if err != nil || result == nil {
			t.Fatal(err)
		}

		_, err = base.Signup(t.Context(), "second@example.com", "a distinct second password")
		if err != nil {
			t.Fatal(err)
		}

		second, err := svc.BeginWebAuthnEnrollment(t.Context(), "second@example.com", "a distinct second password", enrollmentRequirement())
		if err != nil {
			t.Fatal(err)
		}

		var wire map[string]any

		err = json.Unmarshal(response, &wire)
		if err != nil {
			t.Fatal(err)
		}

		client, err := json.Marshal(map[string]any{"type": "webauthn.create", "challenge": base64.RawURLEncoding.EncodeToString(second.Options.Response.Challenge), "origin": "https://example.com", "crossOrigin": false})
		if err != nil {
			t.Fatal(err)
		}

		wire["response"].(map[string]any)["clientDataJSON"] = base64.RawURLEncoding.EncodeToString(client)

		response, err = json.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}

		result, err = svc.FinishWebAuthnEnrollment(t.Context(), second.Token, response, enrollmentRequirement())
		if result != nil || err == nil {
			t.Fatal("credential registered to two subjects")
		}

		var version, factors, attempts int

		err = db.QueryRowContext(t.Context(), "SELECT u.auth_version,(SELECT count(*) FROM authenticators),p.attempts FROM users u JOIN auth_pending p ON u.id=p.user_id WHERE u.email='second@example.com'").Scan(&version, &factors, &attempts)
		if err != nil || version != 1 || factors != 1 || attempts != 1 {
			t.Fatal("uniqueness failure left partial authority or refunded attempt")
		}
	})
	t.Run("caller cancellation while waiting", func(t *testing.T) {
		db, _, _, svc, challenge := enrollmentFixture(t, config.AuthenticatorConfig{})

		holder, err := db.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()

		_, err = holder.ExecContext(t.Context(), "SELECT id FROM users FOR UPDATE")
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		done := make(chan error, 1)

		go func() {
			_, finishErr := svc.FinishWebAuthnEnrollment(ctx, challenge.Token, []byte("{}"), enrollmentRequirement())
			done <- finishErr
		}()

		waitSessionLock(t, t.Context(), db, holder)
		cancel()

		err = <-done
		if err == nil {
			t.Fatal("canceled verification returned success")
		}

		err = holder.Rollback()
		if err != nil {
			t.Fatal(err)
		}

		var attempts int

		err = db.QueryRowContext(t.Context(), "SELECT attempts FROM auth_pending").Scan(&attempts)
		if err != nil || attempts != 0 {
			t.Fatal("cancellation wrote partial reservation")
		}
	})
	t.Run("post-lock expiry and caller deadline", func(t *testing.T) {
		db, _, _, svc, challenge := enrollmentFixture(t, config.AuthenticatorConfig{})

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		holder, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()

		_, err = holder.ExecContext(ctx, "SELECT id FROM users FOR UPDATE")
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan error, 1)

		go func() {
			_, finishErr := svc.FinishWebAuthnEnrollment(ctx, challenge.Token, []byte("{}"), enrollmentRequirement())
			done <- finishErr
		}()

		waitSessionLock(t, ctx, db, holder)

		_, err = holder.ExecContext(ctx, "WITH n AS (SELECT clock_timestamp() AS ts) UPDATE auth_pending SET password_at=n.ts-interval '5 minutes',created_at=n.ts-interval '5 minutes',expires_at=n.ts FROM n")
		if err != nil {
			t.Fatal(err)
		}

		err = holder.Commit()
		if err != nil {
			t.Fatal(err)
		}

		err = <-done
		if !errors.Is(err, core.ErrEnrollment) {
			t.Fatalf("post-lock expiry ignored: %v", err)
		}

		var attempts int

		err = db.QueryRowContext(ctx, "SELECT attempts FROM auth_pending").Scan(&attempts)
		if err != nil || attempts != 0 {
			t.Fatal("expired pending work was charged")
		}
	})
}
