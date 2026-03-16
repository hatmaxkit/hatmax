package mailer

import (
	"context"
	"strings"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/settings"
)

const (
	SettingEnabled            = "mailer.enabled"
	SettingMode               = "mailer.mode"
	SettingProvider           = "mailer.provider"
	SettingFromEmail          = "mailer.from_email"
	SettingFromName           = "mailer.from_name"
	SettingSMTPHost           = "mailer.smtp_host"
	SettingSMTPPort           = "mailer.smtp_port"
	SettingSMTPUsername       = "mailer.smtp_username"
	SettingSMTPPassword       = "mailer.smtp_password"
	SettingSMTPTLS            = "mailer.smtp_tls"
	SettingSMTPStartTLS       = "mailer.smtp_starttls"
	SettingSMTPInsecureSkip   = "mailer.smtp_insecure_skip_verify"
	SettingMailgunAPIKey      = "mailer.mailgun_api_key"
	SettingMailgunDomain      = "mailer.mailgun_domain"
	SettingMailgunBaseURL     = "mailer.mailgun_base_url"
	SettingSendGridAPIKey     = "mailer.sendgrid_api_key"
	SettingSESRegion          = "mailer.ses_region"
	SettingSESAccessKeyID     = "mailer.ses_access_key_id"
	SettingSESSecretAccessKey = "mailer.ses_secret_access_key"
	SettingSESConfigSetName   = "mailer.ses_configuration_set_name"
)

const (
	ModeDisabled = "disabled"
	ModeDryRun   = "dry_run"
	ModeActive   = "active"
)

var Schemas = []settings.Schema{
	{
		Key:         SettingEnabled,
		Type:        settings.Bool,
		Default:     "false",
		Label:       "Enable Mailer",
		Description: "Enable mail delivery provider",
	},
	{
		Key:         SettingMode,
		Type:        settings.Enum,
		Default:     ModeDisabled,
		Label:       "Mailer Runtime Mode",
		Description: "Controls mail delivery execution mode",
		Options:     []string{ModeDisabled, ModeDryRun, ModeActive},
		Labels:      []string{"Disabled", "Dry Run", "Active"},
	},
	{
		Key:         SettingProvider,
		Type:        settings.Enum,
		Default:     "smtp",
		Label:       "Mailer Provider",
		Description: "Mail delivery provider",
		Options:     []string{"smtp", "mailgun", "sendgrid", "ses"},
		Labels:      []string{"SMTP", "Mailgun", "SendGrid", "AWS SES"},
	},
	{Key: SettingFromEmail, Type: settings.String, Default: "noreply@localhost", Label: "From Email", MaxLength: 255},
	{Key: SettingFromName, Type: settings.String, Label: "From Name", MaxLength: 255},
	{Key: SettingSMTPHost, Type: settings.String, Label: "SMTP Host", MaxLength: 255},
	{Key: SettingSMTPPort, Type: settings.Int, Default: "587", Label: "SMTP Port"},
	{Key: SettingSMTPUsername, Type: settings.String, Label: "SMTP Username", MaxLength: 255},
	{Key: SettingSMTPPassword, Type: settings.String, Label: "SMTP Password", Secret: true, MaxLength: 255},
	{Key: SettingSMTPTLS, Type: settings.Bool, Default: "false", Label: "SMTP TLS"},
	{Key: SettingSMTPStartTLS, Type: settings.Bool, Default: "false", Label: "SMTP STARTTLS"},
	{Key: SettingSMTPInsecureSkip, Type: settings.Bool, Default: "false", Label: "SMTP Insecure Skip Verify"},
	{Key: SettingMailgunAPIKey, Type: settings.String, Label: "Mailgun API Key", Secret: true, MaxLength: 255},
	{Key: SettingMailgunDomain, Type: settings.String, Label: "Mailgun Domain", MaxLength: 255},
	{Key: SettingMailgunBaseURL, Type: settings.String, Label: "Mailgun Base URL", MaxLength: 255},
	{Key: SettingSendGridAPIKey, Type: settings.String, Label: "SendGrid API Key", Secret: true, MaxLength: 255},
	{Key: SettingSESRegion, Type: settings.String, Default: "us-east-1", Label: "SES Region", MaxLength: 64},
	{Key: SettingSESAccessKeyID, Type: settings.String, Label: "SES Access Key ID", Secret: true, MaxLength: 255},
	{Key: SettingSESSecretAccessKey, Type: settings.String, Label: "SES Secret Access Key", Secret: true, MaxLength: 255},
	{Key: SettingSESConfigSetName, Type: settings.String, Label: "SES Configuration Set Name", MaxLength: 255},
}

// SettingsProvider provides dynamic settings used to resolve mailer runtime configuration.
type SettingsProvider interface {
	GetString(ctx context.Context, key string) (string, error)
	GetInt(ctx context.Context, key string) (int, error)
	GetBool(ctx context.Context, key string) (bool, error)
}

// RegisterSchemas registers mailer settings schemas.
func RegisterSchemas(r *settings.Registry) {
	for _, s := range Schemas {
		r.Register(s)
	}
}

// New resolves mailer from static configuration.
func New(cfg *config.Config, log log.Logger) Mailer {
	return NewWithSettings(nil, cfg, log)
}

// NewWithSettings resolves mailer from static config and dynamic settings overrides.
func NewWithSettings(settings SettingsProvider, cfg *config.Config, log log.Logger) Mailer {
	resolved := resolveFromConfig(cfg)
	resolved = applySettings(context.Background(), settings, resolved)

	return resolved.build(context.Background(), log)
}

type resolvedConfig struct {
	enabled  bool
	mode     string
	provider string
	from     Address

	smtp     SMTPConfig
	mailgun  MailgunConfig
	sendgrid SendGridConfig
	ses      SESConfig
}

func resolveFromConfig(cfg *config.Config) resolvedConfig {
	base := config.New()
	if cfg != nil {
		base = cfg
	}

	from := Address{
		Email: base.Mailer.DefaultFrom.Email,
		Name:  base.Mailer.DefaultFrom.Name,
	}

	smtpCfg := SMTPConfig{
		Config: Config{
			DefaultFrom: from,
			Provider:    "smtp",
		},
		Host:               base.Mailer.SMTP.Host,
		Port:               base.Mailer.SMTP.Port,
		Username:           base.Mailer.SMTP.Username,
		Password:           base.Mailer.SMTP.Password,
		TLS:                base.Mailer.SMTP.TLS,
		StartTLS:           base.Mailer.SMTP.StartTLS,
		InsecureSkipVerify: base.Mailer.SMTP.InsecureSkipVerify,
	}

	sendGridCfg := SendGridConfig{
		Config: Config{
			DefaultFrom: from,
			Provider:    "sendgrid",
		},
		APIKey: base.Mailer.SendGrid.APIKey,
	}

	mailgunCfg := MailgunConfig{
		Config: Config{
			DefaultFrom: from,
			Provider:    "mailgun",
		},
		APIKey:  base.Mailer.Mailgun.APIKey,
		Domain:  base.Mailer.Mailgun.Domain,
		BaseURL: base.Mailer.Mailgun.BaseURL,
	}

	sesCfg := SESConfig{
		Config: Config{
			DefaultFrom: from,
			Provider:    "ses",
		},
		Region:               base.Mailer.SES.Region,
		AccessKeyID:          base.Mailer.SES.AccessKeyID,
		SecretAccessKey:      base.Mailer.SES.SecretAccessKey,
		ConfigurationSetName: base.Mailer.SES.ConfigurationSetName,
	}

	return resolvedConfig{
		enabled:  base.Mailer.Enabled,
		mode:     strings.ToLower(strings.TrimSpace(base.Mailer.Mode)),
		provider: strings.ToLower(strings.TrimSpace(base.Mailer.Provider)),
		from:     from,
		smtp:     smtpCfg,
		mailgun:  mailgunCfg,
		sendgrid: sendGridCfg,
		ses:      sesCfg,
	}
}

func applySettings(ctx context.Context, settings SettingsProvider, current resolvedConfig) resolvedConfig {
	if settings == nil {
		return current
	}

	if v, ok := getBool(ctx, settings, SettingEnabled); ok {
		current.enabled = v
	}

	if v, ok := getString(ctx, settings, SettingMode); ok {
		current.mode = strings.ToLower(strings.TrimSpace(v))
	}

	if v, ok := getString(ctx, settings, SettingProvider); ok {
		current.provider = strings.ToLower(strings.TrimSpace(v))
	}

	if v, ok := getString(ctx, settings, SettingFromEmail); ok {
		current.from.Email = v
	}

	if v, ok := getString(ctx, settings, SettingFromName); ok {
		current.from.Name = v
	}

	current.smtp.DefaultFrom = current.from
	current.mailgun.DefaultFrom = current.from
	current.sendgrid.DefaultFrom = current.from
	current.ses.DefaultFrom = current.from

	if v, ok := getString(ctx, settings, SettingSMTPHost); ok {
		current.smtp.Host = v
	}

	if v, ok := getInt(ctx, settings, SettingSMTPPort); ok {
		current.smtp.Port = v
	}

	if v, ok := getString(ctx, settings, SettingSMTPUsername); ok {
		current.smtp.Username = v
	}

	if v, ok := getString(ctx, settings, SettingSMTPPassword); ok {
		current.smtp.Password = v
	}

	if v, ok := getBool(ctx, settings, SettingSMTPTLS); ok {
		current.smtp.TLS = v
	}

	if v, ok := getBool(ctx, settings, SettingSMTPStartTLS); ok {
		current.smtp.StartTLS = v
	}

	if v, ok := getBool(ctx, settings, SettingSMTPInsecureSkip); ok {
		current.smtp.InsecureSkipVerify = v
	}

	if v, ok := getString(ctx, settings, SettingMailgunAPIKey); ok {
		current.mailgun.APIKey = v
	}

	if v, ok := getString(ctx, settings, SettingMailgunDomain); ok {
		current.mailgun.Domain = v
	}

	if v, ok := getString(ctx, settings, SettingMailgunBaseURL); ok {
		current.mailgun.BaseURL = v
	}

	if v, ok := getString(ctx, settings, SettingSendGridAPIKey); ok {
		current.sendgrid.APIKey = v
	}

	if v, ok := getString(ctx, settings, SettingSESRegion); ok {
		current.ses.Region = v
	}

	if v, ok := getString(ctx, settings, SettingSESAccessKeyID); ok {
		current.ses.AccessKeyID = v
	}

	if v, ok := getString(ctx, settings, SettingSESSecretAccessKey); ok {
		current.ses.SecretAccessKey = v
	}

	if v, ok := getString(ctx, settings, SettingSESConfigSetName); ok {
		current.ses.ConfigurationSetName = v
	}

	return current
}

func (c resolvedConfig) build(ctx context.Context, log log.Logger) Mailer {
	if !c.enabled {
		return NewNoopMailer(log)
	}

	mode := c.mode
	if mode == "" {
		mode = ModeActive
	}

	switch mode {
	case ModeDisabled, ModeDryRun:
		return NewNoopMailer(log)
	case ModeActive:
	default:
		if log != nil {
			log.Infof("mailer: unknown mode %q, using noop", mode)
		}

		return NewNoopMailer(log)
	}

	provider := c.provider
	if provider == "" {
		provider = "smtp"
	}

	switch provider {
	case "smtp":
		return NewSMTPMailer(c.smtp)
	case "mailgun":
		if strings.TrimSpace(c.mailgun.APIKey) == "" || strings.TrimSpace(c.mailgun.Domain) == "" {
			if log != nil {
				log.Info("mailer: mailgun selected but api key/domain are missing, using noop")
			}

			return NewNoopMailer(log)
		}

		return NewMailgunMailer(c.mailgun)
	case "sendgrid":
		if strings.TrimSpace(c.sendgrid.APIKey) == "" {
			if log != nil {
				log.Info("mailer: sendgrid selected but api key is empty, using noop")
			}

			return NewNoopMailer(log)
		}

		return NewSendGridMailer(c.sendgrid)
	case "ses":
		m, err := NewSESMailer(ctx, c.ses)
		if err != nil {
			if log != nil {
				log.Errorf("mailer: ses init failed: %v", err)
			}

			return NewNoopMailer(log)
		}

		return m
	default:
		if log != nil {
			log.Infof("mailer: unknown provider %q, using noop", provider)
		}

		return NewNoopMailer(log)
	}
}

func getString(ctx context.Context, settings SettingsProvider, key string) (string, bool) {
	v, err := settings.GetString(ctx, key)
	if err != nil {
		return "", false
	}

	return strings.TrimSpace(v), true
}

func getInt(ctx context.Context, settings SettingsProvider, key string) (int, bool) {
	v, err := settings.GetInt(ctx, key)
	if err != nil {
		return 0, false
	}

	return v, true
}

func getBool(ctx context.Context, settings SettingsProvider, key string) (bool, bool) {
	v, err := settings.GetBool(ctx, key)
	if err != nil {
		return false, false
	}

	return v, true
}
