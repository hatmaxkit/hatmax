<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# model

Base types for domain models: IDs, timestamps, passwords, roles.

## Usage

```go
// IDs
id := model.NewID()                    // "550e8400-e29b-41d4-a716-446655440000"
model.GenerateID(&user.ID)             // sets in place
parsed, err := model.ParseID(str)

// Nullable UUIDs (for optional foreign keys)
nullID := model.NullUUID(&parentID)    // string ptr -> uuid.NullUUID
strPtr := model.FromNullUUID(nullID)   // uuid.NullUUID -> string ptr

// Timestamps
now := model.Now()                     // time.Time truncated to seconds

// Passwords
hash, err := model.HashPassword(password)
ok := model.ComparePassword(hash, password)

// Roles
role := model.RoleAdmin
if role.HasPermission(model.PermWrite) { ... }
```

See individual files: `id.go`, `time.go`, `password.go`, `roles.go`.

## Versioned credentials

`NewPasswordVerifier(PasswordVerifierConfig{})` provides independently salted
PHC Argon2id records through `Hash(ctx, password)` and classified verification
through `Verify(ctx, record, password)`. Share one instance to enforce its
resource budget. Both operations use bounded UTF-8/NFC input processing.

See the [password verifier reference](../docs/reference/authentication/README.md#versioned-password-verifier)
for encoding, configuration, errors and cancellation guarantees. Candidate
policy and database operations belong outside this model primitive. The existing
bcrypt helpers remain in use by the auth service until its integration changes.
