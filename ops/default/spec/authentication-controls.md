<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication Attempts, Errors and Security Events

Date: 2026-10-05
Approved: 2026-10-05
Status: Approved
Concern: authentication-controls
Parent: [Authentication security foundation](authentication-security.md)
Model: [Authentication controls model](authentication-controls-model.md)
Prerequisites: [Sessions](authentication-sessions.md), [Authenticators](authenticators.md), [Account recovery](account-recovery.md)
Inspected baseline: `74b4e86d4d3dd4829ad8da61f62fecdb0da187e1`
Implementation status: Complete
Execution gate: Closed

## Purpose and Ownership

Refine AUTH-07 with shared admission before credential work, classified internal
outcomes, safe public responses, bounded security-event observation and finite
ingress resources. Extend the existing `auth`, `config`, `middleware` and Ticked
adapter/transport boundaries. Core owns orchestration and typed contracts;
applications own identity normalization, durable storage, trusted proxy policy,
event destinations and deployment-specific anti-automation.

This concern preserves delivered subject/challenge/recovery budgets, proof
requirements, atomic mutations and the shared password verifier. It adds the
missing password-entry coverage instead of replacing those mechanisms with a
general limiter framework. All-factor-loss proofing and authenticator disabling
or rebinding need a separate lifecycle contract. Temporary admission denial
cannot deactivate an account, revoke factors or grant weaker access.

## Inspected Coverage and Gaps

| Boundary | Current evidence | Required change |
| --- | --- | --- |
| Duplicate registration | `auth.Queries.CreateUser` requires `ErrEmailTaken`; Ticked maps `users_email_key` conflicts; `TestCredentialTransactions` proves one concurrent winner | Preserve the delivered classification; remove raw public error forwarding |
| Password entry | `Service.Signin`, `Service.Reauthenticate`, initial WebAuthn enrollment and `FallbackService.verifyPassword` reach the shared verifier without one shared password-attempt budget | Charge public identity admission before lookup and all credential work before verification |
| Existing finite controls | Authenticator and recovery adapters commit their own attempts before verification; KDF size/concurrency and operation deadlines are delivered | Retain their semantics and distinguish each budget's purpose |
| Public forms | Ticked sign-in groups messages but logs the submitted email and raw errors; signup renders `err.Error()` and signs in a new account automatically | Classified redacted handling, neutral registration and bounded failure timing |
| Peer limiting | `middleware.RateLimiter` retains an unbounded map and starts a cleanup goroutine without cancellation/join | Validated finite counters/capacity and explicit cleanup ownership |
| Recovery ingress | `MailboxHandler` has finite peer/active-request capacity without a background loop | Preserve its guarantees while wiring one common outer authentication guard |
| Event observation | Core has ordinary logs; the Ticked audit subscriber concerns todo events | Add a narrow typed authentication observer; do not repurpose todo payloads |

These are source observations, not new runtime or production-assurance evidence.

## Shared Credential Admission

The application supplies the same canonical identity used by its unique account
lookup. All aliases accepted for that account must converge on this identity. Core does not invent email folding, aliases or tenant selection. Require
valid UTF-8, 1–254 bytes and no control characters before storage. Password input
encoding/byte bounds are checked before admission; existing new-password policy
and NFC processing remain authoritative. Structural rejection performs no KDF.

Use two closed durable purposes: `registration` and `password-proof`. Every
public password-proof start, including sign-in, first WebAuthn enrollment and
password-based TOTP/fallback setup or sign-in, shares `password-proof` for that
canonical identity. Session-derived password reauthentication uses the actual
subject's canonical identity, never a request-selected identity. Internal helper
calls cannot charge the same password check twice. Signup uses `registration`.
Checking a proposed new password during change/reset remains under the delivered
recovery budgets; verifying a backup code retains its existing factor budget.

For structurally valid public identity requests, commit admission before account
lookup, password blocklist calls, hashing or verification. Charge the same durable
budget for missing, inactive and existing identities. No fake subject is created
for a missing account. The finite capacity and ingress rules below bound
unknown-identity storage and the rate of credential work.

One admission permits one bounded operation. It does not establish proof, reserve
a session or relax the current credential-version checks. Cryptography and
external callbacks run after the admission transaction releases its locks.
Admission consumes the existing whole-operation deadline rather than adding
time to it. Earlier caller deadlines remain authoritative. Storage failure or ambiguous
admission commit denies work; never fall back to a process-local counter or
automatically retry the mutation.

Count admitted operations, including successful password checks, policy rejection,
cancellation, checker/KDF failure and subsequent stale credential state. Never
refund a charge or clear it after partial proof, session issuance, password
change/reset, token replacement, restart or policy revision. This is an explicit
resource budget, not a consecutive-failure counter. Existing factor/recovery
attempts may also be charged for their distinct operation; password proof cannot
reset them or reuse their authorization.

The final allowed attempt sets one finite cooldown. Further denied requests do
not increment or extend it. A new window starts only at or after both window end
and cooldown end. All replicas sharing the adapter use trusted database time,
atomic increments and the same capacity/key namespace. Current non-expired
counters survive process restarts. Tightening a limit applies immediately;
loosening it cannot replenish a live window. Tightening below an already spent
count denies the remaining captured window without rewriting the count or
starting a cooldown from each denial. Key changes require preserving live
budgets and are not a reset mechanism.

## Finite Ingress and Cleanup

One shared authentication ingress guard runs before identity lookup and expensive
body/candidate handling. Use the existing `middleware.ClientIP` contract with
explicit trusted-proxy configuration; an untrusted forwarded header cannot pick
a different bucket. Canonical IPv4/IPv6 forms select the same peer identity.
Identity and peer budgets serve different purposes: the peer guard is local to
one process, while durable identity budgets are shared through storage. Neither
is an internet-wide bot-detection service.

Replace per-request timestamp lists with one finite counter/window per peer.
Refuse new peers at capacity without evicting or resetting a live bucket. Reject
new work at active-request capacity without allocating waiting goroutines. Release
active capacity on every exit. Recovery's existing finite guard remains effective;
the common guard cannot weaken its limits or neutral acknowledgment.

Construction validates configuration and starts no goroutine. Cleanup is a
bounded explicit operation; admission may perform at most one bounded cleanup
batch. Applications own any scheduled invocation and its start/cancellation/join.
Do not leave a hidden ticker, launch a goroutine per identity or scan an unbounded
map while holding a request lock. Closing an application stops new admissions
before releasing owned infrastructure; existing request contexts handle shutdown.

Durable cleanup removes only records whose window and cooldown have both ended,
under the same synchronization as admission. It updates finite capacity atomically
and cannot delete a currently charged window. It does not touch account, session,
factor or recovery authority. Unknown identities and denied traffic have the same
retention rule as known identities; denial cannot keep expired data alive.

## Internal Outcomes and Public Transport

Preserve existing `errors.Is` contracts, including `ErrEmailTaken`, and classify
missing/inactive identity, invalid proof, expired/replayed state, incomplete proof,
policy rejection, exhausted attempts, capacity, stale state and operating failure.
Inactive identity needs a stable typed classification rather than a free-form
message. Expected negative authentication is distinct from storage/checker/KDF
failure. A lookup or operating failure cannot be reported as successful proof.
Errors, events and public responses contain no credential or bearer material.

Ticked maps internal outcomes using explicit cases. It must never render or log
raw SQL/driver errors, `err.Error()`, request identities, form bodies or cookies
from these authentication paths. Safe password-policy feedback may be shown when
it depends only on the submitted candidate, not account existence. Internal
uniqueness classification and its actual concurrency regression are retained.

For a valid registration candidate, new-account creation and duplicate identity
return the same status, acknowledgment and navigation to normal sign-in. Neither
response sets an authentication cookie. Remove registration's automatic sign-in:
the public response cannot disclose creation through a new session or redirect.
This does not add mailbox activation or change the selected access policy.

Missing, inactive, incorrect-password and identity-throttled sign-in failures have
the same status/message and acknowledgment target. Password-based enrollment and
fallback entry similarly group account/proof/admission failures without changing
their successful challenge contracts. Existing recovery initiation keeps its
neutral response. Authenticated reauthentication may expose a bounded retry delay
because its subject is already established by an actual current session.

Use a fixed target per public operation, longer than its whole-operation work
deadline plus observation and at least 100 milliseconds of margin, measured from
request acceptance. The target is configuration-owned and independent of the
lookup result; routes with longer service deadlines require a longer target.
Hold active ingress capacity through the acknowledgment wait, obey cancellation
and never spawn a
detached timer worker. Public failure bodies do not reveal remaining identity
attempts, cooldowns or subject references. Syntax/body rejection and peer/global
capacity responses occur before identity lookup and may have distinct statuses.
Uniform operating-unavailable responses may remain distinct from expected denial;
they must not disclose account-dependent causes. An ambiguous account/session
commit never triggers automatic replay or a success cookie.

## Security Events

Provide one typed observer shared by the actual authentication services. Emit a
terminal observation for registration, sign-in/reauthentication, sign-out/session
rotation or revocation, factor/pending begin and finish, factor management and
mailbox/change/reset operations, including admission denial and operating failure.
Ordinary session lookup/listing and page GET are not security mutations. An
internal helper emits no duplicate; independent public operations remain separate.

The closed event shape carries event ID, trusted occurrence time, operation,
classified outcome and optional validated subject/record references and actual
proof method. References come from trusted storage, never request claims. Pending
or password-only proof cannot be called completed MFA; issuance is distinguished
from consumption and committed completion. Unknown/ambiguous storage commit is
classified as operating/unknown outcome, not rolled back or completed.

No event contains passwords, password records, bearer tokens or their digests,
TOTP keys/outputs, backup codes, recovery links, raw identity/IP/user-agent values,
free-form errors or arbitrary metadata maps. Safe bounded enums and validated
references prevent log injection. Consumers own contextual risk decisions,
personal-data handling, retention, provider mapping and notification delivery.

Observation happens after database locks are released. Use finite non-waiting
callback admission and a deadline shorter than the remaining caller deadline.
The caller-owned observer must honor cancellation and be concurrency-safe, as
the existing password checker contract requires. No detached callback goroutine,
unbounded queue or automatic retry is permitted. Capacity, rejection, timeout and
delivery errors yield a bounded redacted observation-failure signal; they do not
alter the authentication result, roll back a security mutation or repeat it.
Late observer success is not successful delivery. Tests and documentation must
state that cancellation is cooperative; a callback that ignores context violates
the adapter contract. An explicit discard observer is permitted but establishes
no audit-delivery assurance.

This hook is best effort, not an atomic audit outbox or exactly-once delivery.
Delivered durable recovery notification intents retain their stronger mutation
contract. Applications needing mandatory audit persistence must review that
requirement separately instead of treating this observer as durable evidence.

## Proposed Defaults and Hard Bounds

These behavior and default decisions were approved on 2026-10-05. Invalid configuration fails construction;
zero optional values select documented defaults, not unlimited behavior.

| Boundary | Default | Supported bound |
| --- | --- | --- |
| Password-proof admission | 10 operations per 10-minute window; 10-minute cooldown | 1–20 attempts; window/cooldown each 1–60 minutes |
| Registration admission | 3 operations per one-hour window; one-hour cooldown | 1–10 attempts; window/cooldown each 1–60 minutes |
| Durable identity records across both purposes | 10000 | 100–100000; atomic shared capacity, no live eviction |
| Durable admission/cleanup deadline | 1 second | 100 milliseconds–5 seconds; never extends caller deadline |
| Peer admission | 12 requests per minute, 1024 peer records | 1–1000 requests; window 1 second–1 hour; capacity 1–10000 |
| Active authentication requests, including acknowledgment waits | 32 | 1–128; no waiting queue |
| Cleanup batch | 128 peers / 1000 durable records | 1–1000 peers / 1–1000 durable records per call |
| Public acknowledgment | 6 seconds with default 5-second credential work | Work plus observation plus margin must fit; maximum 31 seconds |
| Observer calls | 100-millisecond deadline; 2 simultaneous callbacks | Deadline 1–100 milliseconds; concurrency 1–16; no waiting queue |
| Event payload | At most 1024 serialized bytes | References at most 128 bytes each; closed enums; no arbitrary payload |

The password KDF's existing parameter/concurrency envelope remains unchanged.
At default active capacity, at most 32 requests hold the acknowledgment wait;
new traffic is denied before database/crypto work. Storage is bounded by the
record cap even under unique-identity floods. Measure actual index/storage cost,
contention and cleanup work before finalizing the adapter, rather than claiming
a memory or throughput guarantee from record counts alone.

## Acceptance Criteria

| ID | Required evidence |
| --- | --- |
| AC-01 | Inventory every actual password-verification entry; public sign-in/enrollment/fallback charge before account lookup; session-derived reauthentication shares the same budget before KDF, while registration and delivered factor/recovery budgets retain their distinct purposes |
| AC-02 | Two service instances and actual PostgreSQL prove exact atomic capacity/attempt limits, one last admitted operation, stable restart state, no refunds/resets and exact window/cooldown equality |
| AC-03 | Unknown identities, malformed/oversized input, stale state, canceled/ambiguous storage, policy tightening, capacity floods and concurrent bounded cleanup cannot cause unauthorized work or unbounded records |
| AC-04 | Peer limiter unit/race and actual handler tests prove canonical/trusted-proxy identity, finite maps/active work, no live eviction, no constructor worker, bounded cleanup and correct release/shutdown ownership |
| AC-05 | Actual registration races retain one user/one classified conflict; HTTP/browser journeys prove equal new/duplicate acknowledgment, no automatic session, safe policy feedback and no raw identity/driver error disclosure |
| AC-06 | Actual PostgreSQL/HTTP password-entry failures meet neutral message/status/timing policy; accepted proof/challenges and current-policy session/MFA semantics remain valid; real caller cancellation and capacity behavior are recorded |
| AC-07 | Actual service/adapter events classify committed, pending, denied and unknown outcomes correctly; malicious inputs, failed/late observers, saturation and callback re-entry cannot leak secrets, hold DB locks or change authentication authority |
| AC-08 | A finite affected regression preserves all-session invalidation, token/replay/backup consumption and required proof. Core/configuration/example/User Guide contracts agree; one exact integrated candidate passes the repository gate after all approved slices merge |

Fake stores cannot establish distributed counters or transaction semantics.
Captured events prove the observer boundary, not external audit/notification
delivery. Bounded timing evidence is not a production side-channel audit. Record
finite selectors, mandatory real database/browser prerequisites and resource
measurements in the later approved delivery plan before running validation.

## References and Scope Decision

Sources verified on 2026-10-05:

- [ASVS 5.0.0 V6](https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x15-V6-Authentication.md): 6.1.1/6.1.3 and 6.3.1/6.3.4 concern documented abuse controls across authentication paths; 6.3.8 includes registration and failure disclosure.
- [NIST SP 800-63B-4, rate limiting](https://pages.nist.gov/800-63-4/sp800-63b.html#rate-limiting): cumulative authenticator failure limits, disabling and rebinding exceed the fixed-window resource admission defined here. Do not claim that these budgets alone meet that lifecycle requirement or establish an AAL.
- [OWASP authentication guidance](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html#authentication-and-error-messages): public outcomes/timing and account-based throttling need review together with malicious lockout risks.
- [OWASP logging guidance](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html#data-to-exclude): redact sensitive authentication data, bound event fields and handle observation failure without changing application authority.

The bounded behavior/defaults and implementation scope were approved on
2026-10-05, including charged operations without refunds, neutral registration
without automatic sign-in and best-effort observation. The internal
[delivery plan](../plan/authentication-controls.md) and
[tracker](../tracker/authentication-controls.md) record five slices and ten tasks.
The parent remains partially implemented; all-factor-loss/lifecycle and consumer
assurance boundaries are not closed by scope approval.

## Implementation Coverage

Slices 1–4 are merged and delivered: mandatory shared durable admission, guarded
password entry, finite public ingress, neutral transport and typed best-effort
security observations. Slice 5 supplies actual Chromium/PostgreSQL journeys and
finite affected adapter/HTTP/race regression. Registration races retain one user
and equal public navigation without a session. Shared final proof admission,
capacity refusals, classified redacted events and retained actual MFA/recovery
passed. The [tracker](../tracker/authentication-controls.md#requirement-evidence)
maps AC-01 through AC-08 to exact evidence and remaining obligations.

All five slices are merged and delivered. The exact integrated candidate
`d598c465521384424e511f761019b351eaac69e1` passed the single full `make check`
with 80.7% coverage against 80% and zero strict-lint issues. The concern is
complete; [delivery-set closure](../tracker/authentication-controls.md#delivery-set-closure)
records the exact result and remaining application obligations. The
supported AUTH-07 mechanisms do not close cumulative authenticator disabling/
rebinding, all-factor-loss identity proofing, durable external audit delivery or
consumer/deployment assurance. The parent foundation remains Partial.
