<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# config

Application configuration from YAML, env vars, and flags.

## Usage

This fragment belongs in the bootstrap composition root. Import `config`,
`fmt` and `os`; pass a readable YAML file and the complete `os.Args`.

```go
cfg, err := config.Load("config.yaml", "MYAPP_", os.Args)
if err != nil {
    return err
}

if err := cfg.Validate(); err != nil {
    return err
}

// Access config
fmt.Println(cfg.Server.Port)
fmt.Println(cfg.Log.Level)
```

Precedence (highest to lowest):
1. Flags (`--database.host=x`)
2. Env vars (`MYAPP_DATABASE_HOST=x`)
3. YAML file
4. Defaults

## Notes

Static configuration at startup. For dynamic runtime configuration, see `settings/`.

`Load` does not validate automatically. `Validate` checks core startup
constraints; capability constructors validate their own additional settings.
Do not log `ConnectionString()`: it includes the configured database password.
The [Configuration Reference](../docs/reference/configuration/README.md)
describes flag support, environment key mapping and validator boundaries.
