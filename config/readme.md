# config

Application configuration from YAML, env vars, and flags.

## Usage

```go
cfg, err := config.Load("config.yaml", "MYAPP_", os.Args)
if err != nil {
    log.Fatal(err)
}

if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

// Access config
fmt.Println(cfg.Server.Port)
fmt.Println(cfg.Database.ConnectionString())
```

Precedence (highest to lowest):
1. Flags (`--database.host=x`)
2. Env vars (`MYAPP_DATABASE_HOST=x`)
3. YAML file
4. Defaults

## Notes

Static configuration at startup. For dynamic runtime configuration, see `settings/`.
