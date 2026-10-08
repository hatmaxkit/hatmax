<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# log

Structured logging on top of slog.

## Usage

This fragment belongs in the bootstrap/request composition with `cfg`, `port`,
`reqID` and `userID`. Import the Hatmax `log` package. Supply safe identifiers
and messages; the logger does not redact secrets.

```go
logger := log.NewLogger(cfg)

logger.Info("server started")
logger.Infof("listening on %s", port)
logger.Error("connection failed")

// With context
reqLog := logger.With("request_id", reqID, "user_id", userID)
reqLog.Info("processing request")
```

Levels: `debug`, `info`, `error`. Configured via `cfg.Log.Level`.

JSON output if `LOG_FORMAT=json`, human-readable text by default.
Both output format and level are captured when the logger is constructed.

## API

The structural contract below uses the imported `log.Logger` return type:

```go
type Logger interface {
    Debug(v ...any)
    Debugf(format string, a ...any)
    Info(v ...any)
    Infof(format string, a ...any)
    Error(v ...any)
    Errorf(format string, a ...any)
    With(args ...any) log.Logger
}
```

For tests: `log.NewNoopLogger()` or `log.NewTestLogger("debug")`.
