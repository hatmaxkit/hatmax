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
	CreateSession(ctx context.Context, state CredentialState, session Session) (*Session, error)
	ReplacePassword(ctx context.Context, state CredentialState, passwordHash string, changedAt time.Time) (*User, error)
	GetSessionByToken(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteExpiredSessions(ctx context.Context) error
}

// Service owns a shared credential policy/verifier and caller-owned storage.
type Service struct {
	queries         Queries
	cfg             *config.Config
	log             log.Logger
	policy          *PasswordPolicy
	verifier        *model.PasswordVerifier
	passwordTimeout time.Duration
}

// NewService validates credential configuration and requires a bounded checker.
// The current service permits password-only access and always uses a minimum of
// 15 code points. This constructor does not implement an always-MFA flow.
func NewService(queries Queries, cfg *config.Config, checker PasswordChecker, logger log.Logger) (*Service, error) {
	if queries == nil || cfg == nil || logger == nil {
		return nil, errors.New("auth service dependencies are required")
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

	return &Service{queries: queries, cfg: cfg, log: logger, policy: policy, verifier: verifier, passwordTimeout: settings.Timeout}, nil
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
func (s *Service) Signin(ctx context.Context, email, password string) (*Session, error) {
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

	// Parse session TTL
	ttl, err := time.ParseDuration(s.cfg.Auth.SessionTTL)
	if err != nil {
		ttl = 24 * time.Hour // fallback to 24 hours
	}

	// Create session
	now := model.Now()

	session, err := s.queries.CreateSession(workCtx, state, Session{
		ID: model.NewID(), UserID: state.UserID, Token: model.NewID(), ExpiresAt: now.Add(ttl), CreatedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create session: %w", err)
	}

	s.log.Infof("User signed in: %s", user.ID)

	return session, nil
}

// Signout destroys a session.
func (s *Service) Signout(ctx context.Context, sessionToken string) error {
	session, err := s.queries.GetSessionByToken(ctx, sessionToken)
	if err == sql.ErrNoRows {
		return ErrSessionNotFound
	}

	if err != nil {
		return fmt.Errorf("cannot get session: %w", err)
	}

	err = s.queries.DeleteSession(ctx, session.ID)
	if err != nil {
		return fmt.Errorf("cannot delete session: %w", err)
	}

	s.log.Infof("User signed out: %s", session.UserID)

	return nil
}

// ValidateSession checks if a session token is valid and returns the user.
func (s *Service) ValidateSession(ctx context.Context, token string) (*User, error) {
	session, err := s.queries.GetSessionByToken(ctx, token)
	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("cannot get session: %w", err)
	}

	if session.ExpiresAt.Before(model.Now()) {
		return nil, ErrSessionExpired
	}

	user, err := s.queries.GetUserByID(ctx, session.UserID)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("cannot get user: %w", err)
	}

	if !user.Active {
		return nil, errors.New("user is not active")
	}

	return user, nil
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
func (s *Service) CleanupExpiredSessions(ctx context.Context) error {
	err := s.queries.DeleteExpiredSessions(ctx)
	if err != nil {
		return fmt.Errorf("cannot cleanup expired sessions: %w", err)
	}

	return nil
}
