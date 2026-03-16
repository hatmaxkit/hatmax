package auth

import (
	"context"
	"time"

	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/log"
)

// baseAuthService defines the operations needed from hatmax auth.Service.
type baseAuthService interface {
	Signup(ctx context.Context, email, password string) (*auth.User, error)
	Signin(ctx context.Context, email, password string) (*auth.Session, error)
	Signout(ctx context.Context, sessionToken string) error
	ValidateSession(ctx context.Context, token string) (*auth.User, error)
}

// userCounter defines the count users operation for first-user detection.
type userCounter interface {
	CountUsers(ctx context.Context) (int64, error)
	UpdateUserRoles(ctx context.Context, id string, roles []string, updatedAt time.Time) error
}

// Service provides ticked-specific authentication logic.
type Service struct {
	auth    baseAuthService
	counter userCounter
	log     log.Logger
}

// NewService creates a new ticked auth service.
func NewService(authSvc *auth.Service, queries *Queries, log log.Logger) *Service {
	return &Service{
		auth:    authSvc,
		counter: queries,
		log:     log,
	}
}

// Signup creates a new user. First user is promoted to superadmin.
func (s *Service) Signup(ctx context.Context, email, password string) (*auth.User, error) {
	count, err := s.counter.CountUsers(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.auth.Signup(ctx, email, password)
	if err != nil {
		return nil, err
	}

	// First user becomes superadmin
	if count == 0 {
		err = s.counter.UpdateUserRoles(ctx, user.ID, []string{"superadmin"}, time.Now())
		if err != nil {
			s.log.Errorf("failed to promote first user to superadmin: %v", err)
		} else {
			s.log.Infof("First user %s promoted to superadmin", email)

			user.Roles = []string{"superadmin"}
		}
	}

	return user, nil
}

// Signin delegates to the underlying auth service.
func (s *Service) Signin(ctx context.Context, email, password string) (*auth.Session, error) {
	return s.auth.Signin(ctx, email, password)
}

// Signout delegates to the underlying auth service.
func (s *Service) Signout(ctx context.Context, sessionToken string) error {
	return s.auth.Signout(ctx, sessionToken)
}

// ValidateSession delegates to the underlying auth service.
func (s *Service) ValidateSession(ctx context.Context, token string) (*auth.User, error) {
	return s.auth.ValidateSession(ctx, token)
}
