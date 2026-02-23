package auth

import (
	"context"
	"testing"
	"time"
)

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		userID string
		wantOk bool
	}{
		{
			name:   "user ID in context",
			ctx:    WithUserID(context.Background(), "user-123"),
			userID: "user-123",
			wantOk: true,
		},
		{
			name:   "no user ID in context",
			ctx:    context.Background(),
			userID: "",
			wantOk: false,
		},
		{
			name:   "nil context",
			ctx:    nil,
			userID: "",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := GetUserID(tt.ctx)
			if ok != tt.wantOk {
				t.Errorf("GetUserID() ok = %v, want %v", ok, tt.wantOk)
			}

			if id != tt.userID {
				t.Errorf("GetUserID() id = %v, want %v", id, tt.userID)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	user := &User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tests := []struct {
		name     string
		ctx      context.Context
		wantUser *User
		wantOk   bool
	}{
		{
			name:     "user in context",
			ctx:      WithUser(context.Background(), user),
			wantUser: user,
			wantOk:   true,
		},
		{
			name:     "no user in context",
			ctx:      context.Background(),
			wantUser: nil,
			wantOk:   false,
		},
		{
			name:     "nil context",
			ctx:      nil,
			wantUser: nil,
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, ok := GetUser(tt.ctx)
			if ok != tt.wantOk {
				t.Errorf("GetUser() ok = %v, want %v", ok, tt.wantOk)
			}

			if tt.wantOk && u.ID != tt.wantUser.ID {
				t.Errorf("GetUser() user.ID = %v, want %v", u.ID, tt.wantUser.ID)
			}
		})
	}
}

func TestGetSession(t *testing.T) {
	session := &Session{
		ID:        "session-123",
		UserID:    "user-123",
		Token:     "token-123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		ctx         context.Context
		wantSession *Session
		wantOk      bool
	}{
		{
			name:        "session in context",
			ctx:         WithSession(context.Background(), session),
			wantSession: session,
			wantOk:      true,
		},
		{
			name:        "no session in context",
			ctx:         context.Background(),
			wantSession: nil,
			wantOk:      false,
		},
		{
			name:        "nil context",
			ctx:         nil,
			wantSession: nil,
			wantOk:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := GetSession(tt.ctx)
			if ok != tt.wantOk {
				t.Errorf("GetSession() ok = %v, want %v", ok, tt.wantOk)
			}

			if tt.wantOk && s.ID != tt.wantSession.ID {
				t.Errorf("GetSession() session.ID = %v, want %v", s.ID, tt.wantSession.ID)
			}
		})
	}
}

func TestWithUserID(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-123")

	id, ok := GetUserID(ctx)
	if !ok {
		t.Error("WithUserID() user ID not found in context")
	}

	if id != "user-123" {
		t.Errorf("WithUserID() id = %v, want user-123", id)
	}
}

func TestWithUser(t *testing.T) {
	user := &User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	ctx := WithUser(context.Background(), user)

	u, ok := GetUser(ctx)
	if !ok {
		t.Error("WithUser() user not found in context")
	}

	if u.ID != user.ID {
		t.Errorf("WithUser() user.ID = %v, want %v", u.ID, user.ID)
	}
}

func TestWithSession(t *testing.T) {
	session := &Session{
		ID:        "session-123",
		UserID:    "user-123",
		Token:     "token-123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	ctx := WithSession(context.Background(), session)

	s, ok := GetSession(ctx)
	if !ok {
		t.Error("WithSession() session not found in context")
	}

	if s.ID != session.ID {
		t.Errorf("WithSession() session.ID = %v, want %v", s.ID, session.ID)
	}
}
