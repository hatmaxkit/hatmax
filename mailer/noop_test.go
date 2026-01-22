package mailer

import (
	"context"
	"testing"

	"github.com/hatmaxkit/hatmax/log"
)

func TestNewNoopMailer(t *testing.T) {
	logger := log.NewNoopLogger()
	m := NewNoopMailer(logger)
	if m == nil {
		t.Fatal("expected non-nil mailer")
	}
	if m.log == nil {
		t.Error("expected non-nil logger")
	}
}

func TestNoopMailerSend(t *testing.T) {
	tests := []struct {
		name    string
		logger  log.Logger
		msg     *Message
		wantErr bool
	}{
		{
			name:   "valid text message",
			logger: log.NewNoopLogger(),
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantErr: false,
		},
		{
			name:   "valid html message",
			logger: log.NewNoopLogger(),
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test",
				HTML:    "<p>body</p>",
			},
			wantErr: false,
		},
		{
			name:   "valid multipart message",
			logger: log.NewNoopLogger(),
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
				HTML:    "<p>body</p>",
			},
			wantErr: false,
		},
		{
			name:   "message with attachments",
			logger: log.NewNoopLogger(),
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
				Attachments: []Attachment{
					{Filename: "test.txt", ContentType: "text/plain", Data: []byte("data")},
				},
			},
			wantErr: false,
		},
		{
			name:   "nil logger",
			logger: nil,
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantErr: false,
		},
		{
			name:   "invalid message missing from",
			logger: log.NewNoopLogger(),
			msg: &Message{
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test",
				Text:    "body",
			},
			wantErr: true,
		},
		{
			name:   "invalid message missing recipients",
			logger: log.NewNoopLogger(),
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				Subject: "Test",
				Text:    "body",
			},
			wantErr: true,
		},
		{
			name:   "invalid message missing subject",
			logger: log.NewNoopLogger(),
			msg: &Message{
				From: Address{Email: "from@example.com"},
				To:   []Address{{Email: "to@example.com"}},
				Text: "body",
			},
			wantErr: true,
		},
		{
			name:   "invalid message missing body",
			logger: log.NewNoopLogger(),
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &NoopMailer{log: tt.logger}
			err := m.Send(context.Background(), tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NoopMailer.Send() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNoopMailerImplementsMailer(t *testing.T) {
	var _ Mailer = (*NoopMailer)(nil)
}
