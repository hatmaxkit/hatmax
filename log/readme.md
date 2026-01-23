# log

Structured logging on top of slog.

## Usage

```go
log := log.NewLogger(cfg)

log.Info("server started")
log.Infof("listening on %s", port)
log.Error("connection failed")

// With context
reqLog := log.With("request_id", reqID, "user_id", userID)
reqLog.Info("processing request")
```

Levels: `debug`, `info`, `error`. Configured via `cfg.Log.Level`.

JSON output if `LOG_FORMAT=json`, human-readable text by default.

## API

```go
type Logger interface {
    Debug(v ...any)
    Debugf(format string, a ...any)
    Info(v ...any)
    Infof(format string, a ...any)
    Error(v ...any)
    Errorf(format string, a ...any)
    With(args ...any) Logger
}
```

For tests: `log.NewNoopLogger()` or `log.NewTestLogger("debug")`.
