<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authenticator Tracker

Date: 2026-10-04
Status: Completed
Approved: 2026-10-04
Completed: 2026-10-04
Delivery set: authenticators
Plan: [Delivery plan](../plan/authenticators.md)
Concern: [Authenticator proposal](../spec/authenticators.md)
Model: [Authenticator model](../spec/authenticators-model.md)
Parent: [Authentication security foundation](../spec/authentication-security.md)
Base branch: `dev`
Planning base: `247487f273a54a8e65a8e6b5f86191f38bcb0be8`
Active slice: None
Active tasks: None
Execution gate: Closed

## Slice Status

| Slice | Short name | Status | Branch | PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Restricted enrollment | delivered | `feat/authenticator-enrollment` | `feat(slice-1): add restricted authenticator enrollment` | `#98` | `ops/default/report/slices/authenticators/slice-1-enrollment.md` |
| Slice 2 | WebAuthn completion | delivered | `feat/webauthn-completion` | `feat(slice-2): complete WebAuthn authentication and step-up` | `#99` | `ops/default/report/slices/authenticators/slice-2-webauthn-completion.md` |
| Slice 3 | TOTP and backup proof | delivered | `feat/authenticator-fallback` | `feat(slice-3): add replay-resistant TOTP and backup proof` | `#100` | `ops/default/report/slices/authenticators/slice-3-fallback-proof.md` |
| Slice 4 | Authorized factor changes | delivered | `feat/authenticator-control` | `feat(slice-4): enforce authorized authenticator changes` | `#101` | `ops/default/report/slices/authenticators/slice-4-factor-control.md` |
| Slice 5 | Browser acceptance | delivered | `test/authenticator-acceptance` | `test(slice-5): verify authenticator browser integration` | `#102` | `ops/default/report/slices/authenticators/slice-5-browser-acceptance.md` |

## Tasks

| Task | Status | Expected commit | Commit | Required evidence |
| --- | --- | --- | --- | --- |
| T1.1 | complete | `feat(auth): add restricted WebAuthn enrollment` | `776b4aca0ab80318fa11d86035684d00efca2b5a` | Reviewed dependency/profile; typed bounded pending/budget/setup, actual registration and synchronized adapter |
| T1.2 | complete | `test(auth): verify enrollment isolation and bounds` | `f5b62edccb970f2af7e20f80d6f77f1890a3b437` | Protocol rejection, pending isolation, durable admission/budgets and real PostgreSQL setup races |
| T2.1 | complete | `feat(auth): complete WebAuthn proof atomically` | `f7df235f4411be7a2bb7eaa0f211077295edba61` | Actual assertion, closed facts and atomic counter/pending/session insertion or rotation |
| T2.2 | complete | `test(auth): verify WebAuthn completion transactions` | `1b01ea4ba0a64a87705996b37320e745cd20d863` | Real signatures/one winner, current counter/flags, post-lock time and full rollback |
| T3.1 | complete | `feat(auth): add replay-resistant fallback proof` | `acc2881fa63335b5f15540955e24ed7aecb1ae95` | Encrypted TOTP, exact accepted steps and salted domain-bound one-use backup proof |
| T3.2 | complete | `test(auth): verify fallback replay and policy` | `56af95e82bdc272815c1571db1228a6616e3cf13` | Trusted-time/step and code races, salts/KDF, no weak-policy substitution and durable budget exhaustion |
| T4.1 | complete | `feat(auth): enforce authorized factor changes` | `1f7b73b55d3511d5df9cba62a7a4472c5962842a` | Recent trusted authority, bounded own-factor operations, version/revocation and coherent actor rotation |
| T4.2 | complete | `test(auth): verify authenticator change authority` | `b7a2d6ed1fde75ee861084f0e102109d05289308` | Subject isolation, post-lock freshness, last-factor/current-proof constraints, mutation races and rollback |
| T5.1 | complete | `test(auth): exercise authenticator browser journeys` | `bdffc01cca3f2cf1b520614429caad2cdecec117` | Actual browser/virtual device against production handlers, real PostgreSQL and required stronger proof |
| T5.2 | complete | `test(auth): close authenticator integration evidence` | `6f4b5027870569a65bbbbaa4044f766d51d6a1d8` | Cookie/parser/deadline regression, supported guidance and AU/AUTH evidence mapping |

## Dependencies and Execution Gate

Credential security and authentication sessions are delivered on canonical `dev`.
The concern/model, selected dependency/profile/defaults and five-slice plan were
approved on 2026-10-04. The concern/model are promoted together; Slice 1/T1.1
is delivered. Slices 1-5 are delivered after verified PR #102 merge. All tasks and delivered reports are complete; the exact integrated aggregate gate passed.

Before T1.1 settle matching Go/SQL/wire fields and exact pinned verifier/key
dependencies. Before T1.2 record actual finite integration/fuzz selectors;
before T5.1 record the real browser acceptance invocation/prerequisites.
Use actual protocol/TOTP/KDF verification and real persistence. Pending labels,
setup flags and fake method claims never count as authenticator acceptance.

## Completion Gates

- [x] Concern/model/dependency/profile/defaults and plan reviewed; concern promoted and Slice 1 activated.
- [x] Slice 1 merged; delivered enrollment report and real setup/budget evidence recorded.
- [x] Slice 2 merged; delivered WebAuthn completion/step-up and atomic replay evidence recorded.
- [x] Slice 3 merged; delivered TOTP/backup verification, replay and policy evidence recorded.
- [x] Slice 4 merged; delivered current-authority/factor-change evidence recorded.
- [x] Slice 5 merged; delivered real browser integration and supported documentation recorded.
- [x] AU-01 through AU-10 mapped to actual deterministic, verifier, race, persistence and browser evidence.
- [x] Exact immutable integrated `dev` candidate passes `make check`.
- [x] Supported AUTH-03/AUTH-05 coverage and remaining AUTH-06/AUTH-07/application assurance obligations recorded.

## Planning Validation

- `make docs-check` — passed, including local links and example compilation.
- `make source-license-check` — passed: 764 headers, 104 content-preserving annotations.
- `git diff --check` — passed.
- Spec/model fields, operation purposes, proof authority, slice/task ordering,
  exact branches/titles/report paths and closed execution state — consistent.
- The four proposal artifacts contain no outer-product references.

No runtime, browser or full delivery-set evidence is claimed for this planning
proposal. No dependency has been added and previous completed set gates are not
repeated.

## Slice 1 Focused Validation Bindings

- Protocol/parser fuzz: `go test ./auth -run '^$' -fuzz '^FuzzEnrollmentResponse$' -fuzztime=20s -parallel=2 -timeout=60s`; no KDF/database work in the target.
- Real PostgreSQL: `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=120s`; `DB_HOST` and explicit owned-cluster settings are required.
- T1.1 passed focused auth/config/adapter/HTTP tests, real PostgreSQL registration/replay/revocation, strict lint, licensing, documentation and diff checks. No aggregate delivery-set gate ran.

T1.2 passed protocol rejection/binding tests, finite constructor/HTTP bounds,
actual registration with both supported backup classes, 20-second parser fuzz
(354797 executions), real PostgreSQL races/rollback/budgets/cleanup/uniqueness and
credential/session regression, affected-package race checks, vet, strict lint,
source licensing and docs/example compilation. Pending replay issues no session.
The schema uses a new `004-authenticators.sql` migration for existing databases.
The complete set's integrated gate remains scheduled after Slice 5 merge.

Slice 1 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/98
Report introduction: `3f04dc1c00fce625ead88c7611b97da62176a897`.
Verified PR #98 merge: `d916a27f53e2d0d42d67aed246d018194aacbfcf`.
Next checkpoint: review/merge Slice 2, verify canonical state, then activate the already-approved Slice 3.

## Slice 2 Focused Validation Bindings

- Parser fuzz: `go test ./auth -run '^$' -fuzz '^FuzzAssertionResponse$' -fuzztime=20s -parallel=2 -timeout=60s`; no KDF/database work.
- Real PostgreSQL: `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(WebAuthnTransactions|EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=120s`; explicit owned-cluster connection settings required.
- T2.1 passed actual signed authentication/step-up with PostgreSQL, package and regression races, strict default/tagged lint, source licensing and docs/example compilation. T2.2 passed signed negative protocol, replay/backup/counter/ownership conflicts, post-lock/post-write time checks, durable admission and full rollback evidence. Parser fuzz completed 484362 executions without failure; PostgreSQL regression races, package races, vet, strict lint, licensing and documentation checks passed.


Slice 2 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/99
Report introduction: `379370c2b25b63cc1726ed6a8af88984a179baab`.
Verified PR #99 merge: `3d451cf7a0ba7eaf622ec732cc388c379062b3f2`.
Next: review/merge Slice 3, verify canonical state, then activate approved Slice 4.

## Slice 3 Focused Validation Bindings

- Parser fuzz: `go test ./crypto -run '^$' -fuzz '^FuzzBackupCode$' -fuzztime=20s -parallel=2 -timeout=60s`; no KDF/database work.
- Real PostgreSQL: `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(FallbackTransactions|WebAuthnTransactions|EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=180s`; explicit owned-cluster settings required, missing `DB_HOST` fails.
- T3.1 passed actual TOTP setup/MFA, backup issue/use/reuse denial, PostgreSQL regression with race detection, focused package races, vet, strict lint, documentation/example compilation and source licensing.

T3.2 passed actual replay/rotation/regeneration races, SQL rollback, current factor/account/policy/key conflicts, oldest-proof expiry after real locks/writes, shared budget reissue exhaustion, salted owner/ID-bound PHC records, cleanup and transport isolation. Parser fuzz completed 920632 executions without failure. Full named PostgreSQL regression passed with race detection (54.089 seconds); package races, vet, strict default/tagged lint, licensing and docs/example compilation passed. Tagged formatting correction: `54d75553de6ed8e00616d1cc9bd07d48485f7c18`.

Slice 3 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/100
Report introduction: `70564689cf5858754e3d7072074dfae41838ccce`.
Next: verify maintainer merge, close Slice 3 on `dev`, then implement approved Slice 4/T4.1/T4.2.
The complete set's aggregate gate remains after Slice 5 merge.

Verified PR #100 merge: `a2de0f3cde3f596342979da949902487d13605ce`.
Next: execute approved Slice 4/T4.1/T4.2.

## Slice 4 Focused Validation Bindings

- Real PostgreSQL: `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(FactorAuthority|FactorChanges|FallbackTransactions|WebAuthnTransactions|EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=180s`; explicit owned-cluster settings required, missing `DB_HOST` fails.
- Parser/policy fuzz: `go test ./auth -run '^$' -fuzz '^FuzzFactorSelection$' -fuzztime=20s -parallel=2 -timeout=60s`; no KDF/database work.

T4.1 passed actual additional/replacement WebAuthn and TOTP, TOTP-only upgrade under explicit MFA, last-factor denial, all-session/pending invalidation and coherent actor rotation with PostgreSQL/race detection. Full named PostgreSQL regression passed (57.743 seconds); focused package races, vet, strict default/tagged lint, licensing and docs/example compilation passed. No aggregate gate ran.

T4.2 passed current owner/RP/actor/account/factor/target/policy/lease conflicts,
actual competing completion/removal races, last usable factor policy with available
key identities, post-lock and post-write proof expiry (including the oldest actual
password/TOTP constituent), forced activation/version/session rollback and durable
admission. Runtime checks now restrict listing/targets to the configured RP and
count only current-RP WebAuthn or available-key TOTP as remaining usable primary
factors. Actor-supplied key identities are replaced with the core key-ring snapshot.
Selector fuzz completed 387509 executions in 20 seconds without failure. Final
named PostgreSQL regression passed with race detection (71.093 seconds); focused
package races, HTTP policy/body/origin/cookie tests, vet, strict default/tagged lint,
licensing and docs/example compilation passed. No aggregate gate ran.

Slice 4 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/101
Report introduction: `ddc2c4747dbd04ee76562dd081c039b19f428281`.
Next: verify maintainer merge, close Slice 4 on `dev`, then execute already-approved Slice 5/T5.1/T5.2.


Verified PR #101 merge: `e9563e94cb7b4ff032895fad15f96b083d1ac7e8`.
Slice 5/T5.1/T5.2 is delivered; PR #102 is merged.

## Slice 5 Browser Validation Binding

- Command: `go test -race -tags=browser ./examples/ticked/internal/web -run '^TestAuthenticatorBrowser$' -count=1 -timeout=180s`.
- Mandatory database settings: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_NAME` and
  explicitly supplied `DB_PASSWORD` (empty is permitted for the owned local
  socket cluster). A randomly named owned schema receives production migrations
  and is dropped on normal cleanup; no shared schema is modified.
- Mandatory executables: `CHROMIUM_BIN=/usr/sbin/chromium` and
  `NODE_BIN=/usr/sbin/node`. Verified Chromium 151.0.7922.173 (Arch Linux),
  Node v26.8.1 and its built-in WebSocket implementation; no npm dependency.
- CDP virtual device: CTAP2, internal and second-device USB transport, resident key, user verification,
  automatic presence and verified user. ES256/UV/RK are the production profile.
- The finite harness starts an owned headless browser with an isolated temporary
  profile and connects only to its loopback debugging endpoint. Browser socket
  scratch uses an owned short `/tmp/hatmax-browser-*` path because Unix socket
  names cannot use the longer worktree build path; normal cleanup removes it. Production Ticked
  handlers/core services and real PostgreSQL perform every proof completion.
- Localhost HTTP is explicitly enabled for development in the trusted RP/origin
  configuration. Test-only OTP generation uses the existing OTP library and never
  approves proof; production setup/finish verifies and consumes the actual code.
- Missing executables, database settings, secure context or virtual device fail
  acceptance. Signed fixtures and skipped checks cannot count as browser evidence.
- Default strict lint and separately tagged browser/integration strict lint are
  required before the task commit. The aggregate gate remains after Slice 5 merge.


T5.1 passed actual Chromium 151.0.7922.173 navigator registration and assertion,
password-to-WebAuthn step-up, retired-actor denial, the production add-passkey
control with a second virtual device, safe listing/removal/last-factor checks,
real backup use/reuse denial, and TOTP sign-in/step-up/replay. Browser acceptance
passed with race detection (12.72 seconds test time); package races, vet,
strict default/tagged lint, source licensing and docs/example compilation passed.
No runtime verifier or persistence behavior changed; no dependency was added.


T5.2 browser regression passed with race detection (12.78 seconds test time):
tampered actual browser UP/UV/signature responses, pending-as-cookie, wrong-purpose
finish, unknown/non-object/trailing/oversized JSON and failure cookie preservation.
Browser success still requires actual verification, PostgreSQL and committed proof.
The production backup handler is bound to recent phishing-resistant management;
MFA fallback cannot weaken it. Tagged PostgreSQL regression passed with race
detection (66.488 seconds), including actual lock/write expiry, canceled waits,
replay/mutation races and forced completion rollback. `go mod verify` passed;
selected protocol/OTP/key dependencies and their stored licenses were inspected.


Supported AUTH-03 completion and AUTH-05 mechanisms now have actual browser and
real verifier/transaction evidence for the documented WebAuthn, TOTP and backup
profiles. AUTH-06 recovery and AUTH-07 general abuse/event infrastructure remain
separate concerns. Consumers still own RP/origin/access policy, database atomicity,
key custody, HTTPS, hardware/attestation/AAL claims and domain authorization.
AU-10's exact integrated aggregate condition passed; see the final closure
evidence below.


Slice 5/T5.1/T5.2 is delivered after verified PR #102 merge. The canonical
Slice 5 report maps AU-01 through AU-10 with the successful exact integrated
aggregate evidence. No correction branch was required.


Slice 5 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/102
Report introduction: `7c84792f2a27e7500d626ac2d1951ca7eaea3aff`.
Focused checks, final merge, AU mapping and exact aggregate gate are complete.
Future work must refresh canonical state and use its separately authorized scope.


## Integrated Validation Checkpoint

Verified PR #102 merge: `8685ea4b703b1628eda032f960e481ba64b24e2b`.
All five reports are delivered; the exact immutable integrated candidate passed
`make check`. The gate used Go 1.27.1 and an owned PostgreSQL 18.6 cluster with a
separate owned database. Default checks
remain distinct from the already passed explicitly tagged browser/integration
checks. No main alignment, release, tag, mirror or deployment is authorized.


## Delivery-Set Closure

Slice 5 of 5 completed. Exact integrated `dev` candidate:
`29d5720aa506140e521fe852a37630fa5057499b`.
Verified PR #102 merge: `8685ea4b703b1628eda032f960e481ba64b24e2b`.

- `make check` — passed once on 2026-10-04: source licensing (812 headers,
  109 annotations), format, vet, complete default suite, 81.8% total coverage
  against the required 80%, and strict lint (zero issues).
- `git diff --exit-code -- '*.go'` — passed after the gate; formatting did not
  change tracked runtime files or the candidate identity.
- Tagged actual-browser and PostgreSQL race evidence remains separately recorded
  in the delivered Slice 5 report; it is not inferred from default tests.
- The isolated gate database was dropped and only the owned cluster stopped
  after normal completion. No validation correction was required.
- AU-01 through AU-10 and supported AUTH-03/AUTH-05 profiles are delivered.
  AUTH-06/AUTH-07 and hardware/attestation/deployment/application assurance duties
  remain separate. No main alignment, release, tag, mirror or deployment ran.

Integrated gate Acta: `01M43YXBA25Z1J9CWPS2E3RYEK`.
The following closure commit changes only documentary evidence; the gate above
belongs to the exact candidate named above and is not claimed for another hash.
