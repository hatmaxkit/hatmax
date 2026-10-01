<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Changelog

All notable changes to Hatmax are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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

- Hatmax and newly scaffolded applications now require Go 1.26 or newer.
- Dependencies now use their latest compatible module versions, including
  updates that address known security advisories.
- Hatmax now uses GNU GPL version 3 only (`GPL-3.0-only`) for its libraries,
  generator, examples, and templates. Earlier MIT-licensed versions retain
  their original terms.
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

- PostgreSQL subscribers no longer miss messages whose transactions commit
  after messages with higher IDs. Per-message delivery tracking survives named
  subscriber restarts and preserves independent fan-out delivery.

### Security

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
