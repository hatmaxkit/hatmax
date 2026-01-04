package crypto

import (
	"crypto/ed25519"
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(nil)

	tests := []struct {
		name       string
		claims     TokenClaims
		privateKey ed25519.PrivateKey
		wantErr    error
	}{
		{
			name: "valid token generation",
			claims: TokenClaims{
				Subject:   "user-123",
				SessionID: "session-456",
				Audience:  "hatmax",
				ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			},
			privateKey: privateKey,
			wantErr:    nil,
		},
		{
			name: "with context",
			claims: TokenClaims{
				Subject:   "user-123",
				SessionID: "session-456",
				Audience:  "hatmax",
				Context:   map[string]string{"role": "admin"},
				ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			},
			privateKey: privateKey,
			wantErr:    nil,
		},
		{
			name: "with authz version",
			claims: TokenClaims{
				Subject:      "user-123",
				SessionID:    "session-456",
				Audience:     "hatmax",
				ExpiresAt:    time.Now().Add(24 * time.Hour).Unix(),
				AuthzVersion: 1,
			},
			privateKey: privateKey,
			wantErr:    nil,
		},
		{
			name: "nil private key",
			claims: TokenClaims{
				Subject:   "user-123",
				SessionID: "session-456",
				Audience:  "hatmax",
				ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			},
			privateKey: nil,
			wantErr:    ErrMissingPrivateKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.claims, tt.privateKey)

			if err != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil && token == "" {
				t.Error("GenerateToken() returned empty token")
			}

			if tt.wantErr == nil {
				claims, err := VerifyToken(token, publicKey)
				if err != nil {
					t.Fatalf("VerifyToken() after GenerateToken() failed: %v", err)
				}

				if claims.Subject != tt.claims.Subject {
					t.Errorf("Subject = %v, want %v", claims.Subject, tt.claims.Subject)
				}

				if claims.SessionID != tt.claims.SessionID {
					t.Errorf("SessionID = %v, want %v", claims.SessionID, tt.claims.SessionID)
				}
			}
		})
	}
}

func TestVerifyToken(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(nil)
	_, wrongPrivateKey, _ := ed25519.GenerateKey(nil)

	validClaims := TokenClaims{
		Subject:   "user-123",
		SessionID: "session-456",
		Audience:  "hatmax",
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	validToken, _ := GenerateToken(validClaims, privateKey)

	expiredClaims := TokenClaims{
		Subject:   "user-123",
		SessionID: "session-456",
		Audience:  "hatmax",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}

	expiredToken, _ := GenerateToken(expiredClaims, privateKey)

	wrongKeyToken, _ := GenerateToken(validClaims, wrongPrivateKey)

	tests := []struct {
		name      string
		token     string
		publicKey ed25519.PublicKey
		wantErr   error
	}{
		{
			name:      "valid token",
			token:     validToken,
			publicKey: publicKey,
			wantErr:   nil,
		},
		{
			name:      "expired token",
			token:     expiredToken,
			publicKey: publicKey,
			wantErr:   ErrTokenExpired,
		},
		{
			name:      "wrong public key",
			token:     wrongKeyToken,
			publicKey: publicKey,
			wantErr:   ErrInvalidToken,
		},
		{
			name:      "invalid token string",
			token:     "invalid.token.string",
			publicKey: publicKey,
			wantErr:   ErrInvalidToken,
		},
		{
			name:      "empty token",
			token:     "",
			publicKey: publicKey,
			wantErr:   ErrInvalidToken,
		},
		{
			name:      "nil public key",
			token:     validToken,
			publicKey: nil,
			wantErr:   ErrMissingPublicKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := VerifyToken(tt.token, tt.publicKey)

			if err != tt.wantErr {
				t.Errorf("VerifyToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil {
				if claims.Subject == "" {
					t.Error("VerifyToken() returned empty Subject")
				}
				if claims.SessionID == "" {
					t.Error("VerifyToken() returned empty SessionID")
				}
			}
		})
	}
}

func TestGenerateVerifyRoundtrip(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(nil)

	tests := []struct {
		name   string
		claims TokenClaims
	}{
		{
			name: "basic claims",
			claims: TokenClaims{
				Subject:   "user-123",
				SessionID: "session-456",
				Audience:  "hatmax",
				ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
			},
		},
		{
			name: "with empty context",
			claims: TokenClaims{
				Subject:   "user-123",
				SessionID: "session-456",
				Audience:  "hatmax",
				Context:   map[string]string{},
				ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
			},
		},
		{
			name: "with multiple context values",
			claims: TokenClaims{
				Subject:   "user-123",
				SessionID: "session-456",
				Audience:  "hatmax",
				Context:   map[string]string{"role": "admin", "tenant": "org-789"},
				ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
			},
		},
		{
			name: "with authz version",
			claims: TokenClaims{
				Subject:      "user-123",
				SessionID:    "session-456",
				Audience:     "hatmax",
				ExpiresAt:    time.Now().Add(1 * time.Hour).Unix(),
				AuthzVersion: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.claims, privateKey)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}

			claims, err := VerifyToken(token, publicKey)
			if err != nil {
				t.Fatalf("VerifyToken() error = %v", err)
			}

			if claims.Subject != tt.claims.Subject {
				t.Errorf("Subject = %v, want %v", claims.Subject, tt.claims.Subject)
			}

			if claims.SessionID != tt.claims.SessionID {
				t.Errorf("SessionID = %v, want %v", claims.SessionID, tt.claims.SessionID)
			}

			if claims.Audience != tt.claims.Audience {
				t.Errorf("Audience = %v, want %v", claims.Audience, tt.claims.Audience)
			}

			if tt.claims.AuthzVersion > 0 && claims.AuthzVersion != tt.claims.AuthzVersion {
				t.Errorf("AuthzVersion = %v, want %v", claims.AuthzVersion, tt.claims.AuthzVersion)
			}

			if tt.claims.Context != nil && len(tt.claims.Context) > 0 {
				if claims.Context == nil {
					t.Error("Context is nil, want non-nil")
				}
				for k, v := range tt.claims.Context {
					if claims.Context[k] != v {
						t.Errorf("Context[%s] = %v, want %v", k, claims.Context[k], v)
					}
				}
			}
		})
	}
}

func TestGenerateSessionID(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "generate session ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sid1 := GenerateSessionID()
			sid2 := GenerateSessionID()

			if sid1 == "" {
				t.Error("GenerateSessionID() returned empty string")
			}

			if sid2 == "" {
				t.Error("GenerateSessionID() returned empty string")
			}

			if sid1 == sid2 {
				t.Error("GenerateSessionID() not unique, got same ID twice")
			}

			if len(sid1) != 36 {
				t.Errorf("GenerateSessionID() length = %d, want 36 (UUID format)", len(sid1))
			}
		})
	}
}
