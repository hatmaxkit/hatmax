<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Password Policy

Status: reviewing
Delivery set: credential-security
Plan: [Credential security plan](../../../plan/credential-security.md)
Tracker: [Credential security tracker](../../../tracker/credential-security.md)
Branch: `feat/credential-password-policy`
PR: pending

## Purpose

Provide a complete password-candidate policy primitive for AUTH-01. Credential
hashing and service integration remain in Slices 2 and 3.

## Delivered Behavior

- Reject invalid UTF-8 and oversized raw inputs before normalization.
- Return the accepted NFC-normalized candidate without trimming, case folding,
  truncation or character-composition rules.
- Count normalized Unicode code points. Defaults permit 15 through 1024 code
  points and at most 4096 UTF-8 bytes; at least 64 code points must be supported.
- Permit an eight-character minimum only under trusted always-MFA policy.
  Authentication must enforce the required factors separately.
- Require a caller-owned checker for complete disallowed candidates. Checker
  errors, cancellation, deadlines and late successes cannot approve a password.
- Use Go 1.27.1 for the module and generated applications. CI selects the latest
  Go 1.27 patch; its linter is compiled with that toolchain.

## Implementation Notes

Policy configuration is immutable after construction. The checker owns its
source, coverage, provenance and concurrent safety. Input validation precedes
checking; the earlier caller or configured deadline constrains the check.
Cancellation is cooperative and does not create an abandoned goroutine.

Checker error text excludes provider diagnostics. The wrapped cause remains
available through `errors.Is` and `errors.As`; callers must protect diagnostics.
Deadline tests use `testing/synctest` so correctness does not depend on wall-clock
scheduling. Concurrent tests share one policy across 16 requests.

## Contracts Added or Changed

- `PasswordPolicyConfig` defines validated length, byte and timeout limits and
  the trusted MFA requirement.
- `PasswordChecker.Disallowed(ctx, password)` consumes the complete normalized
  candidate and reports rejection or failure.
- `NewPasswordPolicy` requires a checker and rejects invalid configuration.
- `PasswordPolicy.Prepare` returns a normalized candidate or an empty value and
  classified error. Existing `ErrPasswordTooShort` is reused.
- New errors distinguish invalid policy, invalid encoding, oversized candidates,
  disallowed passwords and checker failure. Context errors remain detectable.

## Files of Interest

- [Policy and checker contracts](../../../../../auth/password_policy.go)
- [Boundary and concurrency tests](../../../../../auth/password_policy_test.go)
- [Authentication reference](../../../../../docs/reference/authentication/README.md#password-policy)
- [Generated application baseline](../../../../../generator/execute/render_application.go)

## Validation

Local checks passed with Go 1.27.1 and golangci-lint 2.12.2 built with Go 1.27.1.
The shell used this build configuration:

```sh
export TMPDIR="$PWD/.tmp/build" GOTMPDIR="$PWD/.tmp/build" GOFLAGS=-p=2
```

- `go test ./auth ./config` — passed.
- `go test -race ./auth ./config` — passed.
- `go test ./generator/execute ./generator/project ./generator/conversation ./generator/interaction ./generator/backend/codex` — passed.
- `make docs-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed, zero issues.
- `git diff --check` — passed.

The full delivery-set gate runs after Slice 3 merges into `dev`.

## Risks and Follow-ups

- Checker coverage and cancellation depend on the supplied implementation; this
  primitive does not supply a breach-password corpus or external provider.
- `Service.Signup` and `Signin` retain their current credential path until
  Slice 3. This slice does not implement an MFA flow or a compliance level.
- Slice 2 starts after Slice 1 is merged and verified on `dev`.
