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

`HashPassword` returns a bcrypt hash at `bcrypt.DefaultCost`, or wraps a
bcrypt error as `cannot hash password`. `ComparePassword` reports whether the
hash matches the password.

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
