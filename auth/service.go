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
	"unicode/utf8"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

var (
	ErrUserInactive      = errors.New("user is not active")
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
	CreateSession(ctx context.Context, state CredentialState, session SessionRecord, requirement AccessRequirement, limit int) (*Session, error)
	RotateSession(ctx context.Context, state CredentialState, current SessionDigest, generation int64, replacement SessionRecord, requirement AccessRequirement) (*Session, error)
	ListSessions(ctx context.Context, actor SessionDigest, requirement AccessRequirement, limit int, cursor string) (*SessionPage, error)
	RevokeSessions(ctx context.Context, actor SessionDigest, requirement AccessRequirement, selection SessionSelection) (int64, error)
	ReplacePassword(ctx context.Context, state CredentialState, passwordHash string, changedAt time.Time) (*User, error)
	ValidateSession(ctx context.Context, digest SessionDigest, requirement AccessRequirement, activity SessionActivity, interval time.Duration) (*ValidatedSession, error)
	DeleteSession(ctx context.Context, digest SessionDigest) error
	DeleteExpiredSessions(ctx context.Context, limit int) (int64, error)
}

// Service owns a shared credential policy/verifier and caller-owned storage.
type Service struct {
	queries         Queries
	admission       *CredentialAdmission
	observations    *SecurityObservations
	cfg             *config.Config
	log             log.Logger
	policy          *PasswordPolicy
	verifier        *model.PasswordVerifier
	passwordTimeout time.Duration
	sessions        config.SessionSettings
	sessionToken    func() (string, SessionDigest, error)
}

// NewService validates credentials and requires a bounded checker and shared
// durable admission and security observations. Child services share both.
// The current service permits password-only access and always uses a minimum of
// 15 code points. This constructor does not implement an always-MFA flow.
func NewService(queries Queries, cfg *config.Config, checker PasswordChecker, admission *CredentialAdmission, observations *SecurityObservations, logger log.Logger) (*Service, error) {
	if queries == nil || cfg == nil || logger == nil || admission == nil || admission.queries == nil || observations == nil || observations.observer == nil || observations.slots == nil {
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

	return &Service{queries: queries, admission: admission, observations: observations, cfg: cfg, log: logger, policy: policy, verifier: verifier, passwordTimeout: settings.Timeout, sessions: sessionSettings, sessionToken: newSessionToken}, nil
}

// admitCredential checks raw structure without running policy or credential work.
// The caller owns the whole-operation deadline and canonical identity.
func (s *Service) admitCredential(ctx context.Context, identity, password string, purpose CredentialAdmissionPurpose) error {
	if len(password) > s.policy.config.MaxBytes || !utf8.ValidString(password) {
		if purpose == CredentialPasswordProof {
			return model.ErrPasswordInput
		}

		if len(password) > s.policy.config.MaxBytes {
			return ErrPasswordTooLong
		}

		return ErrPasswordEncoding
	}

	return s.admission.Admit(ctx, identity, purpose)
}

func (s *Service) signup(ctx context.Context, email, password string) (*User, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}

	workCtx, cancel := context.WithTimeout(ctx, s.passwordTimeout)
	defer cancel()

	err := s.admitCredential(workCtx, email, password, CredentialRegistration)
	if err != nil {
		return nil, err
	}

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

func (s *Service) signin(ctx context.Context, email, password string, requirement AccessRequirement) (*AuthenticationResult, error) {
	err := requirement.Check(s.sessions.TTL)
	if err != nil {
		return nil, err
	}

	workCtx, cancel := context.WithTimeout(ctx, s.passwordTimeout)
	defer cancel()

	err = s.admitCredential(workCtx, email, password, CredentialPasswordProof)
	if err != nil {
		return nil, err
	}

	user, err := s.queries.GetUserByEmail(workCtx, email)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("cannot get user: %w", err)
	}

	securitySubject(workCtx, user.ID)

	if !user.Active {
		return nil, ErrUserInactive
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

	securityProof(workCtx, PasswordProof)

	if requirement.Proof != RequirePassword {
		return unmetRequirement(user, requirement.Proof), nil
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	token, digest, err := s.sessionToken()
	if err != nil {
		return nil, fmt.Errorf("cannot generate session secret: %w", err)
	}

	record := SessionRecord{Session: Session{ID: model.NewID(), UserID: state.UserID, AuthVersion: state.Version, PolicyRevision: requirement.Revision, Generation: 1, Proof: VerifiedProof{Method: PasswordProof, VerifiedAt: now}, AuthenticatedAt: now, CreatedAt: now, LastActivityAt: now, ExpiresAt: now.Add(s.sessions.TTL), InactivityTTL: s.sessions.InactivityTTL}, Digest: digest}

	sessionCtx, sessionCancel := context.WithTimeout(workCtx, s.sessions.Timeout)
	defer sessionCancel()

	session, err := s.queries.CreateSession(sessionCtx, state, record, requirement, s.sessions.MaxPerSubject)
	if err != nil {
		return nil, fmt.Errorf("cannot create session: %w", err)
	}

	err = sessionCtx.Err()
	if err != nil {
		return nil, err
	}

	s.log.Infof("User signed in: %s", user.ID)

	return &AuthenticationResult{Outcome: AuthenticationCompleted, Reason: AuthenticationSatisfied, Issued: &IssuedSession{Session: *session, Token: token}}, nil
}

func (s *Service) signout(ctx context.Context, token string) error {
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

	return workCtx.Err()
}

// ValidateSession checks current stored state and optionally records trusted
// relevant subscriber activity. NoActivity is required for background polling.
func (s *Service) ValidateSession(ctx context.Context, token string, requirement AccessRequirement, activity SessionActivity) (*ValidatedSession, error) {
	err := requirement.Check(s.sessions.TTL)
	if err != nil {
		return nil, err
	}

	if activity != NoActivity && activity != RelevantActivity {
		return nil, ErrSessionActivity
	}

	digest, err := ParseSessionToken(token)
	if err != nil {
		return nil, err
	}

	workCtx, cancel := context.WithTimeout(ctx, s.sessions.Timeout)
	defer cancel()

	session, err := s.queries.ValidateSession(workCtx, digest, requirement, activity, s.sessions.ActivityInterval)
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

	err = workCtx.Err()
	if err != nil {
		return 0, err
	}

	return count, nil
}
