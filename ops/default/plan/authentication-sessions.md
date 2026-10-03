<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication and Sessions Delivery Plan

Date: 2026-10-03
Status: Approved
Approved: 2026-10-03
Delivery set: authentication-sessions
Slice strategy: behavior-first
Reason: secure validation, required-proof outcomes and session control each form
an independently testable behavior. Update affected adapters/callers in each
slice so every merged increment compiles and carries its own persistence evidence.
Parent specification: [Authentication security](../spec/authentication-security.md)
Concern: [Authentication and sessions](../spec/authentication-sessions.md)
Model: [Authentication and sessions model](../spec/authentication-sessions-model.md)
Tracker: [Delivery tracker](../tracker/authentication-sessions.md)
Base branch: `dev`
Planning base: `56b1cb05ea4243fbb87fe41ef3c7bb83b72e4fb5`
Active slice: Slice 3
Execution gate: Review
Go baseline: 1.27.1

## Outcome and Authority

Deliver AUTH-04's stateful session foundation and AUTH-03's required-proof/result
boundary, using the approved concern/model. The concern, model, plan and tracker
were approved on 2026-10-03 under the approved parent scope. Stronger requirements fail closed while their real
AUTH-05 verifier is unavailable. Do not claim complete MFA or full AUTH-03 from
metadata or test fixtures. Recovery remains AUTH-06; wider attempt controls remain
AUTH-07, except finite operation/error limits required here.

The concern/model are promoted together; Slices 1 and 2 are delivered and Slice 3 is reviewing. Exact interfaces and persisted fields are
settled against the model before each affected runtime change. Material scope
changes require review before adding work. Implementation branches follow the recorded map.

## Delivered Prerequisites and Inventory

- The three credential-security slices merged in PRs #92/#93/#94. Exact integrated
  `d3e27373930d4e188c5f27413f6f1822b469986c` passed `make check`; closure is the
  planning baseline. The [completed tracker](../tracker/credential-security.md)
  supplies evidence rather than another credential rewrite.
- `AuthVersion`, owned snapshots, conditional session insertion and atomic
  password replacement already exist. Extend that required storage boundary.
- `auth.Service`, models/context/middleware, configuration and their tests are
  affected; retain the existing auth/model/crypto/config ownership.
- Ticked service/handlers/queries, SQL source/initial migration/generated models,
  example configuration and tests require synchronized updates per changed API.
- Review `middleware` auth consumers, examples and generator output before each
  signature change; update only callers actually affected by the auth contract.
- Existing random-token generation supports 32-byte secrets. No new token framework
  or authenticator dependency is selected by this set.

## Ordered Slices

| Slice | Short name | Exact branch | Expected PR title | Expected report |
| --- | --- | --- | --- | --- |
| Slice 1 | Secure session lifecycle | `feat/auth-session-lifecycle` | `feat(slice-1): enforce secure session lifecycle` | `ops/default/report/slices/authentication-sessions/slice-1-session-lifecycle.md` |
| Slice 2 | Required proof and outcomes | `feat/auth-required-proof` | `feat(slice-2): enforce authentication proof requirements` | `ops/default/report/slices/authentication-sessions/slice-2-required-proof.md` |
| Slice 3 | Reauthentication and control | `feat/auth-session-control` | `feat(slice-3): add atomic session reauthentication and control` | `ops/default/report/slices/authentication-sessions/slice-3-session-control.md` |

Each slice starts from current `dev` in a dedicated canonical worktree, produces
one report and opens one PR to `dev`. Wait for verified merge before continuing.
Update API/configuration references, affected examples and usable Unreleased
behavior in the same slice; the final slice is not a documentation catch-up gate.

## Slice 1: Secure Session Lifecycle

Outcome: password-authenticated sessions use independent secrets, digest-only
storage and current-state/exact-expiry validation with bounded activity updates.

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Settle stored/issued/validated session types, canonical secret namespaces, validated lifetime/cadence and atomic validate/touch storage contracts. Implement generation/digest parsing and password session issue/validation; update adapters, schemas, callers and context together. | `feat(auth): enforce secure session lifecycle` |
| T1.2 | Demonstrate entropy/source failures, canonical parser bounds, digest-only persistence, equality/inactivity/coalescing, post-lock time and account-version invalidation. Add real Postgres race/rollback tests, parser fuzzing and API/configuration guidance. | `test(auth): verify session lifecycle boundaries` |

Acceptance: AS-01, AS-02, AS-06 and affected AS-09 obligations. Existing
password-only access continues under its documented policy; this preparatory
slice cannot be advertised as MFA delivery or promoted as the complete set.

## Slice 2: Required Proof and Outcomes

Prerequisite: Slice 1 merged and current-state/session representation verified.
Outcome: trusted access requirements select completed versus non-authorizing
pending/denied results; middleware cannot infer stronger proof from enrollment.

| Task | Work | Expected commit |
| --- | --- | --- |
| T2.1 | Settle and implement trusted requirement/revision, verified password facts and typed authentication outcomes. Require operation policy at service/middleware boundaries; expose safe validated session metadata and freshness evaluation. Define pending representation/lookup separation without exposing unsupported proof completion. | `feat(auth): enforce required proof and authentication outcomes` |
| T2.2 | Exercise policy revisions, exact freshness, enrollment flags, missing/unsupported methods and pending/full separation; demonstrate password-only completion and strict denial end to end. Update affected example flows/docs to make unavailable continuation explicit. | `test(auth): verify proof outcome isolation` |

Acceptance: AS-03, AS-04 and affected AS-01/AS-09. Production verification remains
password-only. Where no real supported continuation exists, return no pending
secret or stored record. Future pending invariants are documented for
AUTH-05, without placeholder storage methods/tables or an executable MFA workflow. AUTH-03 remains
partial until real completion and factor consumption are delivered.

## Slice 3: Reauthentication and Control

Prerequisite: Slice 2 merged with trusted policy/proof semantics.
Outcome: supported fresh authentication rotates atomically; authorized session
management, admission and cleanup remain finite under concurrent account changes.

| Task | Work | Expected commit |
| --- | --- | --- |
| T3.1 | Implement password reauthentication for supported policy with generation-based atomic rotation, current/other/all subject-session revocation, recent-proof management, safe bounded listing, per-subject retained-row admission and explicit cleanup. Update the required adapter/caller contracts together. | `feat(auth): add atomic session reauthentication and control` |
| T3.2 | Demonstrate competing rotations, revocation/disable/password-change races, post-lock expiry, partial-write rollback, subject isolation and concurrent capacity limits with real Postgres. Verify context/cookie/example behavior, docs/generator consumers and user-visible session control; map evidence to remaining AUTH obligations. | `test(auth): verify session control integration` |

Acceptance: AS-05 through AS-09 plus regression of AS-01 through AS-04.
A stronger current policy cannot be satisfied by password reauthentication.
Management handlers consume validated metadata and trusted application authority;
core does not define application administrator roles.

## Per-Slice Validation

Run only checks required by changed behavior and record exact results in each
slice report:

- `go test ./auth ./config ./crypto ./model ./middleware`
- `go test -race ./auth ./config ./crypto ./model ./middleware`
- `go test ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web`
- `go test ./generator/...` when active API/config/scaffold consumers change.
- `go test -run '^$' ./examples/...`
- Real PostgreSQL session integration under `-tags=integration -race -count=1`;
  settle the exact new test selector before implementing T1.2 and record it in
  the tracker. Supply DB connection settings externally; absence must fail.
- Slice 1 token-parser fuzzing: settle the exact selector before T1.2, run a
  bounded 20-second campaign without database or KDF work and record executions.
- `make source-license-check`
- `make vet`
- `make lint-strict`
- `make docs-check`
- `git diff --check`

Time tests establish equality, inactivity cadence, future/invalid proof times
and current time after lock waits. Persistence tests establish serialization and
rollback rather than copying production predicates into a fake. Pending/strong
proof fixtures never count as real authenticator acceptance evidence.

## Delivery-Set Gate and Completion

After all three slices merge, run `make check` once against the exact integrated
`dev` candidate with an isolated real test database. HatMax has no scheduled
nightly. Record candidate and results, and use the scoped validation-correction
branch/PR route if repository fixes are required. Do not run `make ci` for this
set; badge publication is separate.

Close only when slice reports, task/PR evidence, AS-01 through AS-09 and the full
local gate pass. Record AUTH-04 coverage and AUTH-03's remaining real-authenticator
completion explicitly. Browser/session management presentation, production
policy and complete assurance assessment remain consuming-application evidence.
Publication of a selected core dependency is separate from local set closure;
this plan does not authorize main alignment, release, tagging or deployment.

## Slice 1 Contract

`Session` is safe metadata: ID, subject, captured `AuthVersion`, generation,
completed password authentication time, creation/activity times, absolute expiry
and an inactivity duration. `SessionRecord` pairs it with a fixed-size
`SessionDigest`; `IssuedSession` alone carries the transient raw token.
`ValidatedSession` pairs an owned user snapshot with safe session metadata.
Policy revision/required-proof properties are introduced with Slice 2 rather than
unused placeholder claims in this password-only increment.

`Signin` returns an issued value. `ValidateSession` receives trusted explicit
`SessionActivity` (none or relevant) and returns the validated value. Middleware
and active callers choose activity explicitly and attach safe session context.
`Queries.CreateSession` receives only a record plus expected credential state;
`ValidateSession` receives a digest and activity/cadence and checks current time
under the account/session locks. `DeleteSession` receives a digest, preventing a
stale raw token from revoking a later representation by record ID. Existing
cleanup becomes batch-bounded while its full management contract stays in Slice 3.

Configuration retains validated `session_ttl` (24h), adds `session_inactivity_ttl`
(30m), `session_activity_interval` (1m), `session_timeout` (5s) and
`session_cleanup_batch` (1000). Stored durations use whole microseconds to align
Go checks with PostgreSQL precision. The adapter evaluates `clock_timestamp()`
after locks; it never trusts a timestamp sampled before the wait. Relevant
activity conditionally updates the stored timestamp only when the cadence elapses.

## Slice 2 Contract

`AccessRequirement` is an owned value: `Proof` (password, MFA or phishing-resistant
MFA), `Revision` (1 through 128 printable ASCII bytes) and `MaxAge` (zero disables
freshness; otherwise 1s through absolute lifetime, whole microseconds). It is a
trusted server argument for sign-in, validation and middleware; no request field
can select it. Invalid requirements fail before verification/storage.

`VerifiedProof` has a closed supported `Method` and `VerifiedAt`; only password
is currently supported. Safe `Session` adds policy revision and proof metadata;
SQL adds matching revision, method and verification time with constraints.
The core password verifier produces the facts. Adapters validate stored facts,
current revision, freshness and requirement after locks and before activity.

`AuthenticationResult` distinguishes completed, pending enrollment, pending proof
and denied with finite reasons. Completed results alone contain `IssuedSession`.
Unmet MFA uses enrollment flags only to describe the next required step, never as
proof; phishing-resistant MFA is denied as unavailable. All unmet results carry
no issued/continuation secret or stored row. Invalid credentials remain classified
errors with no result; storage/verifier errors are operating failures. There is
no public completion API, asserted-proof callback, pending secret or table.

`Signin` receives a requirement and returns this result. `CreateSession` receives
the same trusted requirement. `ValidateSession` receives requirement before
activity throughout service/storage/middleware. Ticked owns one explicit
password-only revision and checks completed outcome before setting a cookie.
Pending/unavailable outcomes render a bounded explanatory failure, with no
continuation endpoint or implied MFA completion.

## Slice 3 Contract

`Reauthenticate(ctx, token, password, requirement)` validates a live current
session without demanding already-recent proof, verifies the password again and
passes a replacement record to required `RotateSession` storage. The adapter
rechecks captured account version, current digest/generation, revision and expiry
under subject/session locks, then replaces the digest and increments generation
without changing record ID/creation time. Failed proof or storage work returns no
issued secret. Stronger current requirements cannot complete from password.

`CreateSession` additionally receives the configured retained-row admission cap.
Under the subject lock it reclaims at most 100 expired subject rows, counts all
retained rows and rejects capacity without eviction. New settings are
`session_recent_proof_age` (5m, 1s through absolute lifetime, whole microseconds),
`session_max_per_subject` (10, 1–100) and `session_page_size` (50, 1–100).

`ListSessions(ctx, token, requirement, cursor)` and
`RevokeSessions(ctx, token, requirement, selection)` derive the target subject from
the live actor bearer. Storage revalidates current actor/recent proof under locks;
recent age is the tighter of operation policy and configured management age.
`SessionSelection` has finite current/selected/others/all scopes; selected IDs are
bounded at 128 printable ASCII bytes. `SessionPage` contains only safe metadata,
current record ID and an opaque canonical URL Base64 cursor at most 128 bytes.
Keyset ordering is record ID, scoped to the actor subject; page reads fetch at most
configured size plus one. Multi-record revocation locks IDs in stable order.
Applications own any administrative authority over another subject; self-service
APIs accept no target subject or claimed proof. Bearer-only sign-out remains the
existing current-secret withdrawal operation, separate from recent-proof management.

Ticked adds a password reauthentication form and subject-scoped session screen.
Rotation updates the cookie only after completed issuance; failed work preserves
it. Self-service management failures require reauthentication and never fall back
to lower proof or an administrative role. Cleanup remains explicit and bounded.
