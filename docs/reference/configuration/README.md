<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Configuration

Hatmax has two configuration layers.

Static configuration is a `config.Config` value loaded at process start. It
does not change while the process is running. The implementation note is
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

### Groups

| Group | Fields |
| --- | --- |
| `log` | `level` |
| `server` | `port`, `host` |
| `database` | `host`, `port`, `user`, `password`, `database`, `schema`, `sslmode` |
| `auth` | session lifecycle and credential policy/work fields below, `email_encryption_key`, `email_lookup_key` |
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
| `scheduler.batch_size` is less than 1 | `scheduler.batch_size must be at least 1` |
| `scheduler.workers` is less than 1 | `scheduler.workers must be at least 1` |
| `scheduler.retry_attempts` is less than 1 | `scheduler.retry_attempts must be at least 1` |

### Derived values

`DatabaseConfig.ConnectionString` returns
`host='<host>' port=<port> user='<user>' password='<password>' dbname='<database>' sslmode='<sslmode>'`.
Each string value uses PostgreSQL keyword/value escaping: apostrophes and
backslashes are backslash-escaped, and empty values remain explicit `''`.
Spaces, quotes, and connection-option text remain part of their configured
value. Existing driver validation and defaults still apply.

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

`Schema.Validate` accepts an empty raw value when `Required` is false, without
checking the type. A required empty value returns `setting "<key>" is required`.
`MaxLength` above zero rejects a longer raw value. `bool` uses
`strconv.ParseBool`. `int` uses `strconv.Atoi` and rejects values outside
`Min` or `Max` when those pointers are set. `enum` accepts only an entry in
`Options`. `MaxLength` of zero does not limit length.

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
