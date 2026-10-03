//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/log"
)

type observedProofQueries struct {
	*Queries
	proof chan core.SessionRecord
}

func (q observedProofQueries) CreateSession(ctx context.Context, state core.CredentialState, record core.SessionRecord, required core.AccessRequirement, limit int) (*core.Session, error) {
	q.proof <- record

	return q.Queries.CreateSession(ctx, state, record, required, limit)
}

func waitProofExpiry(t *testing.T, ctx context.Context, db *sql.DB, expiry time.Time) {
	t.Helper()

	for {
		var expired bool

		err := db.QueryRowContext(ctx, "SELECT clock_timestamp()>=$1::timestamptz", expiry).Scan(&expired)
		if err != nil {
			t.Fatal(err)
		}

		if expired {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// Actual verifier output and locked PostgreSQL time establish policy/freshness
// decisions. Rejected requirements cannot insert sessions or renew activity.
func TestProofTransactions(t *testing.T) {
	t.Run("actual password and unavailable methods", func(t *testing.T) {
		db, _, svc, issued := sessionFixture(t)

		stored := storedSession(t, db, issued)
		if stored.PolicyRevision != PasswordRequirement().Revision || stored.ProofMethod != int16(core.PasswordProof) || !stored.ProofVerifiedAt.Equal(issued.Proof.VerifiedAt) {
			t.Fatal("verified password facts not persisted")
		}

		required := PasswordRequirement()
		for _, profile := range []core.RequiredProof{core.RequireMFA, core.RequirePhishingResistantMFA} {
			required.Proof = profile

			result, err := svc.Signin(t.Context(), "session@example.com", "a distinct safe password", required)
			if err != nil || result == nil || result.Outcome == core.AuthenticationCompleted || result.Issued != nil || result.Reason != core.AuthenticationMethodUnavailable {
				t.Fatalf("unsupported profile issued authority: %v", err)
			}
		}

		var sessions, pendingTables int

		err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM sessions").Scan(&sessions)
		if err != nil || sessions != 1 {
			t.Fatal("unavailable methods persisted sessions")
		}

		err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_name LIKE '%pending%'").Scan(&pendingTables)
		if err != nil || pendingTables != 0 {
			t.Fatal("placeholder pending state exists")
		}
		// The initial schema also rejects attempts to label unverified methods.
		_, err = db.ExecContext(t.Context(), "UPDATE sessions SET proof_method=2 WHERE id=$1", issued.ID)
		if err == nil {
			t.Fatal("unsupported stored method accepted")
		}
	})
	t.Run("policy checks precede activity", func(t *testing.T) {
		db, _, svc, issued := sessionFixture(t)
		sessionSQL(t, db, "UPDATE sessions SET created_at=clock_timestamp()-interval '3 minutes',proof_verified_at=clock_timestamp()-interval '150 seconds',authenticated_at=clock_timestamp()-interval '2 minutes',last_activity_at=clock_timestamp()-interval '90 seconds' WHERE id=$1", issued.ID)
		before := storedSession(t, db, issued)

		tests := []struct {
			name     string
			required core.AccessRequirement
			want     error
		}{
			{"revision", core.AccessRequirement{Proof: core.RequirePassword, Revision: "changed-v2"}, core.ErrSessionPolicy},
			{"MFA", core.AccessRequirement{Proof: core.RequireMFA, Revision: PasswordRequirement().Revision}, core.ErrSessionProof},
			{"freshness", core.AccessRequirement{Proof: core.RequirePassword, Revision: PasswordRequirement().Revision, MaxAge: time.Minute}, core.ErrSessionProofExpired},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				validated, err := svc.ValidateSession(t.Context(), issued.Token, test.required, core.RelevantActivity)
				if validated != nil || !errors.Is(err, test.want) {
					t.Fatalf("protected validation: %v", err)
				}

				after := storedSession(t, db, issued)
				if !after.LastActivityAt.Equal(before.LastActivityAt) || !after.ProofVerifiedAt.Equal(before.ProofVerifiedAt) {
					t.Fatal("rejected requirement renewed activity/proof")
				}
			})
		}

		validated, err := svc.ValidateSession(t.Context(), issued.Token, PasswordRequirement(), core.RelevantActivity)
		if err != nil || validated == nil {
			t.Fatalf("password requirement did not pass: %v", err)
		}

		after := storedSession(t, db, issued)
		if !after.LastActivityAt.After(before.LastActivityAt) || !after.ProofVerifiedAt.Equal(before.ProofVerifiedAt) {
			t.Fatal("activity refreshed password proof")
		}
	})
	t.Run("freshness after session lock wait", func(t *testing.T) {
		db, _, svc, issued := sessionFixture(t)
		sessionSQL(t, db, "UPDATE sessions SET created_at=clock_timestamp()-interval '62 seconds',proof_verified_at=clock_timestamp()-interval '60500 milliseconds',authenticated_at=clock_timestamp()-interval '60 seconds',last_activity_at=clock_timestamp()-interval '60 seconds' WHERE id=$1", issued.ID)
		before := storedSession(t, db, issued)
		required := PasswordRequirement()
		required.MaxAge = 61 * time.Second

		metadata, err := toAuthSession(before)
		if err != nil {
			t.Fatal(err)
		}

		now, err := dal.New(db).SessionClock(t.Context())
		if err != nil || required.Evaluate(*metadata, now) != nil {
			t.Fatal("proof expired before lock wait")
		}

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		holder, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()

		_, err = dal.New(holder).GetSessionForUpdate(ctx, before.TokenDigest)
		if err != nil {
			t.Fatal(err)
		}

		result := make(chan error, 1)

		go func() {
			_, validateErr := svc.ValidateSession(ctx, issued.Token, required, core.RelevantActivity)
			result <- validateErr
		}()

		waitSessionLock(t, ctx, db, holder)
		waitProofExpiry(t, ctx, db, before.ProofVerifiedAt.Add(required.MaxAge))

		err = holder.Commit()
		if err != nil {
			t.Fatal(err)
		}

		err = <-result
		if !errors.Is(err, core.ErrSessionProofExpired) {
			t.Fatalf("pre-lock freshness authorized request: %v", err)
		}

		after := storedSession(t, db, issued)
		if !after.LastActivityAt.Equal(before.LastActivityAt) {
			t.Fatal("expired proof renewed activity")
		}
	})
	t.Run("issuance freshness after user lock wait", func(t *testing.T) {
		db, q, cfg := credentialDatabase(t)
		observed := observedProofQueries{Queries: q, proof: make(chan core.SessionRecord, 1)}

		svc, err := core.NewService(observed, cfg, NewPasswordChecker(), log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}

		user, err := svc.Signup(t.Context(), "issue@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		holder, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()

		_, err = dal.New(holder).GetUserForAuth(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		required := PasswordRequirement()
		required.MaxAge = time.Second
		result := make(chan error, 1)

		go func() {
			outcome, signinErr := svc.Signin(ctx, user.Email, "a distinct safe password", required)
			if outcome != nil {
				signinErr = errors.New("stale proof produced result")
			}

			result <- signinErr
		}()

		var record core.SessionRecord
		select {
		case record = <-observed.proof:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}

		waitSessionLock(t, ctx, db, holder)
		waitProofExpiry(t, ctx, db, record.Proof.VerifiedAt.Add(required.MaxAge))

		err = holder.Commit()
		if err != nil {
			t.Fatal(err)
		}

		err = <-result
		if !errors.Is(err, core.ErrSessionProofExpired) {
			t.Fatalf("stale issuance proof: %v", err)
		}

		var sessions int

		err = db.QueryRowContext(ctx, "SELECT count(*) FROM sessions").Scan(&sessions)
		if err != nil || sessions != 0 {
			t.Fatal("stale proof persisted session")
		}
	})
}
