// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

// Config holds the application configuration.
type Config struct {
	Log           LogConfig           `koanf:"log"`
	Server        ServerConfig        `koanf:"server"`
	Database      DatabaseConfig      `koanf:"database"`
	Auth          AuthConfig          `koanf:"auth"`
	Authenticator AuthenticatorConfig `koanf:"authenticator"`
	Recovery      RecoveryConfig      `koanf:"recovery"`
	Contact       ContactConfig       `koanf:"contact"`
	Property      PropertyConfig      `koanf:"property"`
	PubSub        PubSubConfig        `koanf:"pubsub"`
	Scheduler     SchedulerConfig     `koanf:"scheduler"`
	Mailer        MailerConfig        `koanf:"mailer"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level string `koanf:"level"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port string `koanf:"port"`
	Host string `koanf:"host"`
}

// DatabaseConfig holds PostgreSQL connection configuration.
type DatabaseConfig struct {
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	User     string `koanf:"user"`
	Password string `koanf:"password"`
	Database string `koanf:"database"`
	Schema   string `koanf:"schema"`
	SSLMode  string `koanf:"sslmode"`
}

// AuthConfig holds authentication and session configuration.
type AuthConfig struct {
	SessionRecentProofAge   string `koanf:"session_recent_proof_age"`
	SessionMaxPerSubject    int    `koanf:"session_max_per_subject"`
	SessionPageSize         int    `koanf:"session_page_size"`
	SessionTTL              string `koanf:"session_ttl"`
	SessionInactivityTTL    string `koanf:"session_inactivity_ttl"`
	SessionActivityInterval string `koanf:"session_activity_interval"`
	SessionTimeout          string `koanf:"session_timeout"`
	SessionCleanupBatch     int    `koanf:"session_cleanup_batch"`
	PasswordMinLen          int    `koanf:"password_min_len"`
	PasswordMaxLen          int    `koanf:"password_max_len"`
	PasswordMaxBytes        int    `koanf:"password_max_bytes"`
	PasswordCheckTimeout    string `koanf:"password_check_timeout"`
	PasswordTimeout         string `koanf:"password_timeout"`
	ArgonMemoryKiB          uint64 `koanf:"argon_memory_kib"`
	ArgonIterations         uint64 `koanf:"argon_iterations"`
	ArgonParallelism        uint64 `koanf:"argon_parallelism"`
	ArgonMaxMemoryKiB       uint64 `koanf:"argon_max_memory_kib"`
	ArgonMaxIterations      uint64 `koanf:"argon_max_iterations"`
	ArgonMaxParallelism     uint64 `koanf:"argon_max_parallelism"`
	PasswordMaxConcurrent   int    `koanf:"password_max_concurrent"`
	EmailEncryptionKey      string `koanf:"email_encryption_key"`
	EmailLookupKey          string `koanf:"email_lookup_key"`
}

// ContactConfig holds contact data-protection configuration.
type ContactConfig struct {
	PIIEncryptionKey string `koanf:"pii_encryption_key"`
	EmailLookupKey   string `koanf:"email_lookup_key"`
}

// PropertyConfig holds managed-property data-protection configuration.
type PropertyConfig struct {
	NotesProtectionKey string `koanf:"notes_protection_key"`
}

// PubSubConfig holds pub/sub configuration.
type PubSubConfig struct {
	Enabled      bool   `koanf:"enabled"`
	PollInterval string `koanf:"poll_interval"`
	BatchSize    int    `koanf:"batch_size"`
}

// SchedulerConfig holds scheduler configuration.
type SchedulerConfig struct {
	Enabled       bool   `koanf:"enabled"`
	Interval      string `koanf:"interval"`
	BatchSize     int    `koanf:"batch_size"`
	Workers       int    `koanf:"workers"`
	RetryAttempts int    `koanf:"retry_attempts"`
	RetryBackoff  string `koanf:"retry_backoff"`
}

// MailerConfig holds mailer configuration.
type MailerConfig struct {
	Enabled     bool                 `koanf:"enabled"`
	Mode        string               `koanf:"mode"`
	Provider    string               `koanf:"provider"`
	DefaultFrom MailerAddressConfig  `koanf:"default_from"`
	SMTP        MailerSMTPConfig     `koanf:"smtp"`
	Mailgun     MailerMailgunConfig  `koanf:"mailgun"`
	SendGrid    MailerSendGridConfig `koanf:"sendgrid"`
	SES         MailerSESConfig      `koanf:"ses"`
}

// MailerAddressConfig holds default sender address configuration.
type MailerAddressConfig struct {
	Email string `koanf:"email"`
	Name  string `koanf:"name"`
}

// MailerSMTPConfig holds SMTP provider configuration.
type MailerSMTPConfig struct {
	Host               string `koanf:"host"`
	Port               int    `koanf:"port"`
	Username           string `koanf:"username"`
	Password           string `koanf:"password"`
	TLS                bool   `koanf:"tls"`
	StartTLS           bool   `koanf:"starttls"`
	InsecureSkipVerify bool   `koanf:"insecure_skip_verify"`
}

// MailerSendGridConfig holds SendGrid provider configuration.
type MailerSendGridConfig struct {
	APIKey string `koanf:"api_key"`
}

// MailerMailgunConfig holds Mailgun provider configuration.
type MailerMailgunConfig struct {
	APIKey  string `koanf:"api_key"`
	Domain  string `koanf:"domain"`
	BaseURL string `koanf:"base_url"`
}

// MailerSESConfig holds SES provider configuration.
type MailerSESConfig struct {
	Region               string `koanf:"region"`
	AccessKeyID          string `koanf:"access_key_id"`
	SecretAccessKey      string `koanf:"secret_access_key"`
	ConfigurationSetName string `koanf:"configuration_set_name"`
}

// PollIntervalDuration parses the PollInterval string as a duration.
// Returns 100ms if parsing fails.
func (c PubSubConfig) PollIntervalDuration() time.Duration {
	d, err := time.ParseDuration(c.PollInterval)
	if err != nil {
		return 100 * time.Millisecond
	}

	return d
}

// IntervalDuration parses the scheduler interval.
// Returns 1m if parsing fails.
func (c SchedulerConfig) IntervalDuration() time.Duration {
	d, err := time.ParseDuration(c.Interval)
	if err != nil {
		return time.Minute
	}

	return d
}

// RetryBackoffDuration parses the scheduler retry backoff.
// Returns 1m if parsing fails.
func (c SchedulerConfig) RetryBackoffDuration() time.Duration {
	d, err := time.ParseDuration(c.RetryBackoff)
	if err != nil {
		return time.Minute
	}

	return d
}

// New creates a new Config with sensible defaults.
func New() *Config {
	return &Config{
		Log: LogConfig{
			Level: "info",
		},
		Server: ServerConfig{
			Port: ":8080",
			Host: "localhost",
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "dev",
			Password: "dev",
			Database: "dev",
			Schema:   "",
			SSLMode:  "disable",
		},
		Auth: AuthConfig{
			SessionRecentProofAge: "5m", SessionMaxPerSubject: 10, SessionPageSize: 50,
			SessionTTL:              "24h",
			SessionInactivityTTL:    "30m",
			SessionActivityInterval: "1m",
			SessionTimeout:          "5s",
			SessionCleanupBatch:     1000,
			PasswordMinLen:          15,
			PasswordMaxLen:          1024, PasswordMaxBytes: 4096,
			PasswordCheckTimeout: "2s", PasswordTimeout: "5s",
			ArgonMemoryKiB: 65536, ArgonIterations: 3, ArgonParallelism: 4,
			ArgonMaxMemoryKiB: 65536, ArgonMaxIterations: 3, ArgonMaxParallelism: 4,
			PasswordMaxConcurrent: 2,
		},
		PubSub: PubSubConfig{
			Enabled:      false,
			PollInterval: "100ms",
			BatchSize:    100,
		},
		Scheduler: SchedulerConfig{
			Enabled:       false,
			Interval:      "1m",
			BatchSize:     20,
			Workers:       1,
			RetryAttempts: 3,
			RetryBackoff:  "1m",
		},
		Mailer: MailerConfig{
			Enabled:  false,
			Mode:     "disabled",
			Provider: "smtp",
			DefaultFrom: MailerAddressConfig{
				Email: "noreply@localhost",
				Name:  "",
			},
			SMTP: MailerSMTPConfig{
				Port: 587,
			},
			SES: MailerSESConfig{
				Region: "us-east-1",
			},
		},
	}
}

// Load loads configuration from a YAML file with environment variable
// and command-line flag overrides.
//
// Configuration precedence (highest to lowest):
//  1. Command-line flags
//  2. Environment variables (with envPrefix)
//  3. YAML file (with env var expansion)
//  4. Default values
//
// Parameters:
//   - path: Path to YAML config file
//   - envPrefix: Prefix for environment variables (e.g., "HATMAX_")
//   - args: Command-line arguments (typically os.Args)
//
// Returns the loaded configuration or an error.
func Load(path, envPrefix string, args []string) (*Config, error) {
	k := koanf.New(".")
	cfg := New()

	fs := pflag.NewFlagSet(args[0], pflag.ExitOnError)
	fs.String("log.level", "info", "Log level (debug, info, error)")
	fs.String("server.port", ":8080", "HTTP server port")
	fs.String("server.host", "localhost", "HTTP server host")
	fs.String("database.host", "localhost", "Database host")
	fs.Int("database.port", 5432, "Database port")
	fs.String("database.user", "dev", "Database user")
	fs.String("database.password", "dev", "Database password")
	fs.String("database.database", "dev", "Database name")
	fs.String("database.schema", "", "Database schema")
	fs.String("database.sslmode", "disable", "Database SSL mode")
	fs.String("auth.session_recent_proof_age", "5m", "Maximum proof age for session management")
	fs.Int("auth.session_max_per_subject", 10, "Maximum retained sessions per subject")
	fs.Int("auth.session_page_size", 50, "Maximum sessions per management page")
	fs.String("auth.session_ttl", "24h", "Absolute session lifetime")
	fs.String("auth.session_inactivity_ttl", "30m", "Session inactivity lifetime")
	fs.String("auth.session_activity_interval", "1m", "Relevant session activity persistence interval")
	fs.String("auth.session_timeout", "5s", "Session storage operation timeout")
	fs.Int("auth.session_cleanup_batch", 1000, "Maximum expired sessions removed per cleanup")
	fs.Int("auth.password_min_len", 15, "Minimum normalized password code points")
	fs.Int("auth.password_max_len", 1024, "Maximum normalized password code points")
	fs.Int("auth.password_max_bytes", 4096, "Maximum raw and normalized password bytes")
	fs.String("auth.password_check_timeout", "2s", "Password checker timeout")
	fs.String("auth.password_timeout", "5s", "Total credential operation timeout")
	fs.Uint64("auth.argon_memory_kib", 65536, "Argon2id creation memory in KiB")
	fs.Uint64("auth.argon_iterations", 3, "Argon2id creation iterations")
	fs.Uint64("auth.argon_parallelism", 4, "Argon2id creation lanes")
	fs.Uint64("auth.argon_max_memory_kib", 65536, "Maximum verification memory in KiB")
	fs.Uint64("auth.argon_max_iterations", 3, "Maximum verification iterations")
	fs.Uint64("auth.argon_max_parallelism", 4, "Maximum verification lanes")
	fs.Int("auth.password_max_concurrent", 2, "Maximum active credential KDF operations")
	fs.String("auth.email_encryption_key", "", "Email encryption key")
	fs.String("auth.email_lookup_key", "", "Email lookup key")
	fs.String("contact.pii_encryption_key", "", "Contact PII encryption key")
	fs.String("contact.email_lookup_key", "", "Contact email lookup key")
	fs.String("property.notes_protection_key", "", "Property notes protection key")
	fs.String("recovery.verification_ttl", "24h", "Mailbox verification lifetime")
	fs.String("recovery.reset_ttl", "1h", "Password reset lifetime")
	fs.String("recovery.timeout", "5s", "Recovery operation timeout")
	fs.String("recovery.lease", "5s", "Recovery reservation lifetime")
	fs.Int("recovery.token_attempts", 5, "Maximum attempts per recovery token")
	fs.Int("recovery.issuance_attempts", 3, "Recovery issues per subject and purpose per fifteen minutes")
	fs.Int("recovery.completion_attempts", 10, "Recovery completions per subject per fifteen minutes")
	fs.Int("recovery.cleanup_batch", 1000, "Maximum retained tokens removed per cleanup call")
	fs.Bool("pubsub.enabled", false, "Enable pub/sub")
	fs.String("pubsub.poll_interval", "100ms", "Pub/sub poll interval")
	fs.Int("pubsub.batch_size", 100, "Pub/sub batch size")
	fs.Bool("scheduler.enabled", false, "Enable scheduler")
	fs.String("scheduler.interval", "1m", "Scheduler poll interval")
	fs.Int("scheduler.batch_size", 20, "Scheduler batch size")
	fs.Int("scheduler.workers", 1, "Scheduler workers")
	fs.Int("scheduler.retry_attempts", 3, "Maximum total scheduler attempts per slot, including the first")
	fs.String("scheduler.retry_backoff", "1m", "Fixed scheduler delay after a failed attempt")
	fs.Bool("mailer.enabled", false, "Enable mailer")
	fs.String("mailer.mode", "disabled", "Mailer runtime mode (disabled, dry_run, active)")
	fs.String("mailer.provider", "smtp", "Mailer provider (smtp, mailgun, sendgrid, ses)")
	fs.String("mailer.default_from.email", "noreply@localhost", "Default from email")
	fs.String("mailer.default_from.name", "", "Default from name")
	fs.String("mailer.smtp.host", "", "SMTP host")
	fs.Int("mailer.smtp.port", 587, "SMTP port")
	fs.String("mailer.smtp.username", "", "SMTP username")
	fs.String("mailer.smtp.password", "", "SMTP password")
	fs.Bool("mailer.smtp.tls", false, "SMTP implicit TLS")
	fs.Bool("mailer.smtp.starttls", false, "SMTP STARTTLS")
	fs.Bool("mailer.smtp.insecure_skip_verify", false, "Skip SMTP TLS cert verification")
	fs.String("mailer.mailgun.api_key", "", "Mailgun API key")
	fs.String("mailer.mailgun.domain", "", "Mailgun domain")
	fs.String("mailer.mailgun.base_url", "", "Mailgun API base URL")
	fs.String("mailer.sendgrid.api_key", "", "SendGrid API key")
	fs.String("mailer.ses.region", "us-east-1", "SES region")
	fs.String("mailer.ses.access_key_id", "", "SES access key id")
	fs.String("mailer.ses.secret_access_key", "", "SES secret access key")
	fs.String("mailer.ses.configuration_set_name", "", "SES configuration set name")
	fs.Parse(args[1:])

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	expanded := []byte(os.ExpandEnv(string(raw)))

	err = k.Load(rawbytes.Provider(expanded), yaml.Parser())
	if err != nil {
		return nil, fmt.Errorf("cannot parse yaml: %w", err)
	}

	err = k.Load(env.Provider(envPrefix, ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, envPrefix)), "_", ".", -1)
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("cannot load env vars: %w", err)
	}

	err = k.Load(posflag.Provider(fs, ".", k), nil)
	if err != nil {
		return nil, fmt.Errorf("cannot load flags: %w", err)
	}

	err = k.Unmarshal("", cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal config: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server.port is required")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database.database is required")
	}

	_, err := c.Auth.SessionSettings()
	if err != nil {
		return err
	}

	_, err = c.Auth.PasswordSettings()
	if err != nil {
		return err
	}

	_, err = c.Recovery.RecoverySettings()
	if err != nil {
		return err
	}

	if c.Scheduler.BatchSize < 1 {
		return fmt.Errorf("scheduler.batch_size must be at least 1")
	}

	if c.Scheduler.Workers < 1 {
		return fmt.Errorf("scheduler.workers must be at least 1")
	}

	if c.Scheduler.RetryAttempts < 1 {
		return fmt.Errorf("scheduler.retry_attempts must be at least 1")
	}

	return nil
}

// ConnectionString builds a PostgreSQL keyword/value connection string.
// Values are quoted and escaped; Schema selects one literal SQL identifier.
func (d DatabaseConfig) ConnectionString() string {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		connectionValue(d.Host), d.Port, connectionValue(d.User), connectionValue(d.Password),
		connectionValue(d.Database), connectionValue(d.SSLMode))

	if d.Schema != "" {
		// search_path has SQL identifier syntax inside a connection value.
		// Keep NUL bytes intact so pgx rejects them instead of changing the name.
		schema := `"` + strings.ReplaceAll(d.Schema, `"`, `""`) + `"`
		connStr += " search_path=" + connectionValue(schema)
	}

	return connStr
}

func connectionValue(value string) string {
	// PostgreSQL keyword values escape backslashes and apostrophes with \.
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)

	return "'" + value + "'"
}
