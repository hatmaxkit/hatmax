package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hatmaxkit/hatmax/config"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/model"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrEmailTaken       = errors.New("email already taken")
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionExpired   = errors.New("session expired")
	ErrPasswordTooShort = errors.New("password too short")
	ErrInvalidEmail     = errors.New("invalid email")
)

// Queries defines the interface for auth database operations.
// Users should implement this interface using sqlc-generated code.
type Queries interface {
	CreateUser(ctx context.Context, id, email, passwordHash string, createdAt, updatedAt time.Time) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	CreateSession(ctx context.Context, id, userID, token string, expiresAt, createdAt time.Time) (*Session, error)
	GetSessionByToken(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteExpiredSessions(ctx context.Context) error
}

// Service provides authentication functionality.
type Service struct {
	queries Queries
	cfg     *config.Config
	log     logger.Logger
}

// NewService creates a new auth service.
func NewService(queries Queries, cfg *config.Config, log logger.Logger) *Service {
	return &Service{
		queries: queries,
		cfg:     cfg,
		log:     log,
	}
}

// Signup creates a new user account.
func (s *Service) Signup(ctx context.Context, email, password string) (*User, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}

	if len(password) < s.cfg.Auth.PasswordMinLen {
		return nil, ErrPasswordTooShort
	}

	// Check if email is already taken
	_, err := s.queries.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailTaken
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("cannot check email: %w", err)
	}

	// Hash password
	passwordHash, err := model.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("cannot hash password: %w", err)
	}

	// Create user
	now := model.Now()
	user, err := s.queries.CreateUser(ctx, model.NewID(), email, passwordHash, now, now)
	if err != nil {
		return nil, fmt.Errorf("cannot create user: %w", err)
	}

	s.log.Infof("User signed up: %s", user.ID)

	return user, nil
}

// Signin validates credentials and creates a session.
func (s *Service) Signin(ctx context.Context, email, password string) (*Session, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("cannot get user: %w", err)
	}

	if !user.Active {
		return nil, errors.New("user is not active")
	}

	// Verify password
	if !model.ComparePassword(user.PasswordHash, password) {
		return nil, ErrInvalidPassword
	}

	// Parse session TTL
	ttl, err := time.ParseDuration(s.cfg.Auth.SessionTTL)
	if err != nil {
		ttl = 24 * time.Hour // fallback to 24 hours
	}

	// Create session
	now := model.Now()
	session, err := s.queries.CreateSession(
		ctx,
		model.NewID(),
		user.ID,
		model.NewID(), // token
		now.Add(ttl),
		now,
	)
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

	if err := s.queries.DeleteSession(ctx, session.ID); err != nil {
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
	if err := s.queries.DeleteExpiredSessions(ctx); err != nil {
		return fmt.Errorf("cannot cleanup expired sessions: %w", err)
	}
	return nil
}
