<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Crypto

`crypto` encrypts strings, derives lookup hashes, hashes passwords, signs
PASETO tokens, and generates TOTP material. The implementation note is
[crypto/readme.md](../../../crypto/readme.md).

Signup and sign-in use the versioned `model.PasswordVerifier`, not
`crypto.HashPassword` or PASETO credentials. Sessions use random bytes from
`GenerateSecureToken` and purpose-separated digests. See
[Authentication](../authentication/README.md).

## Authenticated strings

`EncryptedString` has `Ciphertext`, `Nonce`, and `Tag`. Each field is
standard padded Base64. Encryption generates a new 12-byte nonce and a 16-byte
authentication tag; empty plaintext has empty ciphertext and still has a tag.

`EncryptString` requires a 32-byte key. A different length returns
`ErrInvalidKey`. It seals the plaintext with AES-256-GCM. `associatedData` is
authenticated and is not encrypted. The same bytes are required for
`DecryptString`.

`DecryptString` returns `ErrInvalidKey`, `ErrInvalidCiphertext`,
`ErrInvalidNonce`, or `ErrInvalidTag` when the corresponding input cannot be
used. An authentication or decryption failure returns `ErrDecryptionFailed`.
Invalid Base64 tag text is `ErrInvalidTag`; a decoded tag with an invalid length
fails authentication as `ErrDecryptionFailed`. A failure returns no plaintext.

Callers own keys and any rotation/version metadata. Use independent encryption
and lookup keys. Persist the nonce and tag with the ciphertext and retain the
exact associated-data bytes. An HMAC lookup still reveals equal input values
under the same key; encryption does not remove that equality signal.

`EncryptEmail` calls `EncryptString` with nil associated data and returns the
three fields. `DecryptEmail` calls `DecryptString` with nil associated data.
An `ErrInvalidNonce` from that call is returned as `ErrInvalidIV`.

## Lookup hashes

`DeriveLookupHash` requires a 32-byte key and returns `ErrInvalidKey`
otherwise. It returns the standard Base64 HMAC-SHA-256 of the value bytes.
The function does not trim or lowercase the value.

`ComputeLookupHash` returns the same hash, or `""` when the key is invalid.

## Passwords and tokens

The low-level Argon2id primitive uses time 1, memory 65536 KiB (64 MiB),
4 lanes, and a 32-byte output.
`GenerateSalt` returns 32 random bytes. `HashPassword` returns nil when the
salt is not 32 bytes. `VerifyPassword` returns false for a salt of any other
length, and otherwise compares the hashes in constant time.
It hashes password bytes exactly, with no NFC normalization, password policy,
input bounds, context or shared concurrency budget. The caller owns those
controls. Its raw output is not the auth service's PHC credential record.

`GenerateSecureToken` uses at least 32 random bytes and returns them as padded
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

Public PASETO signs rather than encrypts claims. Verification checks signature
and expiration; it does not select an expected audience, authorize a subject,
check a current account version or consult the session store. The consumer must
check those application requirements. `AuthzVersion` is this helper's existing
single-byte encoding, not a general canonical integer claim.

`GenerateSessionID` returns a new UUID string.

## TOTP

`GenerateTOTPKey` uses a 30-second period, six digits, and SHA-1.
`MatchTOTPCode(secret, code, now, skew)` accepts canonical six ASCII digits,
a 32-character secret and trusted time, and returns the matched integer step.
Skew is zero or one. Current step is checked first, then previous and next.
`TOTPInWindow` rechecks the selected step at fresh commit time; acceptance must
also strictly advance the durable last step in the access-completion transaction.
These primitives do not persist replay state or issue authentication proof.
The former boolean helpers using hidden current time are removed.

`GenerateQRCodePNG` returns a PNG of the key's QR image at the supplied size.

`NewBackupSecret` produces `backup1.<UUID>.<22-character Base64url secret>` with
128 random secret bits. `ParseBackupCode` rejects noncanonical input before any
KDF. `BackupVerifierInput` binds purpose, subject, identifier and secret for the
shared PHC Argon2id engine. Do not persist plaintext codes. The former shared-salt,
set-scan/index APIs are removed. Use [actual fallback completion](../authentication/README.md#totp-and-backup-proof)
for independently salted storage, bounded KDF admission and atomic one-use access.
