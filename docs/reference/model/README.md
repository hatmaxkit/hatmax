<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Model

`model` supplies identifiers, Argon2id credentials, UTC timestamps, and a role
check. The implementation note is [model/readme.md](../../../model/readme.md).

## Identifiers

`NewID` returns a new UUID string. `GenerateID` stores one in `*string`.
`ParseID` uses `uuid.Parse`.

`NullUUID` returns an invalid `uuid.NullUUID` for a nil pointer, an empty
string, or a string that is not a UUID. `FromNullUUID` returns nil when
`Valid` is false, and otherwise a pointer to `UUID.String()`.

## Passwords

`NewPasswordVerifier(cfg)` constructs a bounded PHC Argon2id verifier.
`Hash(ctx, password)` creates a salted encoded record; `Verify(ctx, record,
password)` returns nil for a match or a classified error. Both operations use
NFC and finite input/work limits. See the
[credential reference](../authentication/README.md#versioned-password-verifier)
for the encoding, parameters, cancellation and shared concurrency contract.
The former bcrypt hash/compare helpers have been removed; no legacy reader or
rehash path is available.

`GenerateRandomPassword(length)` reads that many random bytes, encodes them
as raw URL Base64, and returns the first `length` characters of that
encoding.

## Time

`Now` returns the current UTC time. `SetCreated` writes that time to both
pointers. `SetUpdated` writes it to one pointer.

## Roles

`HasRole(roles, role)` returns true when `roles` contains `superadmin`, and
otherwise when it contains `role`. This bypass is not used by
`auth.User.HasRole`.
