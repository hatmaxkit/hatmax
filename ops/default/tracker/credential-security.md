<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Credential Security Tracker

Date: 2026-10-03
Status: Completed
Delivery set: credential-security
Plan: [Credential security plan](../plan/credential-security.md)
Specification: [Authentication security](../spec/authentication-security.md)
Model: [Authentication security model](../spec/authentication-security-model.md)
Base branch: `dev`
Planning base: `6ec140af2825`
Active slice: None
Active tasks: None
Execution gate: Closed

## Slice Status

| Slice | Short name | Status | Branch | PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Password policy | delivered | `feat/credential-password-policy` | `feat(slice-1): define bounded password policy` | `#92` | `ops/default/report/slices/credential-security/slice-1-password-policy.md` |
| Slice 2 | Versioned verifier | delivered | `feat/credential-versioned-verifier` | `feat(slice-2): add versioned password verification` | `#93` | `ops/default/report/slices/credential-security/slice-2-versioned-verifier.md` |
| Slice 3 | Credential integration | delivered | `feat/credential-auth-integration` | `feat(slice-3): integrate secure credential storage` | `#94` | `ops/default/report/slices/credential-security/slice-3-credential-integration.md` |

## Tasks

| Task | Status | Expected commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.0 | completed | `build(go): align modules and CI with Go 1.27` | `7ca151646cf3` | Go 1.27.1; generator tests and documentation checks passed |
| T1.1 | completed | `feat(auth): add bounded password policy` | `ec3c7939b3ab` | Policy and checker contracts; auth/config tests and vet passed |
| T1.2 | completed | `test(auth): verify password policy boundaries` | `af94c2ab0c6d` | Unicode/limits/checker failure and cancellation; race and strict lint passed |
| T2.1 | completed | `feat(model): add versioned password verifier` | `6b507e82d78c` | PHC Argon2id, strict bounds, classified outcomes, measured defaults and shared concurrency budget |
| T2.2 | completed | `test(model): verify credential format boundaries` | `194b42196363` | Reference interoperability, Unicode, cancellation, concurrency, 20-second fuzz, benchmarks, race and static checks passed |
| T3.1 | completed | `feat(auth): integrate versioned credential storage` | `500c9228c6bb` | Signup/sign-in integration, replaced storage/configuration contracts and current-state checks |
| T3.2 | completed | `test(auth): validate credential integration` | `abcca95738da` | `DB_HOST=<test-host> DB_PORT=<port> DB_USER=<user> DB_NAME=<database> go test -tags=integration -race -count=1 -run '^TestCredentialTransactions$' ./examples/ticked/internal/feat/auth`; examples/generator, API documentation and Slice 3 gate |

## Dependencies

PR #92 merged on 2026-10-03. Integration was verified on `dev` at
`374609d914a24a5ab82be911ed2320188c2f827e`; Slice 1 is delivered.

Slice 1 validation passed with Go 1.27.1 and golangci-lint 2.12.2 built with
the same toolchain. The [Slice 1 report](../report/slices/credential-security/slice-1-password-policy.md)
records commands and behavior boundaries.

Slice 2 validation passed. Its [report](../report/slices/credential-security/slice-2-versioned-verifier.md)
records measured defaults, exact commands and remaining service integration.
PR #93 merged on 2026-10-03. Integration was verified on `dev` at
`0d0ca4c50972af01dc50a7c844a2bc4a70ca6187`; Slice 2 is delivered. Slice 3 also requires the storage/current-state and configuration contracts. Test infrastructure failures are recorded
without treating missing persistence evidence as a passing check.

Slice 3 merged in PR #94 on 2026-10-03. Integration was verified on `dev` at
`d3e27373930d4e188c5f27413f6f1822b469986c`. Its
[report](../report/slices/credential-security/slice-3-credential-integration.md)
records required storage semantics and real PostgreSQL concurrency evidence.
The integrated set gate passed for that exact candidate; all three slices are delivered.

## Completion Gates

- [x] Slice 1 merges and its delivered report/commit evidence is recorded.
- [x] Slice 2 merges and its delivered report/commit evidence is recorded.
- [x] Storage/current-state and configuration contracts are settled before Slice 3.
- [x] Slice 3 merges and its delivered report/commit evidence is recorded.
- [x] Exact integrated `dev` candidate passes `make check`.
- [x] AUTH-01/AUTH-02 evidence and downstream handoff are recorded.

Completing this set does not mark AUTH-03 through AUTH-07 or the downstream
application account/workspace capability delivered.

## Integrated Validation

Candidate: `d3e27373930d4e188c5f27413f6f1822b469986c`
Gate: `make check` — passed on 2026-10-03 with Go 1.27.1, golangci-lint 2.12.2
and an isolated PostgreSQL 18.6 test database configured through the repository's
`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` and `DB_NAME` variables.
The gate ran source licensing, formatting, vet, the complete default test suite,
coverage and strict lint. Coverage was 84.6% (required: 80%); lint reported zero
issues. Tracked runtime files remained unchanged after formatting.

Build settings were `TMPDIR=$PWD/.tmp/build`, `GOTMPDIR=$PWD/.tmp/build` and
`GOFLAGS=-p=2`. This is local integrated evidence; optional tagged/live acceptance
targets and production validation are not part of `make check`. The real credential
transaction/race tests were completed in Slice 3 and are recorded in its report.

## Requirement Evidence and Handoff

| Requirement | Delivered evidence | Remaining consumer or later-work obligation |
| --- | --- | --- |
| AUTH-01 | Slice 1's bounded Unicode/NFC policy and fail-closed checker; Slice 3's signup enforcement, startup validation and Unicode persistence tests | Select/document common, compromised and application-specific checker coverage; integrate the same primitive into later change/reset workflows; shorter policy requires an actual always-MFA flow |
| AUTH-02 | Slice 2's interoperable PHC Argon2id format, independent salts, bounded parser/KDF and measured admission settings; Slice 3's conditional current-state/session/replacement transactions | Every downstream adapter enforces atomic storage/version semantics; deployments select resource budgets; any FIPS profile requires separate design |

The credential-security set is complete. AUTH-03 through AUTH-07 remain pending
outside this set. Downstream applications can consume these documented contracts
through a deliberately selected published dependency; no local workspace override,
application implementation, main alignment or release is implied by this closure.
