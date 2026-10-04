<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 3: TOTP and Backup Proof

Status: delivered
Delivery set: authenticators
Plan: [Delivery plan](../../../plan/authenticators.md)
Tracker: [Delivery tracker](../../../tracker/authenticators.md)
Branch: `feat/authenticator-fallback`
PR: `#100`
Merged: 2026-10-04
Merge commit: `a2de0f3cde3f596342979da949902487d13605ce`
T3.1: `acc2881fa63335b5f15540955e24ed7aecb1ae95`
T3.2: `56af95e82bdc272815c1571db1228a6616e3cf13`
Tagged test formatting: `54d75553de6ed8e00616d1cc9bd07d48485f7c18`

## Purpose

Complete actual password/TOTP and password/backup MFA under explicit application
policy. Accepted steps/codes and pending consumption commit with session insertion
or bound rotation. These fallback methods never supply phishing-resistant proof.

## Delivered Behavior

- Initial TOTP setup verifies an actual fresh password and requires no established
  factor. Confirmation consumes the actual matched step, advances account version,
  and revokes all sessions/pending without granting application access.
- Seeds use AES-256-GCM with externally supplied 32-byte keys and explicit retained
  identities. Associated data binds subject, factor, purpose/version and key identity;
  ciphertext substitution or unavailable keys fail closed.
- TOTP uses the unchanged pinned OTP library's SHA-1/six-digit/30-second profile.
  Trusted-time verification returns the matched integer step; skew is at most one.
  Final completion requires a current-window step strictly greater than accepted
  state and matches both security and replay revisions.
- Backup codes contain 128 random secret bits and a canonical public identifier.
  Subject/identifier lookup selects one independently salted, purpose/owner/ID-bound
  PHC Argon2id verifier through the credential service's shared bounded engine.
- Both methods capture actual password time, current account/factor/policy/skew,
  finite pending expiry and optional current actor digest/ID/generation. Durable
  shared admission is spent before factor verification and survives failed proof
  and pending reissue. Expired-only retention/cleanup remain bounded.
- Composite proof records both constituent times and evaluates freshness from the
  oldest actual fact. Routine step/code use preserves bound sessions; security
  revision or set changes invalidate them. Password-only reauthentication does
  not preserve or renew factor authority.
- Backup set issue/replacement requires actual recent management MFA or stronger,
  excludes password-only/code actors, and commits set replacement, account version,
  revocation and retained actor rotation together. Plaintexts return once after
  commit; retained proof times and session expiry remain unchanged.
- Ticked exposes bounded same-origin JSON routes with captured server policy,
  purpose-specific pending tokens, `no-store` responses, cookie issue after commit,
  and separate MFA/phishing-resistant proof routes. Failed work preserves cookies.

## Implementation Notes

`NewFallbackService` requires typed `FallbackQueries`, explicit finite
`FallbackConfig` and an owned seed key ring; there is no optional store upgrade,
untyped transaction or caller proof callback. Admission slots and cryptographic
costs are bounded. OTP-only configuration does not require a WebAuthn RP.

TOTP has one typed record per subject, counted with WebAuthn under the subject
lock. Backup sets have one active identity per subject and at most ten codes.
The additive migration 006 extends the shared pending purpose/shape constraints
and closed session proof with typed TOTP/set/code tables. SQL source, SQLC mappings,
current proof validation and production wiring are synchronized. No dependency
version changed; `github.com/pquerna/otp v1.5.0` remains pinned.

Final transactions lock subject before pending/actor and material, use fresh
post-lock/post-write time, conditionally match verifier snapshots, and atomically
consume one-use state with session completion. Forced SQL failures roll back
completion while the separate durable reservation stays charged. Backup generation
also reserves the shared budget before its finite sequence of hashes.

Ticked enables fallback routes only with both explicit `TICKED_TOTP_KEY_ID` and
canonical standard-Base64 `TICKED_TOTP_KEY` encoding 32 key bytes. Neither supplied
leaves these routes disabled; partial/invalid configuration fails initialization.
Its fallback access policy allows MFA, while backup management defaults to recent
phishing-resistant MFA. An application may explicitly allow recent MFA management
for an approved TOTP-only profile; HTTP cannot weaken that policy.

## Contracts Added or Changed

- `FallbackService`: initial TOTP begin/confirm, actual password/factor authentication,
  bound step-up, atomic finish, authorized code-set issue and finite pending cleanup.
- Typed pending/reservation/completion capture policy, method/skew, security/replay
  identity, account version, actual password time and optional actor generation.
- Closed `PasswordTOTPProof` / `PasswordBackupProof` bind factor/set identity and
  revision; `VerifiedAt` is oldest password time and `FactorAt` actual factor time.
- The hidden-clock boolean OTP and shared-salt/index backup APIs are removed.
  New primitives require supplied trusted time, canonical code parsing and durable
  step/code consumption through actual completion.
- Provisioning URL and raw codes are immediate secret results, never user/session
  metadata. AES key identities are printable 1..64 bytes, key rings contain 1..8
  32-byte keys; OTP skew is 0..1, backup issue count 1..10 (default 8).
- Wire code is exactly 67 ASCII bytes with strict canonical Base64url/UUID shape.
  HTTP code is at most 128 bytes, email 254 and password 4096; JSON body at most
  64 KiB. Unknown fields, non-object bodies and trailing values reject.
- API/configuration reference, package notes, User Guide and Unreleased describe
  actual lower-assurance access and migration/consumer requirements.

## Files of Interest

- [Fallback service](../../../../../auth/fallback.go),
  [seed encryption](../../../../../auth/seed.go) and [proof](../../../../../auth/proof.go).
- [OTP primitive](../../../../../crypto/totp.go) and
  [canonical backup secrets](../../../../../crypto/backup.go).
- [Atomic adapter](../../../../../examples/ticked/internal/feat/auth/fallback.go) and
  [additive migration](../../../../../examples/ticked/assets/migration/postgres/006-fallback-proof.sql).
- [Real transaction evidence](../../../../../examples/ticked/internal/feat/auth/fallback_integration_test.go).
- [Production HTTP](../../../../../examples/ticked/internal/web/fallback.go) and
  [reference](../../../../../docs/reference/authentication/README.md#totp-and-backup-proof).

## Validation

Focused local checks passed on Go 1.27.1. Real PostgreSQL uses an owned 18.6
cluster with explicit connection settings; missing `DB_HOST` fails. Build/lint
scratch and Go parallelism were bounded. The owned cluster was stopped after
all tests completed; unrelated services and retained worktrees were preserved.

- `go test ./auth ./config ./crypto ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web`
- `go test -race ./auth ./config ./crypto ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web`
- `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(FallbackTransactions|WebAuthnTransactions|EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=180s` — passed, final regression 54.089 seconds.
- `go test ./crypto -run '^$' -fuzz '^FuzzBackupCode$' -fuzztime=20s -parallel=2 -timeout=60s` — 920632 executions, no failure, no KDF/database work.
- `make source-license-check` — 799 headers, 107 content-preserving annotations before report introduction.
- `make vet`
- `make lint-strict` — zero issues.
- `golangci-lint run --build-tags=integration --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./auth/... ./config/... ./crypto/... ./examples/ticked/...` — zero issues.
- `make docs-check` — includes example compilation.
- `git diff --check`

Actual verification/PostgreSQL proves setup-step consumption and no setup access,
current security/replay/account/policy/lease/key bindings, one TOTP/code winner,
one actor rotation/set replacement winner, reuse denial, shared durable exhaustion
across reissue, current set ownership, independently salted domain-bound records,
out-of-budget PHC rejection, expired-only cleanup and unchanged proof after routine
replay updates. SQL failures preserve accepted step/code/pending and the old set/actor;
successful retry re-verifies actual material. An actual row lock and delayed SQL
write separately expire oldest proof and preserve unconsumed code state.

Exact supplied-time unit cases establish step/skew/expiry equality and actual
matched-code rejection at the next completion-window boundary. Key tests cover
nonce independence, ownership/material/version/key substitution and caller key
mutation. Existing shared verifier admission/concurrency/cancellation tests pass
with race detection; fallback admission fails before storage when full. HTTP-only
fakes independently prove method/policy/purpose/cookie/origin/body isolation,
including the non-object-command regression; they are not cryptographic evidence.
AU-02 through AU-06 and affected AU-08/AU-10 have focused evidence.

## Risks and Follow-ups

Established-factor addition/removal/replacement and fuller management presentation
remain Slice 4; real production browser journeys remain Slice 5. Current default
Ticked backup management requires phishing-resistant proof, so TOTP-only consumers
must explicitly select an allowed recent-MFA management profile. Seed key retention,
secure provisioning/code delivery, TLS and application authorization are consumer
obligations. No hardware/assurance certification is inferred. No full delivery-set
`make check` ran; the aggregate gate remains after Slice 5 merge. Main alignment,
release, tags, deployment and general account/all-factor recovery are outside
this delivery.
