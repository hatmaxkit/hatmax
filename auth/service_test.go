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

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

// mockQueries implements the Queries interface for testing.
type mockQueries struct {
	users    map[string]*User
	sessions map[SessionDigest]*SessionRecord
}

func newMockQueries() *mockQueries {
	return &mockQueries{
		users:    make(map[string]*User),
		sessions: make(map[SessionDigest]*SessionRecord),
	}
}

func (m *mockQueries) CreateUser(ctx context.Context, id, email, passwordHash string, createdAt, updatedAt time.Time) (*User, error) {
	// Check if email already exists
	for _, u := range m.users {
		if u.Email == email {
			return nil, ErrEmailTaken
		}
	}

	user := &User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		AuthVersion:  1,
		Active:       true,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
	m.users[id] = user

	snapshot := *user

	return &snapshot, nil
}

func (m *mockQueries) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	for _, u := range m.users {
		if u.Email == email {
			snapshot := *u

			return &snapshot, nil
		}
	}

	return nil, sql.ErrNoRows
}

func (m *mockQueries) GetUserByID(ctx context.Context, id string) (*User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}

	snapshot := *user

	return &snapshot, nil
}

func (m *mockQueries) CreateSession(ctx context.Context, state CredentialState, session SessionRecord) (*Session, error) {
	user := m.users[state.UserID]
	if user == nil || !user.Active || user.AuthVersion != state.Version {
		return nil, ErrCredentialChanged
	}

	session.UserID = state.UserID
	m.sessions[session.Digest] = &session

	snapshot := session.Session

	return &snapshot, nil
}

func (m *mockQueries) ReplacePassword(ctx context.Context, state CredentialState, hash string, changedAt time.Time) (*User, error) {
	user := m.users[state.UserID]
	if user == nil || !user.Active || user.AuthVersion != state.Version {
		return nil, ErrCredentialChanged
	}

	user.PasswordHash = hash
	user.AuthVersion++
	user.UpdatedAt = changedAt

	for token, session := range m.sessions {
		if session.UserID == state.UserID {
			delete(m.sessions, token)
		}
	}

	copy := *user

	return &copy, nil
}

func (m *mockQueries) ValidateSession(ctx context.Context, digest SessionDigest, activity SessionActivity, interval time.Duration) (*ValidatedSession, error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	session := m.sessions[digest]
	if session == nil {
		return nil, ErrSessionNotFound
	}

	user := m.users[session.UserID]
	if user == nil || !user.Active || user.AuthVersion != session.AuthVersion {
		return nil, ErrCredentialChanged
	}

	now := time.Now()

	err = session.Check(now)
	if err != nil {
		return nil, err
	}

	if activity == RelevantActivity && now.Sub(session.LastActivityAt) >= interval {
		session.LastActivityAt = now
	}

	snapshot := *user
	snapshot.Roles = append([]string(nil), user.Roles...)

	return &ValidatedSession{User: &snapshot, Session: session.Session}, nil
}

func (m *mockQueries) DeleteSession(ctx context.Context, digest SessionDigest) error {
	if m.sessions[digest] == nil {
		return ErrSessionNotFound
	}

	delete(m.sessions, digest)

	return nil
}

func (m *mockQueries) DeleteExpiredSessions(ctx context.Context, limit int) (int64, error) {
	var count int64
	for digest, session := range m.sessions {
		if errors.Is(session.Check(time.Now()), ErrSessionExpired) && count < int64(limit) {
			delete(m.sessions, digest)

			count++
		}
	}

	return count, nil
}

// expiredFixture gives service tests a structurally valid expired record.
func expiredFixture(t *testing.T, queries *mockQueries, user *User) string {
	t.Helper()

	token, digest, err := newSessionToken()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().Add(-2 * time.Hour)
	queries.sessions[digest] = &SessionRecord{Digest: digest, Session: Session{
		ID: "expired-id", UserID: user.ID, AuthVersion: user.AuthVersion, Generation: 1,
		AuthenticatedAt: now, CreatedAt: now, LastActivityAt: now, ExpiresAt: now.Add(time.Hour), InactivityTTL: time.Minute,
	}}

	return token
}

func TestSignup(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := newServiceForTest(t, queries, cfg, logger)

	tests := []struct {
		name      string
		email     string
		password  string
		wantErr   error
		setupFunc func()
	}{
		{
			name:     "valid signup",
			email:    "test@example.com",
			password: "correct-password-123",
			wantErr:  nil,
		},
		{
			name:     "empty email",
			email:    "",
			password: "correct-password-123",
			wantErr:  ErrInvalidEmail,
		},
		{
			name:     "password too short",
			email:    "test2@example.com",
			password: "short",
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "email already taken",
			email:    "existing@example.com",
			password: "correct-password-123",
			wantErr:  ErrEmailTaken,
			setupFunc: func() {
				queries.CreateUser(context.Background(), "existing-id", "existing@example.com", "hash", time.Now(), time.Now())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			user, err := svc.Signup(context.Background(), tt.email, tt.password)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Signup() error = nil, wantErr %v", tt.wantErr)

					return
				}

				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("Signup() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("Signup() unexpected error = %v", err)

				return
			}

			if user == nil {
				t.Error("Signup() returned nil user")

				return
			}

			if user.Email != tt.email {
				t.Errorf("Signup() user.Email = %v, want %v", user.Email, tt.email)
			}

			if !user.Active {
				t.Error("Signup() user.Active = false, want true")
			}
		})
	}
}

func TestSignin(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := newServiceForTest(t, queries, cfg, logger)

	// Create a test user
	user, _ := svc.Signup(context.Background(), "test@example.com", "correct-password-123")

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{
			name:     "valid signin",
			email:    "test@example.com",
			password: "correct-password-123",
			wantErr:  nil,
		},
		{
			name:     "user not found",
			email:    "notfound@example.com",
			password: "correct-password-123",
			wantErr:  ErrUserNotFound,
		},
		{
			name:     "invalid password",
			email:    "test@example.com",
			password: "wrongpassword",
			wantErr:  ErrInvalidPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := svc.Signin(context.Background(), tt.email, tt.password)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Signin() error = nil, wantErr %v", tt.wantErr)

					return
				}

				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("Signin() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("Signin() unexpected error = %v", err)

				return
			}

			if session == nil {
				t.Error("Signin() returned nil session")

				return
			}

			if session.UserID != user.ID {
				t.Errorf("Signin() session.UserID = %v, want %v", session.UserID, user.ID)
			}

			if session.Token == "" {
				t.Error("Signin() session.Token is empty")
			}
		})
	}
}

func TestSigninInactiveUser(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := newServiceForTest(t, queries, cfg, logger)

	// Create a test user and mark as inactive
	user, _ := svc.Signup(context.Background(), "inactive@example.com", "correct-password-123")
	user.Active = false
	queries.users[user.ID] = user

	_, err := svc.Signin(context.Background(), "inactive@example.com", "correct-password-123")
	if err == nil {
		t.Error("Signin() with inactive user should return error")
	}
}

func TestSignout(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := newServiceForTest(t, queries, cfg, logger)

	// Create a test user and session
	_, _ = svc.Signup(context.Background(), "test@example.com", "correct-password-123")
	session, _ := svc.Signin(context.Background(), "test@example.com", "correct-password-123")

	tests := []struct {
		name         string
		sessionToken string
		wantErr      error
	}{
		{
			name:         "valid signout",
			sessionToken: session.Token,
			wantErr:      nil,
		},
		{
			name:         "session not found",
			sessionToken: "invalid-token",
			wantErr:      ErrSessionToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Signout(context.Background(), tt.sessionToken)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Signout() error = nil, wantErr %v", tt.wantErr)

					return
				}

				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("Signout() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("Signout() unexpected error = %v", err)
			}

			// Verify session was deleted
			_, err = svc.ValidateSession(context.Background(), session.Token, NoActivity)
			if !errors.Is(err, ErrSessionNotFound) {
				t.Error("Signout() session was not deleted")
			}
		})
	}
}

func TestValidateSession(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := newServiceForTest(t, queries, cfg, logger)

	// Create a test user and session
	user, _ := svc.Signup(context.Background(), "test@example.com", "correct-password-123")
	session, _ := svc.Signin(context.Background(), "test@example.com", "correct-password-123")

	expiredToken := expiredFixture(t, queries, user)

	tests := []struct {
		name  string
		token string
		want  *User
		err   error
	}{
		{
			name:  "valid session",
			token: session.Token,
			want:  user,
			err:   nil,
		},
		{
			name:  "session not found",
			token: "invalid-token",
			want:  nil,
			err:   ErrSessionToken,
		},
		{
			name:  "expired session",
			token: expiredToken,
			want:  nil,
			err:   ErrSessionExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.ValidateSession(context.Background(), tt.token, NoActivity)

			if tt.err != nil {
				if err == nil {
					t.Errorf("ValidateSession() error = nil, wantErr %v", tt.err)

					return
				}

				if !errors.Is(err, tt.err) && err.Error() != tt.err.Error() {
					t.Errorf("ValidateSession() error = %v, wantErr %v", err, tt.err)
				}

				return
			}

			if err != nil {
				t.Errorf("ValidateSession() unexpected error = %v", err)

				return
			}

			if got.User.ID != tt.want.ID {
				t.Errorf("ValidateSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateSessionInactiveUser(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := newServiceForTest(t, queries, cfg, logger)

	// Create a test user and session
	user, _ := svc.Signup(context.Background(), "test@example.com", "correct-password-123")
	session, _ := svc.Signin(context.Background(), "test@example.com", "correct-password-123")

	// Mark user as inactive
	user.Active = false
	queries.users[user.ID] = user

	_, err := svc.ValidateSession(context.Background(), session.Token, NoActivity)
	if err == nil {
		t.Error("ValidateSession() with inactive user should return error")
	}
}

func TestGetUserByID(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := newServiceForTest(t, queries, cfg, logger)

	// Create a test user
	user, _ := svc.Signup(context.Background(), "test@example.com", "correct-password-123")

	tests := []struct {
		name    string
		userID  string
		want    *User
		wantErr error
	}{
		{
			name:    "valid user ID",
			userID:  user.ID,
			want:    user,
			wantErr: nil,
		},
		{
			name:    "user not found",
			userID:  "invalid-id",
			want:    nil,
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetUserByID(context.Background(), tt.userID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("GetUserByID() error = nil, wantErr %v", tt.wantErr)

					return
				}

				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("GetUserByID() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("GetUserByID() unexpected error = %v", err)

				return
			}

			if got.ID != tt.want.ID {
				t.Errorf("GetUserByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := newServiceForTest(t, queries, cfg, logger)

	// Create a test user
	user, _ := svc.Signup(context.Background(), "test@example.com", "correct-password-123")

	// Create a valid session
	validSession, _ := svc.Signin(context.Background(), "test@example.com", "correct-password-123")

	expiredToken := expiredFixture(t, queries, user)

	// Cleanup expired sessions
	_, err := svc.CleanupExpiredSessions(context.Background())
	if err != nil {
		t.Errorf("CleanupExpiredSessions() unexpected error = %v", err)
	}

	// Verify expired session was deleted
	_, err = svc.ValidateSession(context.Background(), expiredToken, NoActivity)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Error("CleanupExpiredSessions() expired session was not deleted")
	}

	// Verify valid session still exists
	_, err = svc.ValidateSession(context.Background(), validSession.Token, NoActivity)
	if err != nil {
		t.Error("CleanupExpiredSessions() valid session was deleted")
	}
}
