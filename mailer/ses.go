package mailer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

// SESConfig holds AWS SES-specific configuration.
type SESConfig struct {
	Config
	Region               string
	AccessKeyID          string
	SecretAccessKey      string
	ConfigurationSetName string
}

// SESMailer sends emails via AWS Simple Email Service.
type SESMailer struct {
	cfg    SESConfig
	client *ses.Client
}

// NewSESMailer creates a new AWS SES mailer.
func NewSESMailer(ctx context.Context, cfg SESConfig) (*SESMailer, error) {
	var awsCfg aws.Config
	var err error

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			)),
		)
	} else {
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &SESMailer{
		cfg:    cfg,
		client: ses.NewFromConfig(awsCfg),
	}, nil
}

// Send sends an email via AWS SES.
func (m *SESMailer) Send(ctx context.Context, msg *Message) error {
	if err := msg.Validate(); err != nil {
		return err
	}

	from := msg.From
	if from.Email == "" {
		from = m.cfg.DefaultFrom
	}

	if msg.HasAttachments() {
		return m.sendRawEmail(ctx, msg, from)
	}

	return m.sendSimpleEmail(ctx, msg, from)
}

func (m *SESMailer) sendSimpleEmail(ctx context.Context, msg *Message, from Address) error {
	toAddrs := make([]string, len(msg.To))
	for i, addr := range msg.To {
		toAddrs[i] = addr.Email
	}

	dest := &types.Destination{
		ToAddresses: toAddrs,
	}

	if len(msg.CC) > 0 {
		ccAddrs := make([]string, len(msg.CC))
		for i, addr := range msg.CC {
			ccAddrs[i] = addr.Email
		}
		dest.CcAddresses = ccAddrs
	}

	if len(msg.BCC) > 0 {
		bccAddrs := make([]string, len(msg.BCC))
		for i, addr := range msg.BCC {
			bccAddrs[i] = addr.Email
		}
		dest.BccAddresses = bccAddrs
	}

	body := &types.Body{}

	if msg.Text != "" {
		body.Text = &types.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(msg.Text),
		}
	}

	if msg.HTML != "" {
		body.Html = &types.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(msg.HTML),
		}
	}

	emailMsg := &types.Message{
		Subject: &types.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(msg.Subject),
		},
		Body: body,
	}

	input := &ses.SendEmailInput{
		Source:      aws.String(from.String()),
		Destination: dest,
		Message:     emailMsg,
	}

	if msg.ReplyTo != nil {
		input.ReplyToAddresses = []string{msg.ReplyTo.Email}
	}

	if m.cfg.ConfigurationSetName != "" {
		input.ConfigurationSetName = aws.String(m.cfg.ConfigurationSetName)
	}

	_, err := m.client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("ses send email: %w", err)
	}

	return nil
}

func (m *SESMailer) sendRawEmail(ctx context.Context, msg *Message, from Address) error {
	smtpMailer := &SMTPMailer{}
	rawMsg, err := smtpMailer.buildRawMessage(msg)
	if err != nil {
		return fmt.Errorf("build raw message: %w", err)
	}

	destinations := make([]string, 0, len(msg.To)+len(msg.CC)+len(msg.BCC))
	for _, addr := range msg.AllRecipients() {
		destinations = append(destinations, addr.Email)
	}

	input := &ses.SendRawEmailInput{
		Source:       aws.String(from.Email),
		Destinations: destinations,
		RawMessage: &types.RawMessage{
			Data: rawMsg,
		},
	}

	if m.cfg.ConfigurationSetName != "" {
		input.ConfigurationSetName = aws.String(m.cfg.ConfigurationSetName)
	}

	_, err = m.client.SendRawEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("ses send raw email: %w", err)
	}

	return nil
}

// Verify sends a verification email to an address (required for sandbox mode).
func (m *SESMailer) Verify(ctx context.Context, email string) error {
	input := &ses.VerifyEmailIdentityInput{
		EmailAddress: aws.String(email),
	}

	_, err := m.client.VerifyEmailIdentity(ctx, input)
	if err != nil {
		return fmt.Errorf("ses verify email: %w", err)
	}

	return nil
}

// GetSendQuota returns the current sending quota.
func (m *SESMailer) GetSendQuota(ctx context.Context) (*SESQuota, error) {
	output, err := m.client.GetSendQuota(ctx, &ses.GetSendQuotaInput{})
	if err != nil {
		return nil, fmt.Errorf("ses get quota: %w", err)
	}

	return &SESQuota{
		Max24HourSend:   output.Max24HourSend,
		SentLast24Hours: output.SentLast24Hours,
		MaxSendRate:     output.MaxSendRate,
	}, nil
}

// SESQuota represents SES sending quota.
type SESQuota struct {
	Max24HourSend   float64
	SentLast24Hours float64
	MaxSendRate     float64
}

// RemainingQuota returns the remaining emails that can be sent in 24h.
func (q *SESQuota) RemainingQuota() float64 {
	return q.Max24HourSend - q.SentLast24Hours
}

var _ Mailer = (*SESMailer)(nil)
