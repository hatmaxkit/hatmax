// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Configured values must survive pgx parsing without becoming extra connection
// options; a schema is one literal identifier, not a search-path expression.
func TestConnectionValues(t *testing.T) {
	t.Setenv("PGPASSFILE", filepath.Join(t.TempDir(), "absent"))
	t.Setenv("PGSERVICE", "")
	t.Setenv("PGPASSWORD", "environment-password")

	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "plain", value: "ledger"},
		{name: "spaces", value: "ledger account"},
		{name: "whitespace", value: "ledger\taccount\n"},
		{name: "apostrophe", value: "ledger's account"},
		{name: "backslash", value: `ledger\account`},
		{name: "double quote", value: `ledger"account`},
		{name: "combined", value: ` ledger's\"account `},
		{name: "option text", value: "ledger sslmode=disable application_name=unexpected"},
		{name: "unicode", value: "ledger-é"},
		{name: "empty", value: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DatabaseConfig{Host: "db.example.test", Port: 5433, User: tc.value, Password: tc.value, Database: tc.value, Schema: tc.value, SSLMode: "require"}
			if tc.value == "" {
				// User and database are required by Config.Validate, unlike an
				// empty password/schema; keep valid application values here.
				cfg.User, cfg.Database = "ledger", "ledger"
			}

			parsed, err := pgx.ParseConfig(cfg.ConnectionString())
			if err != nil {
				t.Fatalf("parse configured values: %v", err)
			}

			if parsed.Host != cfg.Host || parsed.Port != uint16(cfg.Port) || parsed.User != cfg.User || parsed.Password != cfg.Password || parsed.Database != cfg.Database {
				t.Fatalf("connection values did not round-trip for %s", tc.name)
			}

			wantSchema := ""
			if cfg.Schema != "" {
				wantSchema = pgx.Identifier{cfg.Schema}.Sanitize()
			}

			if parsed.RuntimeParams["search_path"] != wantSchema {
				t.Errorf("search_path = %q; want %q", parsed.RuntimeParams["search_path"], wantSchema)
			}

			if parsed.RuntimeParams["application_name"] == "unexpected" {
				t.Error("configured value became an injected option")
			}
		})
	}
}

// Socket paths and host values use the same keyword escaping as credentials.
func TestConnectionHosts(t *testing.T) {
	for _, host := range []string{"localhost", "::1", "/tmp/socket path", `/tmp/socket'\path`} {
		t.Run(host, func(t *testing.T) {
			cfg := DatabaseConfig{Host: host, Port: 5432, User: "ledger", Password: "secret", Database: "ledger", SSLMode: "disable"}

			parsed, err := pgx.ParseConfig(cfg.ConnectionString())
			if err != nil {
				t.Fatal(err)
			}

			if parsed.Host != host {
				t.Errorf("host = %q; want %q", parsed.Host, host)
			}
		})
	}
}

// Empty fields remain explicit keyword values, and malformed sslmode values
// must not be reinterpreted as a valid mode followed by injected parameters.
func TestConnectionOptions(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  DatabaseConfig
		want string
	}{
		{name: "empty fields", want: "host='' port=0 user='' password='' dbname='' sslmode=''"},
		{name: "empty host", cfg: DatabaseConfig{Port: 5432, SSLMode: "disable"}, want: "host=''"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.cfg.ConnectionString(), tc.want) {
				t.Errorf("connection string does not retain explicit empty values")
			}
		})
	}

	t.Run("invalid sslmode", func(t *testing.T) {
		cfg := DatabaseConfig{Host: "localhost", Port: 5432, User: "ledger", Database: "ledger", SSLMode: "disable application_name=unexpected"}

		_, err := pgx.ParseConfig(cfg.ConnectionString())
		if err == nil {
			t.Error("invalid sslmode became connection syntax")
		}
	})

	t.Run("nul schema", func(t *testing.T) {
		cfg := DatabaseConfig{Host: "localhost", Port: 5432, User: "ledger", Database: "ledger", Schema: "ledger\x00archive", SSLMode: "disable"}

		_, err := pgx.ParseConfig(cfg.ConnectionString())
		if err == nil {
			t.Error("invalid NUL-containing schema was silently normalized")
		}
	})
}

// Keyword and nested identifier escaping must round-trip arbitrary non-NUL
// values through the actual driver parser, without adding connection options.
func FuzzConnectionValues(f *testing.F) {
	f.Setenv("PGSERVICE", "")

	for _, value := range []string{"", "account name", `account's\"name`, "sslmode=disable application_name=unexpected", "\t\n", "é", "\x00"} {
		f.Add(value)
	}

	f.Fuzz(func(t *testing.T, value string) {
		value = "ledger_" + value
		cfg := DatabaseConfig{Host: "localhost", Port: 5432, User: value, Password: value, Database: value, Schema: value, SSLMode: "disable"}

		parsed, err := pgx.ParseConfig(cfg.ConnectionString())
		if strings.ContainsRune(value, 0) {
			if err == nil {
				t.Fatal("NUL-containing connection string accepted")
			}

			return
		}

		if err != nil {
			t.Fatalf("round-trip parse: %v", err)
		}

		if parsed.User != value || parsed.Password != value || parsed.Database != value || parsed.RuntimeParams["search_path"] != (pgx.Identifier{value}).Sanitize() {
			t.Fatal("configured value was not retained")
		}
	})
}
