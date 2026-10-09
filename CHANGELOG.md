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

## [0.6.0] - 2026-10-09

### Added

- `hm` extends assisted generation with a conversational terminal UI, application
  creation from scratch, optional initial features and persistent conversation
  history. Resume a project to add features, fields, validation and explicitly
  requested documentation. Every change requires its own inspectable plan and
  approval; `hm generate` remains available for focused terminal requests.
- Accounts can verify an email address, change a password after recent
  authentication and recover a forgotten password through a one-use email link.
  These changes sign out existing sessions and preserve configured MFA.
  Ticked provides verification, change and recovery forms with security notices
  and bounded mail retries.
- Applications can enroll and manage passkeys, sign in with WebAuthn and confirm
  sensitive actions with recent passkey proof. Authenticator changes revoke
  other sessions and pending ceremonies while protecting the last usable factor.
- Applications can use password/TOTP and one-use backup codes for MFA and
  recent confirmation. TOTP requires application-owned encryption keys; backup
  codes are shown once. These fallback methods do not provide phishing-resistant
  proof. Applications can list and revoke their own sessions after confirmation.
- Registration and password authentication can enforce durable per-identity
  attempt budgets across replicas and restarts. Ticked requires stable
  `TICKED_CREDENTIAL_NAMESPACE` and `TICKED_CREDENTIAL_KEY` values and bounds
  peers, active requests and request bodies. Registration does not automatically
  sign in; duplicate registrations and public password failures avoid exposing
  account existence.
- Applications can observe classified authentication outcomes through a bounded
  callback that excludes credentials and raw identities. Delivery failures do
  not change authentication results.
- Scheduled jobs support bounded retries with a fixed delay. PostgreSQL retry
  waits and attempt budgets survive restarts; recurring schedules advance after
  terminal completion.
- The User Guide, setup instructions, package reference and generator guidance
  now cover the delivered behavior with verified examples and reader workflows.

### Changed

- Password authentication uses salted Argon2id credentials and checks complete
  NFC-normalized passwords before signup. Password-only access requires at least
  15 Unicode code points and supports long Unicode passwords. Custom stores must
  implement atomic credential-state checks and applications must supply a bounded
  password checker. Bcrypt credentials, helpers and `auth.bcrypt_cost` are no
  longer supported.
- Sessions expire after 24 hours or 30 minutes of inactivity by default and store
  only token digests. Custom stores must atomically validate session/account state
  and update relevant activity; background polling must not extend sessions.
  Reauthentication rotates the bearer. Sign-in and validation require an explicit
  proof policy and revision; password-only proof cannot authorize MFA access.
- `app.Serve` takes a caller-owned `*http.Server`; pass that same instance to
  `app.Shutdown`. Missing header and idle timeouts default to 5 and 60 seconds.
  Manually assembled startup lists use `app.StartupStep` with an optional stop
  function; calls using the outputs of `app.Setup` keep their source form.
- Custom scheduler stores must atomically claim retries, complete runs and advance
  schedules. Existing PostgreSQL job tables require the additive retry-column
  upgrade. Non-empty opaque recurring schedule specifications must migrate to
  the documented JSON format.
- The former hidden-clock TOTP and shared-salt backup helpers are removed.
  Fallback authentication requires explicit seed keys and storage that consumes
  accepted steps or codes atomically with session completion.
- `hm` is the canonical command. `hatmax` remains an equivalent compatibility
  alias for the first two tagged minor releases containing `hm`.
- The generator admits Hatmax v0.6.x projects. New application scaffolds retain
  their explicitly pinned, published v0.5.0 dependency baseline.
- Dependencies use updated compatible versions, including security corrections.
- Hatmax's libraries, generator, examples and templates are licensed under
  Apache License 2.0, effective 2026-10-02.

### Fixed

- Generated applications can create and evolve features without moving wiring
  into their entrypoint. Required timestamp additions preserve compatible SQLC
  rows and stored values; generated validation tests pass project lint.
- Applications created with initial HTMX features load their required template
  helpers. Complete application requests preserve an explicitly supplied Go
  module path, and generator diagnostics retain useful context within finite
  output limits.
- Concurrent signups consistently report duplicate email addresses. Sign-in
  rejects credentials or account state changed during verification; stale
  credential writes cannot overwrite newer state.
- PostgreSQL connection values preserve spaces, quotes, backslashes and empty
  passwords. Schema names retain their literal case and punctuation.
- SMTP delivery respects cancellation and a total 30-second limit. Mailgun sends
  attachments with their filenames and MIME types. Email providers apply the
  configured default sender when a message omits `From`.
- Runtime settings use defaults only for missing keys. Custom stores must report
  absence with `settings.ErrNotFound`; explicit empty values are retained and
  persistence or cancellation failures are returned. Delete a key to reset it.
- Scheduler shutdown is repeatable and respects deadlines. Cancellation stops
  polling and active work; repeated starts do not create duplicate loops.
  Handler panics fail their run without terminating healthy jobs. Completed
  one-shot jobs retire and recurring jobs advance without becoming stuck.
- Application startup failure stops successfully started components in reverse
  order without stopping unrelated components or losing the original error.
- PostgreSQL subscribers retain messages whose transactions commit out of order
  and retry failed handlers after named restarts without replaying successful
  messages from the same batch.

### Security

- New application scaffolds select corrected PostgreSQL driver and Unicode text
  dependencies while retaining their published Hatmax baseline. Existing
  applications must update those dependencies separately.
- Email delivery rejects invalid custom header names and values containing CR
  or LF. Pass unfolded values instead of folded lines.
- Image processing and S3 uploads reject encoded inputs above 20 MiB. The standard
  image processor rejects images above 25 million pixels before full decoding
  and checks cancellation between processing phases.
- SMTP with `StartTLS=true` rejects servers that do not advertise STARTTLS.
  STARTTLS and implicit TLS verify certificates unless explicitly disabled.
- Forwarded-IP headers cannot bypass internal-network restrictions or reset rate
  limits. Proxy deployments must explicitly configure trusted proxy networks;
  internal restrictions check the connection peer.
- Local image storage rejects traversal and symbolic-link escapes from its root.
- HTMX out-of-band swaps escape selector values. Wrappers accept supported paired
  HTML tags and reject malformed or unsupported tags.
- UI links, navigation, breadcrumbs and form actions apply Go template URL
  filtering, neutralizing unsupported schemes such as `javascript:` and `data:`.

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
