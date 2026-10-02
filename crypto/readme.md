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

```go
context := []byte("contact:contact-123:notes")
value, err := crypto.EncryptString("private note", key, context)
plaintext, err := crypto.DecryptString(value, key, context)
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
```

## Other utilities

```go
// Argon2id password hashing
salt, _ := crypto.GenerateSalt()
hash := crypto.HashPassword(password, salt)
ok := crypto.VerifyPassword(password, hash, salt)

// Secure random tokens
token, _ := crypto.GenerateSecureToken(32)
```

For TOTP/MFA, see `totp.go`. For PASETO tokens, see `tokens.go`.

`EncryptEmail`, `DecryptEmail`, and `ComputeLookupHash` remain available for
compatibility. New code should use `EncryptString`, `DecryptString`, and
`DeriveLookupHash`.
