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

// These names belong to the application compositions named in the guides.
// Context compilation grants no transport, persistence or shutdown assurance.
const infrastructureContext = `
var ctx = context.Background()
var cfg, appCfg = config.New(), config.New()
var logger = hatlog.NewNoopLogger()
var database = &fixtureDatabase{}
type fixtureDatabase struct{connection *sql.DB}
func (database *fixtureDatabase) GetDB() *sql.DB{return database.connection}
var router = chi.NewRouter()
var broker pubsub.Broker = pubsub.NewNoopBroker()
var subscriber = &AuditService{subscriber:broker}
type AuditService struct{subscriber pubsub.Subscriber}
func (service *AuditService) handle(context.Context,pubsub.Envelope)error{return nil}
var service = struct{publisher pubsub.Publisher}{broker}
var invoice = struct{ID string}{"invoice-1"}
var order = map[string]string{"id":"order-1"}
var handleOrder pubsub.Handler = func(context.Context,pubsub.Envelope)error{return nil}
var settingsService = &fixtureSettings{}
type fixtureSettings struct{}
func (*fixtureSettings) GetString(context.Context,string)(string,error){return "",settings.ErrNotFound}
func (*fixtureSettings) GetInt(context.Context,string)(int,error){return 0,settings.ErrNotFound}
func (*fixtureSettings) GetBool(context.Context,string)(bool,error){return false,settings.ErrNotFound}
var store = schedulerpostgres.NewStore(database.GetDB())
var runner, sched = scheduler.New(store,scheduler.Config{},logger),scheduler.New(store,scheduler.Config{},logger)
var mail = mailer.New(cfg,logger)
var source io.Reader = strings.NewReader("")
var contentType = "image/png"
var baseTime = time.Date(2026,1,1,0,0,0,0,time.UTC)
var counter = telemetry.NewCounter()
var crashes = telemetry.NewCrashCollector()
var t *testing.T
`

func infrastructureSource(id, body string) (string, error) {
	if strings.HasPrefix(body, "package ") {
		return body, nil
	}

	declarations := ""

	context := infrastructureContext
	if id == "example:docs/how-to/store-images/README.md#block-3" {
		context = strings.ReplaceAll(context, "var store = schedulerpostgres.NewStore(database.GetDB())", "var jobStore = schedulerpostgres.NewStore(database.GetDB())")
		context = strings.ReplaceAll(context, "scheduler.New(store,", "scheduler.New(jobStore,")
		context += "var store=local.NewStore(\"var/uploads\",\"/uploads\")\n"
	}

	if id == "example:scheduler/README.md#block-6" {
		context = strings.ReplaceAll(context, "var database = &fixtureDatabase{}", "var database *sql.DB")
		context = strings.ReplaceAll(context, "database.GetDB()", "database")
	}

	switch {
	case strings.Contains(body, "type Mailer interface"):
		declarations = "type Message = mailer.Message\n" + body
		body = ""
	case strings.Contains(body, "type Store interface"):
		declarations = body
		body = ""
	case strings.Contains(body, "type Publisher interface"):
		declarations = "type Envelope = pubsub.Envelope\ntype Handler = pubsub.Handler\ntype SubscribeOptions = pubsub.SubscribeOptions\n" + body
		body = ""
	case strings.Contains(body, "type RequestCounter interface"), strings.Contains(body, "type CrashRecorder interface"), strings.Contains(body, "func (service *AuditService) Start"), strings.Contains(body, "func TestStore"):
		declarations = body
		body = ""
	}

	parsed, err := parser.ParseFile(token.NewFileSet(), "fragment.go", "package fixture\nfunc fragment()(fragmentError error){\n"+body+"\nreturn\n}\n", 0)
	if err != nil {
		return "", fmt.Errorf("invalid infrastructure fragment %s: %w", id, err)
	}

	var sinks strings.Builder

	for _, statement := range parsed.Decls[0].(*ast.FuncDecl).Body.List {
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

	text := declarations + context + "func fragment()(fragmentError error){\n" + body + sinks.String() + "\nreturn\n}\n"

	imports, err := importsFor(text)
	if err != nil {
		return "", err
	}

	return "package fixture\n" + imports + text, nil
}

func compileInfrastructureContexts(v *verification, discovered inventory) error {
	directory := filepath.Join(v.fixture, "infrastructure-contexts")

	err := v.module(directory)
	if err != nil {
		return err
	}

	// Extend the existing source-aware importer with this slice's actual owners.
	for _, owner := range []string{"fake", "image", "mailer", "pubsub", "scheduler", "telemetry", "testhelper"} {
		importPaths[owner] = "hatmax.adrianpk.com/" + owner
	}

	for name, path := range map[string]string{
		"io": "io", "testing": "testing", "local": "hatmax.adrianpk.com/image/local",
		"s3": "hatmax.adrianpk.com/image/s3", "stdprocessor": "hatmax.adrianpk.com/image/stdprocessor",
		"postgres": "hatmax.adrianpk.com/pubsub/postgres", "schedulerpostgres": "hatmax.adrianpk.com/scheduler/postgres",
	} {
		importPaths[name] = path
	}

	var selected []row

	for _, r := range discovered.rows {
		if r.slice != 5 || !strings.HasPrefix(r.id, "example:") {
			continue
		}

		block, err := rowBlock(discovered, r)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(block, "```go\n") {
			continue
		}

		source, err := infrastructureSource(r.id, blockBody(block))
		if err != nil {
			return err
		}

		file := filepath.Join(directory, fmt.Sprintf("fragment-%03d", len(selected)), "fragment.go")
		if strings.Contains(source, "func Test") {
			file = strings.TrimSuffix(file, ".go") + "_test.go"
		}

		err = writeFile(file, source)
		if err != nil {
			return err
		}

		selected = append(selected, r)
	}

	for _, arguments := range [][]string{{"mod", "tidy"}, {"test", "-run", "^$", "./..."}} {
		err = v.exec(directory, "go", arguments...)
		if err != nil {
			return err
		}
	}

	for _, r := range selected {
		v.record(r, "compiled", "exact published Go bytes compile in their named application context; executable outcomes require the separate owned workflow")
	}

	return nil
}
