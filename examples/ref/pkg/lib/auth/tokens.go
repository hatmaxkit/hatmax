package auth

import (
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
