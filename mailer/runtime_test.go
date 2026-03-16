package mailer

import (
	"context"
	"testing"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/settings"
)

type fakeSettings struct {
	strings map[string]string
	ints    map[string]int
	bools   map[string]bool
}

func (f *fakeSettings) GetString(ctx context.Context, key string) (string, error) {
	if f == nil || f.strings == nil {
		return "", nil
	}

	return f.strings[key], nil
}

func (f *fakeSettings) GetInt(ctx context.Context, key string) (int, error) {
	if f == nil || f.ints == nil {
		return 0, nil
	}

	return f.ints[key], nil
}

func (f *fakeSettings) GetBool(ctx context.Context, key string) (bool, error) {
	if f == nil || f.bools == nil {
		return false, nil
	}

	return f.bools[key], nil
}

func TestNewSMTP(t *testing.T) {
	cfg := config.New()
	cfg.Mailer.Enabled = true
	cfg.Mailer.Mode = ModeActive
	cfg.Mailer.Provider = "smtp"
	cfg.Mailer.DefaultFrom.Email = "noreply@example.com"
	cfg.Mailer.SMTP.Host = "smtp.example.com"
	cfg.Mailer.SMTP.Port = 2525
	cfg.Mailer.SMTP.Username = "u"
	cfg.Mailer.SMTP.Password = "p"

	m := New(cfg, nil)

	smtp, ok := m.(*SMTPMailer)
	if !ok {
		t.Fatalf("New() type = %T, want *SMTPMailer", m)
	}

	if smtp.cfg.Host != "smtp.example.com" {
		t.Errorf("SMTP host = %q, want smtp.example.com", smtp.cfg.Host)
	}

	if smtp.cfg.Port != 2525 {
		t.Errorf("SMTP port = %d, want 2525", smtp.cfg.Port)
	}

	if smtp.cfg.DefaultFrom.Email != "noreply@example.com" {
		t.Errorf("SMTP default from = %q, want noreply@example.com", smtp.cfg.DefaultFrom.Email)
	}
}

func TestNewWithSettingsOverride(t *testing.T) {
	cfg := config.New()
	cfg.Mailer.Enabled = true
	cfg.Mailer.Mode = ModeActive
	cfg.Mailer.Provider = "smtp"
	cfg.Mailer.SMTP.Host = "smtp.example.com"

	settings := &fakeSettings{
		strings: map[string]string{
			SettingProvider:       "sendgrid",
			SettingSendGridAPIKey: "sg-key",
			SettingFromEmail:      "settings@example.com",
		},
		bools: map[string]bool{
			SettingEnabled: true,
		},
	}

	m := NewWithSettings(settings, cfg, nil)

	sendgrid, ok := m.(*SendGridMailer)
	if !ok {
		t.Fatalf("NewWithSettings() type = %T, want *SendGridMailer", m)
	}

	if sendgrid.cfg.APIKey != "sg-key" {
		t.Errorf("SendGrid api key = %q, want sg-key", sendgrid.cfg.APIKey)
	}

	if sendgrid.cfg.DefaultFrom.Email != "settings@example.com" {
		t.Errorf("SendGrid default from = %q, want settings@example.com", sendgrid.cfg.DefaultFrom.Email)
	}
}

func TestNewWithSettingsModeDisabled(t *testing.T) {
	cfg := config.New()
	cfg.Mailer.Enabled = true
	cfg.Mailer.Mode = ModeActive
	cfg.Mailer.Provider = "smtp"
	cfg.Mailer.SMTP.Host = "smtp.example.com"

	settings := &fakeSettings{
		strings: map[string]string{
			SettingMode: ModeDisabled,
		},
		bools: map[string]bool{
			SettingEnabled: true,
		},
	}

	m := NewWithSettings(settings, cfg, nil)
	if _, ok := m.(*NoopMailer); !ok {
		t.Fatalf("NewWithSettings() type = %T, want *NoopMailer", m)
	}
}

func TestNewUnknownProvider(t *testing.T) {
	cfg := config.New()
	cfg.Mailer.Enabled = true
	cfg.Mailer.Mode = ModeActive
	cfg.Mailer.Provider = "unknown"

	m := New(cfg, nil)
	if _, ok := m.(*NoopMailer); !ok {
		t.Fatalf("New() type = %T, want *NoopMailer", m)
	}
}

func TestNewMailgun(t *testing.T) {
	cfg := config.New()
	cfg.Mailer.Enabled = true
	cfg.Mailer.Mode = ModeActive
	cfg.Mailer.Provider = "mailgun"
	cfg.Mailer.DefaultFrom.Email = "noreply@example.com"
	cfg.Mailer.Mailgun.APIKey = "mg-key"
	cfg.Mailer.Mailgun.Domain = "mg.example.com"

	m := New(cfg, nil)

	mg, ok := m.(*MailgunMailer)
	if !ok {
		t.Fatalf("New() type = %T, want *MailgunMailer", m)
	}

	if mg.cfg.APIKey != "mg-key" {
		t.Errorf("Mailgun api key = %q, want mg-key", mg.cfg.APIKey)
	}

	if mg.cfg.Domain != "mg.example.com" {
		t.Errorf("Mailgun domain = %q, want mg.example.com", mg.cfg.Domain)
	}
}

func TestRegisterSchemas(t *testing.T) {
	r := settings.NewRegistry()
	RegisterSchemas(r)

	if _, ok := r.Get(SettingProvider); !ok {
		t.Fatalf("schema %q not registered", SettingProvider)
	}

	if _, ok := r.Get(SettingSMTPHost); !ok {
		t.Fatalf("schema %q not registered", SettingSMTPHost)
	}
}
