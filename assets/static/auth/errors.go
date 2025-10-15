package auth

import "errors"

var (
	ErrInvalidEmail      = errors.New("invalid email address")
	ErrWeakPassword      = errors.New("password does not meet security requirements")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound      = errors.New("user not found")
	ErrUserSuspended     = errors.New("user account suspended")
	ErrUserDeleted       = errors.New("user account deleted")
	ErrSessionExpired    = errors.New("session expired")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
	ErrInvalidAudience   = errors.New("invalid token audience")
	ErrInvalidScope      = errors.New("invalid scope")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrPolicyNotFound    = errors.New("policy not found")
	ErrInvalidPolicy     = errors.New("invalid policy format")
	ErrGrantExpired      = errors.New("grant expired")
	ErrRoleNotFound      = errors.New("role not found")
)

type ValidationError struct {
	Field   string
	Code    string
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	return e[0].Error()
}

func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}