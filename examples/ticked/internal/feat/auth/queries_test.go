package auth

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/hatmaxkit/hatmax/auth"
	"github.com/hatmaxkit/hatmax/examples/ticked/internal/dal"
)

// fakeQuerier implements the querier interface for testing.
type fakeQuerier struct {
	countUsers      int64
	countErr        error
	createUser      dal.User
	createUserErr   error
	getUserByEmail  dal.User
	getUserEmailErr error
	getUserByID     dal.User
	getUserIDErr    error
	createSession   dal.Session
	createSessionErr error
	getSessionByToken dal.Session
	getSessionErr   error
	deleteSessionErr error
	deleteExpiredErr error
	listUsers       []dal.User
	listUsersErr    error
	updateRolesErr  error
	updateActiveErr error
}

func (f *fakeQuerier) CountUsers(ctx context.Context) (int64, error) {
	return f.countUsers, f.countErr
}

func (f *fakeQuerier) CreateUser(ctx context.Context, arg dal.CreateUserParams) (dal.User, error) {
	return f.createUser, f.createUserErr
}

func (f *fakeQuerier) GetUserByEmail(ctx context.Context, email string) (dal.User, error) {
	return f.getUserByEmail, f.getUserEmailErr
}

func (f *fakeQuerier) GetUserByID(ctx context.Context, id string) (dal.User, error) {
	return f.getUserByID, f.getUserIDErr
}

func (f *fakeQuerier) CreateSession(ctx context.Context, arg dal.CreateSessionParams) (dal.Session, error) {
	return f.createSession, f.createSessionErr
}

func (f *fakeQuerier) GetSessionByToken(ctx context.Context, token string) (dal.Session, error) {
	return f.getSessionByToken, f.getSessionErr
}

func (f *fakeQuerier) DeleteSession(ctx context.Context, id string) error {
	return f.deleteSessionErr
}

func (f *fakeQuerier) DeleteExpiredSessions(ctx context.Context) error {
	return f.deleteExpiredErr
}

func (f *fakeQuerier) ListUsers(ctx context.Context) ([]dal.User, error) {
	return f.listUsers, f.listUsersErr
}

func (f *fakeQuerier) UpdateUserRoles(ctx context.Context, arg dal.UpdateUserRolesParams) error {
	return f.updateRolesErr
}

func (f *fakeQuerier) UpdateUserActive(ctx context.Context, arg dal.UpdateUserActiveParams) error {
	return f.updateActiveErr
}

func TestQueries_CreateUser(t *testing.T) {
	now := time.Now()
	fq := &fakeQuerier{
		createUser: dal.User{
			ID:           "user1",
			Email:        "test@example.com",
			PasswordHash: "hash",
			Roles:        []string{},
			Active:       true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	q := &Queries{}
	q.SetQuerier(fq)

	user, err := q.CreateUser(context.Background(), "user1", "test@example.com", "hash", now, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID != "user1" {
		t.Errorf("expected ID user1, got %s", user.ID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
}

func TestQueries_CreateUser_Error(t *testing.T) {
	fq := &fakeQuerier{
		createUserErr: errors.New("duplicate email"),
	}

	q := &Queries{}
	q.SetQuerier(fq)

	_, err := q.CreateUser(context.Background(), "user1", "test@example.com", "hash", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestQueries_GetUserByEmail(t *testing.T) {
	now := time.Now()
	fq := &fakeQuerier{
		getUserByEmail: dal.User{
			ID:           "user1",
			Email:        "test@example.com",
			PasswordHash: "hash",
			Roles:        []string{"admin"},
			Active:       true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	q := &Queries{}
	q.SetQuerier(fq)

	user, err := q.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
}

func TestQueries_GetUserByEmail_Error(t *testing.T) {
	fq := &fakeQuerier{
		getUserEmailErr: sql.ErrNoRows,
	}

	q := &Queries{}
	q.SetQuerier(fq)

	_, err := q.GetUserByEmail(context.Background(), "notfound@example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestQueries_GetUserByID(t *testing.T) {
	now := time.Now()
	fq := &fakeQuerier{
		getUserByID: dal.User{
			ID:           "user1",
			Email:        "test@example.com",
			PasswordHash: "hash",
			Roles:        []string{},
			Active:       true,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	q := &Queries{}
	q.SetQuerier(fq)

	user, err := q.GetUserByID(context.Background(), "user1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID != "user1" {
		t.Errorf("expected ID user1, got %s", user.ID)
	}
}

func TestQueries_GetUserByID_Error(t *testing.T) {
	fq := &fakeQuerier{
		getUserIDErr: sql.ErrNoRows,
	}

	q := &Queries{}
	q.SetQuerier(fq)

	_, err := q.GetUserByID(context.Background(), "notfound")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestQueries_CreateSession(t *testing.T) {
	now := time.Now()
	expires := now.Add(24 * time.Hour)
	fq := &fakeQuerier{
		createSession: dal.Session{
			ID:        "sess1",
			UserID:    "user1",
			Token:     "token123",
			ExpiresAt: expires,
			CreatedAt: now,
		},
	}

	q := &Queries{}
	q.SetQuerier(fq)

	session, err := q.CreateSession(context.Background(), "sess1", "user1", "token123", expires, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if session.Token != "token123" {
		t.Errorf("expected token token123, got %s", session.Token)
	}
}

func TestQueries_CreateSession_Error(t *testing.T) {
	fq := &fakeQuerier{
		createSessionErr: errors.New("db error"),
	}

	q := &Queries{}
	q.SetQuerier(fq)

	_, err := q.CreateSession(context.Background(), "sess1", "user1", "token", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestQueries_GetSessionByToken(t *testing.T) {
	now := time.Now()
	fq := &fakeQuerier{
		getSessionByToken: dal.Session{
			ID:        "sess1",
			UserID:    "user1",
			Token:     "token123",
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now,
		},
	}

	q := &Queries{}
	q.SetQuerier(fq)

	session, err := q.GetSessionByToken(context.Background(), "token123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if session.Token != "token123" {
		t.Errorf("expected token token123, got %s", session.Token)
	}
}

func TestQueries_GetSessionByToken_Error(t *testing.T) {
	fq := &fakeQuerier{
		getSessionErr: sql.ErrNoRows,
	}

	q := &Queries{}
	q.SetQuerier(fq)

	_, err := q.GetSessionByToken(context.Background(), "invalid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestQueries_DeleteSession(t *testing.T) {
	fq := &fakeQuerier{}

	q := &Queries{}
	q.SetQuerier(fq)

	err := q.DeleteSession(context.Background(), "sess1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestQueries_DeleteSession_Error(t *testing.T) {
	fq := &fakeQuerier{
		deleteSessionErr: errors.New("db error"),
	}

	q := &Queries{}
	q.SetQuerier(fq)

	err := q.DeleteSession(context.Background(), "sess1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestQueries_DeleteExpiredSessions(t *testing.T) {
	fq := &fakeQuerier{}

	q := &Queries{}
	q.SetQuerier(fq)

	err := q.DeleteExpiredSessions(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestQueries_DeleteExpiredSessions_Error(t *testing.T) {
	fq := &fakeQuerier{
		deleteExpiredErr: errors.New("db error"),
	}

	q := &Queries{}
	q.SetQuerier(fq)

	err := q.DeleteExpiredSessions(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestQueries_ListUsers(t *testing.T) {
	now := time.Now()
	fq := &fakeQuerier{
		listUsers: []dal.User{
			{ID: "user1", Email: "user1@example.com", CreatedAt: now, UpdatedAt: now},
			{ID: "user2", Email: "user2@example.com", CreatedAt: now, UpdatedAt: now},
		},
	}

	q := &Queries{}
	q.SetQuerier(fq)

	users, err := q.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestQueries_ListUsers_Error(t *testing.T) {
	fq := &fakeQuerier{
		listUsersErr: errors.New("db error"),
	}

	q := &Queries{}
	q.SetQuerier(fq)

	_, err := q.ListUsers(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestQueries_UpdateUserRoles(t *testing.T) {
	fq := &fakeQuerier{}

	q := &Queries{}
	q.SetQuerier(fq)

	err := q.UpdateUserRoles(context.Background(), "user1", []string{"admin"}, time.Now())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestQueries_UpdateUserRoles_Error(t *testing.T) {
	fq := &fakeQuerier{
		updateRolesErr: errors.New("db error"),
	}

	q := &Queries{}
	q.SetQuerier(fq)

	err := q.UpdateUserRoles(context.Background(), "user1", []string{"admin"}, time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestQueries_UpdateUserActive(t *testing.T) {
	fq := &fakeQuerier{}

	q := &Queries{}
	q.SetQuerier(fq)

	err := q.UpdateUserActive(context.Background(), "user1", false, time.Now())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestQueries_UpdateUserActive_Error(t *testing.T) {
	fq := &fakeQuerier{
		updateActiveErr: errors.New("db error"),
	}

	q := &Queries{}
	q.SetQuerier(fq)

	err := q.UpdateUserActive(context.Background(), "user1", false, time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestQueries_CountUsers(t *testing.T) {
	fq := &fakeQuerier{countUsers: 10}

	q := &Queries{}
	q.SetQuerier(fq)

	count, err := q.CountUsers(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if count != 10 {
		t.Errorf("expected count 10, got %d", count)
	}
}

func TestQueries_CountUsers_Error(t *testing.T) {
	fq := &fakeQuerier{countErr: errors.New("db error")}

	q := &Queries{}
	q.SetQuerier(fq)

	_, err := q.CountUsers(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewQueries(t *testing.T) {
	// NewQueries needs a real dal.Queries but we can't easily create one without a DB
	// This test just verifies the function exists and can be called
	_ = NewQueries(nil) // Will panic if used, but tests the constructor exists
}

// Helper to verify toAuthUser conversion
func TestToAuthUser(t *testing.T) {
	now := time.Now()
	dalUser := dal.User{
		ID:           "user1",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Roles:        []string{"admin", "user"},
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	authUser := toAuthUser(dalUser)

	if authUser.ID != dalUser.ID {
		t.Errorf("expected ID %s, got %s", dalUser.ID, authUser.ID)
	}
	if authUser.Email != dalUser.Email {
		t.Errorf("expected Email %s, got %s", dalUser.Email, authUser.Email)
	}
	if authUser.PasswordHash != dalUser.PasswordHash {
		t.Errorf("expected PasswordHash %s, got %s", dalUser.PasswordHash, authUser.PasswordHash)
	}
	if len(authUser.Roles) != len(dalUser.Roles) {
		t.Errorf("expected %d roles, got %d", len(dalUser.Roles), len(authUser.Roles))
	}
	if authUser.Active != dalUser.Active {
		t.Errorf("expected Active %v, got %v", dalUser.Active, authUser.Active)
	}
}

func TestToAuthSession(t *testing.T) {
	now := time.Now()
	expires := now.Add(24 * time.Hour)
	dalSession := dal.Session{
		ID:        "sess1",
		UserID:    "user1",
		Token:     "token123",
		ExpiresAt: expires,
		CreatedAt: now,
	}

	authSession := toAuthSession(dalSession)

	if authSession.ID != dalSession.ID {
		t.Errorf("expected ID %s, got %s", dalSession.ID, authSession.ID)
	}
	if authSession.UserID != dalSession.UserID {
		t.Errorf("expected UserID %s, got %s", dalSession.UserID, authSession.UserID)
	}
	if authSession.Token != dalSession.Token {
		t.Errorf("expected Token %s, got %s", dalSession.Token, authSession.Token)
	}
}

// Test that the Queries implements the auth.Queries interface
func TestQueries_ImplementsAuthQueries(t *testing.T) {
	var _ auth.Queries = (*Queries)(nil)
}
