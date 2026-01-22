package mailer

import (
	"fmt"
	"net/mail"
)

// Address represents an email address with optional display name.
type Address struct {
	Email string
	Name  string
}

// String returns the RFC 5322 formatted address.
func (a Address) String() string {
	if a.Name == "" {
		return a.Email
	}
	addr := mail.Address{Name: a.Name, Address: a.Email}
	return addr.String()
}

// Attachment represents an email attachment.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Message represents an email message.
type Message struct {
	From        Address
	To          []Address
	CC          []Address
	BCC         []Address
	ReplyTo     *Address
	Subject     string
	Text        string
	HTML        string
	Attachments []Attachment
	Headers     map[string]string
}

// Validate checks if the message has minimum required fields.
func (m *Message) Validate() error {
	if m.From.Email == "" {
		return fmt.Errorf("from address is required")
	}
	if len(m.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	for i, addr := range m.To {
		if addr.Email == "" {
			return fmt.Errorf("recipient %d has empty email", i)
		}
	}
	if m.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if m.Text == "" && m.HTML == "" {
		return fmt.Errorf("message body is required (text or html)")
	}
	return nil
}

// IsMultipart returns true if the message has both text and HTML parts.
func (m *Message) IsMultipart() bool {
	return m.Text != "" && m.HTML != ""
}

// HasAttachments returns true if the message has attachments.
func (m *Message) HasAttachments() bool {
	return len(m.Attachments) > 0
}

// AllRecipients returns all recipients (To + CC + BCC).
func (m *Message) AllRecipients() []Address {
	all := make([]Address, 0, len(m.To)+len(m.CC)+len(m.BCC))
	all = append(all, m.To...)
	all = append(all, m.CC...)
	all = append(all, m.BCC...)
	return all
}
