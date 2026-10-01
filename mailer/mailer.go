// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package mailer

import "context"

// Mailer defines the interface for sending emails.
type Mailer interface {
	Send(ctx context.Context, msg *Message) error
}

// Config holds common mailer configuration.
type Config struct {
	// DefaultFrom replaces an empty Message.From.Email before active-provider
	// validation. Send does not change the caller's message.
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

// MailgunConfig holds Mailgun-specific configuration.
type MailgunConfig struct {
	Config
	APIKey  string
	Domain  string
	BaseURL string
}
