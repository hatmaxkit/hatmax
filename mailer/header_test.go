// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package mailer

import (
	"context"
	"io"
	"mime"
	"net/mail"
	"reflect"
	"strings"
	"testing"
)

// Untrusted metadata must not create another field or a premature body. Both
// public validation and the shared raw builder reject it without partial output.
func TestHeaderValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "empty name"},
		{name: "space name", key: "X Bad"},
		{name: "tab name", key: "X\tBad"},
		{name: "colon name", key: "X:Bad"},
		{name: "CR name", key: "X\rBad"},
		{name: "LF name", key: "X\nBad"},
		{name: "CRLF name", key: "X\r\nInjected"},
		{name: "NUL name", key: "X\x00Bad"},
		{name: "DEL name", key: "X\x7fBad"},
		{name: "high byte name", key: "X\xffBad"},
		{name: "Unicode name", key: "X-\u00e9"},
		{name: "CR value", key: "X-Notice", value: "safe\rInjected"},
		{name: "LF value", key: "X-Notice", value: "safe\nInjected"},
		{name: "new header", key: "X-Notice", value: "safe\r\nX-Injected: yes"},
		{name: "body boundary", key: "X-Notice", value: "safe\r\n\r\nInjected body"},
		{name: "folded value", key: "X-Notice", value: "safe\r\n continued"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := senderMessage(Address{Email: "sender@example.com"}, false, "")
			msg.Headers = map[string]string{tc.key: tc.value}
			before := map[string]string{tc.key: tc.value}

			err := msg.Validate()
			if err == nil {
				t.Fatal("Validate accepted unsafe headers")
			}

			raw, err := (&SMTPMailer{}).buildRawMessage(msg)
			if err == nil || raw != nil {
				t.Fatalf("buildRawMessage returned %q, %v; want no bytes and an error", raw, err)
			}

			if !reflect.DeepEqual(msg.Headers, before) {
				t.Fatal("validation changed custom headers")
			}
		})
	}
}

// Field-name bounds are email ftext, not the narrower HTTP token grammar.
// Values retain whitespace, punctuation, MIME encoded words, and Unicode.
func TestHeaderValues(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers map[string]string
	}{
		{name: "nil"},
		{name: "empty map", headers: map[string]string{}},
		{name: "empty value", headers: map[string]string{"X-Empty": ""}},
		{name: "whitespace", headers: map[string]string{"X-Notice": "first\tsecond third"}},
		{name: "punctuation", headers: map[string]string{"X-Notice": "a:b; c=d (comment) <notice@example.com>"}},
		{name: "encoded word", headers: map[string]string{"X-Notice": mime.QEncoding.Encode("utf-8", "Invoice \u00e9")}},
		{name: "Unicode value", headers: map[string]string{"X-Notice": "Invoice \u00e9"}},
		{name: "name bounds", headers: map[string]string{"!": "first", "~": "last"}},
		{name: "email punctuation", headers: map[string]string{"X()[]@,;=": "allowed"}},
		{name: "multiple", headers: map[string]string{"X-First": "one", "X-Second": "two"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := senderMessage(Address{Email: "sender@example.com"}, false, "")
			msg.Headers = tc.headers

			err := msg.Validate()
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}

			raw, err := (&SMTPMailer{}).buildRawMessage(msg)
			if err != nil {
				t.Fatal(err)
			}

			parsed, err := mail.ReadMessage(strings.NewReader(string(raw)))
			if err != nil {
				t.Fatal(err)
			}

			for key, value := range tc.headers {
				if got := parsed.Header.Get(key); got != value {
					t.Errorf("header %q = %q; want %q", key, got, value)
				}
			}
		})
	}
}

// Unsafe headers must fail before SMTP or provider API delivery. Defaults are
// applied to a local copy and cannot mutate caller data even on rejection.
func TestHeaderRejection(t *testing.T) {
	for _, provider := range []string{"smtp", "mailgun", "sendgrid", "ses", "ses raw", "noop"} {
		t.Run(provider, func(t *testing.T) {
			deliveries := make(chan senderDelivery, 1)

			var m Mailer = NewNoopMailer(nil)

			if provider != "noop" {
				m = senderProvider(t, provider, Address{Email: "default@example.com"}, deliveries)
			}

			from := Address{}
			if provider == "noop" {
				from.Email = "sender@example.com"
			}

			msg := senderMessage(from, provider == "ses raw", "")
			msg.Headers = map[string]string{"X-Notice": "safe\r\nX-Injected: yes"}
			before := senderMessage(from, provider == "ses raw", "")
			before.Headers = map[string]string{"X-Notice": "safe\r\nX-Injected: yes"}

			err := m.Send(context.Background(), msg)
			if err == nil || err.Error() != "custom header value contains a line break" {
				t.Fatalf("Send: %v; want custom-header rejection", err)
			}

			if !reflect.DeepEqual(msg, before) {
				t.Fatal("Send changed the caller's message")
			}

			select {
			case got := <-deliveries:
				t.Fatalf("unsafe headers reached transport: %+v", got)
			default:
			}
		})
	}
}

// Parse captured SMTP DATA and SES raw payloads to verify actual delivery still
// preserves headers and encoded subjects across each MIME body shape.
func TestHeaderDelivery(t *testing.T) {
	for _, provider := range []string{"smtp", "ses raw"} {
		for _, body := range []string{"text", "HTML", "alternative", "attachment"} {
			t.Run(provider+"/"+body, func(t *testing.T) {
				deliveries := make(chan senderDelivery, 1)
				m := senderProvider(t, provider, Address{Email: "default@example.com"}, deliveries)
				msg := senderMessage(Address{}, provider == "ses raw" || body == "attachment", "")
				msg.Subject = "Invoice \u00e9"
				msg.Headers = map[string]string{"X-Notice": "invoice: draft; revision=2"}

				switch body {
				case "text":
					msg.HTML = ""
				case "HTML":
					msg.Text = ""
				}

				err := m.Send(context.Background(), msg)
				if err != nil {
					t.Fatal(err)
				}

				got := <-deliveries
				if got.err != nil {
					t.Fatal(got.err)
				}

				parsed, err := mail.ReadMessage(strings.NewReader(got.raw))
				if err != nil {
					t.Fatal(err)
				}

				if parsed.Header.Get("X-Notice") != msg.Headers["X-Notice"] || parsed.Header.Get("X-Injected") != "" {
					t.Fatalf("unexpected delivered headers: %v", parsed.Header)
				}

				subject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
				if err != nil || subject != msg.Subject {
					t.Fatalf("subject = %q, %v; want %q", subject, err, msg.Subject)
				}

				content, err := io.ReadAll(parsed.Body)
				if err != nil || len(content) == 0 {
					t.Fatalf("message body = %q, %v", content, err)
				}
			})
		}
	}
}
