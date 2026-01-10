package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hatmaxkit/hatmax/auth"
	"github.com/hatmaxkit/hatmax/log"
)

// fakeBaseAuth is a fake for baseAuthService.
type fakeBaseAuth struct {
	user        *auth.User
	session     *auth.Session
	signupErr   error
	signinErr   error
	signoutErr  error
	validateErr error
}

func (f *fakeBaseAuth) Signup(ctx context.Context, email, password string) (*auth.User, error) {
	return f.user, f.signupErr
}

func (f *fakeBaseAuth) Signin(ctx context.Context, email, password string) (*auth.Session, error) {
	return f.session, f.signinErr
}

func (f *fakeBaseAuth) Signout(ctx context.Context, sessionToken string) error {
	return f.signoutErr
}

func (f *fakeBaseAuth) ValidateSession(ctx context.Context, token string) (*auth.User, error) {
	return f.user, f.validateErr
}

// fakeCounter is a fake for userCounter.
type fakeCounter struct {
	countUsers   int64
	countErr     error
	updateErr    error
	updatedRoles []string
}

func (f *fakeCounter) CountUsers(ctx context.Context) (int64, error) {
	return f.countUsers, f.countErr
}

func (f *fakeCounter) UpdateUserRoles(ctx context.Context, id string, roles []string, updatedAt time.Time) error {
	f.updatedRoles = roles
	return f.updateErr
}

func TestService_Signup_FirstUser(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{
		user: &auth.User{ID: "user1", Email: "first@example.com"},
	}
	counter := &fakeCounter{countUsers: 0} // first user

	svc := &Service{
		auth:    baseAuth,
		counter: counter,
		log:     logger,
	}

	user, err := svc.Signup(context.Background(), "first@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID != "user1" {
		t.Errorf("expected user ID user1, got %s", user.ID)
	}

	// First user should have superadmin role
	if len(counter.updatedRoles) != 1 || counter.updatedRoles[0] != "superadmin" {
		t.Errorf("expected superadmin role, got %v", counter.updatedRoles)
	}

	// User object should have updated roles
	if len(user.Roles) != 1 || user.Roles[0] != "superadmin" {
		t.Errorf("expected user to have superadmin role, got %v", user.Roles)
	}
}

func TestService_Signup_NotFirstUser(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{
		user: &auth.User{ID: "user2", Email: "second@example.com"},
	}
	counter := &fakeCounter{countUsers: 5} // not first user

	svc := &Service{
		auth:    baseAuth,
		counter: counter,
		log:     logger,
	}

	user, err := svc.Signup(context.Background(), "second@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID != "user2" {
		t.Errorf("expected user ID user2, got %s", user.ID)
	}

	// Non-first user should NOT have roles updated
	if counter.updatedRoles != nil {
		t.Errorf("expected no role update for non-first user, got %v", counter.updatedRoles)
	}
}

func TestService_Signup_CountError(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{
		user: &auth.User{ID: "user1", Email: "test@example.com"},
	}
	counter := &fakeCounter{countErr: errors.New("db error")}

	svc := &Service{
		auth:    baseAuth,
		counter: counter,
		log:     logger,
	}

	_, err := svc.Signup(context.Background(), "test@example.com", "password123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_Signup_BaseAuthError(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{
		signupErr: errors.New("signup failed"),
	}
	counter := &fakeCounter{countUsers: 0}

	svc := &Service{
		auth:    baseAuth,
		counter: counter,
		log:     logger,
	}

	_, err := svc.Signup(context.Background(), "test@example.com", "password123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_Signup_UpdateRolesError(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{
		user: &auth.User{ID: "user1", Email: "first@example.com"},
	}
	counter := &fakeCounter{countUsers: 0, updateErr: errors.New("update failed")}

	svc := &Service{
		auth:    baseAuth,
		counter: counter,
		log:     logger,
	}

	// Should not return error, but log it
	user, err := svc.Signup(context.Background(), "first@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// User should be returned without superadmin role
	if user.ID != "user1" {
		t.Errorf("expected user ID user1, got %s", user.ID)
	}
}

func TestService_Signin(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{
		session: &auth.Session{ID: "sess1", Token: "token123"},
	}

	svc := &Service{
		auth:    baseAuth,
		counter: &fakeCounter{},
		log:     logger,
	}

	session, err := svc.Signin(context.Background(), "test@example.com", "password")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if session.Token != "token123" {
		t.Errorf("expected token token123, got %s", session.Token)
	}
}

func TestService_Signin_Error(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{
		signinErr: errors.New("invalid credentials"),
	}

	svc := &Service{
		auth:    baseAuth,
		counter: &fakeCounter{},
		log:     logger,
	}

	_, err := svc.Signin(context.Background(), "test@example.com", "wrong")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_Signout(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{}

	svc := &Service{
		auth:    baseAuth,
		counter: &fakeCounter{},
		log:     logger,
	}

	err := svc.Signout(context.Background(), "token123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestService_Signout_Error(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{
		signoutErr: errors.New("signout failed"),
	}

	svc := &Service{
		auth:    baseAuth,
		counter: &fakeCounter{},
		log:     logger,
	}

	err := svc.Signout(context.Background(), "token123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_ValidateSession(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{
		user: &auth.User{ID: "user1", Email: "test@example.com"},
	}

	svc := &Service{
		auth:    baseAuth,
		counter: &fakeCounter{},
		log:     logger,
	}

	user, err := svc.ValidateSession(context.Background(), "token123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID != "user1" {
		t.Errorf("expected user ID user1, got %s", user.ID)
	}
}

func TestService_ValidateSession_Error(t *testing.T) {
	logger := log.NewTestLogger("error")

	baseAuth := &fakeBaseAuth{
		validateErr: errors.New("invalid session"),
	}

	svc := &Service{
		auth:    baseAuth,
		counter: &fakeCounter{},
		log:     logger,
	}

	_, err := svc.ValidateSession(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
