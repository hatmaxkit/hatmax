<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# fake

Test doubles for mail and telemetry. Put this complete test in your package's
`mailer_test.go`; it checks the adapter directly without a transport:

## Usage

```go
package example

import (
	"context"
	"testing"

	"hatmax.adrianpk.com/fake"
	"hatmax.adrianpk.com/mailer"
)

// TestWelcome verifies the validated message recorded by the fake mailer.
func TestWelcome(t *testing.T) {
	fm := fake.NewMailer().WithValidation()
	message := &mailer.Message{
		From:    mailer.Address{Email: "noreply@example.com"},
		To:      []mailer.Address{{Email: "user@example.com"}},
		Subject: "Welcome",
		Text:    "Hello",
	}
	if err := fm.Send(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if fm.SendCount() != 1 || !fm.HasMessageTo("user@example.com") ||
		!fm.HasMessageWithSubject("Welcome") || fm.LastMessage() != message {
		t.Fatal("welcome message was not recorded")
	}
	fm.Reset()
	if fm.SendCount() != 0 || fm.LastMessage() != nil {
		t.Fatal("reset retained mail")
	}
}
```

Inject the fake into the consumer service when testing an application workflow.
`SendCount` counts attempted calls, including validation or `SendFunc` failures;
only successful calls appear in `GetMessages`. Messages are retained as pointers,
so the fake does not snapshot later mutation. Configure hooks and output before
concurrent use. Hooks run while holding the fake's mutex and must not reenter it.
`WithOutput(writer)` prints message bodies; use only test data. See the
[Fake Reference](../docs/reference/fake/README.md) for telemetry doubles.
