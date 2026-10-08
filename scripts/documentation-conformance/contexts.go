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
	"slices"
	"strings"
)

// These adapters supply only names explicitly owned by the illustrative application.
// Building them checks API compatibility, not database, authentication or CRUD behavior.
const commonContext = `
var ctx = context.Background()
var cfg = &config.Config{}
var logger = hatlog.NewLogger(cfg)
var r = func() *http.Request {
 request := httptest.NewRequest("POST", "http://localhost/submit", strings.NewReader("name=Alice"))
 request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
 return request
}()
var w = httptest.NewRecorder()
var router = chi.NewRouter()
var mux = http.NewServeMux()
var srv = &http.Server{Handler: router}
var database, migrator, broker, featureStore, listener, featureHandler any
var tmplMgr = web.NewTemplateManager(assetsFS, logger, web.WithFuncMap(ui.FuncMap()))
var templates = tmplMgr
var staticFS fs.FS = assetsFS
var embeddedFS = assetsFS
var myFuncMap = template.FuncMap{"own": func() string { return "owned" }}
var tokenFromContext = func(context.Context) string { return "fixture-token" }
var settingsSvc *settings.Service
var csrfToken = "fixture-token"
var currentValues = map[string]string{"app.name": "Fixture"}
var validationErrors = map[string]string{}
var saveBtn = ui.NewButton("Save")
var cancelBtn = ui.NewButton("Cancel")
var footer = ui.NewContainer()
var content, html, actions template.HTML = "<p>fixture</p>", "<p>fixture</p>", "<button>fixture</button>"
var page, pageSize, totalCount = 2, 20, 41
var items = []string{"fixture"}
var data = map[string]any{"Name": "Alice"}
`

const embedContext = "//go:embed assets\nvar assetsFS embed.FS\n"

var contextOutcomes = map[string]string{
	"example:format/readme.md#block-1":     `if format.Number(1234567) != "1,234,567" || format.Integer(1234.56) != "1,235" || format.Price(150000, "EUR") != "150,000 €" || format.PriceWithDecimals(99.99, "USD") != "$99.99" { return fmt.Errorf("documented number/price outputs changed") }`,
	"example:htmx/readme.md#block-1":       `if attrs.Map()["hx-post"] != "/items" || attrs.Map()["hx-target"] != "#list" || attrs.Map()["hx-swap"] != "outerHTML" { return fmt.Errorf("documented HTMX attributes changed") }`,
	"example:htmx/readme.md#block-3":       `r.Header.Set("HX-Request", "true"); handler(w, r); if w.Code != 286 || w.Header().Get("HX-Redirect") != "/dashboard" || w.Header().Get("HX-Retarget") != "#notifications" { return fmt.Errorf("documented response headers/status changed") }`,
	"example:htmx/readme.md#block-4":       `if string(wrapper.Open()) != "<div id=\"counter\" hx-swap-oob=\"innerHTML\">" || string(wrapper.Close()) != "</div>" { return fmt.Errorf("documented OOB wrapper changed") }`,
	"example:i18n/readme.md#block-1":       `if text != "Buscar" || missing != "missing.key" || fn("common.search") != "Buscar" || translator.Get("en", "common.search") != "Search" || len(locales) != 2 || translator.HasLocale("de") { return fmt.Errorf("documented translation/fallback outcomes changed") }`,
	"example:modal/readme.md#block-1":      `if cfg.ID != "delete-modal" || cfg.Title != "Confirm Delete" || cfg.Size != modal.SizeLarge { return fmt.Errorf("documented modal configuration changed") }`,
	"example:pagination/readme.md#block-1": `if params.Offset() != 20 || params.Limit() != 20 || result.TotalCount != 41 { return fmt.Errorf("documented pagination context changed") }`,
	"example:web/readme.md#block-1":        `if !strings.Contains(w.Body.String(), "Alice") || w.Header().Get("Content-Type") != "text/html; charset=utf-8" { return fmt.Errorf("documented full/partial rendering changed") }; if err := tm.Stop(ctx); err != nil { return err }`,
	"example:web/readme.md#block-3":        `if name != "Alice" || message != "A contact with this email already exists." || w.Code != 303 { return fmt.Errorf("documented form/redirect outcomes changed") }`,
}

var importPaths = map[string]string{
	"context": "context", "embed": "embed", "errors": "errors", "fmt": "fmt",
	"http": "net/http", "httptest": "net/http/httptest", "netip": "net/netip",
	"sql": "database/sql", "strconv": "strconv",
	"fs": "io/fs", "os": "os", "strings": "strings", "template": "html/template", "time": "time",
	"chi": "github.com/go-chi/chi/v5", "hatlog": "hatmax.adrianpk.com/log", "log": "hatmax.adrianpk.com/log",
	"invoicefeat": "example.com/docfixture/invoicefeat",
}

func init() {
	for _, owner := range []string{"app", "auth", "crypto", "config", "db", "format", "htmx", "i18n", "middleware", "modal", "model", "pagination", "render", "seed", "settings", "slug", "ui", "validation", "web"} {
		importPaths[owner] = "hatmax.adrianpk.com/" + owner
	}
}

func blockBody(block string) string {
	first := strings.IndexByte(block, '\n')
	last := strings.LastIndex(strings.TrimSpace(block), "\n")

	return block[first+1:last] + "\n"
}

func sourceBlock(contents map[string]string, page string, number int) (string, error) {
	all := blocks(contents[page])
	if number < 1 || number > len(all) {
		return "", fmt.Errorf("missing documented block: %s#block-%d", page, number)
	}

	return blockBody(all[number-1]), nil
}

func importsFor(source string) (string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), "context.go", "package fixture\n"+source, 0)
	if err != nil {
		return "", err
	}

	selectors := make(map[string]bool)

	ast.Inspect(parsed, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if ok {
			name, ok := selector.X.(*ast.Ident)
			if ok && name.Obj == nil {
				selectors[name.Name] = true
			}
		}

		return true
	})

	var names []string

	for name := range importPaths {
		if selectors[name] {
			names = append(names, name)
		}
	}

	slices.Sort(names)

	var result strings.Builder
	result.WriteString("import (\n")

	for _, name := range names {
		fmt.Fprintf(&result, "%s %q\n", name, importPaths[name])
	}

	result.WriteString(")\n")

	return result.String(), nil
}

// Exact fragment bytes are preserved. Only application context and unused-result sinks are added.
func contextualSource(id, body string) (string, error) {
	declarations := embedContext + "//go:embed locales\nvar localesFS embed.FS\n"
	context := commonContext

	if strings.Contains(body, embedContext) {
		body = strings.Replace(body, embedContext, "", 1)
	}

	switch {
	case id == "example:app/readme.md#block-1":
		marker := strings.Index(body, "// Setup discovers")
		if marker < 0 {
			return "", fmt.Errorf("app declarations/assembly boundary changed")
		}

		declarations += body[:marker]
		body = body[marker:]
		context += "var dbPool any\nvar myService, anotherService MyService\n"
	case id == "example:app/readme.md#block-2":
		declarations += body + "\nvar _ Startable = app.Startable(nil)\nvar _ Stoppable = app.Stoppable(nil)\nvar _ RouteRegistrar = app.RouteRegistrar(nil)\nvar _ StartupStep = StartupStep(app.StartupStep{})\n"
		body = ""
	case id == "example:htmx/readme.md#block-3":
		declarations += body
		body = ""
	case id == "example:middleware/readme.md#block-2":
		body = "r := chi.NewRouter()\n" + body
	case strings.Contains(id, "pages-and-partials.md"):
		declarations += `
type invoiceView struct { Number string }
type fixtureService struct{}
func (fixtureService) List(context.Context) ([]invoiceView, error) { return nil, nil }
type Handler struct { service fixtureService; templates *web.TemplateManager; log log.Logger }
var h = &Handler{templates: tmplMgr, log: logger}
var view = map[string]any{}
var invoice invoiceView
`
		if strings.HasPrefix(strings.TrimSpace(body), "type pageView") {
			declarations += body
			body = ""
		}
	case id == "example:docs/tutorials/user-guide/request-boundary.md#block-3":
		declarations += `
type Handler struct{}
func (*Handler) list(http.ResponseWriter, *http.Request) {}
func (*Handler) create(http.ResponseWriter, *http.Request) {}
func (*Handler) requireAccount(next http.Handler) http.Handler { return next }
` + body
		body = ""
	case strings.Contains(id, "presentation-primitives.md"):
		declarations += `
type fixtureService struct{}
func (fixtureService) List(context.Context, int, int) ([]string, int, error) { return nil, 0, nil }
var h = struct{ service fixtureService }{}
var view = struct{ ConfirmDelete modal.Config }{}
`
	}

	if !strings.Contains(body+declarations, "log.") {
		context += "var log = logger\n"
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
			if ok && name.Name != "_" && name.Name != "err" {
				fmt.Fprintf(&sinks, "_ = %s\n", name.Name)
			}
		}
	}

	wrapped = "func fragment() (fragmentError error) {\n" + body + sinks.String() + contextOutcomes[id] + "\nreturn\n}\n"
	source := declarations + context + wrapped

	imports, err := importsFor(source)
	if err != nil {
		return "", err
	}

	return "package fixture\n" + imports + source, nil
}

func compileContexts(v *verification, discovered inventory) error {
	directory := filepath.Join(v.fixture, "contexts")

	err := v.module(directory)
	if err != nil {
		return err
	}
	// The feature chapter explicitly requires these application-owned implementations.
	var feature strings.Builder

	page := "docs/tutorials/user-guide/feature-anatomy.md"
	for number := 4; number <= 8; number++ {
		body, err := sourceBlock(discovered.contents, page, number)
		if err != nil {
			return err
		}

		feature.WriteString(body)
	}

	feature.WriteString(`
func (*Invoice) Validate() error { return nil }
func (*Handler) page(http.ResponseWriter, *http.Request) {}
func (*Handler) create(http.ResponseWriter, *http.Request) {}
func (*Handler) update(http.ResponseWriter, *http.Request) {}
func (*Handler) delete(http.ResponseWriter, *http.Request) {}
func NewPostgresStore(any) Store { return nil }
func NewHandler(*Service, *web.TemplateManager, log.Logger) *Handler { return &Handler{} }
`)
	featureSource := feature.String()

	imports, err := importsFor(featureSource)
	if err != nil {
		return err
	}

	err = writeFile(filepath.Join(directory, "invoicefeat", "feature.go"), "package invoicefeat\n"+imports+featureSource)
	if err != nil {
		return err
	}

	var compiled []row

	for _, r := range discovered.rows {
		if r.slice != 2 || !strings.HasPrefix(r.id, "example:") {
			continue
		}

		block, err := rowBlock(discovered, r)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(block, "```go\n") || strings.Contains(block, "package main") {
			continue
		}

		if strings.Contains(r.id, "feature-anatomy.md#block-") {
			// Declarations 4–8 are compiled together in invoicefeat; assembly 9 is a consumer.
			if !strings.HasSuffix(r.id, "#block-9") {
				compiled = append(compiled, r)

				continue
			}
		}

		body := blockBody(block)

		source, err := contextualSource(r.id, body)
		if err != nil {
			return err
		}

		snippetDir := filepath.Join(directory, fmt.Sprintf("fragment-%03d", len(compiled)))

		err = writeFile(filepath.Join(snippetDir, "fragment.go"), source)
		if err != nil {
			return err
		}

		if _, executable := contextOutcomes[r.id]; executable {
			err = writeFile(filepath.Join(snippetDir, "fragment_test.go"), "package fixture\nimport \"testing\"\nfunc TestDocumentedOutcome(t *testing.T) { if err := fragment(); err != nil { t.Fatal(err) } }\n")
			if err != nil {
				return err
			}
		}

		for file, text := range map[string]string{
			"assets/locales/en.yml": "common:\n  search: Search\n", "assets/locales/es.yml": "common:\n  search: Buscar\n",
			"assets/templates/auth/login.html": "<p>{{.Name}}</p>", "assets/templates/tasks/row.html": "<p>{{.Name}}</p>",
			"assets/css/style.css": "body {color: black}", "locales/en.yml": "common:\n  search: Search\n",
		} {
			if file == "assets/locales/en.yml" {
				text, err = sourceBlock(discovered.contents, "i18n/readme.md", 3)
				if err != nil {
					return err
				}
			}

			err = writeFile(filepath.Join(snippetDir, file), text)
			if err != nil {
				return err
			}
		}

		compiled = append(compiled, r)
	}

	if len(compiled) == 0 {
		return fmt.Errorf("no contextual Go fragments selected")
	}

	err = v.exec(directory, "go", "mod", "tidy")
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "build", "./...")
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "test", "-count=1", "-timeout=2m", "./...")
	if err != nil {
		return err
	}

	for _, r := range compiled {
		if _, executable := contextOutcomes[r.id]; executable {
			v.record(r, "executed", "exact published fragment executed with explicit fixture inputs; documented observable outputs asserted")
		} else {
			v.record(r, "compiled", "published bytes compile in typed application context; fixture methods do not prove CRUD or policy")
		}
	}

	for _, r := range discovered.rows {
		if r.id == "example:i18n/readme.md#block-3" {
			v.record(r, "executed", "exact published locale YAML loaded through i18n.LoadFromFS; documented translation observed")
		}
	}

	return nil
}
