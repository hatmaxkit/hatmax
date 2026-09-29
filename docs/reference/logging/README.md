# Logging

`log` writes to standard output through `slog`. The implementation note is
[log/readme.md](../../../log/readme.md).

## Logger

```text
Debug(v ...any)
Debugf(format string, a ...any)
Info(v ...any)
Infof(format string, a ...any)
Error(v ...any)
Errorf(format string, a ...any)
With(args ...any) Logger
```

`With` returns a logger that carries the extra `slog` fields and keeps the
same level. The unformatted methods join their arguments with `fmt.Sprint`.
The formatted methods use `fmt.Sprintf`.

## Level

`NewLogger(cfg)` reads `cfg.Log.Level`. `NewTestLogger(level)` uses the given
string. Recognized values are case-insensitive:

| Values | Level |
| --- | --- |
| `debug`, `dbg` | debug |
| `info`, `inf` | info |
| `error`, `err` | error |

Any other string, including an empty string, selects info.

A message is emitted when the logger's level is at or below that message's
level. Debug is level 0, info is level 1, and error is level 2. An info logger
emits info and error. An error logger emits error.

## Output

`NewLogger` and `NewTestLogger` read `LOG_FORMAT` from the environment, not
from `config.Config`. The value `json` selects `slog`'s JSON handler. Any
other value selects the text handler. Both handlers write to standard output
and use the same level limit as the `Logger` method.

## Noop logger

`NewNoopLogger` returns a logger whose methods do nothing. `With` on that
logger returns the same noop logger.
