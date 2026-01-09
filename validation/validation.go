package validation

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

const (
	emailMaxLength      = 254
	emailLocalMaxLength = 64
	passwordMinLength   = 8
	passwordMaxLength   = 128
	usernameMinLength   = 3
	usernameMaxLength   = 32
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

	ErrEmailInvalid        = errors.New("invalid email format")
	ErrEmailTooLong        = errors.New("email too long")
	ErrEmailLocalTooLong   = errors.New("email local part too long")
	ErrPasswordTooShort    = errors.New("password too short")
	ErrPasswordTooLong     = errors.New("password too long")
	ErrPasswordNoUppercase = errors.New("password must contain uppercase letter")
	ErrPasswordNoLowercase = errors.New("password must contain lowercase letter")
	ErrPasswordNoDigit     = errors.New("password must contain digit")
	ErrPasswordNoSpecial   = errors.New("password must contain special character")
	ErrUsernameTooShort    = errors.New("username too short")
	ErrUsernameTooLong     = errors.New("username too long")
	ErrUsernameInvalid     = errors.New("invalid username format")
)

func ValidateEmail(email string) error {
	if len(email) > emailMaxLength {
		return ErrEmailTooLong
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ErrEmailInvalid
	}

	if len(parts[0]) > emailLocalMaxLength {
		return ErrEmailLocalTooLong
	}

	if !emailRegex.MatchString(email) {
		return ErrEmailInvalid
	}

	return nil
}

func ValidatePassword(password string) error {
	if len(password) < passwordMinLength {
		return ErrPasswordTooShort
	}

	if len(password) > passwordMaxLength {
		return ErrPasswordTooLong
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)

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
		return ErrPasswordNoUppercase
	}

	if !hasLower {
		return ErrPasswordNoLowercase
	}

	if !hasDigit {
		return ErrPasswordNoDigit
	}

	if !hasSpecial {
		return ErrPasswordNoSpecial
	}

	return nil
}

func ValidateUsername(username string) error {
	if len(username) < usernameMinLength {
		return ErrUsernameTooShort
	}

	if len(username) > usernameMaxLength {
		return ErrUsernameTooLong
	}

	if !usernameRegex.MatchString(username) {
		return ErrUsernameInvalid
	}

	return nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
