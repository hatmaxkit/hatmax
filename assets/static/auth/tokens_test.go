package auth

import (
	"testing"
	"time"
)

func TestValidateTokenClaims(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		claims        TokenClaims
		expectedCount int
		expectedCodes []string
	}{
		{
			name: "valid token claims",
			claims: TokenClaims{
				Subject:      "user123",
				SessionID:    "session456",
				Audience:     "orders",
				Context:      map[string]string{"type": "team", "id": "123"},
				ExpiresAt:    now.Add(time.Hour).Unix(),
				AuthzVersion: 1,
			},
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name: "missing subject",
			claims: TokenClaims{
				SessionID:    "session456",
				Audience:     "orders",
				ExpiresAt:    now.Add(time.Hour).Unix(),
				AuthzVersion: 1,
			},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name: "missing session ID",
			claims: TokenClaims{
				Subject:      "user123",
				Audience:     "orders",
				ExpiresAt:    now.Add(time.Hour).Unix(),
				AuthzVersion: 1,
			},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name: "missing audience",
			claims: TokenClaims{
				Subject:      "user123",
				SessionID:    "session456",
				ExpiresAt:    now.Add(time.Hour).Unix(),
				AuthzVersion: 1,
			},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name: "missing expires at",
			claims: TokenClaims{
				Subject:      "user123",
				SessionID:    "session456",
				Audience:     "orders",
				AuthzVersion: 1,
			},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name: "negative authz version",
			claims: TokenClaims{
				Subject:      "user123",
				SessionID:    "session456",
				Audience:     "orders",
				ExpiresAt:    now.Add(time.Hour).Unix(),
				AuthzVersion: -1,
			},
			expectedCount: 1,
			expectedCodes: []string{"invalid_value"},
		},
		{
			name: "multiple validation errors",
			claims: TokenClaims{
				AuthzVersion: -1,
			},
			expectedCount: 5,
			expectedCodes: []string{"required", "required", "required", "required", "invalid_value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateTokenClaims(tt.claims, now)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateTokenClaims() returned %d errors, want %d", len(errors), tt.expectedCount)
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}

func TestIsTokenExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		claims   TokenClaims
		expected bool
	}{
		{
			name: "token not expired",
			claims: TokenClaims{
				ExpiresAt: now.Add(time.Hour).Unix(),
			},
			expected: false,
		},
		{
			name: "token expired",
			claims: TokenClaims{
				ExpiresAt: now.Add(-time.Hour).Unix(),
			},
			expected: true,
		},
		{
			name: "token expires exactly now",
			claims: TokenClaims{
				ExpiresAt: now.Unix(),
			},
			expected: true,
		},
		{
			name: "token with zero expiration",
			claims: TokenClaims{
				ExpiresAt: 0,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTokenExpired(tt.claims, now)
			if result != tt.expected {
				t.Errorf("IsTokenExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateTokenExpiration(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		claims        TokenClaims
		expectedCount int
		expectedCodes []string
	}{
		{
			name: "token not expired",
			claims: TokenClaims{
				ExpiresAt: now.Add(time.Hour).Unix(),
			},
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name: "token expired",
			claims: TokenClaims{
				ExpiresAt: now.Add(-time.Hour).Unix(),
			},
			expectedCount: 1,
			expectedCodes: []string{"expired"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateTokenExpiration(tt.claims, now)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateTokenExpiration() returned %d errors, want %d", len(errors), tt.expectedCount)
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}

func TestValidateTokenAudience(t *testing.T) {
	tests := []struct {
		name             string
		claims           TokenClaims
		expectedAudience string
		expectedCount    int
		expectedCodes    []string
	}{
		{
			name: "matching audience",
			claims: TokenClaims{
				Audience: "orders",
			},
			expectedAudience: "orders",
			expectedCount:    0,
			expectedCodes:    []string{},
		},
		{
			name: "non-matching audience",
			claims: TokenClaims{
				Audience: "orders",
			},
			expectedAudience: "users",
			expectedCount:    1,
			expectedCodes:    []string{"invalid_audience"},
		},
		{
			name: "empty token audience",
			claims: TokenClaims{
				Audience: "",
			},
			expectedAudience: "orders",
			expectedCount:    1,
			expectedCodes:    []string{"invalid_audience"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateTokenAudience(tt.claims, tt.expectedAudience)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateTokenAudience() returned %d errors, want %d", len(errors), tt.expectedCount)
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}

func TestValidateTokenContext(t *testing.T) {
	tests := []struct {
		name            string
		claims          TokenClaims
		expectedContext map[string]string
		expectedCount   int
		expectedCodes   []string
	}{
		{
			name: "matching context",
			claims: TokenClaims{
				Context: map[string]string{
					"type": "team",
					"id":   "123",
				},
			},
			expectedContext: map[string]string{
				"type": "team",
				"id":   "123",
			},
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name: "missing context key",
			claims: TokenClaims{
				Context: map[string]string{
					"type": "team",
				},
			},
			expectedContext: map[string]string{
				"type": "team",
				"id":   "123",
			},
			expectedCount: 1,
			expectedCodes: []string{"missing_context"},
		},
		{
			name: "wrong context value",
			claims: TokenClaims{
				Context: map[string]string{
					"type": "team",
					"id":   "456",
				},
			},
			expectedContext: map[string]string{
				"type": "team",
				"id":   "123",
			},
			expectedCount: 1,
			expectedCodes: []string{"invalid_context"},
		},
		{
			name: "multiple context errors",
			claims: TokenClaims{
				Context: map[string]string{
					"type": "organization",
				},
			},
			expectedContext: map[string]string{
				"type": "team",
				"id":   "123",
			},
			expectedCount: 2,
			expectedCodes: []string{"invalid_context", "missing_context"},
		},
		{
			name:            "empty expected context",
			claims:          TokenClaims{Context: map[string]string{"type": "team"}},
			expectedContext: map[string]string{},
			expectedCount:   0,
			expectedCodes:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateTokenContext(tt.claims, tt.expectedContext)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateTokenContext() returned %d errors, want %d", len(errors), tt.expectedCount)
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}

func TestIsTokenValidForService(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		claims   TokenClaims
		service  string
		expected bool
	}{
		{
			name: "valid token for service",
			claims: TokenClaims{
				Subject:      "user123",
				SessionID:    "session456",
				Audience:     "orders",
				Context:      map[string]string{"type": "team", "id": "123"},
				ExpiresAt:    now.Add(time.Hour).Unix(),
				AuthzVersion: 1,
			},
			service:  "orders",
			expected: true,
		},
		{
			name: "wrong audience",
			claims: TokenClaims{
				Subject:      "user123",
				SessionID:    "session456",
				Audience:     "users",
				Context:      map[string]string{"type": "team", "id": "123"},
				ExpiresAt:    now.Add(time.Hour).Unix(),
				AuthzVersion: 1,
			},
			service:  "orders",
			expected: false,
		},
		{
			name: "expired token",
			claims: TokenClaims{
				Subject:      "user123",
				SessionID:    "session456",
				Audience:     "orders",
				Context:      map[string]string{"type": "team", "id": "123"},
				ExpiresAt:    now.Add(-time.Hour).Unix(),
				AuthzVersion: 1,
			},
			service:  "orders",
			expected: false,
		},
		{
			name: "missing required claims",
			claims: TokenClaims{
				Audience:     "orders",
				ExpiresAt:    now.Add(time.Hour).Unix(),
				AuthzVersion: 1,
			},
			service:  "orders",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTokenValidForService(tt.claims, tt.service, now)
			if result != tt.expected {
				t.Errorf("IsTokenValidForService() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetTokenTimeToLive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		claims   TokenClaims
		expected time.Duration
	}{
		{
			name: "token with 1 hour TTL",
			claims: TokenClaims{
				ExpiresAt: now.Add(time.Hour).Unix(),
			},
			expected: time.Hour,
		},
		{
			name: "expired token",
			claims: TokenClaims{
				ExpiresAt: now.Add(-time.Hour).Unix(),
			},
			expected: 0,
		},
		{
			name: "token with zero expiration",
			claims: TokenClaims{
				ExpiresAt: 0,
			},
			expected: 0,
		},
		{
			name: "token expiring in 5 minutes",
			claims: TokenClaims{
				ExpiresAt: now.Add(5 * time.Minute).Unix(),
			},
			expected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTokenTimeToLive(tt.claims, now)
			
			// Allow for small time differences due to execution time
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Second {
				t.Errorf("GetTokenTimeToLive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsTokenNearExpiry(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		claims    TokenClaims
		threshold time.Duration
		expected  bool
	}{
		{
			name: "token expiring within threshold",
			claims: TokenClaims{
				ExpiresAt: now.Add(2 * time.Minute).Unix(),
			},
			threshold: 5 * time.Minute,
			expected:  true,
		},
		{
			name: "token expiring outside threshold",
			claims: TokenClaims{
				ExpiresAt: now.Add(10 * time.Minute).Unix(),
			},
			threshold: 5 * time.Minute,
			expected:  false,
		},
		{
			name: "expired token",
			claims: TokenClaims{
				ExpiresAt: now.Add(-time.Hour).Unix(),
			},
			threshold: 5 * time.Minute,
			expected:  false,
		},
		{
			name: "token with zero expiration",
			claims: TokenClaims{
				ExpiresAt: 0,
			},
			threshold: 5 * time.Minute,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTokenNearExpiry(tt.claims, now, tt.threshold)
			if result != tt.expected {
				t.Errorf("IsTokenNearExpiry() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractScopeFromTokenContext(t *testing.T) {
	tests := []struct {
		name          string
		claims        TokenClaims
		expectedScope Scope
		expectedOk    bool
	}{
		{
			name: "global scope",
			claims: TokenClaims{
				Context: map[string]string{
					"type": "global",
				},
			},
			expectedScope: Scope{Type: "global", ID: ""},
			expectedOk:    true,
		},
		{
			name: "team scope with ID",
			claims: TokenClaims{
				Context: map[string]string{
					"type": "team",
					"id":   "123",
				},
			},
			expectedScope: Scope{Type: "team", ID: "123"},
			expectedOk:    true,
		},
		{
			name: "missing type",
			claims: TokenClaims{
				Context: map[string]string{
					"id": "123",
				},
			},
			expectedScope: Scope{},
			expectedOk:    false,
		},
		{
			name: "team scope without ID",
			claims: TokenClaims{
				Context: map[string]string{
					"type": "team",
				},
			},
			expectedScope: Scope{},
			expectedOk:    false,
		},
		{
			name: "global scope with ID (should work)",
			claims: TokenClaims{
				Context: map[string]string{
					"type": "global",
					"id":   "should-be-ignored",
				},
			},
			expectedScope: Scope{Type: "global", ID: ""},
			expectedOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, ok := ExtractScopeFromTokenContext(tt.claims)
			if ok != tt.expectedOk {
				t.Errorf("ExtractScopeFromTokenContext() ok = %v, want %v", ok, tt.expectedOk)
			}
			if scope != tt.expectedScope {
				t.Errorf("ExtractScopeFromTokenContext() scope = %v, want %v", scope, tt.expectedScope)
			}
		})
	}
}

func TestCreateTokenClaims(t *testing.T) {
	subject := "user123"
	sessionID := "session456"
	audience := "orders"
	context := map[string]string{"type": "team", "id": "123"}
	ttl := time.Hour
	authzVersion := 1

	claims := CreateTokenClaims(subject, sessionID, audience, context, ttl, authzVersion)

	if claims.Subject != subject {
		t.Errorf("CreateTokenClaims() Subject = %v, want %v", claims.Subject, subject)
	}
	if claims.SessionID != sessionID {
		t.Errorf("CreateTokenClaims() SessionID = %v, want %v", claims.SessionID, sessionID)
	}
	if claims.Audience != audience {
		t.Errorf("CreateTokenClaims() Audience = %v, want %v", claims.Audience, audience)
	}
	if claims.AuthzVersion != authzVersion {
		t.Errorf("CreateTokenClaims() AuthzVersion = %v, want %v", claims.AuthzVersion, authzVersion)
	}
	
	if claims.ExpiresAt == 0 {
		t.Error("CreateTokenClaims() ExpiresAt should not be zero")
	}

	expTime := time.Unix(claims.ExpiresAt, 0)
	expectedExp := time.Now().Add(ttl)
	if expTime.Sub(expectedExp) > time.Second {
		t.Errorf("CreateTokenClaims() expiration time is not within expected range")
	}
}