<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 4: Identity and Recovery

Status: reviewing
Delivery set: documentation-conformance
Plan: [Delivery plan](../../../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../../../tracker/documentation-conformance.md)
Branch: `docs/hatmax-identity-recovery`
PR: `#115`
Report introduction: `1cb59fa18fb16e50b0a56c64b9e1fbdb808ee713`
Activation revision: `ba9f95b9272d9e80695599d2b9524e9fe2d4b7ff`
Documentation task: `f96565f609da32a7430cc617138d72d016376fcf`
Merged revision: pending

## Purpose

Reconcile identity, cryptography, authenticator management and account recovery
guidance with current APIs and real local proof, denial and lifecycle outcomes.

## Delivered Behavior

Auth examples handle construction/settings errors and use typed owning contexts.
Reference and learning-path guidance distinguishes registration from sign-in,
ordinary password policy from stronger proof, and illustrative invoice wiring
from Ticked's delivered adapter. Crypto guidance states actual key, nonce, tag,
Argon2 and token contracts, with separate ownership of encryption and lookup keys.
Ticked setup uses its actual database, required private admission material,
foreground startup and embedded migrations; legacy bcrypt seeding is identified
as incompatible with the PHC-only verifier.

Fifteen exact published Go fragments compile against current exported contracts.
Three crypto fragments execute round-trip, lookup and password/token outcomes;
the standalone key generator executes with its output privately captured and
canonical encoding asserted. Context-only invoice handlers grant compilation
assurance rather than application CRUD or authorization assurance.

All three published race-enabled browser selectors execute actual navigator
ceremonies, production handlers/core services and real PostgreSQL adapters.
Enrollment uses the exact published JSON-conversion/create/toJSON/fetch fragment.
Journeys verify cookie changes, WebAuthn/TOTP/one-use backup proof, retired actor
and stronger-policy denials, factor management and preservation, neutral mailbox
flows, current role/MFA operator initiation, failed dispatch/notification retry,
lost committed reset response, replay refusal and required-MFA re-entry.
No successful proof is supplied by a stored credential or signed-response stub.

Ticked's actual entrypoint and published Make targets execute in an isolated
local copy. Native role/database creation, effective configuration, SQLC
generation, testing, build/clean and foreground Make startup are checked.
Registration issues no session; ordinary sign-in issues Secure/HttpOnly/Lax
cookies. A persisted session works after restart with stable admission material,
signout retires it, and owned listeners close after coordinated shutdown.

Coverage reconciles 451 identities. The 50 Slice 4 bindings distinguish five
non-empty package suites, 11 compiled contexts, 21 executed bindings and 13
source-inspected pages or notation. Later groups retain pending evidence.

## Implementation Notes

Go owns Chromium process control and CDP through a private debugging pipe;
JavaScript runs inside the browser for actual navigator and served-form behavior.
Finite calls, callback capacity, private profiles, unique short socket scratch
and owned process shutdown bound the fixture. The driver reports safe stage
labels and redacts protocol values. Negative transport checks reject malformed,
truncated and rejected replies without leaking private response data.

The runner uses Go 1.27.1, SQLC v1.30.0 and native PostgreSQL/Chromium images.
The development invocation used PostgreSQL 18.6 and Chromium 151.0.7922.173.
`HATMAX_DOC_CHROMIUM` supports an explicit native image; the local launcher image
is resolved directly so executable digests bind the actual browser. PostgreSQL
and SQLC retain their documented explicit overrides/worktree-cache behavior.

Every invocation owns its cluster, Unix socket, schemas, HTTP ports and profiles.
Only those fixtures are stopped. The Ticked copy receives exact inventoried
example bytes; owned database/role/port coordinates replace deployment examples
and the YAML RP origin is updated consistently. Private admission material is
stable across restart and omitted from receipts/logs. HTTP requests explicitly
submit the cookie; actual browser Secure-cookie behavior is verified separately.

## Contracts Added or Changed

The slice runner accepts slices 1 through 4, reruns earlier runtime/data groups
and rejects future/integrated modes. Identity receipts bind exact HEAD, source,
snippet and harness inputs, compiler/tool identities, actual ordered commands,
diagnostic digests, methods and outcomes. Missing, stale, duplicate or
misclassified bindings, failed commands, empty/cached/skipped browser execution
and changed selectors cannot pass. Receipt reconciliation requires owned cluster
stop evidence and Ticked process configuration/completion evidence.

Browser fixture files now contain browser JavaScript rather than external
runtime orchestration; content-preserving licensing metadata follows their
renames. Historical reports remain factual. Toolkit runtime APIs and module
dependencies are unchanged. The generic approved native-tooling closure policy
is preserved; integrated constrained-toolchain evidence remains required.

## Files of Interest

- [Authentication reference](../../../../../docs/reference/authentication/README.md)
- [Crypto reference](../../../../../docs/reference/crypto/README.md)
- [Identity learning path](../../../../../docs/tutorials/user-guide/identity-and-sessions.md)
- [Ticked companion](../../../../../examples/ticked/README.md)
- [Native browser driver](../../../../../examples/ticked/internal/web/browser_driver_test.go)
- [Identity contexts](../../../../../scripts/documentation-conformance/identity_contexts.go)
- [Identity evidence](../../../../../scripts/documentation-conformance/identity_evidence.go)
- [Ticked workflow](../../../../../scripts/documentation-conformance/identity_ticked.go)
- [Coverage record](../../documentation-conformance-coverage.md)

## Validation

Focused implementation checks passed:

- `GOWORK=off make docs-check` — documentation/navigation and compiled example checks.
- `GOWORK=off make source-license-check` — headers and content-preserving annotations.
- `GOWORK=off make lint-strict` — zero issues.
- `GOWORK=off GOFLAGS=-p=2 go test -count=1 -timeout=2m ./scripts/documentation-conformance` — evidence, context, selector and fixture-ownership controls.
- `GOWORK=off GOFLAGS=-p=2 go test -tags=browser -race -run '^TestBrowserPipe$' -count=1 -timeout=30s ./examples/ticked/internal/web` — safe native transport failure handling.
- `GOWORK=off go run ./scripts/documentation-conformance check` — 451 identities and navigation.
- `bash -n scripts/check-documentation-conformance.sh scripts/documentation-conformance/data-fixture.sh` — passed.

The owned development identity invocation produced 50 bindings after actual
package, context, Ticked and all three published browser-selector executions.
Fixtures stopped after checks. Development receipts are historical evidence;
the final controller invocation regenerates evidence on the clean pushed head
after PR references:

- `GOWORK=off scripts/check-documentation-conformance.sh slice 4`
- `git diff --check`

The slice command includes required docs/source-license/strict-lint checks,
cumulative runtime/data evidence and active identity/browser receipt checks.
The report introduction precedes that final head; canonical PR/controller
evidence owns the final exact-head result.

Development failures exposed a missing native assertion helper, a settings API
error-handling mismatch, an incorrect signout status expectation and licensing
paths left at the old fixture names. Corrections passed their focused checks
and actual workflows; no production proof bypass or lint-policy change was used.

The first controller run on `c7f1b8e1dd820fdc1392570e056e7850795d110d`
passed all workflows but rejected foreground Make's actual `signal: interrupt`
exit. Its application had completed coordinator/database shutdown and closed
the listener. The checker now recognizes that Ctrl+C wrapper outcome alongside
success/interrupted-recipe exits, while still requiring shutdown diagnostics and
strict success for direct application processes. A regression check rejects
unrelated signals and failed/empty outcomes; exact-head evidence is regenerated.

## Risks and Follow-ups

DC05/DC08 integrated acceptance and DC09's native-toolchain closure demonstration
remain with Slice 7. Virtual devices prove browser/library/adapter integration,
not hardware identity, attestation, deployment custody or compliance. Captured
mail proves local dispatch/notification behavior rather than external delivery.
Context-only fragments and conceptual references retain their stated assurance
limits. Changed source/head/tool inputs require fresh execution evidence.
