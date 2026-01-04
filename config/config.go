package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

// Config holds the application configuration.
type Config struct {
	Log      LogConfig      `koanf:"log"`
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"database"`
	Auth     AuthConfig     `koanf:"auth"`
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
	fs.Parse(args[1:])

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}
	expanded := []byte(os.ExpandEnv(string(raw)))

	if err := k.Load(rawbytes.Provider(expanded), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("cannot parse yaml: %w", err)
	}

	if err := k.Load(env.Provider(envPrefix, ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, envPrefix)), "_", ".", -1)
	}), nil); err != nil {
		return nil, fmt.Errorf("cannot load env vars: %w", err)
	}

	if err := k.Load(posflag.Provider(fs, ".", k), nil); err != nil {
		return nil, fmt.Errorf("cannot load flags: %w", err)
	}

	if err := k.Unmarshal("", cfg); err != nil {
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
