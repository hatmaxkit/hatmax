<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Fake

`fake` records mail and telemetry calls for tests. The implementation note is
[fake/readme.md](../../../fake/readme.md).

## Mailer

`NewMailer` returns a `Mailer` that implements `mailer.Mailer`. `Send`
appends a `MailerSendCall` with the context and message. When
`FailOnValidation` is true, a `Validate` error is returned and the message is
not stored. `SendFunc`, when set, decides the returned error. A nil error
stores the message. Without `SendFunc`, `Send` stores the message, prints it
when `Output` is set, and returns nil.

`SendCount` counts attempted calls, including failures. Calls are recorded before
validation; `GetMessages` includes only successful sends. Stored calls and
messages retain the original message pointer, not a deep copy. Configure hooks,
validation and output before concurrent use. Hooks execute while holding the
fake's mutex and must not call back into that same fake. Printed diagnostics
include message bodies; use only fixture data.

`Reset` clears calls and messages. `GetSendCalls`, `GetMessages`,
`SendCount`, `LastMessage`, `HasMessageTo`, `HasMessageWithSubject`, and
`HasMessageContaining` inspect the stored messages. `SetOutput` and
`WithOutput` set the writer. `WithValidation` sets `FailOnValidation`.

## Telemetry

`NewCounter` implements `telemetry.RequestCounter`. `IncrementRequests`
records the call, adds one, and calls `IncrementFunc` when it is set.
`GetAndResetRequests` returns the count and sets it to zero, or returns
`GetAndResetFunc` when it is set.
`WithCount` sets the count. `Reset` clears the count and the call counts.
An installed `GetAndResetFunc` supplies the result instead of clearing the
simulated count. The telemetry fake hooks also run under their fake's mutex.

`NewCrashCollector` implements `RecordPanic`. Each call is stored, and
`RecordPanicFunc` runs after that when it is set. `LastRecord` returns
the latest call, or nil. `HasRecordFor` reports an endpoint match.
`GetCalls` returns a copy. `Reset` clears the calls.
