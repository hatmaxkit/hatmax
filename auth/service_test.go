package auth

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/hatmaxkit/hatmax/config"
	"github.com/hatmaxkit/hatmax/log"
)

// mockQueries implements the Queries interface for testing.
type mockQueries struct {
	users    map[string]*User
	sessions map[string]*Session
}

func newMockQueries() *mockQueries {
	return &mockQueries{
		users:    make(map[string]*User),
		sessions: make(map[string]*Session),
	}
}

func (m *mockQueries) CreateUser(ctx context.Context, id, email, passwordHash string, createdAt, updatedAt time.Time) (*User, error) {
	// Check if email already exists
	for _, u := range m.users {
		if u.Email == email {
			return nil, errors.New("email already exists")
		}
	}

	user := &User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Active:       true,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
	m.users[id] = user

	return user, nil
}

func (m *mockQueries) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}

	return nil, sql.ErrNoRows
}

func (m *mockQueries) GetUserByID(ctx context.Context, id string) (*User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}

	return user, nil
}

func (m *mockQueries) CreateSession(ctx context.Context, id, userID, token string, expiresAt, createdAt time.Time) (*Session, error) {
	session := &Session{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	}
	m.sessions[token] = session

	return session, nil
}

func (m *mockQueries) GetSessionByToken(ctx context.Context, token string) (*Session, error) {
	session, ok := m.sessions[token]
	if !ok {
		return nil, sql.ErrNoRows
	}

	return session, nil
}

func (m *mockQueries) DeleteSession(ctx context.Context, sessionID string) error {
	for token, s := range m.sessions {
		if s.ID == sessionID {
			delete(m.sessions, token)

			return nil
		}
	}

	return sql.ErrNoRows
}

func (m *mockQueries) DeleteExpiredSessions(ctx context.Context) error {
	now := time.Now()
	for token, s := range m.sessions {
		if s.ExpiresAt.Before(now) {
			delete(m.sessions, token)
		}
	}

	return nil
}

func TestSignup(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := NewService(queries, cfg, logger)

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
			password: "password123",
			wantErr:  nil,
		},
		{
			name:     "empty email",
			email:    "",
			password: "password123",
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
			password: "password123",
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
	svc := NewService(queries, cfg, logger)

	// Create a test user
	user, _ := svc.Signup(context.Background(), "test@example.com", "password123")

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{
			name:     "valid signin",
			email:    "test@example.com",
			password: "password123",
			wantErr:  nil,
		},
		{
			name:     "user not found",
			email:    "notfound@example.com",
			password: "password123",
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
	svc := NewService(queries, cfg, logger)

	// Create a test user and mark as inactive
	user, _ := svc.Signup(context.Background(), "inactive@example.com", "password123")
	user.Active = false
	queries.users[user.ID] = user

	_, err := svc.Signin(context.Background(), "inactive@example.com", "password123")
	if err == nil {
		t.Error("Signin() with inactive user should return error")
	}
}

func TestSignout(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := NewService(queries, cfg, logger)

	// Create a test user and session
	_, _ = svc.Signup(context.Background(), "test@example.com", "password123")
	session, _ := svc.Signin(context.Background(), "test@example.com", "password123")

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
			wantErr:      ErrSessionNotFound,
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
			_, err = queries.GetSessionByToken(context.Background(), session.Token)
			if err != sql.ErrNoRows {
				t.Error("Signout() session was not deleted")
			}
		})
	}
}

func TestValidateSession(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := NewService(queries, cfg, logger)

	// Create a test user and session
	user, _ := svc.Signup(context.Background(), "test@example.com", "password123")
	session, _ := svc.Signin(context.Background(), "test@example.com", "password123")

	// Create an expired session
	expiredSession := &Session{
		ID:        "expired-id",
		UserID:    user.ID,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	queries.sessions[expiredSession.Token] = expiredSession

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
			err:   ErrSessionNotFound,
		},
		{
			name:  "expired session",
			token: expiredSession.Token,
			want:  nil,
			err:   ErrSessionExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.ValidateSession(context.Background(), tt.token)

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

			if got.ID != tt.want.ID {
				t.Errorf("ValidateSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateSessionInactiveUser(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := NewService(queries, cfg, logger)

	// Create a test user and session
	user, _ := svc.Signup(context.Background(), "test@example.com", "password123")
	session, _ := svc.Signin(context.Background(), "test@example.com", "password123")

	// Mark user as inactive
	user.Active = false
	queries.users[user.ID] = user

	_, err := svc.ValidateSession(context.Background(), session.Token)
	if err == nil {
		t.Error("ValidateSession() with inactive user should return error")
	}
}

func TestGetUserByID(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := NewService(queries, cfg, logger)

	// Create a test user
	user, _ := svc.Signup(context.Background(), "test@example.com", "password123")

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
	svc := NewService(queries, cfg, logger)

	// Create a test user
	user, _ := svc.Signup(context.Background(), "test@example.com", "password123")

	// Create a valid session
	validSession, _ := svc.Signin(context.Background(), "test@example.com", "password123")

	// Create an expired session
	expiredSession := &Session{
		ID:        "expired-id",
		UserID:    user.ID,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	queries.sessions[expiredSession.Token] = expiredSession

	// Cleanup expired sessions
	err := svc.CleanupExpiredSessions(context.Background())
	if err != nil {
		t.Errorf("CleanupExpiredSessions() unexpected error = %v", err)
	}

	// Verify expired session was deleted
	_, err = queries.GetSessionByToken(context.Background(), expiredSession.Token)
	if err != sql.ErrNoRows {
		t.Error("CleanupExpiredSessions() expired session was not deleted")
	}

	// Verify valid session still exists
	_, err = queries.GetSessionByToken(context.Background(), validSession.Token)
	if err != nil {
		t.Error("CleanupExpiredSessions() valid session was deleted")
	}
}
