<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Telemetry

`telemetry` counts requests and aggregates panics. The implementation note is
[telemetry/readme.md](../../../telemetry/readme.md).

`RequestCounter` is `IncrementRequests` and `GetAndResetRequests`.
`NewCounter` returns an `AtomicCounter`. `GetAndResetRequests` returns the
current count and sets it to zero.

`CrashRecorder` is `RecordPanic(message, endpoint, method)`.
`NewCrashCollector` stores events by `message + "|" + endpoint`. A repeat
increments `Count` and updates `LastSeen`. The message stored on the first
event is cut at 200 bytes and may end inside a UTF-8 sequence. This truncation
does not redact secrets. Method is recorded on the first event and does not
participate in the group key. The stack keeps at most 10 function names.
`GetAndResetCrashes` returns the events and clears the map. An empty
collector returns nil.

There is no bound on distinct group keys, message key length, or endpoint length.
The key retains the original untruncated message. Supply safe messages and
endpoints and drain periodically; the collector is not an admission or redaction
boundary. Request counters and crash collectors synchronize their own updates.

`KeyMode` is `telemetry.mode`. Its values are `off`, `basic`, `full`, and
`debug`. The default is `basic`. `KeyInstanceID` is `telemetry.instance_id`.
`RegisterSchemas` registers both settings.
These are exporter policy values, not automatic behavior: collectors and
middleware do not read modes, schedule export, or send reports. The application
owns settings lookup, safe export and any retry or retention policy.
