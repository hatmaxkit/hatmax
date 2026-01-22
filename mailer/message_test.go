package mailer

import (
	"testing"
)

func TestAddressString(t *testing.T) {
	tests := []struct {
		name  string
		addr  Address
		want  string
	}{
		{
			name:  "email only",
			addr:  Address{Email: "test@example.com"},
			want:  "test@example.com",
		},
		{
			name:  "email with name",
			addr:  Address{Email: "test@example.com", Name: "Test User"},
			want:  `"Test User" <test@example.com>`,
		},
		{
			name:  "empty address",
			addr:  Address{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.addr.String()
			if got != tt.want {
				t.Errorf("Address.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMessageValidate(t *testing.T) {
	tests := []struct {
		name    string
		msg     *Message
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message with text",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				Text:    "Test body",
			},
			wantErr: false,
		},
		{
			name: "valid message with html",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				HTML:    "<p>Test body</p>",
			},
			wantErr: false,
		},
		{
			name: "valid multipart message",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				Text:    "Test body",
				HTML:    "<p>Test body</p>",
			},
			wantErr: false,
		},
		{
			name: "missing from",
			msg: &Message{
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
				Text:    "Test body",
			},
			wantErr: true,
			errMsg:  "from address is required",
		},
		{
			name: "missing recipients",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{},
				Subject: "Test Subject",
				Text:    "Test body",
			},
			wantErr: true,
			errMsg:  "at least one recipient is required",
		},
		{
			name: "empty recipient email",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: ""}},
				Subject: "Test Subject",
				Text:    "Test body",
			},
			wantErr: true,
			errMsg:  "recipient 0 has empty email",
		},
		{
			name: "missing subject",
			msg: &Message{
				From: Address{Email: "from@example.com"},
				To:   []Address{{Email: "to@example.com"}},
				Text: "Test body",
			},
			wantErr: true,
			errMsg:  "subject is required",
		},
		{
			name: "missing body",
			msg: &Message{
				From:    Address{Email: "from@example.com"},
				To:      []Address{{Email: "to@example.com"}},
				Subject: "Test Subject",
			},
			wantErr: true,
			errMsg:  "message body is required (text or html)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("Message.Validate() error = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestMessageIsMultipart(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
		want bool
	}{
		{
			name: "text only",
			msg:  &Message{Text: "text"},
			want: false,
		},
		{
			name: "html only",
			msg:  &Message{HTML: "<p>html</p>"},
			want: false,
		},
		{
			name: "both text and html",
			msg:  &Message{Text: "text", HTML: "<p>html</p>"},
			want: true,
		},
		{
			name: "empty message",
			msg:  &Message{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.IsMultipart(); got != tt.want {
				t.Errorf("Message.IsMultipart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageHasAttachments(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
		want bool
	}{
		{
			name: "no attachments",
			msg:  &Message{},
			want: false,
		},
		{
			name: "with attachments",
			msg: &Message{
				Attachments: []Attachment{
					{Filename: "test.txt", Data: []byte("data")},
				},
			},
			want: true,
		},
		{
			name: "empty attachments slice",
			msg:  &Message{Attachments: []Attachment{}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.HasAttachments(); got != tt.want {
				t.Errorf("Message.HasAttachments() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageAllRecipients(t *testing.T) {
	tests := []struct {
		name      string
		msg       *Message
		wantCount int
	}{
		{
			name: "to only",
			msg: &Message{
				To: []Address{{Email: "to@example.com"}},
			},
			wantCount: 1,
		},
		{
			name: "to and cc",
			msg: &Message{
				To: []Address{{Email: "to@example.com"}},
				CC: []Address{{Email: "cc@example.com"}},
			},
			wantCount: 2,
		},
		{
			name: "to cc and bcc",
			msg: &Message{
				To:  []Address{{Email: "to@example.com"}},
				CC:  []Address{{Email: "cc@example.com"}},
				BCC: []Address{{Email: "bcc@example.com"}},
			},
			wantCount: 3,
		},
		{
			name: "multiple in each",
			msg: &Message{
				To:  []Address{{Email: "to1@example.com"}, {Email: "to2@example.com"}},
				CC:  []Address{{Email: "cc1@example.com"}},
				BCC: []Address{{Email: "bcc1@example.com"}, {Email: "bcc2@example.com"}},
			},
			wantCount: 5,
		},
		{
			name:      "empty message",
			msg:       &Message{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.msg.AllRecipients()
			if len(got) != tt.wantCount {
				t.Errorf("Message.AllRecipients() count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}
