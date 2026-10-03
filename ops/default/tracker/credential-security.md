<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Credential Security Tracker

Date: 2026-10-03
Status: Active
Delivery set: credential-security
Plan: [Credential security plan](../plan/credential-security.md)
Specification: [Authentication security](../spec/authentication-security.md)
Model: [Authentication security model](../spec/authentication-security-model.md)
Base branch: `dev`
Planning base: `6ec140af2825`
Active slice: Slice 1
Active tasks: None
Execution gate: Review

## Slice Status

| Slice | Short name | Status | Branch | PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Password policy | reviewing | `feat/credential-password-policy` | `feat(slice-1): define bounded password policy` | pending | `ops/default/report/slices/credential-security/slice-1-password-policy.md` |
| Slice 2 | Versioned verifier | planned | `feat/credential-versioned-verifier` | `feat(slice-2): add versioned password verification` | pending | `ops/default/report/slices/credential-security/slice-2-versioned-verifier.md` |
| Slice 3 | Credential integration | planned | `feat/credential-auth-integration` | `feat(slice-3): integrate secure credential storage` | pending | `ops/default/report/slices/credential-security/slice-3-credential-integration.md` |

## Tasks

| Task | Status | Expected commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.0 | completed | `build(go): align modules and CI with Go 1.27` | `7ca151646cf3` | Go 1.27.1; generator tests and documentation checks passed |
| T1.1 | completed | `feat(auth): add bounded password policy` | `ec3c7939b3ab` | Policy and checker contracts; auth/config tests and vet passed |
| T1.2 | completed | `test(auth): verify password policy boundaries` | `af94c2ab0c6d` | Unicode/limits/checker failure and cancellation; race and strict lint passed |
| T2.1 | pending | `feat(model): add versioned password verifier` | pending | Encoded format, parameter bounds, measured KDF cost |
| T2.2 | pending | `test(model): verify credential format boundaries` | pending | Unsupported-format rejection, parser fuzzing, salts and Slice 2 gate |
| T3.1 | pending | `feat(auth): integrate versioned credential storage` | pending | Signup/sign-in integration, replaced storage/configuration contracts and current-state checks |
| T3.2 | pending | `test(auth): validate credential integration` | pending | Real Postgres concurrency, examples/generator, API documentation and Slice 3 gate |

## Dependencies

Slice 1 validation passed with Go 1.27.1 and golangci-lint 2.12.2 built with
the same toolchain. The [Slice 1 report](../report/slices/credential-security/slice-1-password-policy.md)
records commands and behavior boundaries.

Slice 2 depends on delivered Slice 1. Slice 3 also requires the storage/current-
state and configuration contracts. Test infrastructure failures are recorded
without treating missing persistence evidence as a passing check.

## Completion Gates

- [ ] Slice 1 merges and its delivered report/commit evidence is recorded.
- [ ] Slice 2 merges and its delivered report/commit evidence is recorded.
- [ ] Storage/current-state and configuration contracts are settled before Slice 3.
- [ ] Slice 3 merges and its delivered report/commit evidence is recorded.
- [ ] Exact integrated `dev` candidate passes `make check`.
- [ ] AUTH-01/AUTH-02 evidence and downstream handoff are recorded.

Completing this set does not mark AUTH-03 through AUTH-07 or the downstream
application account/workspace capability delivered.
