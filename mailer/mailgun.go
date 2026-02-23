package mailer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// MailgunMailer sends emails via Mailgun API.
type MailgunMailer struct {
	cfg    MailgunConfig
	client *http.Client
}

// NewMailgunMailer creates a new Mailgun mailer.
func NewMailgunMailer(cfg MailgunConfig) *MailgunMailer {
	return &MailgunMailer{
		cfg:    cfg,
		client: &http.Client{},
	}
}

// Send sends an email via Mailgun.
func (m *MailgunMailer) Send(ctx context.Context, msg *Message) error {
	err := msg.Validate()
	if err != nil {
		return err
	}

	from := msg.From
	if from.Email == "" {
		from = m.cfg.DefaultFrom
	}

	if from.Email == "" {
		return fmt.Errorf("mailgun send: from address is required")
	}

	if strings.TrimSpace(m.cfg.APIKey) == "" {
		return fmt.Errorf("mailgun send: api key is required")
	}

	if strings.TrimSpace(m.cfg.Domain) == "" {
		return fmt.Errorf("mailgun send: domain is required")
	}

	values := url.Values{}
	values.Set("from", from.String())
	values.Set("to", joinEmails(msg.To))
	values.Set("subject", msg.Subject)

	if msg.Text != "" {
		values.Set("text", msg.Text)
	}

	if msg.HTML != "" {
		values.Set("html", msg.HTML)
	}

	for _, cc := range msg.CC {
		values.Add("cc", cc.String())
	}

	for _, bcc := range msg.BCC {
		values.Add("bcc", bcc.String())
	}

	if msg.ReplyTo != nil {
		values.Set("h:Reply-To", msg.ReplyTo.String())
	}

	for k, v := range msg.Headers {
		values.Set("h:"+k, v)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(m.cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.mailgun.net"
	}

	endpoint := fmt.Sprintf("%s/v3/%s/messages", baseURL, strings.TrimSpace(m.cfg.Domain))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("mailgun send: cannot create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("api", m.cfg.APIKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("mailgun send: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("mailgun send: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func joinEmails(addrs []Address) string {
	items := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if strings.TrimSpace(a.Email) == "" {
			continue
		}

		items = append(items, a.String())
	}

	return strings.Join(items, ",")
}

var _ Mailer = (*MailgunMailer)(nil)
