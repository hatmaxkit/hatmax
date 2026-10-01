<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
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
| `auth` | `session_ttl`, `password_min_len`, `bcrypt_cost`, `email_encryption_key`, `email_lookup_key` |
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
| `auth.password_min_len` | `8` |
| `auth.bcrypt_cost` | `12` |
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

### Validation

`Validate` returns an error when any of these holds:

| Check | Error |
| --- | --- |
| `server.port` is empty | `server.port is required` |
| `database.host` is empty | `database.host is required` |
| `database.user` is empty | `database.user is required` |
| `database.database` is empty | `database.database is required` |
| `auth.password_min_len` is less than 1 | `auth.password_min_len must be at least 1` |
| `auth.bcrypt_cost` is outside 4 through 31 | `auth.bcrypt_cost must be between 4 and 31` |
| `scheduler.batch_size` is less than 1 | `scheduler.batch_size must be at least 1` |
| `scheduler.workers` is less than 1 | `scheduler.workers must be at least 1` |
| `scheduler.retry_attempts` is less than 1 | `scheduler.retry_attempts must be at least 1` |

### Derived values

`DatabaseConfig.ConnectionString` returns
`host=<host> port=<port> user=<user> password=<password> dbname=<database> sslmode=<sslmode>`.
A non-empty schema appends ` search_path=<schema>`.

`PubSubConfig.PollIntervalDuration` parses `poll_interval`. An invalid value
returns 100 milliseconds.

`SchedulerConfig.IntervalDuration` and `RetryBackoffDuration` parse their
fields. An invalid value returns one minute.

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
