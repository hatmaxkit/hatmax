package mailer

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridConfig holds SendGrid-specific configuration.
type SendGridConfig struct {
	Config
	APIKey string
}

// SendGridMailer sends emails via SendGrid API.
type SendGridMailer struct {
	cfg    SendGridConfig
	client *sendgrid.Client
}

// NewSendGridMailer creates a new SendGrid mailer.
func NewSendGridMailer(cfg SendGridConfig) *SendGridMailer {
	return &SendGridMailer{
		cfg:    cfg,
		client: sendgrid.NewSendClient(cfg.APIKey),
	}
}

// Send sends an email via SendGrid.
func (m *SendGridMailer) Send(ctx context.Context, msg *Message) error {
	err := msg.Validate()
	if err != nil {
		return err
	}

	from := msg.From
	if from.Email == "" {
		from = m.cfg.DefaultFrom
	}

	sgFrom := mail.NewEmail(from.Name, from.Email)

	if len(msg.To) == 0 {
		return fmt.Errorf("at least one recipient required")
	}

	sgTo := mail.NewEmail(msg.To[0].Name, msg.To[0].Email)

	sgMsg := mail.NewSingleEmail(sgFrom, msg.Subject, sgTo, msg.Text, msg.HTML)

	for i := 1; i < len(msg.To); i++ {
		sgMsg.Personalizations[0].AddTos(mail.NewEmail(msg.To[i].Name, msg.To[i].Email))
	}

	for _, addr := range msg.CC {
		sgMsg.Personalizations[0].AddCCs(mail.NewEmail(addr.Name, addr.Email))
	}

	for _, addr := range msg.BCC {
		sgMsg.Personalizations[0].AddBCCs(mail.NewEmail(addr.Name, addr.Email))
	}

	if msg.ReplyTo != nil {
		sgMsg.SetReplyTo(mail.NewEmail(msg.ReplyTo.Name, msg.ReplyTo.Email))
	}

	for k, v := range msg.Headers {
		sgMsg.Headers[k] = v
	}

	for _, att := range msg.Attachments {
		sgAtt := mail.NewAttachment()
		sgAtt.SetContent(base64.StdEncoding.EncodeToString(att.Data))
		sgAtt.SetFilename(att.Filename)

		if att.ContentType != "" {
			sgAtt.SetType(att.ContentType)
		}

		sgAtt.SetDisposition("attachment")
		sgMsg.AddAttachment(sgAtt)
	}

	resp, err := m.client.SendWithContext(ctx, sgMsg)
	if err != nil {
		return fmt.Errorf("sendgrid send: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error: status %d, body: %s", resp.StatusCode, resp.Body)
	}

	return nil
}

var _ Mailer = (*SendGridMailer)(nil)
