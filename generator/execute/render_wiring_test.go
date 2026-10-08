// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

const canonicalCompositionRoot = `package main

import (
	"context"
	"embed"

	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/db"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/web"
)

func main() {
	var assets embed.FS

	configuration := &config.Config{}
	logger := log.NewNoopLogger()
	database := db.New(assets, db.Postgres, configuration, logger)
	migrator := db.NewMigrator(database, assets, db.Postgres, logger)
	tmplMgr := web.NewTemplateManager(assets, logger)
	router := app.NewRouter(logger)
	deps := []any{database, migrator, tmplMgr}
	app.Setup(context.Background(), router, deps...)
}
`

// Recognizing delegated application wiring must not choose between two real
// composition roots or mutate either candidate before that ambiguity is resolved.
func TestCompositionAmbiguity(t *testing.T) {
	root := copyExecutionFixture(t)
	writeExecutionFile(t, root, "main.go", canonicalCompositionRoot)
	writeExecutionFile(t, root, "internal/application/application.go", "package application\nimport \"hatmax.adrianpk.com/app\"\nfunc run() { app.Setup() }\n")
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice", Route: "/invoices", Fields: []intent.Field{{Name: "number", Type: "string", Required: true}},
	}, []string{"postgres_persistence", "runtime_validation"})

	_, err := Prepare(value, inventory, selectedBook)
	requireExecutionCode(t, err, "execution_layout_ambiguous")
}

func TestRenderCreateFeatureRendersAndAppliesCompleteManifest(t *testing.T) {
	root := copyExecutionFixture(t)
	writeExecutionFile(t, root, "main.go", canonicalCompositionRoot)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Label:  "Invoices",
		Fields: []intent.Field{
			{Name: "number", Type: "string", Label: "Number", Required: true},
			{Name: "notes", Type: "text", Label: "Notes"},
		},
	}, []string{"postgres_persistence", "runtime_validation", "htmx_form"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mutations, err := RenderCreateFeature(value, manifest, inventory)
	if err != nil {
		t.Fatalf("RenderCreateFeature() error = %v", err)
	}

	if len(mutations) != len(manifest.Edits) {
		t.Fatalf("rendered mutations = %d, want %d", len(mutations), len(manifest.Edits))
	}

	assertCanonicalRenderedGo(t, manifest, mutations)

	workspace, err := OpenWorkspace(context.Background(), manifest, value, inventory)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	for _, mutation := range mutations {
		_, err = workspace.Stage(mutation)
		if err != nil {
			t.Fatalf("Stage(%q) error = %v", mutation.EditID, err)
		}
	}

	result, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	if result.Status != ExecutionApplied || len(result.Changes) != len(manifest.Edits) {
		t.Fatalf("Commit() = %#v, want %d applied changes", result, len(manifest.Edits))
	}

	mainSource, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	wired := string(mainSource)
	if !strings.Contains(wired, "\t\"embed\"\n\n\tinvoicefeat") {
		t.Errorf("main.go import groups are not canonical:\n%s", wired)
	}

	for _, expected := range []string{
		`invoicefeat "example.com/property/internal/feat/invoice"`,
		"invoiceStore := invoicefeat.NewPostgresStore(database)",
		"invoiceService := invoicefeat.NewService(invoiceStore)",
		"invoiceHandler := invoicefeat.NewHandler(invoiceService, tmplMgr, logger)",
		"database, migrator, tmplMgr, invoiceStore, invoiceHandler",
	} {
		if !strings.Contains(strings.Join(strings.Fields(wired), " "), strings.Join(strings.Fields(expected), " ")) {
			t.Errorf("main.go does not contain %q:\n%s", expected, wired)
		}
	}

	for _, edit := range manifest.Edits {
		_, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(edit.Target)))
		if statErr != nil {
			t.Errorf("target %q was not applied: %v", edit.Target, statErr)
		}
	}
}

func TestRenderCreateFeatureRejectsNoncanonicalCompositionRoot(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{{Name: "number", Type: "string"}},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	_, err = RenderCreateFeature(value, manifest, inventory)
	requireExecutionCode(t, err, "execution_wiring_prerequisite_missing")
}
