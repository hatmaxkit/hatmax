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
	"time"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrEmailTaken        = errors.New("email already taken")
	ErrSessionNotFound   = errors.New("session not found")
	ErrSessionExpired    = errors.New("session expired")
	ErrPasswordTooShort  = errors.New("password too short")
	ErrInvalidEmail      = errors.New("invalid email")
	ErrCredentialChanged = errors.New("credential state changed")
)

// CredentialState identifies the immutable authentication snapshot verified by
// a caller. Password, activation and other auth-state mutations increment Version.
type CredentialState struct {
	UserID  string
	Version int64
}

// Queries owns persistent auth operations and returns caller-owned snapshots.
// CreateUser enforces email uniqueness and classifies conflicts as ErrEmailTaken.
// CreateSession atomically checks active state and expected version before insertion.
// ReplacePassword atomically checks state, replaces the record, increments the
// version and revokes sessions. Stale/inactive/missing state is ErrCredentialChanged.
type Queries interface {
	CreateUser(ctx context.Context, id, email, passwordHash string, createdAt, updatedAt time.Time) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	CreateSession(ctx context.Context, state CredentialState, session SessionRecord) (*Session, error)
	ReplacePassword(ctx context.Context, state CredentialState, passwordHash string, changedAt time.Time) (*User, error)
	ValidateSession(ctx context.Context, digest SessionDigest, activity SessionActivity, interval time.Duration) (*ValidatedSession, error)
	DeleteSession(ctx context.Context, digest SessionDigest) error
	DeleteExpiredSessions(ctx context.Context, limit int) (int64, error)
}

// Service owns a shared credential policy/verifier and caller-owned storage.
type Service struct {
	queries         Queries
	cfg             *config.Config
	log             log.Logger
	policy          *PasswordPolicy
	verifier        *model.PasswordVerifier
	passwordTimeout time.Duration
	sessions        config.SessionSettings
	sessionToken    func() (string, SessionDigest, error)
}

// NewService validates credential configuration and requires a bounded checker.
// The current service permits password-only access and always uses a minimum of
// 15 code points. This constructor does not implement an always-MFA flow.
func NewService(queries Queries, cfg *config.Config, checker PasswordChecker, logger log.Logger) (*Service, error) {
	if queries == nil || cfg == nil || logger == nil {
		return nil, errors.New("auth service dependencies are required")
	}

	sessionSettings, err := cfg.Auth.SessionSettings()
	if err != nil {
		return nil, err
	}

	settings, err := cfg.Auth.PasswordSettings()
	if err != nil {
		return nil, err
	}

	policy, err := NewPasswordPolicy(PasswordPolicyConfig{MinLength: settings.MinLength, MaxLength: settings.MaxLength, MaxBytes: settings.MaxBytes, CheckTimeout: settings.CheckTimeout}, checker)
	if err != nil {
		return nil, err
	}

	verifier, err := model.NewPasswordVerifier(settings.Verifier)
	if err != nil {
		return nil, err
	}

	return &Service{queries: queries, cfg: cfg, log: logger, policy: policy, verifier: verifier, passwordTimeout: settings.Timeout, sessions: sessionSettings, sessionToken: newSessionToken}, nil
}

// Signup applies candidate policy and creates a salted, encoded credential.
// The storage write enforces uniqueness even for simultaneous signup requests.
func (s *Service) Signup(ctx context.Context, email, password string) (*User, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}

	workCtx, cancel := context.WithTimeout(ctx, s.passwordTimeout)
	defer cancel()

	candidate, err := s.policy.Prepare(workCtx, password)
	if err != nil {
		return nil, err
	}

	passwordHash, err := s.verifier.Hash(workCtx, candidate)
	if err != nil {
		return nil, fmt.Errorf("cannot hash password: %w", err)
	}

	now := model.Now()

	user, err := s.queries.CreateUser(workCtx, model.NewID(), email, passwordHash, now, now)
	if err != nil {
		return nil, fmt.Errorf("cannot create user: %w", err)
	}

	s.log.Infof("User signed up: %s", user.ID)

	return user, nil
}

// Signin validates credentials and creates a session.
func (s *Service) Signin(ctx context.Context, email, password string) (*IssuedSession, error) {
	workCtx, cancel := context.WithTimeout(ctx, s.passwordTimeout)
	defer cancel()

	user, err := s.queries.GetUserByEmail(workCtx, email)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("cannot get user: %w", err)
	}

	if !user.Active {
		return nil, errors.New("user is not active")
	}

	// Capture the owned snapshot before expensive verification. The final storage
	// operation checks its version under the same lock as session insertion.
	state := CredentialState{UserID: user.ID, Version: user.AuthVersion}

	err = s.verifier.Verify(workCtx, user.PasswordHash, password)
	if errors.Is(err, model.ErrPasswordMismatch) {
		return nil, ErrInvalidPassword
	}

	if err != nil {
		return nil, fmt.Errorf("cannot verify password: %w", err)
	}

	token, digest, err := s.sessionToken()
	if err != nil {
		return nil, fmt.Errorf("cannot generate session secret: %w", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	record := SessionRecord{Session: Session{ID: model.NewID(), UserID: state.UserID, AuthVersion: state.Version, Generation: 1, AuthenticatedAt: now, CreatedAt: now, LastActivityAt: now, ExpiresAt: now.Add(s.sessions.TTL), InactivityTTL: s.sessions.InactivityTTL}, Digest: digest}

	sessionCtx, sessionCancel := context.WithTimeout(workCtx, s.sessions.Timeout)
	defer sessionCancel()

	session, err := s.queries.CreateSession(sessionCtx, state, record)
	if err != nil {
		return nil, fmt.Errorf("cannot create session: %w", err)
	}

	err = sessionCtx.Err()
	if err != nil {
		return nil, err
	}

	s.log.Infof("User signed in: %s", user.ID)

	return &IssuedSession{Session: *session, Token: token}, nil
}

// Signout destroys a session.
func (s *Service) Signout(ctx context.Context, token string) error {
	digest, err := ParseSessionToken(token)
	if err != nil {
		return err
	}

	workCtx, cancel := context.WithTimeout(ctx, s.sessions.Timeout)
	defer cancel()

	err = s.queries.DeleteSession(workCtx, digest)
	if err != nil {
		return fmt.Errorf("cannot delete session: %w", err)
	}

	return nil
}

// ValidateSession checks current stored state and optionally records trusted
// relevant subscriber activity. NoActivity is required for background polling.
func (s *Service) ValidateSession(ctx context.Context, token string, activity SessionActivity) (*ValidatedSession, error) {
	if activity != NoActivity && activity != RelevantActivity {
		return nil, ErrSessionActivity
	}

	digest, err := ParseSessionToken(token)
	if err != nil {
		return nil, err
	}

	workCtx, cancel := context.WithTimeout(ctx, s.sessions.Timeout)
	defer cancel()

	session, err := s.queries.ValidateSession(workCtx, digest, activity, s.sessions.ActivityInterval)
	if err != nil {
		return nil, fmt.Errorf("cannot validate session: %w", err)
	}

	err = workCtx.Err()
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetUserByID retrieves a user by ID.
func (s *Service) GetUserByID(ctx context.Context, userID string) (*User, error) {
	user, err := s.queries.GetUserByID(ctx, userID)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("cannot get user: %w", err)
	}

	return user, nil
}

// CleanupExpiredSessions removes expired sessions from the database.
func (s *Service) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	workCtx, cancel := context.WithTimeout(ctx, s.sessions.Timeout)
	defer cancel()

	count, err := s.queries.DeleteExpiredSessions(workCtx, s.sessions.CleanupBatch)
	if err != nil {
		return 0, fmt.Errorf("cannot delete expired sessions: %w", err)
	}

	return count, nil
}
