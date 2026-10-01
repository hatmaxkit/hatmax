// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package mailer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
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
	message := msg.withDefaultFrom(m.cfg.DefaultFrom)
	msg = &message

	err := msg.Validate()
	if err != nil {
		return err
	}

	from := msg.From

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

	body, contentType, err := mailgunBody(values, msg.Attachments)
	if err != nil {
		return fmt.Errorf("mailgun send: build request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return fmt.Errorf("mailgun send: cannot create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
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

func mailgunBody(values url.Values, attachments []Attachment) (io.Reader, string, error) {
	if len(attachments) == 0 {
		return strings.NewReader(values.Encode()), "application/x-www-form-urlencoded", nil
	}

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	for key, entries := range values {
		for _, value := range entries {
			header := textproto.MIMEHeader{}
			header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": key}))

			part, err := writer.CreatePart(header)
			if err != nil {
				return nil, "", err
			}

			_, err = io.WriteString(part, value)
			if err != nil {
				return nil, "", err
			}
		}
	}

	for i, attachment := range attachments {
		if attachment.Filename == "" {
			return nil, "", fmt.Errorf("attachment %d: filename is required", i)
		}

		contentType := attachment.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			return nil, "", fmt.Errorf("attachment %d: invalid content type: %w", i, err)
		}

		if !strings.Contains(mediaType, "/") {
			return nil, "", fmt.Errorf("attachment %d: invalid content type", i)
		}

		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "attachment", "filename": attachment.Filename}))
		header.Set("Content-Type", mime.FormatMediaType(mediaType, params))

		part, err := writer.CreatePart(header)
		if err != nil {
			return nil, "", fmt.Errorf("attachment %d: %w", i, err)
		}

		_, err = part.Write(attachment.Data)
		if err != nil {
			return nil, "", fmt.Errorf("attachment %d: %w", i, err)
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, "", err
	}

	return &body, writer.FormDataContentType(), nil
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
