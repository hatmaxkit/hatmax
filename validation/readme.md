# validation

Input validation with field-level errors.

## Usage

```go
// Direct validation (returns error)
err := validation.ValidateEmail(email)
err := validation.ValidatePassword(password)
err := validation.ValidateUsername(username)

// Field validators (return ValidationError for forms)
ve := validation.ValidateEmailField("email", value)
ve := validation.ValidatePhoneField("phone", value)
ve := validation.NoHTML("bio", value)

if !ve.IsEmpty() {
    errors[ve.Field] = ve.Message
}

// Normalization
email := validation.NormalizeEmail(input)  // lowercase, trimmed
```

Password rules: 8-128 chars, requires uppercase, lowercase, digit, special char.
