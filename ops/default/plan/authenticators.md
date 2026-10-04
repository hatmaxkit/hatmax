<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authenticator Delivery Plan

Date: 2026-10-04
Status: Approved
Approved: 2026-10-04
Delivery set: authenticators
Slice strategy: behavior-first
Reason: registration, assertion completion, replay-resistant fallback, authorized
factor changes and browser acceptance are separately reviewable behavior increments.
Each increment updates required storage, active callers and documentation together.
Parent: [Authentication security](../spec/authentication-security.md)
Concern: [Authenticator proposal](../spec/authenticators.md)
Model: [Authenticator model](../spec/authenticators-model.md)
Tracker: [Delivery tracker](../tracker/authenticators.md)
Base branch: `dev`
Planning base: `247487f273a54a8e65a8e6b5f86191f38bcb0be8`
Active slice: Slice 1
Execution gate: Open
Go baseline: 1.27.1

## Outcome and Approval Boundary

Deliver the parent's AUTH-05 mechanisms and actual AUTH-03 factor completion using
the approved credential/session foundations. This proposal selects the dependency,
supported profile, finite budgets and required atomic bindings. The concern/model and this five-slice map were approved on 2026-10-04.

Approval promoted concern/model together, updated direct links and activated
Slice 1/T1.1 in the tracker. Exact Go/SQL bindings are settled against the model
before each changed runtime contract. No branch, dependency or runtime changes
are activated by a pending planning proposal.

Scope covers actual authenticator verification/enrollment and the factor mutations
needed to keep this mechanism coherent. General mailbox/password recovery and
operator-assisted all-factor loss belong to AUTH-06; broader attempt/event
infrastructure belongs to AUTH-07. Endpoint budgets delivered here are mandatory
parts of actual factor submission. Do not ship an unbounded pending endpoint while
waiting for a generic limiter.

## Delivered Prerequisites and Inventory

- [Credential security](../tracker/credential-security.md) and
  [authentication sessions](../tracker/authentication-sessions.md) are delivered.
  Their exact integrated gates are evidence, not work to repeat for planning.
- `auth/proof.go`, service/middleware/models, required queries and their safe
  session context need synchronized proof/pending and factor changes.
- `crypto/totp.go` needs trusted step-returning verification and replacement of
  shared-salt/index-only backup contracts. Keep unrelated cryptographic helpers.
- Ticked SQL source, initial schema, required query implementation, SQLC mappings,
  configuration and handlers provide the concrete persistence/transport adapter.
- Review active generator/scaffold callers at each changed signature; update only
  affected contracts. No runtime dependency on a private application is permitted.
- Inspect the pinned WebAuthn module/license/transitives and bound parser inputs
  before addition. Browser acceptance requires a local Chromium-compatible browser
  and an explicit virtual-authenticator harness; settle its command before T5.1.

## Ordered Slices

| Slice | Short name | Exact branch | Expected PR title | Expected report |
| --- | --- | --- | --- | --- |
| Slice 1 | Restricted enrollment | `feat/authenticator-enrollment` | `feat(slice-1): add restricted authenticator enrollment` | `ops/default/report/slices/authenticators/slice-1-enrollment.md` |
| Slice 2 | WebAuthn completion | `feat/webauthn-completion` | `feat(slice-2): complete WebAuthn authentication and step-up` | `ops/default/report/slices/authenticators/slice-2-webauthn-completion.md` |
| Slice 3 | TOTP and backup proof | `feat/authenticator-fallback` | `feat(slice-3): add replay-resistant TOTP and backup proof` | `ops/default/report/slices/authenticators/slice-3-fallback-proof.md` |
| Slice 4 | Authorized factor changes | `feat/authenticator-control` | `feat(slice-4): enforce authorized authenticator changes` | `ops/default/report/slices/authenticators/slice-4-factor-control.md` |
| Slice 5 | Browser acceptance | `test/authenticator-acceptance` | `test(slice-5): verify authenticator browser integration` | `ops/default/report/slices/authenticators/slice-5-browser-acceptance.md` |

Each slice starts from verified current `dev` in a dedicated canonical worktree
and opens one PR to `dev`. Keep it compiling and passing focused checks; wait for
verified merge before activating the next slice under the same approved set.
Update supported behavior in API/configuration/User Guide and Unreleased when it
becomes usable. Slice 5 owns the browser evidence, not deferred runtime correctness.

## Slice 1: Restricted Enrollment

Deliver real WebAuthn registration from fresh password-authorized initial setup,
with no ordinary access from a pending token. Ship finite pending storage and
durable verification admission with that first real endpoint.

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Review/pin the proposed protocol dependency; settle typed records, encrypted-material dependency, required pending/budget queries and bounded constructor settings. Implement actual initial WebAuthn enrollment with opaque subject handle, captured password authority and version-changing confirmation; update Ticked schema/queries/callers together. | `feat(auth): add restricted WebAuthn enrollment` |
| T1.2 | Demonstrate wrong origin/RP/challenge/subject/algorithm/UP/UV rejection, cross-purpose lookup isolation, retention/admission bounds, durable failure budgets and setup replay/account-state races with actual verification and PostgreSQL. Update supported contracts/docs. | `test(auth): verify enrollment isolation and bounds` |

AU-01/AU-03/AU-07/AU-08 and the affected AU-10 obligations. Existing password
access remains valid under its trusted profile; registration alone cannot be
advertised as completed strong sign-in. Strong assertion completion stays closed
until Slice 2, with no placeholder proof-assertion method.

## Slice 2: WebAuthn Completion

Deliver actual subject-first UV assertion sign-in and bound step-up. Factor
counter/flags, pending consumption and session issue/rotation share one commit.

| Task | Work | Expected commit |
| --- | --- | --- |
| T2.1 | Implement actual library assertion validation and closed proof constituents/freshness; bind pending/current-policy/account/factor and reauthentication generation; implement required atomic completion and update safe middleware/SQL/cookies together. | `feat(auth): complete WebAuthn proof atomically` |
| T2.2 | Establish one concurrent completion/rotation winner, counter policy including zero counters, immutable eligibility and current backup state, wrong/removed credential denial, post-lock expiry/lease/proof rejection and forced rollback with real PostgreSQL. | `test(auth): verify WebAuthn completion transactions` |

AU-01 through AU-04 and affected AU-07/AU-08/AU-10. Use signed real protocol
responses for verifier/transaction tests; final browser evidence remains Slice 5.
No hardware/non-exportability certification is inferred from virtual fixtures.

## Slice 3: TOTP and Backup Proof

Deliver allowed lower-assurance MFA through actual password/TOTP or password/code
verification; one-use state is consumed with completion, never as an earlier write.

| Task | Work | Expected commit |
| --- | --- | --- |
| T3.1 | Implement encrypted TOTP setup/confirmation and trusted matched-step verification, replay-safe pending completion, domain-bound salted backup verifier/generation/consumption and policy-aware combination times. Replace active old contracts and update schemas/configuration/callers together. | `feat(auth): add replay-resistant fallback proof` |
| T3.2 | Demonstrate trusted time/skew/equality, accepted-step setup and replay races, independent salts/KDF admission, concurrent backup reuse/regeneration/rollback and rejection for phishing-resistant requirements. Confirm budget reissue cannot bypass exhaustion. | `test(auth): verify fallback replay and policy` |

AU-02 through AU-06 and affected AU-08/AU-10. Executable code-set issue is bound
to actual recent management proof; any richer replacement/removal flow is Slice 4.
Neither fallback method supplies phishing-resistant authority.

## Slice 4: Authorized Factor Changes

Deliver subject-owned safe listing, additional enrollment, removal/replacement
and backup-set regeneration under explicit recent management proof. Preserve
current factor/policy constraints and rotate a retained actor coherently.

| Task | Work | Expected commit |
| --- | --- | --- |
| T4.1 | Implement finite selections, initial/established enrollment distinction, explicit management policy, factor/version changes, last-factor constraints and atomic revoke/actor rotation. Bind pending targets and proof constituents; add minimal Ticked handlers/presentation. | `feat(auth): enforce authorized factor changes` |
| T4.2 | Prove foreign ownership rejection, stale actor/factor/lease conflicts, old-proof freshness after locks, all-session invalidation, no retained removed-factor proof, last-factor protection and full rollback under races. | `test(auth): verify authenticator change authority` |

AU-07 and affected AU-02/AU-03/AU-04/AU-08/AU-10. A permitted TOTP-only profile
can upgrade using actual current MFA when its trusted policy allows it; policies
requiring phishing-resistant management cannot be weakened by a request.

## Slice 5: Browser Acceptance

Establish production browser begin/finish, pending-cookie isolation and complete
documented flow integration, after all underlying runtime boundaries have focused
real verifier/transaction evidence.

| Task | Work | Expected commit |
| --- | --- | --- |
| T5.1 | Add a finite local browser/virtual-authenticator acceptance harness that uses production Ticked handlers and real PostgreSQL; demonstrate registration, sign-in, strong-route enforcement, step-up, safe controls and relevant TOTP/backup journeys. Supply only integration adjustments required by those approved paths. | `test(auth): exercise authenticator browser journeys` |
| T5.2 | Complete deterministic failure/cookie/deadline/parser evidence, supported reference/User Guide guidance and dependency review evidence; map AU-01 through AU-10 and remaining consumer assurance obligations in the final report. | `test(auth): close authenticator integration evidence` |

AU-09/AU-10 and regression of AU-01 through AU-08. A virtual authenticator proves
browser/library integration, not physical authenticator certification or an AAL.

## Validation and Exit Gates

Per task/slice run relevant focused package tests/race checks, Ticked tests,
affected generator checks and example compilation. Canonical commands are:

- `go test ./auth ./config ./crypto ./model ./middleware`
- `go test -race ./auth ./config ./crypto ./model ./middleware`
- `go test ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web`
- `go test -run '^$' ./examples/...`
- `make source-license-check`
- `make vet`
- `make lint-strict`
- `make docs-check`
- `git diff --check`

Before T1.2 record the exact new real PostgreSQL integration/race selector with a
finite timeout; missing externally supplied connection settings must fail.
Extend it per slice with actual enrollment/completion/fallback/control tests and
the existing credential/session regression. Before parsing work record bounded
20-second fuzz selectors; no KDF or database work in parser fuzz targets. Before
T5.1 record the browser harness and invocation, browser/version, virtual device
profile and required database/settings; missing prerequisites cannot count as pass.

After all five slices merge, run `make check` once for the immutable integrated
`dev` candidate with an isolated real PostgreSQL database. HatMax has no nightly;
this is the approved plan's proposed local full gate after execution approval.
Keep tagged/browser checks as separately named evidence. Do not run `make ci`.
If repository corrections are required, use `fix/authenticators-validation`,
focused correction checks and one PR to `dev`, followed by the exact corrected
local integrated gate after its verified merge.

Close only with five delivered reports, task/commit/PR evidence, AU-01 through
AU-10, actual AUTH-03/AUTH-05 supported-profile evidence and the exact full gate.
Record hardware/attestation, deployment, application authorization and broader
AUTH-06/AUTH-07 obligations explicitly. No main alignment, release, tag or
deployment is authorized by this delivery plan.
