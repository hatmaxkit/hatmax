<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# model

Base types for domain models: IDs, timestamps, passwords, roles.

## Usage

This fragment belongs in an application function with `ctx`, `password`, a
`user` containing `ID string`, and a valid `parentID string`.

```go
// IDs
id := model.NewID()                    // "550e8400-e29b-41d4-a716-446655440000"
model.GenerateID(&user.ID)             // sets in place
parsed, err := model.ParseID(id)
if err != nil {
    return err
}

// Nullable UUIDs (for optional foreign keys)
nullID := model.NullUUID(&parentID)    // string ptr -> uuid.NullUUID
strPtr := model.FromNullUUID(nullID)   // uuid.NullUUID -> string ptr

// Timestamps
now := model.Now()                     // UTC time; precision is retained

// Passwords
verifier, err := model.NewPasswordVerifier(model.PasswordVerifierConfig{})
if err != nil {
    return err
}
hash, err := verifier.Hash(ctx, password)
if err != nil {
    return err
}
err = verifier.Verify(ctx, hash, password)
if err != nil {
    return err
}

// Roles
allowed := model.HasRole([]string{"editor"}, "editor")
```

See individual files: `id.go`, `time.go`, `password.go`, `roles.go`.

`model.HasRole` checks string membership with a `superadmin` bypass. It is a
primitive, not an application authorization policy; `auth.User.HasRole` has
its own contract. `NullUUID` collapses malformed and absent input to an invalid
nullable value, so validate required identifiers before conversion.

## Versioned credentials

`NewPasswordVerifier(PasswordVerifierConfig{})` provides independently salted
PHC Argon2id records through `Hash(ctx, password)` and classified verification
through `Verify(ctx, record, password)`. Share one instance to enforce its
resource budget. Both operations use bounded UTF-8/NFC input processing.

See the [password verifier reference](../docs/reference/authentication/README.md#versioned-password-verifier)
for encoding, configuration, errors and cancellation guarantees. Candidate
policy and database operations belong outside this model primitive. The auth service applies candidate policy before hashing and checks persistent
current state before creating a session.
