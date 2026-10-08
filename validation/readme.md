<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# validation

Input validation with field-level errors.

## Usage

This fragment belongs in an application function with `email`, `password`,
`username`, `confirmation`, `value`, `input` and a field-message map `messages`.

```go
// Direct validation (returns error)
emailErr := validation.ValidateEmail(email)
passwordErr := validation.ValidatePassword(password)
usernameErr := validation.ValidateUsername(username)

errs := validation.ValidateAll(
    validation.Field("email", email).Required().Email(),
    validation.Field("password", password).Required().Password(),
    validation.Field("password_confirm", confirmation).Required().Equal(password),
)

// Field validators (return ValidationError for forms)
emailField := validation.ValidateEmailField("email", value)
phoneField := validation.ValidatePhoneField("phone", value)
htmlField := validation.NoHTML("bio", value)

if !emailField.IsEmpty() {
    messages[emailField.Field] = emailField.Message
}

// Normalization
normalizedEmail := validation.NormalizeEmail(input)  // lowercase, trimmed
```

These standalone password rules use 8–128 UTF-8 bytes and require uppercase,
lowercase, digit and punctuation/symbol. They are separate from the current
authentication candidate policy, which uses normalized code points and a
checker. Do not apply this composition rule as an authentication prerequisite.
`NoHTML` detects a tag-shaped pattern; it does not sanitize HTML or replace
template escaping. See [Validation](../docs/reference/validation/README.md).
