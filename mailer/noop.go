package mailer

import (
	"context"
	"strings"

	"github.com/hatmaxkit/hatmax/log"
)

// NoopMailer is a mailer that logs messages instead of sending them.
type NoopMailer struct {
	log log.Logger
}

// NewNoopMailer creates a new noop mailer.
func NewNoopMailer(log log.Logger) *NoopMailer {
	return &NoopMailer{log: log}
}

// Send logs the message instead of sending it.
func (m *NoopMailer) Send(ctx context.Context, msg *Message) error {
	err := msg.Validate()
	if err != nil {
		return err
	}

	if m.log == nil {
		return nil
	}

	recipients := make([]string, len(msg.To))
	for i, addr := range msg.To {
		recipients[i] = addr.String()
	}

	m.log.Infof("[NOOP MAILER] to=%s subject=%q from=%s",
		strings.Join(recipients, ", "),
		msg.Subject,
		msg.From.String(),
	)

	if msg.IsMultipart() {
		m.log.Debug("[NOOP MAILER] multipart message (text + html)")
	} else if msg.HTML != "" {
		m.log.Debug("[NOOP MAILER] html-only message")
	} else {
		m.log.Debug("[NOOP MAILER] text-only message")
	}

	if msg.HasAttachments() {
		for _, att := range msg.Attachments {
			m.log.Debugf("[NOOP MAILER] attachment: %s (%s, %d bytes)",
				att.Filename, att.ContentType, len(att.Data))
		}
	}

	return nil
}

var _ Mailer = (*NoopMailer)(nil)
