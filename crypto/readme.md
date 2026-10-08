<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# crypto

Cryptographic utilities: authenticated string encryption, deterministic lookup
hashes, Argon2id hashing, TOTP, and secure tokens.

## Authenticated string encryption

`EncryptString` encrypts arbitrary strings with AES-256-GCM. It returns
Base64-encoded ciphertext, nonce, and authentication tag as an
`EncryptedString`. It requires a 32-byte key.

Associated data is authenticated but not encrypted. Supply stable context that
identifies the record and field being protected, then supply precisely the same
bytes to `DecryptString`. This prevents a valid encrypted value from being
substituted into another record or field that uses the same key.

These fragments run inside an application function returning an error. The
application supplies distinct private 32-byte encryption and lookup keys and
owns their persistence, access and rotation. See the [crypto reference](../docs/reference/crypto/README.md).

```go
context := []byte("contact:contact-123:notes")
value, err := crypto.EncryptString("private note", key, context)
if err != nil { return err }
plaintext, err := crypto.DecryptString(value, key, context)
if err != nil { return err }
```

## Deterministic lookup hashes

`DeriveLookupHash` derives a Base64-encoded HMAC-SHA-256 hash from arbitrary
string data and a 32-byte key. It returns an error for an invalid key.

The function hashes the input bytes exactly. Callers must normalize a value
before both encryption and lookup derivation when their domain treats multiple
representations as equivalent.

```go
normalizedEmail := strings.ToLower(strings.TrimSpace(email))
lookup, err := crypto.DeriveLookupHash(normalizedEmail, lookupKey)
if err != nil { return err }
```

## Other utilities

```go
// Low-level Argon2id bytes, not the auth service's encoded credential format.
salt, err := crypto.GenerateSalt()
if err != nil { return err }
hash := crypto.HashPassword(password, salt)
ok := crypto.VerifyPassword(password, hash, salt)

// Secure random tokens
token, err := crypto.GenerateSecureToken(32)
if err != nil { return err }
```

`HashPassword` uses 65536 KiB (64 MiB), one iteration and four lanes, with
caller-owned 32-byte salt and raw 32-byte output. It performs no normalization,
candidate policy, encoded-record validation or shared KDF admission. Use
`model.NewPasswordVerifier` and the authentication policy for the current
credential workflow; these raw bytes cannot be passed to `auth` as a PHC record.

TOTP primitives and backup-code parsing generate or verify cryptographic
material; the authentication service and transactional adapter own actual
one-use completion. PASETO signing/verification is independent of digest-based
authentication sessions. Signed public-token claims remain readable.

`EncryptEmail`, `DecryptEmail`, and `ComputeLookupHash` remain available for
compatibility. New code should use `EncryptString`, `DecryptString`, and
`DeriveLookupHash`.
