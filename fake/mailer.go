package fake

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/hatmaxkit/hatmax/mailer"
)

// MailerSendCall represents a recorded call to Send.
type MailerSendCall struct {
	Ctx     context.Context
	Message *mailer.Message
}

// Mailer is a test double for mailer.Mailer that records all calls.
type Mailer struct {
	mu sync.Mutex

	SendFunc         func(ctx context.Context, msg *mailer.Message) error
	SendCalls        []MailerSendCall
	Messages         []*mailer.Message
	Output           io.Writer
	FailOnValidation bool
}

// NewMailer creates a new fake mailer.
func NewMailer() *Mailer {
	return &Mailer{}
}

// Send records the call and optionally executes SendFunc.
func (m *Mailer) Send(ctx context.Context, msg *mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SendCalls = append(m.SendCalls, MailerSendCall{
		Ctx:     ctx,
		Message: msg,
	})

	if m.FailOnValidation {
		err := msg.Validate()
		if err != nil {
			return err
		}
	}

	if m.SendFunc != nil {
		err := m.SendFunc(ctx, msg)
		if err == nil {
			m.Messages = append(m.Messages, msg)
		}

		return err
	}

	m.Messages = append(m.Messages, msg)

	if m.Output != nil {
		m.printMessage(msg)
	}

	return nil
}

func (m *Mailer) printMessage(msg *mailer.Message) {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("==== EMAIL SENT ====\n")
	b.WriteString(fmt.Sprintf("From:    %s\n", m.truncate(msg.From.String(), 50)))
	b.WriteString(fmt.Sprintf("To:      %s\n", m.truncate(m.formatAddresses(msg.To), 50)))

	if len(msg.CC) > 0 {
		b.WriteString(fmt.Sprintf("CC:      %s\n", m.truncate(m.formatAddresses(msg.CC), 50)))
	}

	if len(msg.BCC) > 0 {
		b.WriteString(fmt.Sprintf("BCC:     %s\n", m.truncate(m.formatAddresses(msg.BCC), 50)))
	}

	if msg.ReplyTo != nil {
		b.WriteString(fmt.Sprintf("ReplyTo: %s\n", m.truncate(msg.ReplyTo.String(), 50)))
	}

	b.WriteString(fmt.Sprintf("Subject: %s\n", m.truncate(msg.Subject, 50)))
	b.WriteString("--------------------\n")

	if msg.Text != "" {
		b.WriteString("[TEXT BODY]\n")

		for _, line := range strings.Split(msg.Text, "\n") {
			b.WriteString(fmt.Sprintf("%s\n", m.truncate(line, 60)))
		}
	}

	if msg.HTML != "" {
		b.WriteString(fmt.Sprintf("[HTML BODY] (%d bytes)\n", len(msg.HTML)))
	}

	if len(msg.Attachments) > 0 {
		b.WriteString("[ATTACHMENTS]\n")

		for _, att := range msg.Attachments {
			b.WriteString(fmt.Sprintf("  - %s (%s, %d bytes)\n",
				att.Filename, att.ContentType, len(att.Data)))
		}
	}

	b.WriteString("====================\n")

	fmt.Fprint(m.Output, b.String())
}

func (m *Mailer) formatAddresses(addrs []mailer.Address) string {
	parts := make([]string, len(addrs))
	for i, addr := range addrs {
		parts[i] = addr.String()
	}

	return strings.Join(parts, ", ")
}

func (m *Mailer) truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}

	return s[:max-3] + "..."
}

// Reset clears all recorded calls and messages.
func (m *Mailer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SendCalls = nil
	m.Messages = nil
}

// GetSendCalls returns a copy of all recorded send calls.
func (m *Mailer) GetSendCalls() []MailerSendCall {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]MailerSendCall{}, m.SendCalls...)
}

// GetMessages returns a copy of all sent messages.
func (m *Mailer) GetMessages() []*mailer.Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]*mailer.Message{}, m.Messages...)
}

// SendCount returns the number of Send calls.
func (m *Mailer) SendCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.SendCalls)
}

// LastMessage returns the last sent message or nil.
func (m *Mailer) LastMessage() *mailer.Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Messages) == 0 {
		return nil
	}

	return m.Messages[len(m.Messages)-1]
}

// HasMessageTo checks if any message was sent to the given email.
func (m *Mailer) HasMessageTo(email string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, msg := range m.Messages {
		for _, addr := range msg.AllRecipients() {
			if addr.Email == email {
				return true
			}
		}
	}

	return false
}

// HasMessageWithSubject checks if any message has the given subject.
func (m *Mailer) HasMessageWithSubject(subject string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, msg := range m.Messages {
		if msg.Subject == subject {
			return true
		}
	}

	return false
}

// HasMessageContaining checks if any message contains the text.
func (m *Mailer) HasMessageContaining(text string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, msg := range m.Messages {
		if strings.Contains(msg.Text, text) || strings.Contains(msg.HTML, text) {
			return true
		}
	}

	return false
}

// SetOutput sets the output writer for message printing.
func (m *Mailer) SetOutput(w io.Writer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Output = w
}

// WithOutput sets output and returns the mailer for chaining.
func (m *Mailer) WithOutput(w io.Writer) *Mailer {
	m.Output = w

	return m
}

// WithValidation enables validation on Send.
func (m *Mailer) WithValidation() *Mailer {
	m.FailOnValidation = true

	return m
}

var _ mailer.Mailer = (*Mailer)(nil)
