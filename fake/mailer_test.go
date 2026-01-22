package fake

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/hatmaxkit/hatmax/mailer"
)

func TestNewMailer(t *testing.T) {
	m := NewMailer()
	if m == nil {
		t.Fatal("expected non-nil mailer")
	}
}

func TestMailerSend(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Mailer)
		msg     *mailer.Message
		wantErr bool
	}{
		{
			name:  "basic send",
			setup: func(m *Mailer) {},
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantErr: false,
		},
		{
			name: "with custom SendFunc success",
			setup: func(m *Mailer) {
				m.SendFunc = func(ctx context.Context, msg *mailer.Message) error {
					return nil
				}
			},
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantErr: false,
		},
		{
			name: "with custom SendFunc error",
			setup: func(m *Mailer) {
				m.SendFunc = func(ctx context.Context, msg *mailer.Message) error {
					return errors.New("send failed")
				}
			},
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantErr: true,
		},
		{
			name: "with validation enabled valid message",
			setup: func(m *Mailer) {
				m.FailOnValidation = true
			},
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantErr: false,
		},
		{
			name: "with validation enabled invalid message",
			setup: func(m *Mailer) {
				m.FailOnValidation = true
			},
			msg: &mailer.Message{
				To:      []mailer.Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantErr: true,
		},
		{
			name: "with output writer",
			setup: func(m *Mailer) {
				m.Output = &bytes.Buffer{}
			},
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMailer()
			tt.setup(m)

			err := m.Send(context.Background(), tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMailerSendRecordsCall(t *testing.T) {
	m := NewMailer()
	msg := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		Subject: "Test",
		Text:    "body",
	}

	ctx := context.Background()
	m.Send(ctx, msg)

	if len(m.SendCalls) != 1 {
		t.Errorf("expected 1 call, got %d", len(m.SendCalls))
	}
	if m.SendCalls[0].Message != msg {
		t.Error("recorded message does not match")
	}
}

func TestMailerPrintMessage(t *testing.T) {
	tests := []struct {
		name         string
		msg          *mailer.Message
		wantContains []string
	}{
		{
			name: "basic message",
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				Text:    "Hello\nWorld",
			},
			wantContains: []string{"from@example.com", "to@example.com", "Test Subject", "Hello", "World"},
		},
		{
			name: "message with cc",
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				CC:      []mailer.Address{{Email: "cc@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantContains: []string{"CC:", "cc@example.com"},
		},
		{
			name: "message with bcc",
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				BCC:     []mailer.Address{{Email: "bcc@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantContains: []string{"BCC:", "bcc@example.com"},
		},
		{
			name: "message with reply-to",
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				ReplyTo: &mailer.Address{Email: "reply@example.com"},
				Subject: "Test",
				Text:    "body",
			},
			wantContains: []string{"ReplyTo:", "reply@example.com"},
		},
		{
			name: "message with html",
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				Subject: "Test",
				HTML:    "<p>Hello</p>",
			},
			wantContains: []string{"[HTML BODY]"},
		},
		{
			name: "message with attachments",
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
				Attachments: []mailer.Attachment{
					{Filename: "test.txt", ContentType: "text/plain", Data: []byte("data")},
				},
			},
			wantContains: []string{"[ATTACHMENTS]", "test.txt", "text/plain"},
		},
		{
			name: "long subject truncated",
			msg: &mailer.Message{
				From:    mailer.Address{Email: "from@example.com"},
				To:      []mailer.Address{{Email: "to@example.com"}},
				Subject: "This is a very long subject line that should be truncated at some point",
				Text:    "body",
			},
			wantContains: []string{"..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			m := NewMailer()
			m.Output = &buf

			m.Send(context.Background(), tt.msg)

			output := buf.String()
			for _, want := range tt.wantContains {
				if !bytes.Contains([]byte(output), []byte(want)) {
					t.Errorf("output missing %q", want)
				}
			}
		})
	}
}

func TestMailerReset(t *testing.T) {
	m := NewMailer()
	msg := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		Subject: "Test",
		Text:    "body",
	}

	m.Send(context.Background(), msg)
	if len(m.SendCalls) != 1 {
		t.Fatal("expected 1 call before reset")
	}

	m.Reset()

	if len(m.SendCalls) != 0 {
		t.Error("expected 0 calls after reset")
	}
	if len(m.Messages) != 0 {
		t.Error("expected 0 messages after reset")
	}
}

func TestMailerGetSendCalls(t *testing.T) {
	m := NewMailer()
	msg := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		Subject: "Test",
		Text:    "body",
	}

	m.Send(context.Background(), msg)
	calls := m.GetSendCalls()

	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}

	calls = append(calls, MailerSendCall{})
	if len(m.GetSendCalls()) != 1 {
		t.Error("GetSendCalls should return a copy")
	}
}

func TestMailerGetMessages(t *testing.T) {
	m := NewMailer()
	msg := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		Subject: "Test",
		Text:    "body",
	}

	m.Send(context.Background(), msg)
	messages := m.GetMessages()

	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	messages = append(messages, &mailer.Message{})
	if len(m.GetMessages()) != 1 {
		t.Error("GetMessages should return a copy")
	}
}

func TestMailerSendCount(t *testing.T) {
	m := NewMailer()
	msg := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		Subject: "Test",
		Text:    "body",
	}

	if m.SendCount() != 0 {
		t.Error("expected 0 count initially")
	}

	m.Send(context.Background(), msg)
	if m.SendCount() != 1 {
		t.Error("expected 1 count after send")
	}

	m.Send(context.Background(), msg)
	if m.SendCount() != 2 {
		t.Error("expected 2 count after second send")
	}
}

func TestMailerLastMessage(t *testing.T) {
	m := NewMailer()

	if m.LastMessage() != nil {
		t.Error("expected nil when no messages")
	}

	msg1 := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		Subject: "First",
		Text:    "body",
	}
	msg2 := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		Subject: "Second",
		Text:    "body",
	}

	m.Send(context.Background(), msg1)
	m.Send(context.Background(), msg2)

	last := m.LastMessage()
	if last.Subject != "Second" {
		t.Errorf("expected last message subject 'Second', got %q", last.Subject)
	}
}

func TestMailerHasMessageTo(t *testing.T) {
	m := NewMailer()
	msg := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		CC:      []mailer.Address{{Email: "cc@example.com"}},
		Subject: "Test",
		Text:    "body",
	}

	m.Send(context.Background(), msg)

	tests := []struct {
		email string
		want  bool
	}{
		{"to@example.com", true},
		{"cc@example.com", true},
		{"unknown@example.com", false},
	}

	for _, tt := range tests {
		if got := m.HasMessageTo(tt.email); got != tt.want {
			t.Errorf("HasMessageTo(%q) = %v, want %v", tt.email, got, tt.want)
		}
	}
}

func TestMailerHasMessageWithSubject(t *testing.T) {
	m := NewMailer()
	msg := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		Subject: "Welcome Email",
		Text:    "body",
	}

	m.Send(context.Background(), msg)

	if !m.HasMessageWithSubject("Welcome Email") {
		t.Error("expected to find message with subject")
	}
	if m.HasMessageWithSubject("Unknown") {
		t.Error("should not find message with unknown subject")
	}
}

func TestMailerHasMessageContaining(t *testing.T) {
	m := NewMailer()

	msg1 := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		Subject: "Test",
		Text:    "Hello World",
	}
	msg2 := &mailer.Message{
		From:    mailer.Address{Email: "from@example.com"},
		To:      []mailer.Address{{Email: "to@example.com"}},
		Subject: "Test",
		HTML:    "<p>Goodbye</p>",
	}

	m.Send(context.Background(), msg1)
	m.Send(context.Background(), msg2)

	tests := []struct {
		text string
		want bool
	}{
		{"Hello", true},
		{"Goodbye", true},
		{"Unknown", false},
	}

	for _, tt := range tests {
		if got := m.HasMessageContaining(tt.text); got != tt.want {
			t.Errorf("HasMessageContaining(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestMailerSetOutput(t *testing.T) {
	m := NewMailer()
	var buf bytes.Buffer

	m.SetOutput(&buf)

	if m.Output != &buf {
		t.Error("SetOutput did not set output")
	}
}

func TestMailerWithOutput(t *testing.T) {
	var buf bytes.Buffer
	m := NewMailer().WithOutput(&buf)

	if m.Output != &buf {
		t.Error("WithOutput did not set output")
	}
}

func TestMailerWithValidation(t *testing.T) {
	m := NewMailer().WithValidation()

	if !m.FailOnValidation {
		t.Error("WithValidation did not set FailOnValidation")
	}
}

func TestMailerFormatAddresses(t *testing.T) {
	m := NewMailer()

	tests := []struct {
		name  string
		addrs []mailer.Address
		want  string
	}{
		{
			name:  "single address",
			addrs: []mailer.Address{{Email: "test@example.com"}},
			want:  "test@example.com",
		},
		{
			name:  "multiple addresses",
			addrs: []mailer.Address{{Email: "a@example.com"}, {Email: "b@example.com"}},
			want:  "a@example.com, b@example.com",
		},
		{
			name:  "empty slice",
			addrs: []mailer.Address{},
			want:  "",
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

func TestMailerTruncate(t *testing.T) {
	m := NewMailer()

	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{
			name: "short string",
			s:    "hello",
			max:  10,
			want: "hello",
		},
		{
			name: "exact length",
			s:    "hello",
			max:  5,
			want: "hello",
		},
		{
			name: "long string",
			s:    "hello world",
			max:  8,
			want: "hello...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMailerImplementsMailerInterface(t *testing.T) {
	var _ mailer.Mailer = (*Mailer)(nil)
}
