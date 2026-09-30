<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Architecture Nit Review

Date: 2026-09-30
Baseline: `e66730f471d33f41da94148891a58557731eece1`
Status: review complete; dependency modernization applied locally; findings open
Publication: proposed for dev integration; no tag or release

## Outcome

The review produced **22 new tickets: 11 high and 11 medium**. Security boundary
failures, pubsub message loss, startup rollback, and scheduler worker failure
isolation are the first implementation priorities.

Dependency modernization and the approved Go 1.26 migration are applied.
The reported product defects are not fixed by this change. Green existing tests
do not cover the reproduced failures.

The checkout has no affected symbols reported by the final vulnerability scan.
A standalone application using published Hatmax `v0.5.0` still resolves affected
dependencies; see [F22](#f22). This distinction remains important until a refreshed
published baseline is selected.

## Scope and method

Reviewed lifecycle and composition, configuration and database assembly,
authentication and password helpers, middleware identity, trusted HTML
rendering, settings, image storage and processing, mail providers, pubsub,
scheduler execution, and generator command validation. Repository-wide tests,
race checks, build, lint, documentation, and generator acceptance complement
the focused source review.

Used the architecture-nit-review procedure and the repository ticket workflow.
Findings separate reproduced behavior from source-confirmed concerns and
already documented capability gaps. No broad package renaming or architectural
rewrite is proposed.

Temporary probes exercised public APIs in isolated directories, a subprocess,
loopback mail servers, and isolated PostgreSQL schemas. They assert the observed
defects, not desired regression behavior, and remain unversioned under
`.tmp/architecture-review/probes/`. Their test names identify evidence in the
tickets. Implementations should add permanent tests of the corrected behavior.

This is a bounded architectural and security review, not a claim that every
possible defect or production deployment condition has been audited.

## Findings

<a id="f1"></a>

### F1 - Bind proxy-derived identity to trusted peers

Severity: high
Area: architecture
File: `middleware/stack.go:19, middleware/stack.go:39, middleware/ratelimit.go:115`
Ticket: [Bind proxy-derived identity to trusted peers](../ticket/solved/20260930211800-trust-proxy-headers-only-from-approved-peers.md)

Problem:
DefaultStack installs chi RealIP without a trusted-proxy boundary. DefaultInternal then checks the rewritten RemoteAddr. A request from 192.0.2.10 with X-Forwarded-For: 127.0.0.1 receives HTTP 200 instead of 403. RateLimit independently trusts X-Forwarded-For and X-Real-IP, so changing these headers also changes the rate bucket.

Why it matters:
Directly exposed services can accept forged internal identities. Rate limiting and network access restrictions do not share one authoritative peer policy.

Recommendation:
Define one trusted-proxy policy. Preserve the socket peer for InternalOnly and accept forwarded client identity only from approved proxies. Use proper IPv4/IPv6 parsing.

Evidence:
Reproduced by TestObservedInternalBypass and TestObservedRateLimitBypass.

<a id="f2"></a>

### F2 - Confine local image storage to its root

Severity: high
Area: persistence
File: `image/local/store.go:31-57`
Ticket: [Confine local image storage to its root](../ticket/solved/20260930211801-confine-local-image-storage-to-its-root.md)

Problem:
Put, Get, and Delete join an unchecked key to basePath. A ../sentinel key accesses the parent directory. A symlink beneath the configured root can also redirect Put outside that root.

Why it matters:
Applications that accept user-derived storage keys can expose filesystem reads, overwrites, and deletion with the process permissions.

Recommendation:
Enforce root-confined operations for all three methods, including symlink traversal. Reject invalid keys and preserve valid nested object paths.

Evidence:
TestObservedStorageEscape reproduced traversal and symlink escape inside owned temporary directories.

<a id="f3"></a>

### F3 - Escape OOB attributes before trusting HTML

Severity: high
Area: architecture
File: `htmx/oob.go:81-82, htmx/oob.go:124-139`
Ticket: [Escape OOB attributes before trusting HTML](../ticket/open/20260930211802-escape-oob-selectors-before-trusting-html.md)

Problem:
OOB.Attr and OOBWrapper.Open interpolate selectors into quoted attributes and return trusted template types. A selector containing a quote creates an additional HTML attribute.

Why it matters:
Untrusted selector data can escape its attribute context and introduce executable markup. Returning template.HTMLAttr or template.HTML bypasses the template engine's escaping.

Recommendation:
Escape attribute values at the final rendering boundary and constrain wrapper tag names. Keep ordinary HTMX selector syntax intact.

Evidence:
TestObservedOOBInjection reproduced a new data-review-injected attribute through both public rendering methods.

<a id="f4"></a>

### F4 - Filter unsafe URL schemes in trusted link rendering

Severity: high
Area: architecture
File: `ui/link.go:117-154`
Ticket: [Filter unsafe URL schemes in trusted link rendering](../ticket/open/20260930211803-filter-unsafe-link-url-schemes.md)

Problem:
Link.Render HTML-escapes href but does not apply URL-context filtering before returning template.HTML. javascript:alert(1) is rendered unchanged; a normal html/template href renders the same value as #ZgotmplZ.

Why it matters:
User-provided URLs can execute script when a rendered link is followed. HTML escaping does not make an executable URL safe.

Recommendation:
Apply a defined safe URL policy before constructing trusted HTML. Review equivalent URL-bearing UI primitives for the same boundary.

Evidence:
TestObservedLinkScheme compares the public link renderer with html/template's URL filtering.

<a id="f5"></a>

### F5 - Prevent pubsub loss across out-of-order commits

Severity: high
Area: persistence
File: `pubsub/postgres/broker.go:296-375, pubsub/postgres/schema.go`
Ticket: [Prevent pubsub loss across out-of-order commits](../ticket/open/20260930211804-make-pubsub-cursors-safe-for-late-commits.md)

Problem:
The subscriber selects id > lastOffset and advances to the highest delivered ID. PostgreSQL assigns sequence IDs before commit. A lower-ID transaction committed after a higher ID has been acknowledged becomes permanently invisible to the subscriber.

Why it matters:
Durable subscribers lose committed messages even when every invoked handler succeeds. Sequence allocation order is not commit order.

Recommendation:
Use a durable delivery or acknowledgement strategy that cannot skip late-committed rows. Preserve named-subscriber restart semantics and bounded polling.

Evidence:
TestObservedPubsubCommitOrder held one insert transaction open, delivered a later insert, advanced the cursor, committed the earlier insert, and observed that it remained undispatched.

<a id="f6"></a>

### F6 - Do not acknowledge failed pubsub handlers

Severity: high
Area: persistence
File: `pubsub/postgres/broker.go:354-375, docs/reference/pubsub/README.md:10-12`
Ticket: [Do not acknowledge failed pubsub handlers](../ticket/open/20260930211805-retain-failed-pubsub-deliveries-for-retry.md)

Problem:
A handler error is logged, but lastProcessedID still advances and the durable offset is saved. The failed message is skipped on future polls and restarts. Existing handler-error tests assert one attempt rather than redelivery.

Why it matters:
The documented at-least-once delivery guarantee does not hold when processing fails.

Recommendation:
Define failed-delivery acknowledgement and retry behavior, including what happens to later messages in a batch. Persist acknowledgement only when the chosen delivery policy permits it.

Evidence:
TestObservedFailedPubsubAck verified a nonzero durable offset after a handler returned an error. TestBrokerHandlerError currently codifies the non-retry behavior.

<a id="f7"></a>

### F7 - Contain scheduler handler panics in each worker

Severity: high
Area: architecture
File: `scheduler/runner.go:124-137, scheduler/runner.go:175-189`
Ticket: [Contain scheduler handler panics in each worker](../ticket/open/20260930211806-contain-scheduler-worker-panics.md)

Problem:
The recover in tick protects only its own goroutine. With Workers greater than one, a handler runs in a child goroutine and its panic terminates the process.

Why it matters:
One background job can crash the complete single-binary application. A failed run can also remain marked running.

Recommendation:
Put the failure boundary around each handler invocation. Persist a failed run and release worker resources without masking normal store failures.

Evidence:
TestObservedWorkerPanic ran a two-worker scheduler in a subprocess and confirmed process termination with the handler panic.

<a id="f8"></a>

### F8 - Pair startup rollback with component identity

Severity: high
Area: architecture
File: `app/lifecycle.go:39-84`
Ticket: [Pair startup rollback with component identity](../ticket/open/20260930211807-pair-startup-rollback-with-component-identity.md)

Problem:
Setup collects start and stop capabilities independently, while Start indexes stops using a start index. A start-only component before a failing component causes an index panic; other mixed capability layouts can stop the wrong component.

Why it matters:
Startup failure loses its original error and can perform incorrect cleanup. The positional restriction is documented, but the public independent capability model remains fragile.

Recommendation:
Preserve component identity when assembling rollback. Stop only successfully started components that own a stop capability, in reverse order.

Evidence:
TestObservedRollbackPanic reproduced the index panic. docs/explanation/component-order-and-startup/README.md already documents the positional limitation.

<a id="f9"></a>

### F9 - Advance or retire completed scheduler slots

Severity: high
Area: persistence
File: `scheduler/runner.go:194-240, scheduler/postgres/store.go:25-72, scheduler/postgres/schema.go:40`
Ticket: [Advance or retire completed scheduler slots](../ticket/open/20260930211808-advance-or-retire-completed-scheduler-slots.md)

Problem:
A completed job remains due because process never calls UpdateNextRun or retires a one-shot job. PostgreSQL then rejects the same slot through UNIQUE(job_id, scheduled_for). The fake store instead permits the handler to run repeatedly.

Why it matters:
The built-in runner/store pair stalls after one successful run and repeatedly polls the exhausted slot. The fake backend does not expose the production behavior.

Recommendation:
Make ownership of next-run advancement explicit and implement the corresponding terminal or recurring transition. Align fake slot uniqueness with PostgreSQL.

Evidence:
TestObservedPostgresScheduleStall ran two ticks: the handler ran once, but the job remained due. TestObservedRepeatedSchedule ran the same fake-store slot twice. The lack of automatic advancement is already documented.

<a id="f14"></a>

### F14 - Honor SMTP STARTTLS and cancellation policy

Severity: high
Area: architecture
File: `mailer/smtp.go:34-85, mailer/mailer.go:22-32`
Ticket: [Honor SMTP STARTTLS and cancellation policy](../ticket/open/20260930211813-honor-smtp-tls-and-cancellation-policy.md)

Problem:
SMTP Send ignores ctx and never consults StartTLS. With TLS false, smtp.SendMail upgrades opportunistically but does not require STARTTLS. StartTLS true therefore permits plaintext delivery to a server that does not advertise the extension.

Why it matters:
A configuration that appears to require transport encryption does not enforce it. A canceled or stalled send can continue indefinitely.

Recommendation:
Own SMTP connection establishment with a context-aware dialer and deadlines. Define and enforce implicit TLS and required STARTTLS behavior, failing before sending message data when required encryption is unavailable.

Evidence:
TestObservedSMTPTransport delivered to a plaintext loopback server with StartTLS true, including with an already canceled context. The unused StartTLS field is also documented as a current limitation.

<a id="f22"></a>

### F22 - Refresh the published scaffold dependency baseline

Severity: high
Area: tooling
File: `generator/execute/render_application.go:46-63, generator/book`
Ticket: [Refresh the published scaffold dependency baseline](../ticket/open/20260930211821-refresh-the-published-scaffold-dependency-baseline.md)

Problem:
The updated scaffold uses chi v5.3.2 but still requires the Book-selected Hatmax v0.5.0. A standalone consumer with the same application lifecycle imports resolves older pgx and x/text versions from that published module. govulncheck reports reachable GO-2026-5004 and GO-2026-5970; upgrading the checkout does not upgrade the published module.

Why it matters:
A clean vulnerability scan of this checkout does not prove newly generated applications use the refreshed dependency baseline.

Recommendation:
After an explicitly approved publication, update the Book-supported Hatmax release, scaffold module requirement, and checksums together. Verify the generated consumer without a local replace. If publication is deferred, choose and test explicit dependency overrides rather than assuming the checkout's go.mod affects consumers.

Evidence:
A separate consumer requiring Hatmax v0.5.0 and chi v5.3.2 was tidied and scanned. The scan found four advisory IDs, including two with reachable symbols. It had no local replacement.

<a id="f10"></a>

### F10 - Make scheduler shutdown bounded and idempotent

Severity: medium
Area: architecture
File: `scheduler/runner.go:82-121`
Ticket: [Make scheduler shutdown bounded and idempotent](../ticket/open/20260930211809-bound-and-idempotently-stop-the-scheduler.md)

Problem:
Stop closes the same channel on every call, so a second call panics. It ignores its context while waiting. The run loop does not select ctx.Done, and repeated Start calls launch additional loops.

Why it matters:
Lifecycle callers cannot safely retry shutdown or enforce a shutdown deadline. Cancellation alone leaves polling active.

Recommendation:
Define runner lifecycle states, make repeated shutdown safe, and honor cancellation and shutdown deadlines. Prevent duplicate active loops.

Evidence:
TestObservedRepeatedStopPanic reproduced the closed-channel panic. Cancellation and repeated-start behavior are source-confirmed.

<a id="f11"></a>

### F11 - Distinguish missing settings from persistence failures

Severity: medium
Area: persistence
File: `settings/service.go:27-58`
Ticket: [Distinguish missing settings from persistence failures](../ticket/open/20260930211810-propagate-settings-persistence-failures.md)

Problem:
GetString, GetInt, and GetBool replace every store error with a default value and a nil error. A storage outage is indistinguishable from an absent setting.

Why it matters:
Runtime switches can silently revert during failures. Callers cannot choose whether to preserve prior state, fail closed, or report degraded operation.

Recommendation:
Define a not-found contract for Store. Apply defaults only for the intended missing-value case and propagate other errors, including context cancellation.

Evidence:
TestObservedSettingsFailure returned a valid true default and no error after the fake store reported a storage failure.

<a id="f12"></a>

### F12 - Apply default mail senders before validation

Severity: medium
Area: architecture
File: `mailer/message.go:55-58, mailer/smtp.go:34-42, mailer/mailgun.go:32-41, mailer/sendgrid.go, mailer/ses.go`
Ticket: [Apply default mail senders before validation](../ticket/open/20260930211811-apply-mailer-default-senders-before-validation.md)

Problem:
Every active provider validates Message before applying its configured DefaultFrom. Validate rejects an empty From, so the fallback branch cannot supply a missing sender.

Why it matters:
The documented default sender behavior fails despite a correctly configured provider.

Recommendation:
Normalize the sender before sender-dependent validation in each active provider. Avoid unexpected mutation of the caller's message.

Evidence:
TestObservedDefaultSender received 'from address is required' despite a configured default. docs/reference/mailer/README.md promises this fallback.

<a id="f13"></a>

### F13 - Do not silently drop Mailgun attachments

Severity: medium
Area: architecture
File: `mailer/mailgun.go:32-119`
Ticket: [Do not silently drop Mailgun attachments](../ticket/open/20260930211812-preserve-mailgun-attachments.md)

Problem:
Send serializes a URL-encoded message without reading Message.Attachments and returns success after HTTP 2xx.

Why it matters:
Switching providers can silently remove files from delivered messages while reporting a successful send.

Recommendation:
Implement Mailgun multipart attachments or return an explicit unsupported-content error until attachment support exists. Do not silently report success for omitted content.

Evidence:
TestObservedMailgunAttachment sent to a loopback HTTP server and confirmed that attachment filename and data were absent from the accepted request.

<a id="f15"></a>

### F15 - Bound image ingestion before buffering and decoding

Severity: medium
Area: architecture
File: `image/stdprocessor/processor.go:49-64, image/s3/store.go:77-93`
Ticket: [Bound image ingestion before buffering and decoding](../ticket/open/20260930211814-bound-image-ingestion-resources.md)

Problem:
Resize reads the complete input without a byte limit and decodes the full image before inspecting dimensions. S3 Put also buffers the entire input and creates an additional string copy. Resize does not consult ctx.

Why it matters:
Applications processing untrusted uploads have no primitive-owned limit on encoded bytes or decoded pixel allocation. Output resize limits do not bound input memory.

Recommendation:
Define encoded-size and decoded-pixel limits, inspect image configuration before full decode, and support cancellation where the underlying operation permits it. Avoid unnecessary S3 buffer copies.

Evidence:
Source-confirmed; no unbounded allocation or denial-of-service probe was run.

<a id="f16"></a>

### F16 - Bound generator command output during capture

Severity: medium
Area: tooling
File: `generator/execute/report.go:146-183, generator/execute/application_workspace.go:409-430`
Ticket: [Bound generator command output during capture](../ticket/open/20260930211815-bound-generator-command-output-during-capture.md)

Problem:
Repository validation captures stdout and stderr in an unbounded bytes.Buffer or CombinedOutput. The evidence limit is applied only after the command exits; application staging retains full output.

Why it matters:
A noisy or failing tool can consume unbounded process memory and produce oversized diagnostic state even though the final evidence helper looks bounded.

Recommendation:
Use bounded output capture while continuously draining the child process. Preserve a useful failure summary and truncation marker across both execution paths.

Evidence:
Source-confirmed buffer ownership and post-execution truncation; no unbounded-output process was launched.

<a id="f17"></a>

### F17 - Encode PostgreSQL connection values and schema identifiers

Severity: medium
Area: persistence
File: `config/config.go:375-384, db/database.go:82-95`
Ticket: [Encode PostgreSQL connection values and schema identifiers](../ticket/open/20260930211816-encode-postgres-values-and-schema-identifiers.md)

Problem:
ConnectionString interpolates keyword values without quoting. A password containing a space does not round-trip through pgx.ParseConfig. ensureSchema interpolates a schema name into SQL without identifier quoting.

Why it matters:
Valid credentials or schema names can fail or be interpreted as connection syntax or SQL. Configuration is a trusted input here; no remote injection path was established.

Recommendation:
Use a PostgreSQL-aware connection encoder and quote schema identifiers with the driver's identifier support. Preserve literal configured values rather than treating them as syntax.

Evidence:
TestObservedDSNEncoding reproduced the space-containing password failure. Schema identifier interpolation is source-confirmed.

<a id="f18"></a>

### F18 - Honor configured bcrypt cost during signup

Severity: medium
Area: architecture
File: `auth/service.go:79, model/password.go:17-18, config/config.go:198`
Ticket: [Honor configured bcrypt cost during signup](../ticket/open/20260930211817-honor-configured-bcrypt-cost-during-signup.md)

Problem:
Signup calls model.HashPassword, which always uses bcrypt.DefaultCost. Auth.BCryptCost is validated and defaults to 12, but the resulting password hash has cost 10.

Why it matters:
Operators cannot apply the documented password hashing work factor; configuration gives a false impression of the policy being used.

Recommendation:
Pass the configured cost through the signup hashing boundary while preserving password comparison and the existing standalone helper's compatibility.

Evidence:
TestObservedIgnoredPasswordCost configured cost 12, created a user through Signup, and inspected the resulting bcrypt hash at cost 10.

<a id="f19"></a>

### F19 - Resolve inert scheduler retry configuration

Severity: medium
Area: architecture
File: `scheduler/config.go:14-20, scheduler/runner.go:194-240, scheduler/postgres/store.go:64-72`
Ticket: [Resolve inert scheduler retry configuration](../ticket/open/20260930211818-resolve-the-scheduler-retry-contract.md)

Problem:
RetryAttempts and RetryBackoff are accepted and defaulted, but never used by process. CreateRun always records attempt 1. scheduler/README.md presents these as retry settings while the reference correctly states that retries are not implemented.

Why it matters:
The API advertises operational controls that have no effect. This is a documented capability gap, not a newly discovered hidden implementation promise.

Recommendation:
Choose one explicit retry contract: implement bounded retry/backoff and attempt persistence, or remove/deprecate unsupported controls and align package guidance. Keep this independent of recurring slot advancement.

Evidence:
Source review found no retry execution path; docs/reference/scheduler/README.md:53 explicitly acknowledges the limitation.

<a id="f20"></a>

### F20 - Allow owned and bounded HTTP serving

Severity: medium
Area: architecture
File: `app/lifecycle.go:91-104`
Ticket: [Allow owned and bounded HTTP serving](../ticket/open/20260930211819-allow-owned-and-bounded-http-serving.md)

Problem:
Serve creates a private http.Server with no header timeout or idle timeout and does not expose that server to callers. Shutdown accepts a different caller-owned server, so it cannot shut down the server created by Serve.

Why it matters:
The convenience serving path cannot share the normal lifecycle's server ownership or enforce basic connection bounds. Generated applications already use their own server and do not depend on this helper.

Recommendation:
Provide a serving path with caller-owned server configuration and shutdown identity, or explicitly deprecate the disconnected helper. Set or require an appropriate header timeout without breaking streaming behavior.

Evidence:
Source-confirmed. The canonical generated application is a non-affected comparison, not a reproduction of this helper.

<a id="f21"></a>

### F21 - Reject SMTP header line injection

Severity: medium
Area: architecture
File: `mailer/smtp.go:323-328, mailer/message.go:55-85`
Ticket: [Reject SMTP header line injection](../ticket/open/20260930211820-reject-smtp-header-line-injection.md)

Problem:
Custom Message.Headers are written as raw key/value lines without CR/LF validation. A header value containing CRLF creates an additional header in the delivered MIME message.

Why it matters:
Applications passing user-derived header metadata can introduce unintended headers or terminate the header section. Subject encoding is a separate path and was not identified as vulnerable by this probe.

Recommendation:
Validate custom header names and values before SMTP serialization. Reject line delimiters and invalid field names while preserving legitimate MIME values.

Evidence:
TestObservedSMTPTransport captured X-Review-Injected as a separate header from a CRLF-containing X-Review value.

## Existing ticket retained

[Own generated-project dependencies](../ticket/open/20260929200201-own-generated-project-dependencies.md)
already covers generator tool provisioning and module reconciliation. Source
review confirmed that application staging runs `go mod tidy`, but an external
generation tool can still be invoked before its prerequisites are provisioned.
No duplicate ticket was created. This concern is independent of F22's published
library dependency baseline.

## Dependency modernization

- Root `go.mod` now declares `go 1.26.0`; validation used Go `1.26.7`.
- All three CI jobs select the latest Go 1.26 patch with `go-version: "1.26.x"`
  and `check-latest: true`, rather than selecting the minimum `1.26.0` patch.
- Scaffold output, workspace fixtures, test-created module manifests, and current
  setup guidance use Go 1.26. Historical operational records were not rewritten.
- Updated 23 direct dependency versions and 90 existing selected module versions.
  The final graph adds three module paths and removes 31 obsolete paths.
- `go mod tidy` now classifies the already imported ANSI dependency as direct.
- Testcontainers `v0.44.0` changed the mapped-port type; both test database helpers
  now use `int(port.Num())` instead of the removed `port.Int()`.
- New scaffold chi pins and checksums use `v5.3.2`.
- Unreleased notes record the compatibility and dependency changes. No version
  has been selected.

| Direct module | Before | After |
| --- | --- | --- |
| `charm.land/bubbles/v2` | v2.0.0 | v2.2.1 |
| `charm.land/bubbletea/v2` | v2.0.0 | v2.0.10 |
| `charm.land/lipgloss/v2` | v2.0.0 | v2.0.6 |
| `github.com/aws/aws-sdk-go-v2` | v1.41.1 | v1.47.1 |
| `github.com/aws/aws-sdk-go-v2/config` | v1.32.7 | v1.33.6 |
| `github.com/aws/aws-sdk-go-v2/credentials` | v1.19.7 | v1.20.6 |
| `github.com/aws/aws-sdk-go-v2/service/s3` | v1.95.1 | v1.114.0 |
| `github.com/aws/aws-sdk-go-v2/service/ses` | v1.34.18 | v1.42.1 |
| `github.com/charmbracelet/x/ansi` | v0.11.6 | v0.11.8 |
| `github.com/go-chi/chi/v5` | v5.2.3 | v5.3.2 |
| `github.com/jackc/pgx/v5` | v5.8.0 | v5.11.0 |
| `github.com/knadh/koanf/parsers/yaml` | v1.1.0 | v1.1.1 |
| `github.com/knadh/koanf/providers/posflag` | v1.0.1 | v1.0.2 |
| `github.com/knadh/koanf/providers/rawbytes` | v1.0.0 | v1.0.1 |
| `github.com/knadh/koanf/v2` | v2.3.0 | v2.3.7 |
| `github.com/lib/pq` | v1.10.9 | v1.12.3 |
| `github.com/testcontainers/testcontainers-go` | v0.40.0 | v0.44.0 |
| `github.com/testcontainers/testcontainers-go/modules/postgres` | v0.40.0 | v0.44.0 |
| `golang.org/x/crypto` | v0.46.0 | v0.57.0 |
| `golang.org/x/image` | v0.35.0 | v0.46.0 |
| `golang.org/x/mod` | v0.31.0 | v0.41.0 |
| `golang.org/x/sys` | v0.41.0 | v0.48.0 |
| `golang.org/x/text` | v0.33.0 | v0.42.0 |

The seven unchanged direct modules already resolve to their latest versions
on the current module paths. No major-version path migration was attempted.
After tidy, `go list -m -u all` still lists nine updates for modules that are
not imported by the checkout or its tests. These are dependency-graph metadata,
not older packages used by Hatmax. Explicit upgrades were tried; tidy removes
unnecessary pins. Adding artificial requirements solely to suppress that list
would not update Hatmax code.

## Vulnerability assessment

The baseline source scan reported 46 advisory IDs at module/package/symbol
levels, including 17 with affected-symbol traces. The final checkout scan has
one module-level advisory and no affected-package or affected-symbol findings.

The remaining [GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932) applies to
`golang.org/x/crypto/openpgp`. Hatmax does not import OpenPGP:
`go list -deps ./...` has no matching package. Removing bcrypt or Argon2 would
not be an appropriate response to an unrelated module-level advisory.

A separate consumer with the scaffold's published Hatmax pin and lifecycle
imports reports four advisory IDs. Two have affected-symbol traces:
[GO-2026-5004](https://pkg.go.dev/vuln/GO-2026-5004) in pgx and
[GO-2026-5970](https://pkg.go.dev/vuln/GO-2026-5970) in x/text.
The pgx advisory requires non-default simple protocol and a specific
dollar-quoted-query pattern; reachability alone does not establish that the
generated application is exploitable. F22 tracks refreshing and rechecking
this consumer baseline.

Vulnerability analysis follows the
[Go vulnerability workflow](https://go.dev/doc/security/vuln/).
Dependency selection uses Go's
[module management commands](https://go.dev/doc/modules/managing-dependencies).

## Validation

All successful repository commands used `GOTOOLCHAIN=go1.26.7`.
Database checks used a disposable loopback PostgreSQL 18 cluster and isolated
schemas through the supported `DB_HOST` test path. The owned cluster was stopped
after validation. No shared database or Docker configuration was changed.

| Check | Result |
| --- | --- |
| `make check` | PASS: SPDX coverage, formatting, vet, tests, 80.0% coverage, strict lint |
| `go test -race ./...` | PASS |
| `make docs-check` | PASS |
| `make generator-project-acceptance` | PASS: real generation/build/test/lint commands |
| `make generator-scaffold-acceptance` | PASS |
| `make generator-conversation-acceptance` | PASS |
| `go build ./...` | PASS |
| `go mod verify` | PASS |
| `govulncheck -json ./...` | PASS: no affected symbols; unrelated module-level OpenPGP advisory |
| `make generator-live-smoke` | PASS: authenticated Codex 0.159.2, resident reuse and project thread isolation |
| Temporary defect probes | PASS: 18 top-level tests confirmed observed behavior |
| Ticket metadata, IDs, paths, and report links | PASS |
| `git diff --check` | PASS |

The live smoke took approximately 126 seconds and exercised interpretation,
dialogue, documentation intent, application clarification/planning, and thread
isolation. It does not replace the separate execution acceptance gates.

CI's PostgreSQL 16 service and container-backed test setup were not executed
here. The container mapped-port migration compiled; local database behavior
was validated through the existing native PostgreSQL test mode.

## Non-issues and constraints

- `dal` is the intentional sqlc-generated persistence boundary, not an
  architecture defect.
- Explicit component ordering, server-rendered HTMX, and feature-local
  model/store/service/handler boundaries remain the canonical Hatmax model.
- Runtime settings/clock mutation during scheduler work is explicitly
  unsupported; no false concurrency finding is raised for that constraint.
- PostgreSQL run-slot uniqueness already prevents duplicate claimed slots;
  the weak lifetime of the `ListDue` row lock is not by itself proof of
  duplicate job effects.
- Existing documentation accurately acknowledges several limitations:
  positional rollback, no automatic recurrence or retry, and unused STARTTLS.
  Tickets track closing those gaps rather than reopening documentation delivery.
- No network exploit, unbounded allocation attack, production deployment,
  remote CI result, or new release is claimed.

## Implementation order

1. Close externally influenced identity, filesystem, and trusted HTML boundaries
   (F1-F4).
2. Restore durable delivery and process-failure isolation (F5-F8).
3. Resolve the scheduler's slot ownership and lifecycle contracts (F9-F10).
4. Address the remaining bounded provider, configuration, and resource issues.
5. Refresh the published dependency baseline with explicit release authority
   (F22), then scan a generated consumer without a local replacement.

The dependency and Go migration can be reviewed independently of the open
hardening tickets. It does not resolve them.
