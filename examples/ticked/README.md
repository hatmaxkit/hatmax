<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Ticked - HatMax Reference Example

Single-binary todo list application demonstrating HatMax framework patterns with authentication, event-driven architecture, and Postgres-based pub/sub.

A todo list is the archetypical example for a reason: it's familiar, simple to understand, and lets us focus on the framework patterns rather than complex business logic.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       Single Binary :8080                       │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │    Auth     │  │    List     │  │   Admin     │              │
│  │   Handler   │  │   Handler   │  │   Handler   │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
│         │                │                │                     │
│         ▼                ▼                ▼                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │    Auth     │  │    List     │  │   Admin     │              │
│  │   Service   │  │   Service   │  │   Service   │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
│         │                │                │                     │
│         │                │ publish        │                     │
│         │                ▼                │                     │
│         │         ┌─────────────┐         │                     │
│         │         │   PubSub    │─────────┼──────┐              │
│         │         │  (NOTIFY)   │         │      │ subscribe    │
│         │         └──────┬──────┘         │      ▼              │
│         │                │                │ ┌─────────────┐     │
│         │                │                │ │   Audit     │     │
│         │                │                │ │   Service   │     │
│         │                │                │ └──────┬──────┘     │
│         │                │                │        │            │
│         └────────────────┼────────────────┴────────┘            │
│                          ▼                                      │
│                   ┌─────────────┐                               │
│                   │    sqlc     │  Type-safe queries            │
│                   └──────┬──────┘                               │
└──────────────────────────┼──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                        PostgreSQL                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │    auth     │  │   ticked    │  │    audit    │              │
│  │   schema    │  │   schema    │  │   schema    │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

### Patterns Demonstrated

- **Single Binary**: All features compile to one executable with embedded assets
- **HTML + HTMX**: Server-side rendering with dynamic interactions
- **Postgres PubSub**: Domain events via LISTEN/NOTIFY (no external broker)
- **Store Pattern**: Type-safe queries with sqlc
- **Lifecycle Management**: Automatic component discovery via `app.Setup()`
- **Feature-based Organization**: Code organized under `internal/feat/`

## Prerequisites

- Go 1.26.0+
- PostgreSQL running locally
- Make
- sqlc (for regenerating queries)

## Quick Start

### 1. Setup PostgreSQL

Create database and user:

```bash
createdb ticked
createuser -P dev  # password: dev
```

### 2. Run migrations

```bash
make migrate
```

### 3. Build and run

```bash
make run
```

Access the application at http://localhost:8080

## Available Commands

### Application
```bash
make build      # Build the binary
make run        # Build and run
make clean      # Remove binary and logs
```

### Database
```bash
make migrate    # Run database migrations
make sqlc       # Regenerate sqlc queries
```

### Development
```bash
make test       # Run tests
```

## Configuration

Default configuration in `config.yaml`:

```yaml
server:
  port: ":8080"

database:
  host: localhost
  port: 5432
  user: dev
  password: dev
  database: tickedhm
  sslmode: disable

log:
  level: debug
```

Override with environment variables (prefix `TICKED_`):

```bash
TICKED_SERVER_PORT=:9000 make run
TICKED_DATABASE_HOST=db.example.com make run
```

## API Endpoints

Routes follow CQRS-light pattern: verb-noun style, GET for queries, POST for commands.

### Auth
- `GET /signup` - Signup page
- `POST /signup` - Register user
- `GET /signin` - Login page
- `POST /signin` - Authenticate user
- `POST /signout` - Logout

### List
- `GET /list-items` - Todo list view
- `POST /add-item` - Add item
- `POST /toggle-item` - Toggle item completion
- `POST /delete-item` - Delete item

### Admin
- `GET /admin` - Dashboard
- `GET /admin/list-users` - Users list
- `GET /admin/get-user?id=xxx` - User details
- `POST /admin/update-roles` - Update user roles
- `POST /admin/toggle-user` - Toggle user active status
- `GET /admin/list-events` - Audit events

> Use `GET /debug/routes` to list all registered endpoints.

## Credential checking

Signup checks NFC-normalized passwords and stores PHC Argon2id records. The
service requires at least 15 code points, permits long Unicode passwords and
validates finite checking, hashing and concurrency settings at startup.

The example supplies `NewPasswordChecker`, a finite curated list of common
phrases and Ticked-specific values. It retains no submitted candidates, performs
no network work and supports concurrent exact-match checks. This demonstration
list is not a compromised-password corpus and does not establish production
breach coverage. Production consumers must choose and document their own source.

The Postgres adapter enforces email uniqueness and uses versioned user-row
transactions for credential replacement, session insertion and validation.
Ticked owns one explicit password-only policy revision. Only completed outcomes
set a cookie; stronger unavailable requirements grant no session/continuation.
Sessions store purpose-separated token digests, the policy revision and actual
password verification metadata. Validation checks current
account version and expiry after account/session locks, and coalesces relevant
activity without extending absolute expiry. Route consumers select activity
explicitly; background polling must use `NoActivity`. Configuration validates
24-hour absolute and 30-minute inactivity defaults at startup. See the
[authentication storage reference](../../docs/reference/authentication/README.md#queries).

### Session Control

Open **Sessions** from the list page. Reauthenticate with the current password
when recent proof is required. Successful reauthentication replaces the cookie
and invalidates the previous bearer; failure preserves the existing cookie.
The screen lists safe metadata and revokes selected, other or all own sessions.
Policy comes from server wiring, and account roles grant no cross-subject session
management through these routes. New session admission is capped at 10 retained
rows per subject; expired rows are reclaimed without evicting live sessions.

### Authenticator fallback

Optional TOTP/backup routes require externally supplied `TICKED_TOTP_KEY_ID` and
`TICKED_TOTP_KEY` (canonical standard Base64 of 32 bytes). Both absent disables
fallback routes; partial/invalid values fail startup. Never commit key material.
Migration 006 stores encrypted seeds, accepted steps, salted code verifiers and
composite proof. TOTP/backup completion requires an actual password and grants
MFA, while `/authenticators/proof` still requires phishing-resistant proof.
Backup issue keeps recent phishing-resistant management policy. Follow the
[User Guide](../../docs/tutorials/user-guide/identity-and-sessions.md#sign-in-with-totp-or-a-backup-code)
and [reference](../../docs/reference/authentication/README.md#totp-and-backup-proof).

After a recent passkey sign-in or step-up, open `/authenticators/manage` to list
owned primary factors, add or replace a passkey/TOTP authenticator, remove a
factor or regenerate backup codes. TOTP and backup actions require the configured
fallback key. Removing or replacing the factor used by the current session signs
it out; other retained sessions rotate without changing actual proof freshness.
The example fixes phishing-resistant management policy in trusted assembly code.
A TOTP-only application profile must explicitly select permitted MFA management
to upgrade; the request cannot choose weaker policy. Initial setup endpoints
remain restricted to accounts without established factors.

## Mailbox verification

Configure `mailer.enabled: true`, `mailer.mode: active` and an active provider,
then set `TICKED_RECOVERY_ORIGIN` to the application's trusted HTTPS origin.
For explicit localhost development, `authenticator.localhost_development: true`
allows a loopback HTTP origin. No-op/dry-run mail cannot enable verification.

Use `/account/mailbox` to request a link for the current account address, then
submit the form at `/account/mailbox/confirm`. GET never consumes a token.
Confirmation revokes sessions and unfinished security flows while preserving
MFA; sign in again through normal authentication. Migration
`008-account-recovery.sql` adds current verification metadata, bounded token
slots, shared budgets and durable notification intent.

The example performs one bounded notification dispatch after confirmation and
exposes `MailboxDelivery.DispatchMailboxNotices` for application-owned retries.
It starts no retry worker. See the
[mailbox reference](../../docs/reference/authentication/README.md#mailbox-verification)
for the explicit dispatch and deployment limits. Password reset remains unavailable.

## Change a password

The same explicit recovery-origin and active-mail configuration enables
`/account/password`. Submit its protected form from a current, recently authenticated
session. Ticked permits password proof only without established factors; an account
with factors needs phishing-resistant MFA under the fixed server policy. A stale
session must complete the appropriate actual reauthentication or step-up first.

Success replaces the password and signs out every session, including the current
browser. It cancels unfinished security flows and mailbox links while preserving
mailbox verification, authenticators and replay/backup state. No new authentication
cookie is issued. A notification failure leaves the committed change intact;
`DispatchMailboxNotices` supports bounded explicit retries. See the
[password change reference](../../docs/reference/authentication/README.md#recent-proof-password-change).
