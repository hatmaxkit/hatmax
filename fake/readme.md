# fake

Test doubles for hatmax interfaces.

## Usage

```go
// Fake mailer records all sent messages
fm := fake.NewMailer()
fm.WithOutput(os.Stdout)  // print emails to stdout
fm.WithValidation()       // fail on invalid messages

// Use in tests
svc := NewService(fm)
svc.SendWelcome(ctx, user)

// Assert
if fm.SendCount() != 1 { t.Error("expected 1 email") }
if !fm.HasMessageTo("user@example.com") { t.Error("wrong recipient") }
if !fm.HasMessageWithSubject("Welcome") { t.Error("wrong subject") }

msg := fm.LastMessage()
fm.Reset()  // clear for next test
```
