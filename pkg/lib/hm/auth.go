package hm

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/adrianpk/hatmax/pkg/lib/core"
)

type Authenticator interface {
	ValidateToken(token string) (string, error)
}

type FakeAuthenticator struct {
	tokens map[string]string
}

func NewFakeAuthenticator() *FakeAuthenticator {
	return &FakeAuthenticator{
		tokens: map[string]string{
			"dev-admin":  "user-admin-123",
			"dev-user":   "user-456",
			"dev-viewer": "user-viewer-789",
		},
	}
}

func NewFakeAuthenticatorWithTokens(tokens map[string]string) *FakeAuthenticator {
	clone := make(map[string]string, len(tokens))
	for k, v := range tokens {
		clone[strings.TrimSpace(k)] = v
	}
	return &FakeAuthenticator{tokens: clone}
}

func (f *FakeAuthenticator) ValidateToken(token string) (string, error) {
	userID, ok := f.tokens[token]
	if !ok {
		return "", errors.New("invalid token")
	}
	return userID, nil
}

type contextKey string

const userIDContextKey contextKey = "hm_auth_user_id"

func AuthMiddleware(auth Authenticator, log core.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				log.Debug("missing authorization header")
				Error(w, http.StatusUnauthorized, "unauthorized", "Missing authorization header")
				return
			}

			userID, err := auth.ValidateToken(token)
			if err != nil {
				log.Debug("invalid token", "error", err, "token", token)
				Error(w, http.StatusUnauthorized, "unauthorized", "Invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}
