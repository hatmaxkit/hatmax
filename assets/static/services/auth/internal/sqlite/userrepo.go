package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/username/repo/services/auth/internal/auth"
	"github.com/username/repo/services/auth/internal/config"
	authpkg "github.com/username/repo/pkg/lib/auth"
)

// UserSQLiteRepo implements the UserRepo interface using SQLite.
type UserSQLiteRepo struct {
	db      *sql.DB
	xparams config.XParams
}

// NewUserSQLiteRepo creates a new SQLite repository for User entities.
func NewUserSQLiteRepo(xparams config.XParams) *UserSQLiteRepo {
	return &UserSQLiteRepo{
		xparams: xparams,
	}
}

// Start opens the database connection and pings it.
func (r *UserSQLiteRepo) Start(ctx context.Context) error {
	appCfg := r.xparams.Cfg

	dbPath := appCfg.Database.Path

	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", dbPath))
	if err != nil {
		return fmt.Errorf("cannot open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("cannot connect to database: %w", err)
	}
	r.db = db
	
	// Create users table if it doesn't exist
	if err := r.createUsersTable(ctx); err != nil {
		return fmt.Errorf("cannot create users table: %w", err)
	}
	
	return nil
}

// Stop closes the database connection.
func (r *UserSQLiteRepo) Stop(ctx context.Context) error {
	if r.db != nil {
		if err := r.db.Close(); err != nil {
			return fmt.Errorf("cannot close database: %w", err)
		}
	}
	return nil
}

// createUsersTable creates the users table with proper schema
func (r *UserSQLiteRepo) createUsersTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email_ct BLOB,
		email_iv BLOB,
		email_tag BLOB,
		email_lookup BLOB NOT NULL,
		password_hash BLOB NOT NULL,
		password_salt BLOB NOT NULL,
		mfa_secret_ct BLOB,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME NOT NULL,
		created_by TEXT DEFAULT '',
		updated_at DATETIME NOT NULL,
		updated_by TEXT DEFAULT ''
	);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lookup ON users(email_lookup);
	CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
	CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
	`

	if _, err := r.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	return nil
}

// Create creates a new User in SQLite.
func (r *UserSQLiteRepo) Create(ctx context.Context, user *auth.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	user.EnsureID()
	user.BeforeCreate()

	query := `
	INSERT INTO users (
		id, email_ct, email_iv, email_tag, email_lookup,
		password_hash, password_salt, mfa_secret_ct, status,
		created_at, created_by, updated_at, updated_by
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID.String(),
		user.EmailCT,
		user.EmailIV,
		user.EmailTag,
		user.EmailLookup,
		user.PasswordHash,
		user.PasswordSalt,
		user.MFASecretCT,
		string(user.Status),
		user.CreatedAt,
		user.CreatedBy,
		user.UpdatedAt,
		user.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Get retrieves a User by ID from SQLite.
func (r *UserSQLiteRepo) Get(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	query := `
	SELECT id, email_ct, email_iv, email_tag, email_lookup,
	       password_hash, password_salt, mfa_secret_ct, status,
	       created_at, created_by, updated_at, updated_by
	FROM users WHERE id = ?
	`

	user := &auth.User{}
	var statusStr string

	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&user.ID,
		&user.EmailCT,
		&user.EmailIV,
		&user.EmailTag,
		&user.EmailLookup,
		&user.PasswordHash,
		&user.PasswordSalt,
		&user.MFASecretCT,
		&statusStr,
		&user.CreatedAt,
		&user.CreatedBy,
		&user.UpdatedAt,
		&user.UpdatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Convert status string back to enum type
	user.Status = authpkg.UserStatus(statusStr)

	return user, nil
}

// GetByEmailLookup retrieves a User by encrypted email lookup hash.
func (r *UserSQLiteRepo) GetByEmailLookup(ctx context.Context, lookup []byte) (*auth.User, error) {
	query := `
	SELECT id, email_ct, email_iv, email_tag, email_lookup,
	       password_hash, password_salt, mfa_secret_ct, status,
	       created_at, created_by, updated_at, updated_by
	FROM users WHERE email_lookup = ?
	`

	user := &auth.User{}
	var statusStr string

	err := r.db.QueryRowContext(ctx, query, lookup).Scan(
		&user.ID,
		&user.EmailCT,
		&user.EmailIV,
		&user.EmailTag,
		&user.EmailLookup,
		&user.PasswordHash,
		&user.PasswordSalt,
		&user.MFASecretCT,
		&statusStr,
		&user.CreatedAt,
		&user.CreatedBy,
		&user.UpdatedAt,
		&user.UpdatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email lookup: %w", err)
	}

	// Convert status string back to enum type
	user.Status = authpkg.UserStatus(statusStr)

	return user, nil
}

// Save updates an existing User in SQLite.
func (r *UserSQLiteRepo) Save(ctx context.Context, user *auth.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	user.BeforeUpdate()

	query := `
	UPDATE users SET
		email_ct = ?, email_iv = ?, email_tag = ?, email_lookup = ?,
		password_hash = ?, password_salt = ?, mfa_secret_ct = ?, status = ?,
		updated_at = ?, updated_by = ?
	WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		user.EmailCT,
		user.EmailIV,
		user.EmailTag,
		user.EmailLookup,
		user.PasswordHash,
		user.PasswordSalt,
		user.MFASecretCT,
		string(user.Status),
		user.UpdatedAt,
		user.UpdatedBy,
		user.ID.String(),
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with ID %s not found for update", user.ID.String())
	}

	return nil
}

// Delete performs a soft delete by changing the user status to deleted.
func (r *UserSQLiteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
	UPDATE users SET
		status = 'deleted',
		updated_at = datetime('now')
	WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with ID %s not found for deletion", id.String())
	}

	return nil
}

// List retrieves all active Users from SQLite.
func (r *UserSQLiteRepo) List(ctx context.Context) ([]*auth.User, error) {
	query := `
	SELECT id, email_ct, email_iv, email_tag, email_lookup,
	       password_hash, password_salt, mfa_secret_ct, status,
	       created_at, created_by, updated_at, updated_by
	FROM users WHERE status != 'deleted'
	ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*auth.User

	for rows.Next() {
		user := &auth.User{}
		var statusStr string

		err := rows.Scan(
			&user.ID,
			&user.EmailCT,
			&user.EmailIV,
			&user.EmailTag,
			&user.EmailLookup,
			&user.PasswordHash,
			&user.PasswordSalt,
			&user.MFASecretCT,
			&statusStr,
			&user.CreatedAt,
			&user.CreatedBy,
			&user.UpdatedAt,
			&user.UpdatedBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		user.Status = authpkg.UserStatus(statusStr)
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// ListByStatus retrieves Users filtered by status from SQLite.
func (r *UserSQLiteRepo) ListByStatus(ctx context.Context, status string) ([]*auth.User, error) {
	query := `
	SELECT id, email_ct, email_iv, email_tag, email_lookup,
	       password_hash, password_salt, mfa_secret_ct, status,
	       created_at, created_by, updated_at, updated_by
	FROM users WHERE status = ?
	ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by status: %w", err)
	}
	defer rows.Close()

	var users []*auth.User

	for rows.Next() {
		user := &auth.User{}
		var statusStr string

		err := rows.Scan(
			&user.ID,
			&user.EmailCT,
			&user.EmailIV,
			&user.EmailTag,
			&user.EmailLookup,
			&user.PasswordHash,
			&user.PasswordSalt,
			&user.MFASecretCT,
			&statusStr,
			&user.CreatedAt,
			&user.CreatedBy,
			&user.UpdatedAt,
			&user.UpdatedBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		user.Status = authpkg.UserStatus(statusStr)
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users by status: %w", err)
	}

	return users, nil
}