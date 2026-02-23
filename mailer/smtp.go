package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"strings"
)

// SMTPMailer sends emails via SMTP.
type SMTPMailer struct {
	cfg SMTPConfig
}

// NewSMTPMailer creates a new SMTP mailer.
func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

// Send sends an email via SMTP.
func (m *SMTPMailer) Send(ctx context.Context, msg *Message) error {
	err := msg.Validate()
	if err != nil {
		return err
	}

	if msg.From.Email == "" {
		msg.From = m.cfg.DefaultFrom
	}

	raw, err := m.buildRawMessage(msg)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	recipients := make([]string, 0, len(msg.To)+len(msg.CC)+len(msg.BCC))
	for _, addr := range msg.AllRecipients() {
		recipients = append(recipients, addr.Email)
	}

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	if m.cfg.TLS {
		return m.sendTLS(addr, auth, msg.From.Email, recipients, raw)
	}

	return smtp.SendMail(addr, auth, msg.From.Email, recipients, raw)
}

func (m *SMTPMailer) sendTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName:         m.cfg.Host,
		InsecureSkipVerify: m.cfg.InsecureSkipVerify,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		err = client.Auth(auth)
		if err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	err = client.Mail(from)
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
