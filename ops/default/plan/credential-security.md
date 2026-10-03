<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Credential Security Delivery Plan

Date: 2026-10-03
Status: Approved
Delivery set: credential-security
Slice strategy: behavior-first
Reason: policy validation, encoded verifier behavior and auth integration each
have independently testable contracts. Introduce complete primitives before
switching the existing service to the new credential path.
Specification: [Authentication security](../spec/authentication-security.md)
Model: [Authentication security model](../spec/authentication-security-model.md)
Tracker: [Credential security tracker](../tracker/credential-security.md)
Base branch: `dev`
Planning base: `6ec140af2825`
Active slice: Slice 3
Execution gate: Open
Go baseline: 1.27.1

## Outcome and Boundary

Deliver AUTH-01 and AUTH-02's credential foundation: Unicode-aware password
policy, bounded disallowed-password checking, self-describing Argon2id records
and the corresponding auth service/storage contracts. Replace the current
bcrypt path without legacy verification, migration or compatibility wrappers.
Keep the work in existing auth, model and configuration boundaries.

This set does not implement the complete authentication-security specification.
Session/challenge, WebAuthn and recovery changes require later bounded delivery
plans. Credential delivery alone cannot establish a secure MFA flow or a complete
application verification level.

## Verified Prerequisites

- The approved behavior and conceptual model define credential ownership and
  implementation obligations, with no legacy datasets or supported APIs to preserve.
- `model/password.go` owns bcrypt hash/compare helpers. `auth.Service.Signup`
  applies a byte-count minimum and configured bcrypt cost; `Signin` verifies
  through the model helper.
- `auth.Queries` has no conditional password-record update. Existing adapters,
  examples and tests must be inventoried before integration changes.
- `golang.org/x/crypto` and `golang.org/x/text` are existing dependencies.
  Password checking uses a caller-supplied source with documented provenance.

## Design Obligations

The current password-only service requires a 15-character default for newly
established passwords. A shorter minimum is valid only after an actual
always-MFA flow exists. No existing password dataset requires acceptance of
the old policy or bcrypt format.

Choose finite character/byte limits, blocklist input limits, cancellation/error
semantics and resource limits before implementing the corresponding primitive.
At least 64 Unicode characters must be supported by new records without
truncation. Define exact input processing/version semantics and use them
consistently for the supported credential format.

Argon2id is the preferred new-record direction. Measure CPU, memory and bounded
concurrent verification before finalizing its default parameters. Bound encoded
record size, salt/output sizes and all work parameters before invoking a KDF.
Cryptographic format and verification use established primitives rather than a
custom password prehash or cipher. FIPS-dependent profiles need separate design.

Disallowed-password checks use a narrow caller-supplied boundary with documented
data provenance and work limits. Unavailable required checking fails explicitly;
an empty/no-op checker cannot be presented as breach-password protection.

Credential writes enforce uniqueness and current-state checks; concurrent changes
cannot overwrite newer state through a stale request. Sign-in cannot issue a
session from a credential/account state invalidated during verification. Define
the generic storage boundary directly rather than adding an optional upgrade
interface. Do not run database operations inside pure model helpers.

Update constructors, queries and configuration when the design requires it.
Update active repository-owned callers, tests and generated examples
together. Remove superseded bcrypt helpers/configuration in the integration
slice; do not preserve their semantics through fallback paths.

## Ordered Slices

| Slice | Short name | Exact branch | Expected PR title | Expected report |
| --- | --- | --- | --- | --- |
| Slice 1 | Password policy | `feat/credential-password-policy` | `feat(slice-1): define bounded password policy` | `ops/default/report/slices/credential-security/slice-1-password-policy.md` |
| Slice 2 | Versioned verifier | `feat/credential-versioned-verifier` | `feat(slice-2): add versioned password verification` | `ops/default/report/slices/credential-security/slice-2-versioned-verifier.md` |
| Slice 3 | Credential integration | `feat/credential-auth-integration` | `feat(slice-3): integrate secure credential storage` | `ops/default/report/slices/credential-security/slice-3-credential-integration.md` |

Each slice branches from current `dev` in its own worktree, has one report and
one PR targeting `dev`. Activate the next slice after integration is verified.

## Slice 1: Password Policy

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.0 | Align the module, generated module/workspace fixtures, CI and documentation with Go 1.27.1. Validate the affected generator packages and rebuild the existing linter with the same toolchain. | `build(go): align modules and CI with Go 1.27` |
| T1.1 | Define and implement password policy validation in auth: valid UTF-8, code-point length, finite input limits, no truncation/composition rules, and explicit password-only versus always-MFA requirements. Define the caller-supplied disallowed-password checker, cancellation and failure contract. | `feat(auth): add bounded password policy` |
| T1.2 | Demonstrate ASCII/Unicode boundaries, common/compromised rejection, checker failure/cancellation and malformed/oversized inputs. Document the API and clarify that the existing signup service is switched only in Slice 3. | `test(auth): verify password policy boundaries` |

### Validation

- `go test ./auth ./config`
- `go test -race ./auth ./config`
- `go test ./generator/execute ./generator/project ./generator/conversation ./generator/interaction ./generator/backend/codex`
- `make docs-check`
- `make vet`
- `make lint-strict`
- `git diff --check`

This slice delivers a complete policy primitive, not new service signup behavior.
Service integration happens in Slice 3; each preparatory slice still compiles
and passes its checks. This sequencing does not require legacy readers in the
new verifier.

## Slice 2: Versioned Verifier

Prerequisite: Slice 1 merged and verified on `dev`.

| Task | Work | Expected commit |
| --- | --- | --- |
| T2.1 | Define a standard encoded Argon2id record and implement salted hash/verify using existing primitives. Validate parameter and record limits before allocation; return classified mismatch, invalid-record and operating outcomes. Measure cost and finalize bounded defaults. | `feat(model): add versioned password verifier` |
| T2.2 | Cover malformed records, unsupported formats/versions, excessive work parameters, distinct salts and correct/incorrect verification. Add parser fuzzing and benchmarks justified by these risks; no bcrypt reader or rehash path. | `test(model): verify credential format boundaries` |

### Validation

- `go test ./model ./crypto ./auth ./config`
- `go test -race ./model ./crypto ./auth ./config`
- Bounded fuzzing of the encoded-record parser; record its exact command/results.
- KDF benchmarks including bounded concurrency; record exact command/results and
  the configuration used before selecting defaults.
- `make vet`
- `make lint-strict`
- `git diff --check`

## Slice 3: Credential Integration

Prerequisites: Slice 2 merged; storage/current-state and configuration contracts
defined.

| Task | Work | Expected commit |
| --- | --- | --- |
| T3.1 | Integrate policy and encoded credentials into signup/sign-in. Define storage/current-state contracts and startup resource configuration. Remove superseded bcrypt helpers/configuration and update affected adapters/tests together; no legacy fallback. | `feat(auth): integrate versioned credential storage` |
| T3.2 | Demonstrate persistent uniqueness/current-state concurrency with a real Postgres adapter, failure behavior, Unicode signup and supported-format sign-in. Update examples, generator consumers and public configuration/API guidance; record completed user-visible behavior in Unreleased. | `test(auth): validate credential integration` |

### Validation

- `go test ./auth ./model ./crypto ./config`
- `go test -race ./auth ./model ./crypto ./config`
- Real Postgres uniqueness/current-state integration; specify the exact repository test
  command in the tracker after the adapter test is defined.
- `go test ./generator/...`
- `go test -run '^$' ./examples/...`
- `make vet`
- `make lint-strict`
- `make docs-check`
- `git diff --check`

## Completion and Handoff

The set completes only after all three slices merge and the exact integrated
`dev` candidate passes `make check`. HatMax has no scheduled nightly, so the
repository-owned local aggregate gate is used once at set closure. Record its
candidate hash and outcome; do not infer release, `main` alignment or tagging.

Map delivered evidence to AUTH-01/AUTH-02 and remaining limitations. Later auth
sets can reuse the credential contracts. Application integration selects a core
dependency deliberately and validates without local workspace overrides.
