package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func ValidateTokenClaims(claims TokenClaims, now time.Time) ValidationErrors {
	var errors ValidationErrors

	if claims.Subject == "" {
		errors = append(errors, ValidationError{
			Field:   "sub",
			Code:    "required",
			Message: "Subject claim is required",
		})
	}

	if claims.SessionID == "" {
		errors = append(errors, ValidationError{
			Field:   "sid",
			Code:    "required",
			Message: "Session ID claim is required",
		})
	}

	if claims.Audience == "" {
		errors = append(errors, ValidationError{
			Field:   "aud",
			Code:    "required",
			Message: "Audience claim is required",
		})
	}

	if claims.ExpiresAt == 0 {
		errors = append(errors, ValidationError{
			Field:   "exp",
			Code:    "required",
			Message: "Expiration time claim is required",
		})
	}

	if claims.AuthzVersion < 0 {
		errors = append(errors, ValidationError{
			Field:   "authz_ver",
			Code:    "invalid_value",
			Message: "Authorization version must be non-negative",
		})
	}

	return errors
}

func IsTokenExpired(claims TokenClaims, now time.Time) bool {
	if claims.ExpiresAt == 0 {
		return true
	}
	expTime := time.Unix(claims.ExpiresAt, 0)
	return now.After(expTime)
}

func ValidateTokenExpiration(claims TokenClaims, now time.Time) ValidationErrors {
	var errors ValidationErrors

	if IsTokenExpired(claims, now) {
		errors = append(errors, ValidationError{
			Field:   "exp",
			Code:    "expired",
			Message: "Token has expired",
		})
	}

	return errors
}

func ValidateTokenAudience(claims TokenClaims, expectedAudience string) ValidationErrors {
	var errors ValidationErrors

	if claims.Audience != expectedAudience {
		errors = append(errors, ValidationError{
			Field:   "aud",
			Code:    "invalid_audience",
			Message: "Token audience does not match expected value",
		})
	}

	return errors
}

func ValidateTokenContext(claims TokenClaims, expectedContext map[string]string) ValidationErrors {
	var errors ValidationErrors

	for key, expectedValue := range expectedContext {
		actualValue, exists := claims.Context[key]
		if !exists {
			errors = append(errors, ValidationError{
				Field:   "ctx." + key,
				Code:    "missing_context",
				Message: "Required context key is missing",
			})
			continue
		}

		if actualValue != expectedValue {
			errors = append(errors, ValidationError{
				Field:   "ctx." + key,
				Code:    "invalid_context",
				Message: "Context value does not match expected value",
			})
		}
	}

	return errors
}

func ValidateTokenForService(claims TokenClaims, service string, now time.Time) ValidationErrors {
	var errors ValidationErrors

	claimErrors := ValidateTokenClaims(claims, now)
	errors = append(errors, claimErrors...)

	expErrors := ValidateTokenExpiration(claims, now)
	errors = append(errors, expErrors...)

	audErrors := ValidateTokenAudience(claims, service)
	errors = append(errors, audErrors...)

	return errors
}

func IsTokenValidForService(claims TokenClaims, service string, now time.Time) bool {
	errors := ValidateTokenForService(claims, service, now)
	return len(errors) == 0
}

func GetTokenTimeToLive(claims TokenClaims, now time.Time) time.Duration {
	if claims.ExpiresAt == 0 {
		return 0
	}

	expTime := time.Unix(claims.ExpiresAt, 0)
	if now.After(expTime) {
		return 0
	}

	return expTime.Sub(now)
}

func IsTokenNearExpiry(claims TokenClaims, now time.Time, threshold time.Duration) bool {
	ttl := GetTokenTimeToLive(claims, now)
	return ttl > 0 && ttl <= threshold
}

func ValidateTokenSubject(claims TokenClaims, expectedSubject string) ValidationErrors {
	var errors ValidationErrors

	if claims.Subject != expectedSubject {
		errors = append(errors, ValidationError{
			Field:   "sub",
			Code:    "invalid_subject",
			Message: "Token subject does not match expected value",
		})
	}

	return errors
}

func ValidateTokenSession(claims TokenClaims, expectedSessionID string) ValidationErrors {
	var errors ValidationErrors

	if claims.SessionID != expectedSessionID {
		errors = append(errors, ValidationError{
			Field:   "sid",
			Code:    "invalid_session",
			Message: "Token session ID does not match expected value",
		})
	}

	return errors
}

func ValidateTokenAuthzVersion(claims TokenClaims, minVersion int) ValidationErrors {
	var errors ValidationErrors

	if claims.AuthzVersion < minVersion {
		errors = append(errors, ValidationError{
			Field:   "authz_ver",
			Code:    "outdated_version",
			Message: "Token authorization version is outdated",
		})
	}

	return errors
}

func ExtractScopeFromTokenContext(claims TokenClaims) (Scope, bool) {
	contextType, hasType := claims.Context["type"]
	if !hasType {
		return Scope{}, false
	}

	contextID, hasID := claims.Context["id"]
	if contextType != "global" && !hasID {
		return Scope{}, false
	}

	if contextType == "global" {
		return Scope{Type: "global", ID: ""}, true
	}

	return Scope{Type: contextType, ID: contextID}, true
}

func TokenSupportsScope(claims TokenClaims, requiredScope Scope) bool {
	tokenScope, hasScope := ExtractScopeFromTokenContext(claims)
	if !hasScope {
		return false
	}

	return ScopeMatches(tokenScope, requiredScope)
}

func ValidateTokenScope(claims TokenClaims, requiredScope Scope) ValidationErrors {
	var errors ValidationErrors

	if !TokenSupportsScope(claims, requiredScope) {
		errors = append(errors, ValidationError{
			Field:   "ctx",
			Code:    "invalid_scope",
			Message: "Token scope does not match required scope",
		})
	}

	return errors
}

func CreateTokenClaims(subject, sessionID, audience string, context map[string]string, ttl time.Duration, authzVersion int) TokenClaims {
	now := time.Now()
	return TokenClaims{
		Subject:      subject,
		SessionID:    sessionID,
		Audience:     audience,
		Context:      context,
		ExpiresAt:    now.Add(ttl).Unix(),
		AuthzVersion: authzVersion,
	}
}

func IsTokenFresh(claims TokenClaims, issuedAt time.Time, maxAge time.Duration) bool {
	if claims.ExpiresAt == 0 {
		return false
	}

	age := time.Since(issuedAt)
	return age <= maxAge
}

// GenerateSessionToken creates a PASETO token for user session
func GenerateSessionToken(userID, sessionID string, privateKey ed25519.PrivateKey, ttl time.Duration) (string, error) {
	claims := CreateTokenClaims(userID, sessionID, "session", map[string]string{"type": "global"}, ttl, 1)
	return GeneratePASETOToken(claims, privateKey)
}

// GenerateInternalToken creates a PASETO token for internal service communication
func GenerateInternalToken(userID, sessionID, audience string, context map[string]string, privateKey ed25519.PrivateKey, ttl time.Duration) (string, error) {
	claims := CreateTokenClaims(userID, sessionID, audience, context, ttl, 1)
	return GeneratePASETOToken(claims, privateKey)
}

// GeneratePASETOToken creates a PASETO token from claims
func GeneratePASETOToken(claims TokenClaims, privateKey ed25519.PrivateKey) (string, error) {
	// Create PASETO payload
	payload := map[string]interface{}{
		"sub":       claims.Subject,
		"sid":       claims.SessionID,
		"aud":       claims.Audience,
		"exp":       claims.ExpiresAt,
		"authz_ver": claims.AuthzVersion,
	}

	if claims.Context != nil && len(claims.Context) > 0 {
		payload["ctx"] = claims.Context
	}

	// Marshal to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token payload: %w", err)
	}

	// Sign with Ed25519 (simplified PASETO v4.public implementation)
	signature := ed25519.Sign(privateKey, payloadJSON)

	// Create PASETO token format: v4.public.payload.signature
	token := fmt.Sprintf("v4.public.%s.%s",
		encodeBase64URL(payloadJSON),
		encodeBase64URL(signature))

	return token, nil
}

// VerifyPASETOToken verifies and parses a PASETO token
func VerifyPASETOToken(token string, publicKey ed25519.PublicKey) (*TokenClaims, error) {
	// Parse PASETO token format: v4.public.payload.signature
	parts := strings.Split(token, ".")
	if len(parts) != 4 || parts[0] != "v4" || parts[1] != "public" {
		return nil, fmt.Errorf("invalid PASETO token format")
	}

	// Decode payload and signature
	payload, err := decodeBase64URL(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	signature, err := decodeBase64URL(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(publicKey, payload, signature) {
		return nil, fmt.Errorf("invalid token signature")
	}

	// Parse claims
	var rawClaims map[string]interface{}
	if err := json.Unmarshal(payload, &rawClaims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	// Convert to TokenClaims struct
	claims := &TokenClaims{}

	if sub, ok := rawClaims["sub"].(string); ok {
		claims.Subject = sub
	}

	if sid, ok := rawClaims["sid"].(string); ok {
		claims.SessionID = sid
	}

	if aud, ok := rawClaims["aud"].(string); ok {
		claims.Audience = aud
	}

	if exp, ok := rawClaims["exp"].(float64); ok {
		claims.ExpiresAt = int64(exp)
	}

	if authzVer, ok := rawClaims["authz_ver"].(float64); ok {
		claims.AuthzVersion = int(authzVer)
	}

	if ctx, ok := rawClaims["ctx"].(map[string]interface{}); ok {
		claims.Context = make(map[string]string)
		for k, v := range ctx {
			if strVal, ok := v.(string); ok {
				claims.Context[k] = strVal
			}
		}
	}

	return claims, nil
}

// GenerateKeyPair generates an Ed25519 key pair for PASETO tokens
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(nil)
}

// encodeBase64URL encodes data using base64 URL encoding without padding
func encodeBase64URL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// decodeBase64URL decodes base64 URL encoded data without padding
func decodeBase64URL(encoded string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(encoded)
}
