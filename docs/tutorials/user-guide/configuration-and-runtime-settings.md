<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Configuration and Runtime Settings

Hatmax applications have two configuration lifetimes:

```text
static configuration -> loaded and validated before construction -> restart to change
runtime settings      -> schema-checked through a service         -> change while running
```

Keeping them separate makes startup reproducible and prevents mutable product
preferences from quietly changing infrastructure dependencies.

## Load Static Configuration Once

The composition root loads configuration before it constructs components:

The Go fragments use the illustrative invoice bootstrap. Import `config`,
`fmt`, `os`, `log` and `settings`, and supply an application `settings.Store`.
The [Guide companion](../../../examples/guide/README.md) is the runnable
composition with a memory store and greeting-setting routes.

```go
cfg, err := config.Load("config.yaml", "INVOICES_", os.Args)
if err != nil {
	fmt.Fprintf(os.Stderr, "cannot load config: %v\n", err)
	os.Exit(1)
}

if err := cfg.Validate(); err != nil {
	fmt.Fprintf(os.Stderr, "cannot validate config: %v\n", err)
	os.Exit(1)
}
```

Values are resolved from flags, prefixed environment variables, the YAML file,
and defaults, in that precedence order. `Load` and `Validate` are separate;
successful parsing does not mean the values form an operable application.

Use a readable YAML file and a non-empty argument list that starts with the
program name. Unknown flags exit the process. The environment mapper replaces
all underscores with dots; for keys such as `auth.session_ttl`, use YAML or
the registered flag. See the reference for the exact loading boundary.

Pass the resulting `*config.Config` to constructors that own the relevant
capability. Feature handlers and stores should not reload files or read
environment variables independently. This keeps one startup snapshot and one
place to diagnose invalid configuration.

Static configuration includes server addresses, database connection details,
authentication policy, capability enablement, polling intervals, worker
counts, and provider credentials. Secrets can arrive through environment
expansion, but must never be printed in startup errors or logs.

Neither the logger nor `Config` redacts secrets. Log only safe operation names
and selected non-secret fields; `Database.ConnectionString()` includes the
password. The snapshot lifetime is an application convention: the exported
struct is mutable, and consumers can capture fields during construction.

## Construct Logging from the Startup Snapshot

Create the logger immediately after configuration validation:

```go
logger := log.NewLogger(cfg)
```

The configured level controls debug, info, and error output. Use `With` to add
stable context such as a request, feature, or record identifier. Record the
operation and wrapped error at the boundary that handles it; do not emit the
same failure at every layer.

Logging is operational output, not a user response and not a durable audit
model. Sensitive values remain excluded even at debug level. The current
logger reads `LOG_FORMAT=json` directly from the environment to select JSON;
other values use text output.

## Define Mutable Settings with Schemas

Runtime settings begin with an application-owned registry:

```go
registry := settings.NewRegistry()
minPageSize, maxPageSize := 1, 100
registry.Register(settings.Schema{
	Key:       "invoice.page_size",
	Type:      settings.Int,
	Default:   "20",
	Min:       &minPageSize,
	Max:       &maxPageSize,
	Label:     "Invoices per page",
	Required:  true,
})

settingService := settings.NewService(registry, settingStore)
```

Schemas define type, required state, bounds, enum choices, defaults, and
display labels. The application owns the registry and the `settings.Store`
adapter; Hatmax does not provide a Postgres settings store.

Registered values are validated before `Set`. Your adapter must distinguish an
absent key with `settings.ErrNotFound` from a present value or a read failure.
Only absence selects the registered default. Typed getters propagate storage,
cancellation, and parse errors; always check those errors before using a value.

A stored empty string stays empty through `GetString`; empty numbers and
booleans return parse errors. Delete a key to restore its default instead of
storing an empty string. Keep the failure policy in the application workflow:
decide whether to fail the operation, preserve a last known value, or show an
error rather than allowing persistence failure to silently change behavior.

Do not use settings for database credentials, encryption keys, listener ports,
or dependencies whose construction changes. Updating a setting does not
reload `config.Config` or rebuild the component graph.

## Put Setting Changes Behind an Application Workflow

A settings page is still a feature boundary. Its handler parses form values,
the service authorizes the actor and calls `settings.Service.Set`, and the
template renders safe schema feedback. A durable adapter persists the change;
an in-memory adapter deliberately loses it at restart.

Readers of a setting depend on `settings.Service`, not directly on the store.
Pass that service into the component that owns the behavior. This keeps
defaults, parsing, and storage policy consistent across HTTP and background
work.

## Decide by Lifetime

Use this test when placing a value:

| Question | Static configuration | Runtime setting |
| --- | --- | --- |
| Needed to construct infrastructure? | yes | no |
| Must be valid before startup? | yes | no |
| Change requires component restart? | usually | no |
| Edited by an authorized product user? | no | often |
| Stored through an application adapter? | no | yes |

The same conceptual value should not exist independently in both layers. If a
runtime setting overrides a startup default, define that precedence as an
application contract rather than relying on whichever value a caller happens
to read.

For exact loading, schemas, typed access, and logging behavior, see
[Configuration](../../reference/configuration/README.md) and
[Logging](../../reference/logging/README.md). The rationale is in
[Static Configuration and Runtime Settings](../../explanation/configuration-boundaries/README.md).

---

[Previous: Identity and Sessions](identity-and-sessions.md) ·
[User Guide](README.md) ·
[Next: Events and Background Work](events-and-background-work.md)
