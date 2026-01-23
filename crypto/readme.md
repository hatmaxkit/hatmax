# crypto

Cryptographic utilities: AES-256-GCM encryption, Argon2id hashing, HMAC, TOTP.

## Usage

```go
// AES-256-GCM encryption (e.g., for PII)
ciphertext, iv, tag, err := crypto.EncryptEmail(email, key)
plaintext, err := crypto.DecryptEmail(ciphertext, iv, tag, key)

// HMAC for deterministic lookups
hash := crypto.ComputeLookupHash(email, signingKey)

// Argon2id password hashing
salt, _ := crypto.GenerateSalt()
hash := crypto.HashPassword(password, salt)
ok := crypto.VerifyPassword(password, hash, salt)

// Secure random tokens
token, _ := crypto.GenerateSecureToken(32)
```

For TOTP/MFA, see `totp.go`. For PASETO tokens, see `tokens.go`.
