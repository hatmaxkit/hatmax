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

`NewCrashCollector` implements `RecordPanic`. Each call is stored, and
`RecordPanicFunc` runs after that when it is set. `LastRecord` returns
the latest call, or nil. `HasRecordFor` reports an endpoint match.
`GetCalls` returns a copy. `Reset` clears the calls.
