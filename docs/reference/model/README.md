<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Model

`model` supplies identifiers, bcrypt passwords, UTC timestamps, and a role
check. The implementation note is [model/readme.md](../../../model/readme.md).

## Identifiers

`NewID` returns a new UUID string. `GenerateID` stores one in `*string`.
`ParseID` uses `uuid.Parse`.

`NullUUID` returns an invalid `uuid.NullUUID` for a nil pointer, an empty
string, or a string that is not a UUID. `FromNullUUID` returns nil when
`Valid` is false, and otherwise a pointer to `UUID.String()`.

## Passwords

`HashPassword(password)` retains its one-argument signature and returns a
bcrypt hash at `bcrypt.DefaultCost` (10), independent of authentication
configuration.

`HashPasswordWithCost(password, cost)` uses the explicit cost. Values outside
`bcrypt.MinCost` through `bcrypt.MaxCost` (4 through 31) return a wrapped
`bcrypt.InvalidCostError`; there is no default-cost fallback. Both helpers
wrap bcrypt errors as `cannot hash password`, return no hash on failure, and
preserve bcrypt's 72-byte password limit.

`ComparePassword` reports whether the hash matches the password, using the cost
stored in the hash rather than the current authentication configuration.

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
