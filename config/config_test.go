package config

import (
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cfg := New()

	if cfg == nil {
		t.Fatal("New() returned nil")
	}

	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %s, want info", cfg.Log.Level)
	}

	if cfg.Server.Port != ":8080" {
		t.Errorf("Server.Port = %s, want :8080", cfg.Server.Port)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %s, want localhost", cfg.Database.Host)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want 5432", cfg.Database.Port)
	}

	if cfg.Auth.SessionTTL != "24h" {
		t.Errorf("Auth.SessionTTL = %s, want 24h", cfg.Auth.SessionTTL)
	}

	if cfg.Auth.PasswordMinLen != 8 {
		t.Errorf("Auth.PasswordMinLen = %d, want 8", cfg.Auth.PasswordMinLen)
	}

	if cfg.Auth.BCryptCost != 12 {
		t.Errorf("Auth.BCryptCost = %d, want 12", cfg.Auth.BCryptCost)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			cfg:     New(),
			wantErr: false,
		},
		{
			name: "empty server port",
			cfg: &Config{
				Server: ServerConfig{Port: ""},
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "dev",
					Database: "dev",
				},
				Auth: AuthConfig{
					PasswordMinLen: 8,
					BCryptCost:     12,
				},
			},
			wantErr: true,
			errMsg:  "server.port is required",
		},
		{
			name: "empty database host",
			cfg: &Config{
				Server: ServerConfig{Port: ":8080"},
				Database: DatabaseConfig{
					Host:     "",
					User:     "dev",
					Database: "dev",
				},
				Auth: AuthConfig{
					PasswordMinLen: 8,
					BCryptCost:     12,
				},
			},
			wantErr: true,
			errMsg:  "database.host is required",
		},
		{
			name: "empty database user",
			cfg: &Config{
				Server: ServerConfig{Port: ":8080"},
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "",
					Database: "dev",
				},
				Auth: AuthConfig{
					PasswordMinLen: 8,
					BCryptCost:     12,
				},
			},
			wantErr: true,
			errMsg:  "database.user is required",
		},
		{
			name: "empty database name",
			cfg: &Config{
				Server: ServerConfig{Port: ":8080"},
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "dev",
					Database: "",
				},
				Auth: AuthConfig{
					PasswordMinLen: 8,
					BCryptCost:     12,
				},
			},
			wantErr: true,
			errMsg:  "database.database is required",
		},
		{
			name: "invalid password min length",
			cfg: &Config{
				Server: ServerConfig{Port: ":8080"},
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "dev",
					Database: "dev",
				},
				Auth: AuthConfig{
					PasswordMinLen: 0,
					BCryptCost:     12,
				},
			},
			wantErr: true,
			errMsg:  "auth.password_min_len must be at least 1",
		},
		{
			name: "bcrypt cost too low",
			cfg: &Config{
				Server: ServerConfig{Port: ":8080"},
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "dev",
					Database: "dev",
				},
				Auth: AuthConfig{
					PasswordMinLen: 8,
					BCryptCost:     3,
				},
			},
			wantErr: true,
			errMsg:  "auth.bcrypt_cost must be between 4 and 31",
		},
		{
			name: "bcrypt cost too high",
			cfg: &Config{
				Server: ServerConfig{Port: ":8080"},
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "dev",
					Database: "dev",
				},
				Auth: AuthConfig{
					PasswordMinLen: 8,
					BCryptCost:     32,
				},
			},
			wantErr: true,
			errMsg:  "auth.bcrypt_cost must be between 4 and 31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		cfg      DatabaseConfig
		expected string
	}{
		{
			name: "with schema",
			cfg: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "dev",
				Password: "dev",
				Database: "dev",
				Schema:   "hatmax",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=dev password=dev dbname=dev sslmode=disable search_path=hatmax",
		},
		{
			name: "without schema",
			cfg: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "dev",
				Password: "dev",
				Database: "dev",
				Schema:   "",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=dev password=dev dbname=dev sslmode=disable",
		},
		{
			name: "with SSL enabled",
			cfg: DatabaseConfig{
				Host:     "db.example.com",
				Port:     5432,
				User:     "produser",
				Password: "secret",
				Database: "proddb",
				Schema:   "public",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5432 user=produser password=secret dbname=proddb sslmode=require search_path=public",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.ConnectionString()
			if result != tt.expected {
				t.Errorf("ConnectionString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLoadWithYAML(t *testing.T) {
	yamlContent := `log:
  level: debug
server:
  port: ":9090"
  host: "0.0.0.0"
database:
  host: testhost
  port: 5433
  user: testuser
  password: testpass
  database: testdb
  schema: testschema
  sslmode: require
auth:
  session_ttl: "12h"
  password_min_len: 10
  bcrypt_cost: 14
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(yamlContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name(), "HATMAX_", []string{"test"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %s, want debug", cfg.Log.Level)
	}

	if cfg.Server.Port != ":9090" {
		t.Errorf("Server.Port = %s, want :9090", cfg.Server.Port)
	}

	if cfg.Database.Host != "testhost" {
		t.Errorf("Database.Host = %s, want testhost", cfg.Database.Host)
	}

	if cfg.Database.Port != 5433 {
		t.Errorf("Database.Port = %d, want 5433", cfg.Database.Port)
	}

	if cfg.Auth.SessionTTL != "12h" {
		t.Errorf("Auth.SessionTTL = %s, want 12h", cfg.Auth.SessionTTL)
	}

	if cfg.Auth.PasswordMinLen != 10 {
		t.Errorf("Auth.PasswordMinLen = %d, want 10", cfg.Auth.PasswordMinLen)
	}

	if cfg.Auth.BCryptCost != 14 {
		t.Errorf("Auth.BCryptCost = %d, want 14", cfg.Auth.BCryptCost)
	}
}

func TestPollIntervalDuration(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		want     time.Duration
	}{
		{
			name:     "valid duration 200ms",
			interval: "200ms",
			want:     200 * time.Millisecond,
		},
		{
			name:     "valid duration 1s",
			interval: "1s",
			want:     1 * time.Second,
		},
		{
			name:     "invalid duration returns default",
			interval: "invalid",
			want:     100 * time.Millisecond,
		},
		{
			name:     "empty duration returns default",
			interval: "",
			want:     100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := PubSubConfig{PollInterval: tt.interval}
			got := cfg.PollIntervalDuration()
			if got != tt.want {
				t.Errorf("PollIntervalDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
