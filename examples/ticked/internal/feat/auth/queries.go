// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"time"

	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
)

// querier defines the interface for sqlc operations needed by auth.
type querier interface {
	CountUsers(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, arg dal.CreateUserParams) (dal.User, error)
	DeleteExpiredSessions(ctx context.Context, limit int32) (int64, error)
	DeleteSession(ctx context.Context, digest []byte) (int64, error)
	GetUserByEmail(ctx context.Context, email string) (dal.User, error)
	GetUserByID(ctx context.Context, id string) (dal.User, error)
	ListUsers(ctx context.Context) ([]dal.User, error)
	UpdateUserActive(ctx context.Context, arg dal.UpdateUserActiveParams) error
	UpdateUserRoles(ctx context.Context, arg dal.UpdateUserRolesParams) error
}

// DBProvider provides access to a database connection.
type DBProvider interface {
	GetDB() *sql.DB
}

// Queries adapts dal.Queries to hatmax auth.Queries interface.
type Queries struct {
	dbProvider DBProvider
	q          querier
}

// NewQueries creates a new auth queries adapter.
func NewQueries(dbProvider DBProvider) *Queries {
	return &Queries{dbProvider: dbProvider}
}

// Start initializes the queries with the database connection.
func (q *Queries) Start(ctx context.Context) error {
	db := q.dbProvider.GetDB()
	if db == nil {
		return fmt.Errorf("database connection not available")
	}

	q.q = dal.New(db)

	return nil
}

// Stop is a no-op for Queries.
func (q *Queries) Stop(ctx context.Context) error {
	return nil
}

// SetQuerier sets the internal querier (for testing).
func (q *Queries) SetQuerier(querier querier) {
	q.q = querier
}

// CreateUser creates a new user and returns it.
func (q *Queries) CreateUser(ctx context.Context, id, email, passwordHash string, createdAt, updatedAt time.Time) (*auth.User, error) {
	user, err := q.q.CreateUser(ctx, dal.CreateUserParams{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Roles:        []string{},
		Active:       true,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
			return nil, auth.ErrEmailTaken
		}

		return nil, err
	}

	return toAuthUser(user), nil
}

// GetUserByEmail retrieves a user by email.
func (q *Queries) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	user, err := q.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return toAuthUser(user), nil
}

// GetUserByID retrieves a user by ID.
func (q *Queries) GetUserByID(ctx context.Context, id string) (*auth.User, error) {
	user, err := q.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return toAuthUser(user), nil
}

// lockCredential rechecks the subject's auth state under the mutation row lock.
func lockCredential(ctx context.Context, tx *sql.Tx, state auth.CredentialState) (dal.User, error) {
	user, err := dal.New(tx).GetUserForAuth(ctx, state.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		return dal.User{}, auth.ErrCredentialChanged
	}

	if err != nil {
		return dal.User{}, err
	}

	if !user.Active || state.Version < 1 || user.AuthVersion != state.Version {
		return dal.User{}, auth.ErrCredentialChanged
	}

	return user, nil
}

func (q *Queries) beginCredentialTx(ctx context.Context) (*sql.Tx, error) {
	if q.dbProvider == nil || q.dbProvider.GetDB() == nil {
		return nil, errors.New("database connection not available")
	}

	return q.dbProvider.GetDB().BeginTx(ctx, nil)
}

// CreateSession atomically checks current credential/account state and inserts
// the session. Activation/password changes serialize through the user row lock.
func (q *Queries) CreateSession(ctx context.Context, state auth.CredentialState, session auth.SessionRecord, requirement auth.AccessRequirement, limit int) (*auth.Session, error) {
	if limit < 1 || limit > 100 {
		return nil, auth.ErrSessionCapacity
	}

	if session.UserID != state.UserID || session.AuthVersion != state.Version || session.Generation != 1 {
		return nil, auth.ErrCredentialChanged
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = lockCredential(ctx, tx, state)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	_, err = queries.ReclaimSubjectSessions(ctx, state.UserID)
	if err != nil {
		return nil, err
	}

	count, err := queries.CountSubjectSessions(ctx, state.UserID)
	if err != nil {
		return nil, err
	}

	if count >= int64(limit) {
		return nil, auth.ErrSessionCapacity
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = requirement.Evaluate(session.Session, now)
	if err != nil {
		return nil, err
	}

	stored, err := queries.CreateSession(ctx, dal.CreateSessionParams{
		ID: session.ID, UserID: state.UserID, TokenDigest: session.Digest[:], AuthVersion: session.AuthVersion, PolicyRevision: session.PolicyRevision, Generation: session.Generation, ProofMethod: int16(session.Proof.Method), ProofVerifiedAt: session.Proof.VerifiedAt,
		AuthenticatedAt: session.AuthenticatedAt, CreatedAt: session.CreatedAt, LastActivityAt: session.LastActivityAt,
		ExpiresAt: session.ExpiresAt, InactivityUs: session.InactivityTTL.Microseconds(),
	})
	if err != nil {
		return nil, err
	}

	metadata, err := toAuthSession(stored)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// ValidateSession serializes account mutation and session activity. The clock
// is sampled after both row locks so a lock wait cannot revive expired state.
func (q *Queries) ValidateSession(ctx context.Context, digest auth.SessionDigest, requirement auth.AccessRequirement, activity auth.SessionActivity, interval time.Duration) (*auth.ValidatedSession, error) {
	if activity != auth.NoActivity && activity != auth.RelevantActivity {
		return nil, auth.ErrSessionActivity
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	queries := dal.New(tx)

	initial, err := queries.GetSessionByDigest(ctx, digest[:])
	if errors.Is(err, sql.ErrNoRows) {
		return nil, auth.ErrSessionNotFound
	}

	if err != nil {
		return nil, err
	}

	user, err := lockCredential(ctx, tx, auth.CredentialState{UserID: initial.UserID, Version: initial.AuthVersion})
	if err != nil {
		return nil, err
	}

	stored, err := queries.GetSessionForUpdate(ctx, digest[:])
	if errors.Is(err, sql.ErrNoRows) {
		return nil, auth.ErrSessionNotFound
	}

	if err != nil {
		return nil, err
	}

	if stored.UserID != user.ID || stored.AuthVersion != user.AuthVersion {
		return nil, auth.ErrCredentialChanged
	}

	metadata, err := toAuthSession(stored)
	if err != nil {
		return nil, err
	}

	if interval < time.Second || interval > 5*time.Minute || interval > metadata.InactivityTTL/4 || interval%time.Microsecond != 0 {
		return nil, auth.ErrSessionActivity
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = requirement.Evaluate(*metadata, now)
	if err != nil {
		return nil, err
	}

	if activity == auth.RelevantActivity && now.Sub(metadata.LastActivityAt) >= interval {
		stored, err = queries.UpdateSessionActivity(ctx, dal.UpdateSessionActivityParams{ID: metadata.ID, LastActivityAt: now})
		if err != nil {
			return nil, err
		}

		metadata, err = toAuthSession(stored)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &auth.ValidatedSession{User: toAuthUser(user), Session: *metadata}, nil
}

// ReplacePassword checks expected current state, replaces the whole record and
// revokes sessions in one transaction. The caller supplies a policy-approved,
// encoded record and owns authorization for the credential-change operation.
func (q *Queries) ReplacePassword(ctx context.Context, state auth.CredentialState, passwordHash string, changedAt time.Time) (*auth.User, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = lockCredential(ctx, tx, state)
	if err != nil {
		return nil, err
	}

	updated, err := dal.New(tx).ReplacePassword(ctx, dal.ReplacePasswordParams{ID: state.UserID, PasswordHash: passwordHash, UpdatedAt: changedAt})
	if err != nil {
		return nil, err
	}

	err = dal.New(tx).DeleteUserSessions(ctx, state.UserID)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return toAuthUser(updated), nil
}

// DeleteSession revokes only the presented digest.
func (q *Queries) DeleteSession(ctx context.Context, digest auth.SessionDigest) error {
	count, err := q.q.DeleteSession(ctx, digest[:])
	if err != nil {
		return err
	}

	if count == 0 {
		return auth.ErrSessionNotFound
	}

	return nil
}

// DeleteExpiredSessions deletes one bounded batch without waiting on busy rows.
func (q *Queries) DeleteExpiredSessions(ctx context.Context, limit int) (int64, error) {
	if limit < 1 || limit > 1000 {
		return 0, auth.ErrSessionRecord
	}

	return q.q.DeleteExpiredSessions(ctx, int32(limit))
}

// ListUsers returns all users (for admin).
func (q *Queries) ListUsers(ctx context.Context) ([]*auth.User, error) {
	users, err := q.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*auth.User, len(users))
	for i, u := range users {
		result[i] = toAuthUser(u)
	}

	return result, nil
}

// UpdateUserRoles updates a user's roles.
func (q *Queries) UpdateUserRoles(ctx context.Context, id string, roles []string, updatedAt time.Time) error {
	return q.q.UpdateUserRoles(ctx, dal.UpdateUserRolesParams{
		ID:        id,
		Roles:     roles,
		UpdatedAt: updatedAt,
	})
}

// UpdateUserActive updates a user's active status.
func (q *Queries) UpdateUserActive(ctx context.Context, id string, active bool, updatedAt time.Time) error {
	return q.q.UpdateUserActive(ctx, dal.UpdateUserActiveParams{
		ID:        id,
		Active:    active,
		UpdatedAt: updatedAt,
	})
}

// CountUsers returns the total number of users.
func (q *Queries) CountUsers(ctx context.Context) (int64, error) {
	return q.q.CountUsers(ctx)
}

func toAuthUser(u dal.User) *auth.User {
	return &auth.User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		AuthVersion:  u.AuthVersion,
		Roles:        append([]string(nil), u.Roles...),
		Active:       u.Active,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func toAuthSession(s dal.Session) (*auth.Session, error) {
	if s.ProofMethod != int16(auth.PasswordProof) || len(s.TokenDigest) != 32 || s.InactivityUs < time.Minute.Microseconds() || s.InactivityUs > (30*24*time.Hour).Microseconds() {
		return nil, auth.ErrSessionRecord
	}

	return &auth.Session{
		ID: s.ID, UserID: s.UserID, AuthVersion: s.AuthVersion, PolicyRevision: s.PolicyRevision, Generation: s.Generation, Proof: auth.VerifiedProof{Method: auth.ProofMethod(s.ProofMethod), VerifiedAt: s.ProofVerifiedAt},
		AuthenticatedAt: s.AuthenticatedAt, CreatedAt: s.CreatedAt, LastActivityAt: s.LastActivityAt,
		ExpiresAt: s.ExpiresAt, InactivityTTL: time.Duration(s.InactivityUs) * time.Microsecond,
	}, nil
}
