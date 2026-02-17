# Changelog

All notable changes to hatmax will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-17

### Added

- Core packages: `app`, `auth`, `config`, `crypto`, `db`, `htmx`, `log`, `model`, `render`, `validation`, `web`
- Authentication with session management and 2FA enforcement support
- TOTP primitives for two-factor authentication (`crypto/totp`)
- Settings package for runtime configuration with schema validation and UI metadata
- i18n package for internationalization
- Mailer package with SMTP, SendGrid, and AWS SES support
- PubSub package for publish/subscribe messaging
- Middleware: rate limiting, authentication, CORS
- Pagination helpers
- Slug generation package
- Seed reference utilities
- Fake package for test doubles (mailer)
- UUID nullable helpers
- Example application: ticked (todo list with auth and admin)
