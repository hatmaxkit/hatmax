<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Validation

`validation` checks values and collects field errors. The implementation note
is [validation/readme.md](../../../validation/readme.md).

## Errors

`ValidationError` has `Field`, `Rule`, `Message`, and `Params`. `Error`
returns `Message` when `Field` is empty, and otherwise `Field: Message`.

`ValidationErrors` joins its items with `"; "`. An empty slice returns `""`.
`HasErrors` reports a non-empty slice. `Add` appends a field and message
without a rule. `AddError` appends a `ValidationError`. `Merge` appends
another slice. `ForField` returns the messages for one field. `Fields`
returns the field names that appear. `ByField` returns the first message for
a field, or `""`. `First` returns the first error, or a zero error when the
slice is empty. `AsMap` groups messages by field. `NewSingleError` and
`NewError` build a one-item slice. `IsEmpty` is true when `Field`, `Rule`,
and `Message` are empty.

`Validator` is `Validate() ValidationErrors`. `Combine` appends the results
of each validator.

## Whole values

`ValidateEmail` rejects more than 254 characters, a local part longer than 64
characters, and a value that does not match one `@` plus the email pattern.
`ValidatePassword` requires 8 through 128 characters, one uppercase letter,
one lowercase letter, one digit, and one punctuation or symbol character.
`ValidateUsername` requires 3 through 32 characters matching
`[a-zA-Z0-9_-]{3,32}`. `NormalizeEmail` trims space and lowercases the value.

The sentinel errors are `ErrEmailInvalid`, `ErrEmailTooLong`,
`ErrEmailLocalTooLong`, `ErrPasswordTooShort`, `ErrPasswordTooLong`,
`ErrPasswordNoUppercase`, `ErrPasswordNoLowercase`, `ErrPasswordNoDigit`,
`ErrPasswordNoSpecial`, `ErrUsernameTooShort`, `ErrUsernameTooLong`, and
`ErrUsernameInvalid`.

## Fields

`Field`, `UUIDField`, `IntField`, and `FloatField` collect errors for one
value. An empty string skips `MinLength`, `MaxLength`, `Email`, `Password`,
`Phone`, `NoHTML`, and `OneOf`. `Required` trims space and adds `is required`
when the result is empty. Later rules still run.

`ValidateEmailField`, `ValidatePhoneField`, and `NoHTML` return an empty
error for an empty value. Phone validation strips spaces, hyphens, dots, and
parentheses, then requires 7 through 15 characters and a match of the original
value against the phone pattern.

`ValidateAll` appends the errors from each `FieldErrors` value.

## Text

`NormalizeText` turns control characters into spaces, trims, and collapses
whitespace. `NormalizeOptionalText` returns nil for a nil pointer or a result
that is empty. `SanitizeStringSlice` normalizes each item, drops empty items,
and keeps the first occurrence of each remaining value. A nil slice stays nil.
