<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Configuration

Hatmax has two configuration layers.

Static configuration is a `config.Config` value loaded at process start. The
application keeps it as a startup snapshot; its exported fields are mutable,
and constructors may capture them. The implementation note is
[config/readme.md](../../../config/readme.md).

Settings are runtime key-value records checked against a schema. The
implementation note is
[settings/readme.md](../../../settings/readme.md).

## Static configuration

`config.New` returns a `Config` filled with the defaults below. `config.Load`
starts from those defaults and then applies a YAML file, environment
variables, and flags.

Precedence, from highest to lowest:

1. Flags.
2. Environment variables that use the prefix passed to `Load`.
3. The YAML file, after `os.ExpandEnv`.
4. The defaults from `New`.

`Load(path, envPrefix, args)` requires a readable file at `path`. A missing
file or invalid YAML returns an error. `args` must be non-empty. `args[0]` is
the program name and `args[1:]` are the flags. The flag set uses
`pflag.ExitOnError`, so a flag parse failure exits the process instead of
returning an error.

An environment variable is mapped by removing `envPrefix`, lowercasing the
remainder, and turning each `_` into `.`. `HATMAX_DATABASE_HOST` with prefix
`HATMAX_` becomes `database.host`.

`Load` does not call `Validate`.

Environment mapping replaces every underscore, including underscores within a
field name. For example, `HATMAX_AUTH_SESSION_TTL` becomes `auth.session.ttl`,
not `auth.session_ttl`, and does not override that field. Use YAML or the
registered `--auth.session_ttl` flag for such keys. Unknown YAML/environment
keys do not become new `Config` fields. YAML `${NAME}` expansion is a separate
mechanism and can inject a value into a key that contains underscores.

Flags are registered explicitly in `Load`. The groups below have corresponding
flags except `authenticator`, whose fields must come from YAML or supported
environment mappings. An unknown flag exits the process. Omitted flags retain
the loaded YAML/environment value; their defaults fill only missing keys.
An unset variable used in YAML expansion becomes an empty string.

### Groups

| Group | Fields |
| --- | --- |
| `log` | `level` |
| `server` | `port`, `host` |
| `database` | `host`, `port`, `user`, `password`, `database`, `schema`, `sslmode` |
| `auth` | session lifecycle and credential policy/work fields below, `email_encryption_key`, `email_lookup_key` |
| `authenticator` | RP identity/origins, development opt-in and enrollment bounds below |
| `credential_admission` | durable credential admission bounds below |
| `authentication_ingress` | HTTP peer/concurrency/acknowledgment bounds below |
| `security_observation` | bounded event-delivery timeout and concurrency |
| `recovery` | mailbox token lifetimes, lease, timeout, attempt budgets and cleanup bounds below |
| `contact` | `pii_encryption_key`, `email_lookup_key` |
| `property` | `notes_protection_key` |
| `pubsub` | `enabled`, `poll_interval`, `batch_size` |
| `scheduler` | `enabled`, `interval`, `batch_size`, `workers`, `retry_attempts`, `retry_backoff` |
| `mailer` | `enabled`, `mode`, `provider`, `default_from.email`, `default_from.name`, and the `smtp`, `mailgun`, `sendgrid`, and `ses` provider fields |

### Defaults

| Key | Default |
| --- | --- |
| `log.level` | `info` |
| `server.port` | `:8080` |
| `server.host` | `localhost` |
| `database.host` | `localhost` |
| `database.port` | `5432` |
| `database.user` | `dev` |
| `database.password` | `dev` |
| `database.database` | `dev` |
| `database.schema` | empty |
| `database.sslmode` | `disable` |
| `auth.session_ttl` | `24h` |
| `auth.session_inactivity_ttl` | `30m` |
| `auth.session_activity_interval` | `1m` |
| `auth.session_timeout` | `5s` |
| `auth.session_recent_proof_age` | `5m` |
| `auth.session_max_per_subject` | `10` |
| `auth.session_page_size` | `50` |
| `auth.session_cleanup_batch` | `1000` |
| `auth.password_min_len` | `15` |
| `auth.password_max_len` | `1024` |
| `auth.password_max_bytes` | `4096` |
| `auth.password_check_timeout` | `2s` |
| `auth.password_timeout` | `5s` |
| `auth.argon_memory_kib` | `65536` |
| `auth.argon_iterations` | `3` |
| `auth.argon_parallelism` | `4` |
| `auth.argon_max_memory_kib` | `65536` |
| `auth.argon_max_iterations` | `3` |
| `auth.argon_max_parallelism` | `4` |
| `auth.password_max_concurrent` | `2` |
| `pubsub.enabled` | `false` |
| `pubsub.poll_interval` | `100ms` |
| `pubsub.batch_size` | `100` |
| `scheduler.enabled` | `false` |
| `scheduler.interval` | `1m` |
| `scheduler.batch_size` | `20` |
| `scheduler.workers` | `1` |
| `scheduler.retry_attempts` | `3` |
| `scheduler.retry_backoff` | `1m` |
| `mailer.enabled` | `false` |
| `mailer.mode` | `disabled` |
| `mailer.provider` | `smtp` |
| `mailer.default_from.email` | `noreply@localhost` |
| `mailer.smtp.port` | `587` |
| `mailer.ses.region` | `us-east-1` |

`contact` and `property` keys default to empty. Auth encryption and lookup
keys also default to empty.

Credential settings apply to [signup/sign-in](../authentication/README.md).
`AuthConfig.PasswordSettings()` returns a validated snapshot. Minimum is 15
through the character maximum; maximum is 64 through 1024 code points. Byte
capacity covers four bytes per maximum code point and cannot exceed 4096.
Checker and credential timeouts must be positive and at most 30 seconds.
Empty timeout strings and zero optional maxima resolve to documented defaults.

Argon2 creation cost and verification ceilings follow the
[verifier resource contract](../authentication/README.md#versioned-password-verifier).
Zero cost/ceiling/concurrency fields use verifier defaults; nonzero values must
fit supported per-operation and aggregate bounds. Numeric values are range-
checked before conversion to narrower KDF parameter types. The verifier is
shared per service; full admission returns a busy error without a waiting queue.
`auth.bcrypt_cost` and the former bcrypt helpers are removed.

### Validation

`Validate` returns an error when any of these holds:

| Check | Error |
| --- | --- |
| `server.port` is empty | `server.port is required` |
| `database.host` is empty | `database.host is required` |
| `database.user` is empty | `database.user is required` |
| `database.database` is empty | `database.database is required` |
| Credential policy, timeout or Argon2 work settings violate the bounds above | A credential configuration error; startup is rejected |
| Session lifetime, cadence, capacity or cleanup settings violate their bounds | A session configuration error |
| Recovery settings violate their bounds | A recovery configuration error |
| Security observation timeout or concurrency violates its bounds | A security observation configuration error |
| `scheduler.batch_size` is less than 1 | `scheduler.batch_size must be at least 1` |
| `scheduler.workers` is less than 1 | `scheduler.workers must be at least 1` |
| `scheduler.retry_attempts` is less than 1 | `scheduler.retry_attempts must be at least 1` |

This is the complete `Config.Validate` boundary. It does not check database
reachability, port range, SSL mode, log level, provider credentials, or all
capability settings. Authenticator, credential-admission and HTTP-ingress
settings have separate validation methods and constructor checks. Duration
fallback helpers for pubsub/scheduler are separate from strict constructor
validation. A successful `Validate` is not a connection or deployment check.

### Derived values

`DatabaseConfig.ConnectionString` returns
`host='<host>' port=<port> user='<user>' password='<password>' dbname='<database>' sslmode='<sslmode>'`.
Each string value uses PostgreSQL keyword/value escaping: apostrophes and
backslashes are backslash-escaped, and empty values remain explicit `''`.
Spaces, quotes, and connection-option text remain part of their configured
value. Existing driver validation and defaults still apply.

The result contains the password. Do not print it, the full `Config`, or secret
provider fields in diagnostics. The configuration and logging packages do not
redact caller-supplied values automatically.

A non-empty schema appends `search_path` containing one double-quoted SQL
identifier, itself escaped as a connection value. Schema names preserve case,
spaces, punctuation, and embedded quotes; the field is not a search-path list
or SQL expression. An empty schema omits `search_path` and retains the driver's
default. NUL-containing connection strings are rejected by pgx.

`PubSubConfig.PollIntervalDuration` parses `poll_interval`. An invalid value
returns 100 milliseconds.

`SchedulerConfig.IntervalDuration` and `RetryBackoffDuration` parse their
fields. An invalid value returns one minute.

`scheduler.retry_attempts` bounds total attempts for a new job slot, including
the first; `1` disables retries. `retry_backoff` is a fixed delay after a failed
attempt. Persisted retry waits retain their original total budget across runner
restarts. See the [scheduler retry contract](../scheduler/README.md#retries).

## Settings

A setting is a `Value`: `Key`, `Raw`, and `UpdatedAt`.

A `Schema` names the key and its constraints. `Type` is `string`, `int`,
`bool`, or `enum`. `Registry.Register` stores a schema by key and overwrites
an existing key. `Get` reports whether the key exists. `All` returns the
schemas. `ByPrefix` returns the schemas whose key has that prefix.

`All` and `ByPrefix` have no guaranteed ordering. Registry lookup protects the
map with a mutex; schema slice/pointer fields are shallow copies. Treat schemas
as immutable after registration. Registration and typed reads do not validate
defaults. `Set` validates registered values; a typed read parses the stored raw
value without rechecking numeric bounds or enum membership.

`Schema.Validate` accepts an empty raw value when `Required` is false, without
checking the type. A required empty value returns `setting "<key>" is required`.
`MaxLength` above zero rejects a longer raw value. `bool` uses
`strconv.ParseBool`. `int` uses `strconv.Atoi` and rejects values outside
`Min` or `Max` when those pointers are set. `enum` accepts only an entry in
`Options`. `MaxLength` of zero does not limit length.

`MaxLength` counts UTF-8 bytes. `Secret` is UI metadata, not encryption,
redaction or access control. The application owns those boundaries.

`DisplayLabel` returns `Label`, or `Key` when `Label` is empty.
`NamespaceSchema.DisplayLabel` does the same for a namespace key.

`Store` is `Get`, `Set`, `All`, and `Delete`. The settings package does not
supply a Postgres store.

`Store.Get` must return `settings.ErrNotFound` only for an absent key. Adapters
may wrap that sentinel; the service checks it with `errors.Is`. Translate
backend-specific absence, such as `sql.ErrNoRows`, at the adapter boundary.
Present values, including `""`, return nil error. Read failures, cancellation,
and expired deadlines must remain errors rather than being reported as absence.

Custom adapters that previously used `"", nil`, an arbitrary error, or a
backend-specific error for absence must adopt this contract. Matching error
text is not sufficient. Delete a key to restore its default; storing an empty
string is no longer a default-reset operation.

`NewService(registry, store)` returns a `Service`.

| Method | Result |
| --- | --- |
| `GetString` | The stored raw value, including `""`. Only `ErrNotFound` selects the schema default, or `""` for an unregistered key. Any other store error returns `""` and that error unchanged. |
| `GetInt` | The stored value parsed with `Atoi`. Only `ErrNotFound` selects the parsed default. An absent key with no default returns `0`, nil. Invalid stored values, including `""`, or non-empty invalid defaults return parse errors. Other store errors return `0` and the original error. |
| `GetBool` | The stored value parsed with `ParseBool`. Only `ErrNotFound` selects the parsed default. An absent key with no default returns `false`, nil. Invalid stored values, including `""`, or non-empty invalid defaults return parse errors. Other store errors return `false` and the original error. |
| `Set` | Validates the value when the key is registered, then stores it. An unregistered key is stored without validation. |
| `Delete` | Deletes the key through the store. |
| `All` | Returns the stored values. |

Always check a getter's error before using its value. Defaults are not written
back to the store. An invalid default matters only when the key is absent; it
does not replace a present value. `Schema.Validate` still allows optional
empty values, but typed getters do not interpret a stored empty number or
boolean as a valid value.

`ParseBool` and `ParseInt` parse a raw string. An empty string returns the
zero value and a nil error. `FormatBool` and `FormatInt` render a value with
`strconv`.

These standalone parsing helpers retain their empty-to-zero behavior; they
do not implement the service's absence/default policy.

Settings do not reload `config.Config`.

Session lifetime/cadence/batch constraints are in the [authentication reference](../authentication/README.md#session-lifecycle). Invalid configured session settings fail construction.

Session management requires recent proof (1s through absolute lifetime). Retained
session capacity and page size are 1 through 100; zero selects 10 and 50
respectively. Invalid values fail `auth.NewService` construction. See the
[authentication contract](../authentication/README.md#reauthentication-and-control).

## Authenticator Enrollment

`Config.Authenticator` is consumed explicitly by `NewAuthenticatorService`; it
requires `rp_id`, `rp_name` and one through eight exact `origins`. Other applications
that do not construct that service need no RP configuration. The Ticked example
configures localhost development on port 8080 explicitly. Production selects an
HTTPS origin and disables `localhost_development`.

The constructor validates `pending_ttl`, `recent_proof_age`, `timeout`, `lease`,
`budget_window`, `cooldown`, `max_pending`, `max_authenticators`, `pending_attempts`,
`subject_attempts`, `max_concurrent` and `cleanup_batch`. Empty/zero fields select
finite defaults; malformed or out-of-range values fail construction. A lease
shorter than the total operation timeout fails. No request-header fallback exists.
See [enrollment bounds](../authentication/README.md#restricted-webauthn-enrollment).

### Fallback constructor settings

`config.FallbackConfig` is explicit constructor configuration, separate from RP
identity. Supply `Issuer`, optional `StrictStep`, optional `BackupCodes` (default
8, range 1..10), and `Limits` using the same finite authenticator admission values.
`Settings()` validates these without requiring RP/origins. Seed key identities and
32-byte key material are explicit `auth.SeedKeys` dependencies, never defaults in
`Config`. Ticked opts in with `TICKED_TOTP_KEY_ID` and `TICKED_TOTP_KEY` (canonical
standard Base64); both absent leaves fallback routes disabled, partial/invalid
values fail startup. See the [fallback contract](../authentication/README.md#totp-and-backup-proof).

## Recovery limits

`Config.Recovery` is validated through `RecoverySettings()` at construction and
by `Config.Validate`. Empty duration strings and zero integer fields select
these defaults; malformed or out-of-range nonzero values fail.

| Key | Default | Bound |
| --- | --- | --- |
| `recovery.verification_ttl` | `24h` | `1m`–`24h` |
| `recovery.reset_ttl` | `1h` | `1m`–`1h` |
| `recovery.timeout` | `5s` | `1s`–`30s` |
| `recovery.lease` | `5s` | `1s`–`30s`, at least the operation timeout |
| `recovery.token_attempts` | `5` | 1–10 |
| `recovery.issuance_attempts` | `3` | 1–10 per subject/purpose per fixed 15m window |
| `recovery.completion_attempts` | `10` | 1–20 per subject per fixed 15m window |
| `recovery.cleanup_batch` | `1000` | 1–1000 |

Durations must use microsecond precision. Terminal token retention is fixed at
24h. Notification retention and dispatch bounds belong to the application
adapter. See the [mailbox reference](../authentication/README.md#mailbox-verification)
and [password reset contract](../authentication/README.md#mailbox-password-reset).

## Credential admission limits

`credential_admission` supplies validated durable operation budgets. Optional
zero fields resolve to defaults; explicit invalid values fail construction.
`CredentialAdmissionSettings.Check` rejects unresolved or invalid adapter input.

| Field | Default | Bound |
| --- | --- | --- |
| `password_attempts` | 10 | 1–20 |
| `password_window`, `password_cooldown` | `10m` each | `1m`–`1h`, microsecond precision |
| `registration_attempts` | 3 | 1–10 |
| `registration_window`, `registration_cooldown` | `1h` each | `1m`–`1h`, microsecond precision |
| `max_identities` | 10000 | 100–100000 across both purposes |
| `timeout` | `1s` | `100ms`–`5s`, microsecond precision; earlier caller deadlines apply |
| `cleanup_batch` | 1000 | 1–1000 retired records per call |

Namespace/private key provisioning is an application adapter dependency, not a
request field or core key-management service. Restart and access-policy changes
cannot reset a live budget. These settings bound the mandatory shared admission
service supplied to authentication construction. Ticked requires
`TICKED_CREDENTIAL_NAMESPACE` and `TICKED_CREDENTIAL_KEY` (canonical base64,
32–64 decoded bytes). All replicas must use stable application-owned material
and consistent settings. Missing or invalid material fails startup; do not put
the key in source, public examples or logs.

## Authentication HTTP ingress

`authentication_ingress` supplies process-local HTTP admission settings. The
Ticked example installs one `AuthenticationIngress` for all authentication POST
routes after the trusted-proxy middleware and before body parsing/account work.
This complements durable per-identity admission; it does not replace it.
`Config.AuthenticationIngressSettings()` validates and snapshots these values.

| Field | Default | Bound |
| --- | --- | --- |
| `peer_requests` | 12 | 1–1000 accepted requests per fixed window |
| `peer_window` | `1m` | `1s`–`1h` |
| `max_peers` | 1024 | 1–10000 live peer counters |
| `max_active` | 32 | 1–128 requests, including acknowledgment waits |
| `cleanup_batch` | 128 | 1–1000 counters inspected per admission/cleanup |
| `acknowledgment` | `6s` | At least whole work + `200ms`; at most `31s` |

Whole work is the largest configured credential, factor or recovery timeout,
with a `5s` floor for existing recovery transport work. Raising those timeouts
requires a matching acknowledgment target. The margin reserves `100ms` for
observation and `100ms` for response scheduling. Ticked shares one initialized
security observation instance across its authentication services.

Construction starts no worker or ticker. Unknown peers are refused when the
finite table is full; live counters are never evicted. Ports, IPv4-mapped values
and equivalent IPv6 spellings share canonical buckets. Untrusted forwarded
headers cannot change the connection peer. Install explicit `ProxyHeaders`
trust policy before ingress when using a trusted reverse proxy.

Ticked binds body reads to the captured whole-work deadline. Its HTTP server
sets finite header/read timeouts and a write timeout one second longer than the
acknowledgment target. It closes admissions and cancels request contexts before
infrastructure shutdown. Canceled acknowledgment waits release active capacity.
Syntax/body, peer/active capacity and origin errors may differ before account
lookup; public password failures carry no account-specific retry information.

## Security observation

`security_observation` supplies `SecurityObservationConfig` to
`NewSecurityObservations`. Invalid values fail construction and `Config.Validate`.

| Field | Default | Bound |
| --- | --- | --- |
| `timeout` | `100ms` | `1ms`–`100ms`, bounded by the caller deadline |
| `concurrency` | 2 | 1–16 callbacks, non-waiting admission |

Zero optional fields select defaults. The application supplies one cooperative,
concurrency-safe observer shared by authentication services. Neither constructor
starts a worker. Saturation skips callback invocation and increments a fixed
diagnostic counter; callback failure cannot change authentication authority.
See the [security observation contract](../authentication/README.md#security-observations).
