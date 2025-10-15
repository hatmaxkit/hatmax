package auth

import (
	"regexp"
	"strings"
	"unicode"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func ValidateEmail(email string) ValidationErrors {
	var errors ValidationErrors

	normalized := NormalizeEmail(email)

	if normalized == "" {
		errors = append(errors, ValidationError{
			Field:   "email",
			Code:    "required",
			Message: "Email is required",
		})
		return errors
	}

	if len(normalized) > 254 {
		errors = append(errors, ValidationError{
			Field:   "email",
			Code:    "too_long",
			Message: "Email must be less than 255 characters",
		})
	}

	if !emailRegex.MatchString(normalized) {
		errors = append(errors, ValidationError{
			Field:   "email",
			Code:    "invalid_format",
			Message: "Email format is invalid",
		})
	}

	parts := strings.Split(normalized, "@")
	if len(parts) == 2 {
		localPart := parts[0]
		if len(localPart) > 64 {
			errors = append(errors, ValidationError{
				Field:   "email",
				Code:    "local_too_long",
				Message: "Email local part must be less than 65 characters",
			})
		}
	}

	return errors
}

func ValidatePassword(password string) ValidationErrors {
	var errors ValidationErrors

	if password == "" {
		errors = append(errors, ValidationError{
			Field:   "password",
			Code:    "required",
			Message: "Password is required",
		})
		return errors
	}

	if len(password) < 8 {
		errors = append(errors, ValidationError{
			Field:   "password",
			Code:    "too_short",
			Message: "Password must be at least 8 characters long",
		})
	}

	if len(password) > 128 {
		errors = append(errors, ValidationError{
			Field:   "password",
			Code:    "too_long",
			Message: "Password must be less than 129 characters",
		})
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		errors = append(errors, ValidationError{
			Field:   "password",
			Code:    "missing_uppercase",
			Message: "Password must contain at least one uppercase letter",
		})
	}

	if !hasLower {
		errors = append(errors, ValidationError{
			Field:   "password",
			Code:    "missing_lowercase",
			Message: "Password must contain at least one lowercase letter",
		})
	}

	if !hasDigit {
		errors = append(errors, ValidationError{
			Field:   "password",
			Code:    "missing_digit",
			Message: "Password must contain at least one digit",
		})
	}

	if !hasSpecial {
		errors = append(errors, ValidationError{
			Field:   "password",
			Code:    "missing_special",
			Message: "Password must contain at least one special character",
		})
	}

	return errors
}

func ValidateUserStatus(status UserStatus) ValidationErrors {
	var errors ValidationErrors

	switch status {
	case UserStatusActive, UserStatusSuspended, UserStatusDeleted:
		return errors
	default:
		errors = append(errors, ValidationError{
			Field:   "status",
			Code:    "invalid_value",
			Message: "Status must be active, suspended, or deleted",
		})
	}

	return errors
}

func ValidateGrantType(grantType GrantType) ValidationErrors {
	var errors ValidationErrors

	switch grantType {
	case GrantTypeRole, GrantTypePermission:
		return errors
	default:
		errors = append(errors, ValidationError{
			Field:   "grant_type",
			Code:    "invalid_value",
			Message: "Grant type must be role or permission",
		})
	}

	return errors
}

func ValidateScope(scope Scope) ValidationErrors {
	var errors ValidationErrors

	if scope.Type == "" {
		errors = append(errors, ValidationError{
			Field:   "scope.type",
			Code:    "required",
			Message: "Scope type is required",
		})
	}

	switch scope.Type {
	case "global":
		if scope.ID != "" {
			errors = append(errors, ValidationError{
				Field:   "scope.id",
				Code:    "invalid_for_global",
				Message: "Global scope cannot have an ID",
			})
		}
	case "team", "organization":
		if scope.ID == "" {
			errors = append(errors, ValidationError{
				Field:   "scope.id",
				Code:    "required",
				Message: "Scope ID is required for non-global scopes",
			})
		}
	default:
		errors = append(errors, ValidationError{
			Field:   "scope.type",
			Code:    "invalid_value",
			Message: "Scope type must be global, team, or organization",
		})
	}

	return errors
}

func ValidatePermissionCode(permission string) ValidationErrors {
	var errors ValidationErrors

	if permission == "" {
		errors = append(errors, ValidationError{
			Field:   "permission",
			Code:    "required",
			Message: "Permission code is required",
		})
		return errors
	}

	if len(permission) > 100 {
		errors = append(errors, ValidationError{
			Field:   "permission",
			Code:    "too_long",
			Message: "Permission code must be less than 101 characters",
		})
	}

	validPermissionRegex := regexp.MustCompile(`^[a-z][a-z0-9_]*:[a-z][a-z0-9_]*$`)
	if !validPermissionRegex.MatchString(permission) {
		errors = append(errors, ValidationError{
			Field:   "permission",
			Code:    "invalid_format",
			Message: "Permission code must be in format 'resource:action' with lowercase letters, numbers, and underscores",
		})
	}

	return errors
}
