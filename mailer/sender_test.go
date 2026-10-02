// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package mailer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/textproto"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
)

type senderDelivery struct {
	from string
	raw  string
	err  error
}

// Provider boundaries must send the effective sender without changing the
// caller's message. Invalid messages must fail before any transport request.
func TestSenderPolicy(t *testing.T) {
	defaultFrom := Address{Email: "default@example.com", Name: "Default Sender"}
	explicitFrom := Address{Email: "explicit@example.com", Name: "Explicit Sender"}

	for _, provider := range []string{"smtp", "mailgun", "sendgrid", "ses", "ses raw"} {
		for _, tc := range []struct {
			name        string
			from        Address
			defaultFrom Address
			want        Address
			invalid     string
			wantErr     string
		}{
			{name: "default", defaultFrom: defaultFrom, want: defaultFrom},
			{name: "explicit", from: explicitFrom, defaultFrom: defaultFrom, want: explicitFrom},
			{name: "name only", from: Address{Name: "Unused Name"}, defaultFrom: defaultFrom, want: defaultFrom},
			{name: "explicit without default", from: explicitFrom, want: explicitFrom},
			{name: "no sender", wantErr: "from address is required"},
			{name: "default name only", defaultFrom: Address{Name: "No Email"}, wantErr: "from address is required"},
			{name: "missing recipient", defaultFrom: defaultFrom, invalid: "recipient", wantErr: "at least one recipient is required"},
			{name: "empty recipient", defaultFrom: defaultFrom, invalid: "empty recipient", wantErr: "recipient 0 has empty email"},
			{name: "missing subject", defaultFrom: defaultFrom, invalid: "subject", wantErr: "subject is required"},
			{name: "missing body", defaultFrom: defaultFrom, invalid: "body", wantErr: "message body is required (text or html)"},
		} {
			t.Run(provider+"/"+tc.name, func(t *testing.T) {
				deliveries := make(chan senderDelivery, 1)
				m := senderProvider(t, provider, tc.defaultFrom, deliveries)
				msg := senderMessage(tc.from, provider == "ses raw", tc.invalid)
				before := senderMessage(tc.from, provider == "ses raw", tc.invalid)

				err := m.Send(context.Background(), msg)
				if !reflect.DeepEqual(msg, before) {
					t.Fatal("Send changed the caller's message")
				}

				if tc.wantErr != "" {
					if err == nil || err.Error() != tc.wantErr {
						t.Fatalf("got %v; want %q", err, tc.wantErr)
					}

					select {
					case got := <-deliveries:
						t.Fatalf("invalid message reached transport: %+v", got)
					default:
					}

					return
				}

				if err != nil {
					t.Fatalf("Send: %v", err)
				}

				select {
				case got := <-deliveries:
					if got.err != nil {
						t.Fatalf("capture delivery: %v", got.err)
					}

					want := tc.want.String()
					if provider == "smtp" || provider == "ses raw" {
						want = tc.want.Email
					}

					if got.from != want {
						t.Fatalf("sender: got %q; want %q", got.from, want)
					}

					if got.raw != "" {
						raw, parseErr := mail.ReadMessage(strings.NewReader(got.raw))
						if parseErr != nil {
							t.Fatalf("parse MIME: %v", parseErr)
						}

						if raw.Header.Get("From") != tc.want.String() {
							t.Fatalf("MIME sender: got %q; want %q", raw.Header.Get("From"), tc.want.String())
						}
					}
				case <-time.After(5 * time.Second):
					t.Fatal("delivery was not captured")
				}
			})
		}
	}
}

func senderMessage(from Address, raw bool, invalid string) *Message {
	msg := &Message{
		From: from, To: []Address{{Email: "recipient@example.com"}},
		CC: []Address{{Email: "copy@example.com"}}, BCC: []Address{{Email: "blind@example.com"}},
		ReplyTo: &Address{Email: "reply@example.com"},
		Subject: "Notice", Text: "Text body", HTML: "<p>HTML body</p>",
	}
	if raw {
		msg.Attachments = []Attachment{{Filename: "invoice.txt", ContentType: "text/plain", Data: []byte("invoice")}}
	}

	switch invalid {
	case "recipient":
		msg.To = nil
	case "empty recipient":
		msg.To = []Address{{}}
	case "subject":
		msg.Subject = ""
	case "body":
		msg.Text, msg.HTML = "", ""
	}

	return msg
}

func senderProvider(t *testing.T, provider string, from Address, deliveries chan<- senderDelivery) Mailer {
	t.Helper()

	if provider == "smtp" {
		return senderSMTP(t, from, deliveries)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := senderDelivery{}

		switch provider {
		case "sendgrid":
			var payload struct {
				From struct{ Email, Name string }
			}

			got.err = json.NewDecoder(r.Body).Decode(&payload)
			got.from = (Address{Email: payload.From.Email, Name: payload.From.Name}).String()
		case "mailgun", "ses", "ses raw":
			got.err = r.ParseForm()
			got.from = r.Form.Get("from")

			if provider != "mailgun" {
				got.from = r.Form.Get("Source")
			}

			if provider == "ses raw" {
				var raw []byte

				raw, got.err = base64.StdEncoding.DecodeString(r.Form.Get("RawMessage.Data"))
				got.raw = string(raw)
			}
		}

		deliveries <- got

		if strings.HasPrefix(provider, "ses") {
			action := r.Form.Get("Action")

			w.Header().Set("Content-Type", "text/xml")
			fmt.Fprintf(w, "<%sResponse xmlns=\"http://ses.amazonaws.com/doc/2010-12-01/\"><%sResult><MessageId>test-message</MessageId></%sResult></%sResponse>", action, action, action, action)

			return
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	cfg := Config{DefaultFrom: from}

	switch provider {
	case "mailgun":
		return NewMailgunMailer(MailgunConfig{Config: cfg, APIKey: "test-key", Domain: "example.com", BaseURL: server.URL})
	case "sendgrid":
		m := NewSendGridMailer(SendGridConfig{Config: cfg, APIKey: "test-key"})
		m.client.BaseURL = server.URL

		return m
	default:
		client := ses.NewFromConfig(aws.Config{
			Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("test-key", "test-secret", ""),
			HTTPClient: server.Client(),
		}, func(o *ses.Options) { o.BaseEndpoint = aws.String(server.URL) })

		return &SESMailer{cfg: SESConfig{Config: cfg}, client: client}
	}
}

func senderSMTP(t *testing.T, from Address, deliveries chan<- senderDelivery) Mailer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	t.Cleanup(func() {
		listener.Close()
		<-done
	})

	go func() {
		defer close(done)

		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()

		conn.SetDeadline(time.Now().Add(5 * time.Second))
		wire := textproto.NewConn(conn)
		wire.PrintfLine("220 local SMTP ready")

		got := senderDelivery{}

		for {
			line, readErr := wire.ReadLine()
			if readErr != nil {
				return
			}

			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				wire.PrintfLine("250 local SMTP")
			case strings.HasPrefix(line, "MAIL FROM:"):
				got.from = strings.Trim(strings.TrimPrefix(line, "MAIL FROM:"), "<>")

				wire.PrintfLine("250 sender accepted")
			case strings.HasPrefix(line, "RCPT TO:"):
				wire.PrintfLine("250 recipient accepted")
			case line == "DATA":
				wire.PrintfLine("354 send message")

				raw, dataErr := wire.ReadDotBytes()

				got.raw, got.err = string(raw), dataErr
				deliveries <- got

				wire.PrintfLine("250 message accepted")
			case line == "QUIT":
				wire.PrintfLine("221 closing")

				return
			default:
				wire.PrintfLine("500 unsupported command")
			}
		}
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}

	return NewSMTPMailer(SMTPConfig{Config: Config{DefaultFrom: from}, Host: host, Port: portNumber})
}

// Direct validation remains independent of a provider's defaults, and noop
// retains its explicit-sender requirement rather than becoming an active provider.
func TestSenderValidation(t *testing.T) {
	msg := senderMessage(Address{}, false, "")

	for _, validate := range []struct {
		name string
		run  func() error
	}{
		{"message", msg.Validate},
		{"noop", func() error { return NewNoopMailer(nil).Send(context.Background(), msg) }},
	} {
		t.Run(validate.name, func(t *testing.T) {
			err := validate.run()
			if err == nil || err.Error() != "from address is required" {
				t.Fatalf("got %v; want missing sender error", err)
			}
		})
	}
}
