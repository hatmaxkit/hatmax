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

// --- Field Validators (return ValidationError) ---

var (
	// phoneRegex matches common phone formats including E.164
	phoneRegex = regexp.MustCompile(`^[\+]?[(]?[0-9]{1,3}[)]?[-\s\.]?[0-9]{1,4}[-\s\.]?[0-9]{1,4}[-\s\.]?[0-9]{1,9}$`)
	// htmlTagRegex matches HTML tags
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
)

// ValidateEmailField validates email format and returns a ValidationError.
// Returns empty ValidationError if value is empty (use with Required for mandatory fields).
func ValidateEmailField(field, value string) ValidationError {
	if value == "" {
		return ValidationError{}
	}

	err := ValidateEmail(value)
	if err != nil {
		return ValidationError{
			Field:   field,
			Rule:    "Email",
			Message: "must be a valid email address",
		}
	}

	return ValidationError{}
}

// ValidatePhoneField validates phone number format.
// Returns empty ValidationError if value is empty (use with Required for mandatory fields).
func ValidatePhoneField(field, value string) ValidationError {
	if value == "" {
		return ValidationError{}
	}

	// Remove common separators for validation
	cleaned := strings.ReplaceAll(value, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")

	// Check minimum length (country code + some digits)
	if len(cleaned) < 7 || len(cleaned) > 15 {
		return ValidationError{
			Field:   field,
			Rule:    "Phone",
			Message: "must be a valid phone number",
		}
	}

	if !phoneRegex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Rule:    "Phone",
			Message: "must be a valid phone number",
		}
	}

	return ValidationError{}
}

// NoHTML validates that a string contains no HTML tags.
// Returns empty ValidationError if value is empty.
func NoHTML(field, value string) ValidationError {
	if value == "" {
		return ValidationError{}
	}

	if htmlTagRegex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Rule:    "NoHTML",
			Message: "must not contain HTML tags",
		}
	}

	return ValidationError{}
}
