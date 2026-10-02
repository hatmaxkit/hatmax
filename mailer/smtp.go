// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// Bound the entire SMTP transaction even when callers supply no deadline.
const smtpSendTimeout = 30 * time.Second

// SMTPMailer sends emails via SMTP.
type SMTPMailer struct {
	cfg SMTPConfig
}

// NewSMTPMailer creates a new SMTP mailer.
func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

// Send sends an email via SMTP, respecting cancellation and a total 30-second
// transport timeout. A shorter caller deadline takes precedence.
func (m *SMTPMailer) Send(ctx context.Context, msg *Message) error {
	message := msg.withDefaultFrom(m.cfg.DefaultFrom)
	msg = &message

	err := msg.Validate()
	if err != nil {
		return err
	}

	raw, err := m.buildRawMessage(msg)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	recipients := make([]string, 0, len(msg.To)+len(msg.CC)+len(msg.BCC))
	for _, addr := range msg.AllRecipients() {
		recipients = append(recipients, addr.Email)
	}

	return m.send(ctx, msg.From.Email, recipients, raw)
}

func (m *SMTPMailer) send(ctx context.Context, from string, to []string, msg []byte) (err error) {
	ctx, cancel := context.WithTimeout(ctx, smtpSendTimeout)
	defer cancel()

	// Preserve the context cause when a closed socket interrupts an SMTP call.
	defer func() {
		if err == nil {
			return
		}

		cause := ctx.Err()
		if cause != nil {
			err = fmt.Errorf("smtp send: %w", cause)

			return
		}

		var networkError net.Error

		deadline, _ := ctx.Deadline()
		if errors.As(err, &networkError) && networkError.Timeout() && !time.Now().Before(deadline) {
			err = fmt.Errorf("smtp send: %w", context.DeadlineExceeded)
		}
	}()

	addr := net.JoinHostPort(m.cfg.Host, fmt.Sprintf("%d", m.cfg.Port))

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	deadline, _ := ctx.Deadline()

	err = conn.SetDeadline(deadline)
	if err != nil {
		return fmt.Errorf("smtp deadline: %w", err)
	}

	tlsConfig := &tls.Config{
		ServerName:         m.cfg.Host,
		InsecureSkipVerify: m.cfg.InsecureSkipVerify,
	}

	var transport net.Conn = conn
	if m.cfg.TLS {
		tlsConn := tls.Client(conn, tlsConfig)

		err = tlsConn.HandshakeContext(ctx)
		if err != nil {
			return fmt.Errorf("smtp tls: %w", err)
		}

		transport = tlsConn
	}

	client, err := smtp.NewClient(transport, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	err = client.Hello("localhost")
	if err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}

	// Implicit TLS already satisfies encryption, including when both flags are set.
	if !m.cfg.TLS {
		available, _ := client.Extension("STARTTLS")
		if m.cfg.StartTLS && !available {
			return fmt.Errorf("smtp: STARTTLS is required but unavailable")
		}

		if available {
			err = client.StartTLS(tlsConfig)
			if err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	return deliverSMTP(client, auth, from, to, msg)
}

func deliverSMTP(client *smtp.Client, auth smtp.Auth, from string, to []string, msg []byte) error {
	if auth != nil {
		err := client.Auth(auth)
		if err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	err := client.Mail(from)
	if err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}

	for _, rcpt := range to {
		err = client.Rcpt(rcpt)
		if err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}

	return client.Quit()
}

func (m *SMTPMailer) buildRawMessage(msg *Message) ([]byte, error) {
	err := msg.validateHeaders()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	m.writeHeader(&buf, "From", msg.From.String())
	m.writeHeader(&buf, "To", m.formatAddresses(msg.To))

	if len(msg.CC) > 0 {
		m.writeHeader(&buf, "Cc", m.formatAddresses(msg.CC))
	}

	if msg.ReplyTo != nil {
		m.writeHeader(&buf, "Reply-To", msg.ReplyTo.String())
	}

	m.writeHeader(&buf, "Subject", m.encodeSubject(msg.Subject))
	m.writeHeader(&buf, "MIME-Version", "1.0")

	for k, v := range msg.Headers {
		m.writeHeader(&buf, k, v)
	}

	hasAttachments := msg.HasAttachments()
	isMultipart := msg.IsMultipart()

	if hasAttachments {
		return m.buildWithAttachments(&buf, msg)
	} else if isMultipart {
		return m.buildMultipartAlternative(&buf, msg)
	} else if msg.HTML != "" {
		return m.buildHTMLOnly(&buf, msg)
	}

	return m.buildTextOnly(&buf, msg)
}

func (m *SMTPMailer) buildTextOnly(buf *bytes.Buffer, msg *Message) ([]byte, error) {
	m.writeHeader(buf, "Content-Type", "text/plain; charset=utf-8")
	m.writeHeader(buf, "Content-Transfer-Encoding", "quoted-printable")
	buf.WriteString("\r\n")

	qp := quotedprintable.NewWriter(buf)
	qp.Write([]byte(msg.Text))
	qp.Close()

	return buf.Bytes(), nil
}

func (m *SMTPMailer) buildHTMLOnly(buf *bytes.Buffer, msg *Message) ([]byte, error) {
	m.writeHeader(buf, "Content-Type", "text/html; charset=utf-8")
	m.writeHeader(buf, "Content-Transfer-Encoding", "quoted-printable")
	buf.WriteString("\r\n")

	qp := quotedprintable.NewWriter(buf)
	qp.Write([]byte(msg.HTML))
	qp.Close()

	return buf.Bytes(), nil
}

func (m *SMTPMailer) buildMultipartAlternative(buf *bytes.Buffer, msg *Message) ([]byte, error) {
	writer := multipart.NewWriter(buf)
	m.writeHeader(buf, "Content-Type", fmt.Sprintf("multipart/alternative; boundary=%s", writer.Boundary()))
	buf.WriteString("\r\n")

	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=utf-8")
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")

	textPart, err := writer.CreatePart(textHeader)
	if err != nil {
		return nil, err
	}

	qp := quotedprintable.NewWriter(textPart)
	qp.Write([]byte(msg.Text))
	qp.Close()

	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")

	htmlPart, err := writer.CreatePart(htmlHeader)
	if err != nil {
		return nil, err
	}

	qp = quotedprintable.NewWriter(htmlPart)
	qp.Write([]byte(msg.HTML))
	qp.Close()

	writer.Close()

	return buf.Bytes(), nil
}

func (m *SMTPMailer) buildWithAttachments(buf *bytes.Buffer, msg *Message) ([]byte, error) {
	writer := multipart.NewWriter(buf)
	m.writeHeader(buf, "Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary()))
	buf.WriteString("\r\n")

	if msg.IsMultipart() {
		bodyHeader := textproto.MIMEHeader{}
		altWriter := multipart.NewWriter(io.Discard)
		bodyHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%s", altWriter.Boundary()))

		bodyPart, err := writer.CreatePart(bodyHeader)
		if err != nil {
			return nil, err
		}

		altWriter = multipart.NewWriter(bodyPart)

		textHeader := textproto.MIMEHeader{}
		textHeader.Set("Content-Type", "text/plain; charset=utf-8")
		textHeader.Set("Content-Transfer-Encoding", "quoted-printable")

		textPart, err := altWriter.CreatePart(textHeader)
		if err != nil {
			return nil, err
		}

		qp := quotedprintable.NewWriter(textPart)
		qp.Write([]byte(msg.Text))
		qp.Close()

		htmlHeader := textproto.MIMEHeader{}
		htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
		htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")

		htmlPart, err := altWriter.CreatePart(htmlHeader)
		if err != nil {
			return nil, err
		}

		qp = quotedprintable.NewWriter(htmlPart)
		qp.Write([]byte(msg.HTML))
		qp.Close()

		altWriter.Close()
	} else if msg.HTML != "" {
		bodyHeader := textproto.MIMEHeader{}
		bodyHeader.Set("Content-Type", "text/html; charset=utf-8")
		bodyHeader.Set("Content-Transfer-Encoding", "quoted-printable")

		bodyPart, err := writer.CreatePart(bodyHeader)
		if err != nil {
			return nil, err
		}

		qp := quotedprintable.NewWriter(bodyPart)
		qp.Write([]byte(msg.HTML))
		qp.Close()
	} else {
		bodyHeader := textproto.MIMEHeader{}
		bodyHeader.Set("Content-Type", "text/plain; charset=utf-8")
		bodyHeader.Set("Content-Transfer-Encoding", "quoted-printable")

		bodyPart, err := writer.CreatePart(bodyHeader)
		if err != nil {
			return nil, err
		}

		qp := quotedprintable.NewWriter(bodyPart)
		qp.Write([]byte(msg.Text))
		qp.Close()
	}

	for _, att := range msg.Attachments {
		attHeader := textproto.MIMEHeader{}

		contentType := att.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		attHeader.Set("Content-Type", contentType)
		attHeader.Set("Content-Transfer-Encoding", "base64")
		attHeader.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`,
			m.encodeFilename(att.Filename)))

		attPart, err := writer.CreatePart(attHeader)
		if err != nil {
			return nil, err
		}

		encoder := base64.NewEncoder(base64.StdEncoding, attPart)
		encoder.Write(att.Data)
		encoder.Close()
	}

	writer.Close()

	return buf.Bytes(), nil
}

func (m *SMTPMailer) writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

func (m *SMTPMailer) formatAddresses(addrs []Address) string {
	parts := make([]string, len(addrs))
	for i, addr := range addrs {
		parts[i] = addr.String()
	}

	return strings.Join(parts, ", ")
}

func (m *SMTPMailer) encodeSubject(subject string) string {
	return mime.QEncoding.Encode("utf-8", subject)
}

func (m *SMTPMailer) encodeFilename(filename string) string {
	return mime.QEncoding.Encode("utf-8", filename)
}

var _ Mailer = (*SMTPMailer)(nil)
