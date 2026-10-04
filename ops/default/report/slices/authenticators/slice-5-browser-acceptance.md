<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 5: Browser Acceptance

Status: delivered
Delivery set: authenticators
Plan: [Delivery plan](../../../plan/authenticators.md)
Tracker: [Delivery tracker](../../../tracker/authenticators.md)
Branch: `test/authenticator-acceptance`
PR: `#102`
Verified merge: `8685ea4b703b1628eda032f960e481ba64b24e2b`
Delivered: 2026-10-04
T5.1: `bdffc01cca3f2cf1b520614429caad2cdecec117`
T5.2: `6f4b5027870569a65bbbbaa4044f766d51d6a1d8`

## Purpose

Establish actual browser integration across the approved authenticator paths,
using production Ticked handlers/core verification and real PostgreSQL. Bind the
five slices' focused evidence to supported behavior and remaining consumer duties.

## Delivered Behavior

- Chromium executes real `navigator.credentials.create/get` with CTAP2/RK/UV.
  Initial enrollment verifies and revokes without issuing a session; real sign-in
  creates the committed phishing-resistant session accepted by the strong route.
- Password-only sign-in cannot enter strong routes or manage factors. WebAuthn
  step-up preserves the actor cookie during begin, replaces it after commit, and
  rejects the retired bearer through the production HTTP route.
- The production management page adds a passkey on a second virtual device.
  Safe listing, owned removal, retained-actor rotation and last-factor denial
  pass against actual persistence; no registration shortcut manufactures proof.
- Actual TOTP setup/sign-in/step-up and backup-code issue/use pass the MFA route.
  Consumed steps/codes reject replay. These weaker methods cannot satisfy the
  phishing-resistant route or the production strong management policy.
- Tampered UP/UV/signature bytes from actual browser assertions, pending tokens
  presented as cookies, wrong-purpose finishes and unknown/non-object/trailing/
  oversized JSON reject. Failures preserve existing cookies; pending/setup
  results confer no access. Secure/HttpOnly/SameSite=Lax metadata and hidden
  script access are checked without printing bearer/code/seed material.
- Reference and User Guide document invocation, mandatory fixtures, supported
  policy, cookie transitions and the virtual-device assurance boundary.

## Implementation Notes

The `browser` build tag is explicitly opt-in. Missing database settings,
executables, secure context or virtual-device support fail acceptance. A random
owned schema receives production migrations 001 and 004 through 007 and is
dropped on cleanup. Successful proof always uses actual production services and
their atomic PostgreSQL adapter; the fixture creates accounts, not verified facts.

Node's built-in WebSocket drives the
[official CDP WebAuthn interface](https://chromedevtools.github.io/devtools-protocol/tot/WebAuthn/).
No npm or Go dependency is added. Chromium uses a separate temporary profile and
owned loopback debugging endpoint. Unix socket scratch has an owned short
`/tmp/hatmax-browser-*` path to respect socket path limits. Browser and fixture
scratch are closed/removed on normal completion. A second USB virtual device
avoids replacing the same subject's resident credential on the first internal
device; Chromium permits only one internal virtual device per environment.

The test-only OTP generator uses the existing OTP library, never approves proof
or writes factors, and exists only inside the tagged test server. Previous-step
setup followed by current-step authentication demonstrates actual consumption
without a thirty-second wait; setup generation avoids an imminent step boundary.
Options are cloned before binary decoding so retries use the same immutable
begin snapshot. Negative assertions alter real browser response bytes before
HTTP submission; signed verifier fixtures remain separate transaction evidence.

The browser fixture retains the production phishing-resistant management
requirement and explicitly selects phishing-resistant factor access to exercise
the last-passkey restriction. Ordinary MFA fallback remains an explicit weaker
access profile under the same trusted policy revision. No runtime API, migration,
proof verifier, dependency pin or application behavior changes in this slice.

### Dependency Review

Stored module source, version requirements and license texts were inspected;
`go mod verify` reports all modules verified. Selected protocol, decoder, OTP and
cryptographic dependencies remain pinned in `go.mod`/`go.sum`:

| Module | Version | Stored license |
| --- | --- | --- |
| `github.com/go-webauthn/webauthn` | v0.18.2 | BSD-3-Clause |
| `github.com/go-webauthn/x` | v0.3.1 | BSD-3-Clause |
| `github.com/fxamacker/cbor/v2` | v2.9.4 | MIT |
| `github.com/x448/float16` | v0.8.4 | MIT |
| `github.com/google/go-tpm` | v0.9.8 | Apache-2.0 |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | MIT |
| `github.com/pquerna/otp` | v1.5.0 | Apache-2.0 |
| `github.com/boombuler/barcode` | v1.1.0 | MIT |
| `golang.org/x/crypto` | v0.57.0 | BSD-3-Clause |

The selected protocol module requires Go 1.26 and names Go 1.27.1 as toolchain;
the repository and validation use Go 1.27.1. Core's finite JSON/CBOR pre-parser,
ES256-only registration, required UV/RK, exact RP/origin/subject binding and
`none` attestation narrow the supported protocol surface. Its signature/replay
tests and actual browser execution establish the selected integration. Checksum
and license inspection do not constitute an independent security audit or a
current vulnerability-advisory scan; ongoing dependency monitoring stays required.

## Contracts Added or Changed

- Required `browser` test prerequisites and finite invocation now provide actual
  acceptance evidence separately from default tests and signed fixtures.
- Test fixtures own their database schema, browser, temporary profile and device
  code generation. Production typed contracts and current consumers are unchanged.
- Documentation binds supported flows to actual cookie/proof outcomes and explains
  hardware, deployment and application assurance duties.

## Files of Interest

- `examples/ticked/internal/web/authenticator_browser_test.go`: owned PostgreSQL,
  real services, production handlers and finite subprocess fixture.
- `examples/ticked/internal/web/testdata/authenticators.mjs`: CDP devices, real
  browser journeys, production management controls and negative cookie/wire paths.
- `docs/reference/authentication/README.md` and
  `docs/tutorials/user-guide/identity-and-sessions.md`: prerequisites and supported
  consumer flows. `REUSE.toml` records the browser fixture's licensing metadata.

## Validation

Go 1.27.1, PostgreSQL 18.6, Chromium 151.0.7922.173 and Node v26.8.1. Owned local
database settings were supplied explicitly; bounded `.tmp/build`, `TMPDIR`,
`GOTMPDIR` and `GOFLAGS=-p=2` were used. Browser socket scratch is separate and
short. The commands below separate local focused checks from the integrated gate;
none is deployment evidence.

- `make source-license-check` — passed, 812 headers and 109 annotations including report introduction.
- `make vet` — passed.
- `make lint-strict` — passed, zero issues.
- `make docs-check` — passed, including local links and example compilation.
- `go test -race ./auth ./config ./crypto ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -race -tags=browser ./examples/ticked/internal/web -run '^TestAuthenticatorBrowser$' -count=1 -timeout=180s -v` — passed; final task browser test 12.78 seconds, seven named journey/evidence outcomes.
- `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(FactorAuthority|FactorChanges|FallbackTransactions|WebAuthnTransactions|EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=180s` — passed, 66.488 seconds.
- `go vet -tags=browser ./examples/ticked/internal/web` — passed.
- `golangci-lint run --build-tags=browser,integration --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./examples/ticked/internal/web/...` — passed with owned lint cache, zero issues.
- `go mod verify` — passed, all modules verified.
- `git diff --check` — passed.

### Integrated Gate

After verified PR #102 merge and delivered-slice metadata closure, `make check`
passed once on exact integrated `dev` candidate
`29d5720aa506140e521fe852a37630fa5057499b` on 2026-10-04. Go 1.27.1 and an
isolated owned PostgreSQL 18.6 database were used. Source licensing (812 headers,
109 annotations), format, vet, the complete default suite, total coverage 81.8%
(80% required) and strict lint (zero issues) passed. Tracked formatting remained
unchanged. The temporary gate database was dropped and the owned cluster stopped
after normal completion. No correction branch or repeated aggregate run was
required. Acta: `01M43YXBA25Z1J9CWPS2E3RYEK`.

These results are separate from the actual browser and tagged PostgreSQL race
checks above. Subsequent documentary closure does not change tested runtime code
or claim another commit hash as the tested candidate.

### Acceptance Mapping

| Contract | Actual evidence and boundary |
| --- | --- |
| AU-01 | `TestEnrollmentVerification`, `TestEnrollmentTransactions`, `TestWebAuthnTransactions` protocol/signature/binding rejection; real navigator registration/assertion and tampered-response browser denial |
| AU-02 | `TestProofEvaluation`, `TestWebAuthnProof`, `TestCompositeFreshness`; actual browser password/WebAuthn/TOTP/backup routes and stronger-policy denial |
| AU-03 | Purpose/token tests, HTTP finish routing and captured-state PostgreSQL conflicts; pending-as-cookie and unchanged begin/failure cookies in Chromium |
| AU-04 | Real PostgreSQL concurrent winners/rotation/counters/steps/codes and complete forced rollback in WebAuthn/fallback/control transaction tests |
| AU-05 | `TestSeedBinding`, `TestTOTPCompletionWindow`, `TestFallbackTransactions` actual encrypted setup, matched steps, lock/write time and replay; actual browser TOTP sign-in/step-up |
| AU-06 | Real fallback salted verifier/owner/ID, reuse/regeneration/rollback and concurrent code-use evidence; actual browser issue/use/reuse and strong-route denial |
| AU-07 | `TestFactorChanges`, `TestFactorAuthority`, initial enrollment tests; real management-page add, current factor listing/removal, actor rotation and last-factor denial |
| AU-08 | Constructor/parser/admission tests, PostgreSQL budget/retention/cleanup and post-lock/write/canceled-wait checks; browser JSON bounds. Finite parser fuzz evidence is recorded in delivered Slice 1–4 reports |
| AU-09 | Mandatory `TestAuthenticatorBrowser`, production Ticked handlers/core services, actual navigator/device operations and real PostgreSQL; no skipped prerequisite or proof callback |
| AU-10 | Current consumers/example compilation, reference/User Guide, selected dependency review and focused default/tagged checks passed. Exact integrated `make check` passed on `29d5720aa506140e521fe852a37630fa5057499b`, 81.8% total coverage and zero strict-lint issues |

The supported AUTH-03 factor-completion and AUTH-05 mechanism profiles now have
actual browser, verifier, persistence and race evidence. The complete delivery
set is closed after its exact immutable integrated `dev` candidate
`29d5720aa506140e521fe852a37630fa5057499b` passed the approved aggregate gate.

## Risks and Follow-ups

- PR #102 is merged and its exact integrated gate passed. No main alignment,
  release, tag, mirror publication or deployment ran.
- Virtual devices establish integration rather than hardware identity,
  non-exportability, attestation trust or AAL/compliance certification. Consumers
  own trusted RP/origin/access policy, real database atomicity, key custody, HTTPS,
  hardware evidence and application-domain authorization.
- AUTH-06 mailbox/password/all-factor recovery and AUTH-07 general abuse/event
  infrastructure remain separate concerns. Finite endpoint budgets delivered
  here remain mandatory; they do not imply those broader concerns are complete.
