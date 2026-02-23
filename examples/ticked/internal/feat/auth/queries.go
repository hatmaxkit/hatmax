package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hatmaxkit/hatmax/auth"
	"github.com/hatmaxkit/hatmax/examples/ticked/internal/dal"
)

// querier defines the interface for sqlc operations needed by auth.
type querier interface {
	CountUsers(ctx context.Context) (int64, error)
	CreateSession(ctx context.Context, arg dal.CreateSessionParams) (dal.Session, error)
	CreateUser(ctx context.Context, arg dal.CreateUserParams) (dal.User, error)
	DeleteExpiredSessions(ctx context.Context) error
	DeleteSession(ctx context.Context, id string) error
	GetSessionByToken(ctx context.Context, token string) (dal.Session, error)
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

// CreateSession creates a new session and returns it.
func (q *Queries) CreateSession(ctx context.Context, id, userID, token string, expiresAt, createdAt time.Time) (*auth.Session, error) {
	session, err := q.q.CreateSession(ctx, dal.CreateSessionParams{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	})
	if err != nil {
		return nil, err
	}

	return toAuthSession(session), nil
}

// GetSessionByToken retrieves a session by token.
func (q *Queries) GetSessionByToken(ctx context.Context, token string) (*auth.Session, error) {
	session, err := q.q.GetSessionByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return toAuthSession(session), nil
}

// DeleteSession deletes a session by ID.
func (q *Queries) DeleteSession(ctx context.Context, sessionID string) error {
	return q.q.DeleteSession(ctx, sessionID)
}

// DeleteExpiredSessions removes all expired sessions.
func (q *Queries) DeleteExpiredSessions(ctx context.Context) error {
	return q.q.DeleteExpiredSessions(ctx)
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
		Roles:        u.Roles,
		Active:       u.Active,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func toAuthSession(s dal.Session) *auth.Session {
	return &auth.Session{
		ID:        s.ID,
		UserID:    s.UserID,
		Token:     s.Token,
		ExpiresAt: s.ExpiresAt,
		CreatedAt: s.CreatedAt,
	}
}
