package auth

import "context"

type contextKey string

const (
	userIDKey  contextKey = "auth_user_id"
	userKey    contextKey = "auth_user"
	sessionKey contextKey = "auth_session"
)

// GetUserID extracts the user ID from the context.
// Returns the user ID and true if found, empty string and false otherwise.
func GetUserID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id, true
	}
	return "", false
}

// GetUser extracts the full user from the context.
// Returns the user and true if found, nil and false otherwise.
func GetUser(ctx context.Context) (*User, bool) {
	if ctx == nil {
		return nil, false
	}
	if user, ok := ctx.Value(userKey).(*User); ok {
		return user, true
	}
	return nil, false
}

// GetSession extracts the session from the context.
// Returns the session and true if found, nil and false otherwise.
func GetSession(ctx context.Context) (*Session, bool) {
	if ctx == nil {
		return nil, false
	}
	if session, ok := ctx.Value(sessionKey).(*Session); ok {
		return session, true
	}
	return nil, false
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithUser adds a user to the context.
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// WithSession adds a session to the context.
func WithSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, sessionKey, session)
}
