package mailer

import "context"

// Mailer defines the interface for sending emails.
type Mailer interface {
	Send(ctx context.Context, msg *Message) error
}

// Config holds common mailer configuration.
type Config struct {
	DefaultFrom Address
	Provider    string
}

// SMTPConfig holds SMTP-specific configuration.
type SMTPConfig struct {
	Config
	Host               string
	Port               int
	Username           string
	Password           string
	TLS                bool
	StartTLS           bool
	InsecureSkipVerify bool
}
