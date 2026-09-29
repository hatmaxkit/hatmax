# Telemetry

`telemetry` counts requests and aggregates panics. The implementation note is
[telemetry/readme.md](../../../telemetry/readme.md).

`RequestCounter` is `IncrementRequests` and `GetAndResetRequests`.
`NewCounter` returns an `AtomicCounter`. `GetAndResetRequests` returns the
current count and sets it to zero.

`CrashRecorder` is `RecordPanic(message, endpoint, method)`.
`NewCrashCollector` stores events by `message + "|" + endpoint`. A repeat
increments `Count` and updates `LastSeen`. The message stored on the first
event is cut at 200 characters. The stack keeps at most 10 function names.
`GetAndResetCrashes` returns the events and clears the map. An empty
collector returns nil.

`KeyMode` is `telemetry.mode`. Its values are `off`, `basic`, `full`, and
`debug`. The default is `basic`. `KeyInstanceID` is `telemetry.instance_id`.
`RegisterSchemas` registers both settings.
