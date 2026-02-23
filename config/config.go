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
	Log       LogConfig       `koanf:"log"`
	Server    ServerConfig    `koanf:"server"`
	Database  DatabaseConfig  `koanf:"database"`
	Auth      AuthConfig      `koanf:"auth"`
	PubSub    PubSubConfig    `koanf:"pubsub"`
	Scheduler SchedulerConfig `koanf:"scheduler"`
	Mailer    MailerConfig    `koanf:"mailer"`
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
	SessionTTL     string `koanf:"session_ttl"`
	PasswordMinLen int    `koanf:"password_min_len"`
	BCryptCost     int    `koanf:"bcrypt_cost"`
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
			SessionTTL:     "24h",
			PasswordMinLen: 8,
			BCryptCost:     12,
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
	fs.String("auth.session_ttl", "24h", "Session TTL")
	fs.Int("auth.password_min_len", 8, "Minimum password length")
	fs.Int("auth.bcrypt_cost", 12, "BCrypt cost factor")
	fs.Bool("pubsub.enabled", false, "Enable pub/sub")
	fs.String("pubsub.poll_interval", "100ms", "Pub/sub poll interval")
	fs.Int("pubsub.batch_size", 100, "Pub/sub batch size")
	fs.Bool("scheduler.enabled", false, "Enable scheduler")
	fs.String("scheduler.interval", "1m", "Scheduler poll interval")
	fs.Int("scheduler.batch_size", 20, "Scheduler batch size")
	fs.Int("scheduler.workers", 1, "Scheduler workers")
	fs.Int("scheduler.retry_attempts", 3, "Scheduler retry attempts")
	fs.String("scheduler.retry_backoff", "1m", "Scheduler retry backoff")
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

	if c.Auth.PasswordMinLen < 1 {
		return fmt.Errorf("auth.password_min_len must be at least 1")
	}

	if c.Auth.BCryptCost < 4 || c.Auth.BCryptCost > 31 {
		return fmt.Errorf("auth.bcrypt_cost must be between 4 and 31")
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

// ConnectionString builds a PostgreSQL connection string with schema support.
func (d DatabaseConfig) ConnectionString() string {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Database, d.SSLMode)

	if d.Schema != "" {
		connStr += fmt.Sprintf(" search_path=%s", d.Schema)
	}

	return connStr
}
