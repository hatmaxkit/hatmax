// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"hatmax.adrianpk.com/log"
)

// Creation and search_path must use the same literal identifier. Unqualified
// application queries must target it on every pooled connection, not public.
func TestSchemaIdentifiers(t *testing.T) {
	if testing.Short() {
		t.Skip("database integration")
	}

	cfg, cleanup := setupTestDBWithConfig(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin := New(testAssetsFS, Postgres, cfg, log.NewTestLogger("error"))

	err := admin.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Stop(ctx)

	for _, tc := range []struct {
		name   string
		schema string
	}{
		{name: "mixed case", schema: "Ledger_" + randomString(8)},
		{name: "hyphen", schema: "ledger-invoices_" + randomString(8)},
		{name: "reserved word", schema: "select"},
		{name: "double quote", schema: `ledger"archive_` + randomString(8)},
		{name: "space", schema: "ledger archive_" + randomString(8)},
		{name: "comma", schema: "ledger,archive_" + randomString(8)},
		{name: "apostrophe and backslash", schema: `ledger's\archive_` + randomString(8)},
		{name: "SQL punctuation", schema: `ledger;--archive_` + randomString(8)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var exists bool

			err := admin.DB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = $1)", tc.schema).Scan(&exists)
			if err != nil {
				t.Fatal(err)
			}

			if exists {
				t.Skip("schema already exists; preserve caller-owned objects")
			}

			appConfig := *cfg
			appConfig.Database.Schema = tc.schema
			application := New(testAssetsFS, Postgres, &appConfig, log.NewTestLogger("error"))

			t.Cleanup(func() {
				application.Stop(context.Background())

				_, err := admin.DB.ExecContext(context.Background(), "DROP SCHEMA IF EXISTS "+pgx.Identifier{tc.schema}.Sanitize()+" CASCADE")
				if err != nil {
					t.Errorf("drop test-owned schema: %v", err)
				}
			})

			err = application.Start(ctx)
			if err != nil {
				t.Fatal(err)
			}

			// A second ensure must be idempotent and preserve the literal name.
			err = application.ensureSchema(ctx)
			if err != nil {
				t.Fatal(err)
			}

			first, err := application.DB.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer first.Close()

			second, err := application.DB.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer second.Close()

			for _, connection := range []*sql.Conn{first, second} {
				var current string

				err = connection.QueryRowContext(ctx, "SELECT current_schema()").Scan(&current)
				if err != nil || current != tc.schema {
					t.Fatalf("current schema = %q, %v; want %q", current, err, tc.schema)
				}
			}

			_, err = first.ExecContext(ctx, "CREATE TABLE connection_probe (id integer)")
			if err != nil {
				t.Fatal(err)
			}

			err = second.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = 'connection_probe')", tc.schema).Scan(&exists)
			if err != nil || !exists {
				t.Fatalf("unqualified table not created in literal schema: %v", err)
			}
		})
	}
}
