# Changelog

All notable changes to Hatmax are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- General authenticated string encryption with associated-data support and
  deterministic lookup hashes.
- Contact protection configuration keys.
- Managed-property notes protection configuration.

### Changed

- Documentation now provides a reproducible User Guide, focused how-to guides,
  package reference, and design explanations through one Diataxis index.

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
