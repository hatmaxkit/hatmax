# Static Configuration and Runtime Settings

Hatmax has two configuration mechanisms because process configuration and
operator-controlled product settings have different lifetimes.

## Static configuration defines process assembly

`config.Config` is loaded before components are built. It selects server,
database, mailer, scheduler, and other infrastructure values needed to create
the process. File, environment, and flag precedence is resolved once. Changing
the source file does not mutate a running process.

Static configuration is appropriate for values whose change normally requires
new infrastructure or a process restart, such as a database address or TLS
credentials.

## Settings define runtime policy

The `settings` package validates named string values through registered
schemas. A caller supplies the store, so persistence and transaction behavior
remain application decisions. Services such as mailer, scheduler, and
telemetry can read those values while the process is running.

Settings are appropriate for bounded runtime policy such as pausing a
scheduler or selecting a mail delivery mode. They do not rewrite or reload
`config.Config`.

## Defaults need one owner

Static defaults belong to `config.New`. Runtime-setting defaults belong to
their `settings.Schema`. When a service supports both sources, it starts from
static configuration and applies successfully read settings as overrides.
The [Configuration Reference](../../reference/configuration/index.md) lists
the exact precedence and validation behavior.
