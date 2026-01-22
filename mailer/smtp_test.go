package mailer

import (
	"strings"
	"testing"
)

func TestNewSMTPMailer(t *testing.T) {
	cfg := SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
	}
	m := NewSMTPMailer(cfg)
	if m == nil {
		t.Fatal("expected non-nil mailer")
	}
	if m.cfg.Host != cfg.Host {
		t.Errorf("expected host %q, got %q", cfg.Host, m.cfg.Host)
	}
}

func TestSMTPMailerBuildRawMessage(t *testing.T) {
	m := &SMTPMailer{}

	tests := []struct {
		name        string
		msg         *Message
		wantHeaders []string
		wantBody    string
	}{
		{
			name: "text only",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				Text:    "Hello World",
			},
			wantHeaders: []string{
				"From: from@example.com",
				"To: to@example.com",
				"Subject:",
				"Content-Type: text/plain",
			},
		},
		{
			name: "html only",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				HTML:    "<p>Hello World</p>",
			},
			wantHeaders: []string{
				"From: from@example.com",
				"Content-Type: text/html",
			},
		},
		{
			name: "multipart",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				Text:    "Hello World",
				HTML:    "<p>Hello World</p>",
			},
			wantHeaders: []string{
				"Content-Type: multipart/alternative",
			},
		},
		{
			name: "with cc",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				CC:      []Address{{Email: "cc@example.com"}},
				Subject: "Test Subject",
				Text:    "Hello World",
			},
			wantHeaders: []string{
				"Cc: cc@example.com",
			},
		},
		{
			name: "with reply-to",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				ReplyTo: &Address{Email: "reply@example.com"},
				Subject: "Test Subject",
				Text:    "Hello World",
			},
			wantHeaders: []string{
				"Reply-To: reply@example.com",
			},
		},
		{
			name: "with custom headers",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				Text:    "Hello World",
				Headers: map[string]string{"X-Custom": "value"},
			},
			wantHeaders: []string{
				"X-Custom: value",
			},
		},
		{
			name: "with attachments",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				Text:    "Hello World",
				Attachments: []Attachment{
					{Filename: "test.txt", ContentType: "text/plain", Data: []byte("data")},
				},
			},
			wantHeaders: []string{
				"Content-Type: multipart/mixed",
			},
		},
		{
			name: "attachments with html only",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				HTML:    "<p>Hello</p>",
				Attachments: []Attachment{
					{Filename: "test.txt", Data: []byte("data")},
				},
			},
			wantHeaders: []string{
				"Content-Type: multipart/mixed",
			},
		},
		{
			name: "attachments with multipart body",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				Text:    "text",
				HTML:    "<p>html</p>",
				Attachments: []Attachment{
					{Filename: "test.txt", Data: []byte("data")},
				},
			},
			wantHeaders: []string{
				"Content-Type: multipart/mixed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := m.buildRawMessage(tt.msg)
			if err != nil {
				t.Fatalf("buildRawMessage() error = %v", err)
			}

			rawStr := string(raw)
			for _, h := range tt.wantHeaders {
				if !strings.Contains(rawStr, h) {
					t.Errorf("expected header %q not found in message", h)
				}
			}
		})
	}
}

func TestSMTPMailerFormatAddresses(t *testing.T) {
	m := &SMTPMailer{}

	tests := []struct {
		name  string
		addrs []Address
		want  string
	}{
		{
			name:  "single address",
			addrs: []Address{{Email: "test@example.com"}},
			want:  "test@example.com",
		},
		{
			name:  "multiple addresses",
			addrs: []Address{{Email: "a@example.com"}, {Email: "b@example.com"}},
			want:  "a@example.com, b@example.com",
		},
		{
			name:  "address with name",
			addrs: []Address{{Email: "test@example.com", Name: "Test"}},
			want:  `"Test" <test@example.com>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.formatAddresses(tt.addrs)
			if got != tt.want {
				t.Errorf("formatAddresses() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSMTPMailerEncodeSubject(t *testing.T) {
	m := &SMTPMailer{}

	tests := []struct {
		name    string
		subject string
	}{
		{
			name:    "ascii subject",
			subject: "Hello World",
		},
		{
			name:    "unicode subject",
			subject: "Héllo Wörld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.encodeSubject(tt.subject)
			if got == "" {
				t.Error("expected non-empty encoded subject")
			}
		})
	}
}

func TestSMTPMailerImplementsMailer(t *testing.T) {
	var _ Mailer = (*SMTPMailer)(nil)
}
