// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// Application-owned types supply compilation context without asserting invoice
// CRUD or production policy. Executable workflows receive separate receipts.
const dataContext = `
var ctx = context.Background()
var cfg = config.New()
var logger = log.NewNoopLogger()
var router = chi.NewRouter()
var database = db.New(assetsFS, db.Postgres, cfg, logger)
var tmplMgr = web.NewTemplateManager(assetsFS, logger)
var user = struct{ID string}{}
var parentID = model.NewID()
var password = "Fixture password 7!"
var email, username, confirmation, value, input = "reader@example.com", "reader", password, "", " READER@EXAMPLE.COM "
var messages = map[string]string{}
var port, reqID, userID = ":8080", "request-fixture", "user-fixture"
var settingStore settings.Store = &fixtureSettingStore{}
type Value = settings.Value
type fixtureSettingStore struct{}
func (*fixtureSettingStore) Get(context.Context, string) (string, error) {return "", settings.ErrNotFound}
func (*fixtureSettingStore) Set(context.Context, string, string) error {return nil}
func (*fixtureSettingStore) All(context.Context) ([]settings.Value, error) {return nil, nil}
func (*fixtureSettingStore) Delete(context.Context, string) error {return nil}
var reg = settings.NewRegistry()
var r = func() *http.Request {
 request := httptest.NewRequest("POST", "http://localhost/invoices", strings.NewReader("number=INV-1&notes=note"))
 request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
 return request
}()
var w = httptest.NewRecorder()
var form = func() *web.FormValues {values,err:=web.ParseForm(r);if err!=nil{panic(err)};return values}()
type InvoiceInput struct{ Number, Notes string }
type Invoice struct {ID, Number, Notes, Slug string; CreatedAt, UpdatedAt time.Time}
func (invoice *Invoice) Validate() error {
 errs := validation.ValidateAll(validation.Field("number",invoice.Number).Required().MaxLength(40), validation.Field("notes",invoice.Notes).MaxLength(2000).NoHTML())
 if errs.HasErrors() {return errs}
 return nil
}
var invoice = Invoice{ID:model.NewID(),Number:"INV-1"}
type invoiceForm struct {Number, Notes string; Errors web.FormErrors}
type invoiceHandler struct{}
func (*invoiceHandler) renderForm(http.ResponseWriter, invoiceForm) {}
var h = &invoiceHandler{}
`

func dataSource(id, body string, discovered inventory) (string, error) {
	declarations := embedContext
	context := dataContext

	if strings.Contains(body, embedContext) {
		body = strings.Replace(body, embedContext, "", 1)
	}

	switch id {
	case "example:docs/tutorials/user-guide/models-and-data-flow.md#block-2", "example:docs/tutorials/user-guide/models-and-data-flow.md#block-3", "example:docs/tutorials/user-guide/models-and-data-flow.md#block-5", "example:docs/tutorials/user-guide/persistence-and-migrations.md#block-3", "example:seed/readme.md#block-1", "example:settings/readme.md#block-4", "example:log/readme.md#block-2":
		declarations += body
		body = ""
	case "example:seed/readme.md#block-2":
		seeder, err := sourceBlock(discovered.contents, "seed/readme.md", 1)
		if err != nil {
			return "", err
		}

		declarations += seeder
	case "example:docs/how-to/connect-postgres/README.md#block-3":
		context += "var starts []app.StartupStep\nvar stops []func(context.Context) error\nvar registrars []app.RouteRegistrar\n"
	}

	wrapped := "func fragment() (fragmentError error) {\n" + body + "\nreturn\n}\n"

	parsed, err := parser.ParseFile(token.NewFileSet(), "fragment.go", "package fixture\n"+wrapped, 0)
	if err != nil {
		return "", fmt.Errorf("invalid contextual syntax %s: %w", id, err)
	}

	var sinks strings.Builder

	function, ok := parsed.Decls[0].(*ast.FuncDecl)
	if !ok {
		return "", fmt.Errorf("missing contextual function: %s", id)
	}

	for _, statement := range function.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE {
			continue
		}

		for _, lhs := range assignment.Lhs {
			name, ok := lhs.(*ast.Ident)
			if ok && name.Name != "_" {
				fmt.Fprintf(&sinks, "_ = %s\n", name.Name)
			}
		}
	}

	source := declarations + context + "func fragment() (fragmentError error) {\n" + body + sinks.String() + "\nreturn\n}\n"

	imports, err := importsFor(source)
	if err != nil {
		return "", err
	}

	return "package fixture\n" + imports + source, nil
}

func compileDataContexts(v *verification, discovered inventory) error {
	directory := filepath.Join(v.fixture, "data-contexts")

	err := v.module(directory)
	if err != nil {
		return err
	}

	// A typed application DAL supplies only the mapping shape. SQLC generation
	// and real database execution must independently establish persistence.
	err = writeFile(filepath.Join(directory, "dal", "invoice.go"), "package dal\nimport \"time\"\ntype Invoice struct{ID,Number,Notes string;CreatedAt,UpdatedAt time.Time}\n")
	if err != nil {
		return err
	}

	importPaths["dal"] = "example.com/docfixture/dal"
	defer delete(importPaths, "dal")

	var selected []row

	for _, r := range discovered.rows {
		if r.slice != 3 || !strings.HasPrefix(r.id, "example:") {
			continue
		}

		block, err := rowBlock(discovered, r)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(block, "```go\n") {
			continue
		}

		var source string

		if r.id == "example:docs/tutorials/user-guide/persistence-and-migrations.md#block-9" {
			source, err = contextualSource(r.id, blockBody(block))
		} else {
			source, err = dataSource(r.id, blockBody(block), discovered)
		}

		if err != nil {
			return err
		}

		fragment := filepath.Join(directory, fmt.Sprintf("fragment-%03d", len(selected)))

		err = writeFile(filepath.Join(fragment, "fragment.go"), source)
		if err != nil {
			return err
		}

		err = writeFile(filepath.Join(fragment, "assets", "fixture.txt"), "embed fixture\n")
		if err != nil {
			return err
		}

		err = writeFile(filepath.Join(fragment, "locales", "en.yml"), "fixture: value\n")
		if err != nil {
			return err
		}

		selected = append(selected, r)
	}

	// Reuse the Slice 2 chapter's explicitly illustrative feature definitions.
	var feature strings.Builder

	for number := 4; number <= 8; number++ {
		body, err := sourceBlock(discovered.contents, "docs/tutorials/user-guide/feature-anatomy.md", number)
		if err != nil {
			return err
		}

		feature.WriteString(body)
	}

	feature.WriteString("\nfunc (*Invoice) Validate() error{return nil}\nfunc (*Handler) page(http.ResponseWriter,*http.Request){}\nfunc (*Handler) create(http.ResponseWriter,*http.Request){}\nfunc (*Handler) update(http.ResponseWriter,*http.Request){}\nfunc (*Handler) delete(http.ResponseWriter,*http.Request){}\nfunc NewPostgresStore(any) Store{return nil}\nfunc NewHandler(*Service,*web.TemplateManager,log.Logger)*Handler{return &Handler{}}\n")

	imports, err := importsFor(feature.String())
	if err != nil {
		return err
	}

	err = writeFile(filepath.Join(directory, "invoicefeat", "feature.go"), "package invoicefeat\n"+imports+feature.String())
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "mod", "tidy")
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "build", "./...")
	if err != nil {
		return err
	}

	for _, r := range selected {
		v.record(r, "compiled", "exact published Go bytes compile with named application-owned store, handler and DAL adapters; no CRUD execution claim")
	}

	return nil
}
