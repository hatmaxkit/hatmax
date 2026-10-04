<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Changelog

All notable changes to Hatmax are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Users can change a password after recent authentication. Accounts with MFA
  require current qualifying MFA proof; a successful change signs out every
  session and preserves mailbox verification and authenticators. Ticked provides
  a protected change form with durable notification and bounded mail retry.

- Applications can verify a current mailbox through a bounded one-use email
  link. Verification signs out existing sessions and requires normal sign-in
  again, preserving configured MFA. Ticked provides protected verification forms
  when active mail delivery and a trusted link origin are configured.

- Applications can enroll an initial WebAuthn authenticator after fresh password
  verification, sign in with verified user presence and verification, and step
  up a live session for phishing-resistant access. Setup invalidates previous
  sessions; successful step-up replaces the old bearer. Ticked provides bounded
  JSON endpoints with trusted RP/origin configuration. Applications can also
  complete MFA with actual password/TOTP or one-use backup codes. TOTP seeds use
  application-owned encryption keys; backup sets require recent management
  proof and codes are shown once. These methods do not satisfy phishing-resistant
  access. Applications can list, add, replace and remove their own authenticators
  under recent management proof. Changes revoke other sessions and pending
  ceremonies, protect the last usable factor and replace a retained session
  bearer without refreshing its proof. Removing its factor signs the actor out.

- Scheduled jobs now retry handler failures within a configured total attempt
  limit and fixed delay. PostgreSQL retry waits survive restarts without resetting
  their budget; recurring schedules advance only after terminal completion.
- `hm` now provides a resumable conversational TUI for creating canonical
  compiling Hatmax applications, optionally with initial features, and for
  evolving existing Hatmax projects through explicit digest-bound plans.
- User-local conversation state now retains bounded dialogue, proposals,
  diagnostics, reset history, and safe recovery without storing repository
  contents or reusable approval.
- Authenticated acceptance now covers ordinary dialogue, application
  clarification and planning, approval recomputation, scope rebinding,
  resident runtime reuse, and project thread isolation.

### Changed

- The former hidden-clock TOTP and shared-salt backup helpers are removed.
  Applications adopting fallback authentication must provide explicit seed keys
  and typed storage that consumes accepted steps or codes with session completion.

- Sessions now expire after 24 hours or 30 minutes of inactivity by default,
  reject stale account state, and store only token digests. Custom auth stores
  must implement atomic validation and activity updates; middleware must select
  relevant activity explicitly so background polling does not extend sessions.
  Password reauthentication now rotates the bearer atomically, and recent-proof
  management lists and revokes own sessions. New admission is bounded per user
  without evicting live sessions. Invalid session configuration fails construction. Sign-in and validation
  require explicit server proof policy and revision. Password-only verification cannot satisfy MFA; unmet requirements return non-authorizing outcomes,
  and policy changes or stale proof reject access before activity is renewed.

- Authentication now checks complete NFC-normalized passwords before signup and
  stores salted Argon2id credentials. Password-only access requires at least 15
  Unicode code points and supports long Unicode passwords. Supply a bounded
  password checker and implement atomic credential-state checks in custom stores;
  bcrypt credentials, helpers and `auth.bcrypt_cost` are no longer supported.

- `app.Serve` now takes a caller-owned `*http.Server`; use that same instance
  for `app.Shutdown`. Missing header and idle timeouts default to 5 and 60
  seconds, while positive custom limits and streaming response policy remain
  under application control. Existing router-and-port calls must migrate.
- Custom scheduler stores must support atomic retry claims and persisted retry
  waits. Existing PostgreSQL job tables require the additive retry-column upgrade
  included in the scheduler schema before running the upgraded scheduler.
- Custom scheduler stores now own atomic run completion and schedule advancement.
  PostgreSQL recurring jobs use documented JSON schedule specifications;
  existing non-empty opaque specifications must be converted.
- Manually assembled application startup lists now use `app.StartupStep` values
  with an optional stop function per component. Existing calls using the outputs
  of `app.Setup` keep the same source form.
- Dependencies now use their latest compatible module versions, including
  updates that address known security advisories.
- Hatmax's libraries, generator, examples, and templates are now licensed under
  Apache License 2.0 (`Apache-2.0`), allowing use in proprietary applications
  under its terms. Adrian PK authorizes this licensing as the sole author and
  copyright holder, effective 2026-10-02.
- `hm` is the canonical command for interactive and headless generation.
  `hatmax` remains a behaviorally equivalent compatibility alias for the first
  two tagged minor releases containing `hm`.
- The `hm` TUI now uses the full terminal as a restrained chat surface, keeps
  exceptional controls and active phases in one contextual footer, groups
  related clarification questions for one combined reply, animates
  indeterminate work, and presents human plan summaries before optional
  typed-plan and diagnostic details.
- Complete application requests now preserve an explicitly supplied Go module
  path through target inspection instead of asking for it again.

### Fixed

- Concurrent signup requests now report a duplicate-email outcome consistently.
  Sign-in cannot create a session from credentials or account state changed
  during verification; stale credential writes cannot overwrite newer state.

- Applications created with initial HTMX features now load their templates
  instead of failing at startup because required template helpers were missing.
- PostgreSQL connection values now preserve spaces, quotes, backslashes, and
  empty passwords. Schema creation and selection use the configured name
  literally, including case and punctuation, instead of treating it as SQL.
- Generator validation now limits noisy command output during execution while
  retaining initial context and final failure details.
- SMTP delivery now stops when canceled and has a total 30-second limit, with
  shorter caller deadlines respected throughout connection and message delivery.
- Mailgun now sends attachments instead of silently dropping them, preserving
  filenames, file contents, and MIME types alongside the message fields.
- Email providers now apply the configured default sender before validation,
  allowing messages to omit `From` without changing the caller's message.
  Explicit senders continue to override the default.
- Runtime settings now use defaults only for missing keys, without hiding
  persistence or cancellation failures. Custom stores must report absence with
  `settings.ErrNotFound`; explicit empty values no longer reset a setting to
  its default. Delete the key to reset it.
- Scheduler shutdown is now safe to repeat and respects the caller's deadline.
  Startup cancellation stops polling and active execution; repeated starts no
  longer launch duplicate loops. Restarting after shutdown requires a new runner.
- Completed scheduled jobs no longer remain stuck on an exhausted slot.
  One-shot jobs retire; daily, weekly, and interval jobs advance after success
  or failure. Result and schedule changes are saved together, and claimed slots
  no longer consume later due batches.
- Application startup failure now stops only successfully started components
  that own a stop capability, in reverse order, without stopping the wrong
  component or losing the original error when capabilities are mixed.
- Scheduled job handler panics no longer terminate the application when using
  concurrent workers or interrupt a single-worker batch. Failed runs are recorded
  and healthy jobs continue; failed-state persistence errors are logged.
- PostgreSQL subscribers no longer miss messages whose transactions commit
  after messages with higher IDs. Per-message delivery tracking survives named
  subscriber restarts and preserves independent fan-out delivery.
- PostgreSQL subscribers now retry messages whose handlers return an error,
  including after a named restart, without replaying successful messages from
  the same batch. Retries wait for the configured polling interval; persistent
  failures remain pending for application remediation.

### Security

- Newly scaffolded applications now select corrected PostgreSQL driver and
  Unicode text dependencies even when using the published Hatmax v0.5.0 library.
  Existing applications must update those dependencies separately.
- Email delivery now rejects custom header names with invalid characters and
  values containing CR or LF before transport, preventing additional headers
  or a premature message body. Pass unfolded custom values instead of folded lines.
- Image resizing and S3 uploads now reject encoded inputs above 20 MiB.
  The standard processor rejects images above 25 million pixels before full
  decoding and checks cancellation between input reads and processing phases.
- SMTP now enforces `StartTLS=true` instead of silently sending in plaintext
  when the server does not advertise STARTTLS. Both STARTTLS and implicit TLS
  verify certificates unless verification is explicitly disabled.
- Forged forwarded-IP headers no longer bypass internal network restrictions
  or reset rate limits. Proxy deployments must explicitly configure trusted
  proxy networks; internal restrictions always check the connection peer.
- Local image storage now confines reads, writes, and deletes to its configured
  root, rejecting path traversal and preventing escapes through symbolic links.
- Out-of-band HTMX swaps now escape selector values instead of allowing quotes
  to inject HTML attributes or elements. Wrappers accept supported paired HTML
  content tags; malformed or unsupported tags now panic.
- UI links, navigation, breadcrumbs, and form actions now use standard Go
  template URL filtering. Unsupported schemes such as `javascript:` and `data:`
  are neutralized; relative URLs, HTTP, HTTPS, and `mailto:` remain supported.

## [0.5.0] - 2026-09-29

### Added

- `hatmax generate` now turns natural-language feature requests into an
  inspectable Hatmax plan, requires explicit approval, applies only canonical
  Book-owned project changes, reuses a resident Codex session, and reports
  conformance and repository validation evidence.
- Explicit documentation requests now generate the smallest warranted
  tutorial, how-to, reference, or explanation surface from sealed Hatmax
  evidence, preserving user-authored Markdown outside managed sections.
- Generated projects follow the canonical Hatmax package, wiring, persistence,
  rendering, validation, and testing patterns and pass the discovered project
  generation, formatting, build, test, and lint commands before completion.

## [0.4.0] - 2026-09-28

### Added

- Applications can encrypt arbitrary strings with authenticated associated
  data and derive deterministic keyed hashes for normalized lookup values.
  Existing email-encryption helpers remain compatible.
- Router middleware can be installed through application options, and
  state-changing browser requests can be restricted to same-origin sources.
- Form handlers can expose structured, user-safe validation feedback and apply
  password-strength or equality rules to fields.
- Configuration now supports encryption and lookup keys for authentication
  email addresses, contact data, and managed-property notes.
- Database integrations can identify PostgreSQL migration and asset sets
  through a shared engine identifier.

### Changed

- Documentation now provides a reproducible User Guide, focused how-to guides,
  package reference, design explanations, and runnable companion applications
  through one Diataxis index.

## [0.1.0] - 2026-02-17

### Added

- Core packages: `app`, `auth`, `config`, `crypto`, `db`, `htmx`, `log`,
  `model`, `render`, `validation`, and `web`.
- Authentication with session management and two-factor authentication
  enforcement support.
- TOTP primitives for multi-factor authentication.
- Runtime settings with schema validation and UI metadata.
- Internationalization, mail delivery, pubsub, scheduling, pagination,
  slugs, seeding, telemetry, image handling, and test helpers.
- The Ticked reference application with authentication and administration.
