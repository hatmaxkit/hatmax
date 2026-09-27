# Crypto

`crypto` encrypts strings, derives lookup hashes, hashes passwords, signs
PASETO tokens, and generates TOTP material. The implementation note is
[crypto/readme.md](../../../crypto/readme.md).

`auth` does not call these password or token functions for signup and
sign-in. See [Authentication](../authentication/index.md).

## Authenticated strings

`EncryptedString` has `Ciphertext`, `Nonce`, and `Tag`. Each field is
standard Base64.

`EncryptString` requires a 32-byte key. A different length returns
`ErrInvalidKey`. It seals the plaintext with AES-256-GCM. `associatedData` is
authenticated and is not encrypted. The same bytes are required for
`DecryptString`.

`DecryptString` returns `ErrInvalidKey`, `ErrInvalidCiphertext`,
`ErrInvalidNonce`, or `ErrInvalidTag` when the corresponding input cannot be
used. An authentication or decryption failure returns `ErrDecryptionFailed`.

`EncryptEmail` calls `EncryptString` with nil associated data and returns the
three fields. `DecryptEmail` calls `DecryptString` with nil associated data.
An `ErrInvalidNonce` from that call is returned as `ErrInvalidIV`.

## Lookup hashes

`DeriveLookupHash` requires a 32-byte key and returns `ErrInvalidKey`
otherwise. It returns the standard Base64 HMAC-SHA-256 of the value bytes.
The function does not trim or lowercase the value.

`ComputeLookupHash` returns the same hash, or `""` when the key is invalid.

## Passwords and tokens

Argon2id uses time 1, memory 64 KiB, 4 threads, and a 32-byte key.
`GenerateSalt` returns 32 random bytes. `HashPassword` returns nil when the
salt is not 32 bytes. `VerifyPassword` returns false for a salt of any other
length, and otherwise compares the hashes in constant time.

`GenerateSecureToken` uses at least 32 random bytes and returns them as raw
URL Base64. A requested length below 32 is raised to 32.

## PASETO

`TokenClaims` fields are `Subject`, `SessionID`, `Audience`, `Context`,
`ExpiresAt`, and `AuthzVersion`.

`GenerateToken` returns `ErrMissingPrivateKey` for a nil private key. It signs
a v4 public PASETO token. `ExpiresAt` is a Unix timestamp.
`AuthzVersion` is written only when it is greater than zero, as one character
whose code point is that integer plus `'0'`. `VerifyToken` reads
`authz_ver` only when that string has length 1.

`VerifyToken` returns `ErrMissingPublicKey` for a nil public key.
`ErrTokenExpired` when the parser reports a PASETO rule error, and
`ErrInvalidToken` for another parse or key error. A missing claim is left at
its zero value.

`GenerateSessionID` returns a new UUID string.

## TOTP

`GenerateTOTPKey` uses a 30-second period, six digits, and SHA-1.
`ValidateTOTPCode` checks the code against the secret.
`ValidateTOTPCodeWithSkew` allows the supplied number of periods before and
after the current time, with the same period, digits, and algorithm.

`GenerateQRCodePNG` returns a PNG of the key's QR image at the supplied size.

`GenerateBackupCodes` uses 8 codes when `count` is not positive. Each plain
code is the first 8 characters of unpadded Base32 of 6 random bytes, uppercased.
The stored hash is Argon2id of that code with the fixed salt
`hatmax-backup-code-salt`, encoded as padded Base64.
`VerifyBackupCode` trims and uppercases the supplied code and returns the
matching index, or false and `-1`.
