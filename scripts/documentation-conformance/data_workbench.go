// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"fmt"
	"path/filepath"
)

// The workbench executes current public validators and persistence APIs against
// the invocation-owned cluster. Embedded SQL is extracted from published pages.
const dataWorkbenchTests = `package workbench
import (
 "context"
 "database/sql"
 "embed"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
 "strings"
 "testing"
 "time"
 "example.com/docfixture/dal"
 "hatmax.adrianpk.com/config"
 "hatmax.adrianpk.com/db"
 "hatmax.adrianpk.com/log"
 "hatmax.adrianpk.com/model"
 "hatmax.adrianpk.com/seed"
 "hatmax.adrianpk.com/settings"
 "hatmax.adrianpk.com/validation"
)
//go:embed assets
var assetsFS embed.FS

func startupConfig(t *testing.T, schema string) *config.Config {
 t.Helper()
 port,err:=strconv.Atoi(os.Getenv("DB_PORT"));if err!=nil{t.Fatal(err)}
 cfg,err:=config.Load("connection.yaml","DOC_",[]string{"docfixture","--database.host="+os.Getenv("DB_HOST"),"--database.port="+strconv.Itoa(port),"--database.user="+os.Getenv("DB_USER"),"--database.password="+os.Getenv("DB_PASSWORD"),"--database.database="+os.Getenv("DB_NAME"),"--database.schema="+schema})
 if err!=nil{t.Fatal(err)};if err=cfg.Validate();err!=nil{t.Fatal(err)};return cfg
}
func startDB(t *testing.T, schema string) *db.Database {
 t.Helper();d:=db.New(assetsFS,db.Postgres,startupConfig(t,schema),log.NewNoopLogger())
 t.Cleanup(func(){if err:=d.Stop(context.Background());err!=nil{t.Error(err)}})
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
 if err:=d.Start(ctx);err!=nil{t.Fatal(err)};return d
}

// Effective loading must distinguish defaults, YAML expansion, supported env
// keys, explicitly changed flags and the underscore mapping limitation.
func TestConfigurationValues(t *testing.T) {
 t.Setenv("DOC_SERVER_PORT",":8102")
 t.Setenv("DOC_AUTH_SESSION_TTL","1h")
 t.Setenv("DOC_EXPANDED_HOST","expanded-host")
 path:=filepath.Join(t.TempDir(),"config.yaml")
 if err:=os.WriteFile(path,[]byte("server:\n  port: ':8101'\ndatabase:\n  host: '${DOC_EXPANDED_HOST}'\nauth:\n  session_ttl: '2h'\nunknown:\n  value: ignored\n"),0600);err!=nil{t.Fatal(err)}
 for _,tc:=range []struct{name string;args []string;port string}{
  {"environment",[]string{"fixture"},":8102"},
  {"flags",[]string{"fixture","--server.port=:8103"},":8103"},
 }{t.Run(tc.name,func(t *testing.T){cfg,err:=config.Load(path,"DOC_",tc.args);if err!=nil{t.Fatal(err)};if cfg.Server.Port!=tc.port||cfg.Database.Host!="expanded-host"||cfg.Auth.SessionTTL!="2h"||cfg.Database.User!="dev"{t.Fatal("documented loading precedence changed")};if err:=cfg.Validate();err!=nil{t.Fatal(err)}})}
 if _,err:=config.Load(filepath.Join(t.TempDir(),"absent.yaml"),"DOC_",[]string{"fixture"});err==nil{t.Fatal("missing file accepted")}
 if err:=os.WriteFile(path,[]byte("server: ["),0600);err!=nil{t.Fatal(err)}
 if _,err:=config.Load(path,"DOC_",[]string{"fixture"});err==nil{t.Fatal("invalid YAML accepted")}
 if err:=os.WriteFile(path,[]byte("server:\n  port: ''\n"),0600);err!=nil{t.Fatal(err)}
 cfg,err:=config.Load(path,"UNUSED_",[]string{"fixture"});if err!=nil{t.Fatal(err)};if cfg.Validate()==nil{t.Fatal("load performed or bypassed effective validation")}
}

// Parse failure exits its process, rather than returning an error to Load.
func TestFlagExit(t *testing.T) {
 if os.Getenv("DOC_FLAG_CHILD")=="true"{_,_=config.Load("connection.yaml","UNUSED_",[]string{"fixture","--unknown-flag"});os.Exit(0)}
 cmd:=exec.Command(os.Args[0],"-test.run=^TestFlagExit$");cmd.Env=append(os.Environ(),"DOC_FLAG_CHILD=true")
 output,err:=cmd.CombinedOutput();var exit *exec.ExitError
 if !errors.As(err,&exit)||exit.ExitCode()!=2||!strings.Contains(string(output),"unknown flag"){t.Fatalf("flag boundary: %v",err)}
}

// Every versioned config is loaded through the actual public validators.
func TestPublishedConfiguration(t *testing.T) {
 for _,name:=range []string{"connection.yaml","guide.yaml","ticked.yaml"}{t.Run(name,func(t *testing.T){cfg,err:=config.Load(name,"UNUSED_",[]string{"fixture"});if err!=nil{t.Fatal(err)};if err=cfg.Validate();err!=nil{t.Fatal(err)};if _,err=cfg.CredentialAdmission.CredentialAdmissionSettings();err!=nil{t.Fatal(err)};if _,err=cfg.AuthenticationIngressSettings();err!=nil{t.Fatal(err)};if name=="ticked.yaml"{if _,err=cfg.Authenticator.EnrollmentSettings();err!=nil{t.Fatal(err)}}})}
}

// Real transactions must keep prior files, roll back a failed file, skip
// already-tracked history even if its SQL changes, and ignore Down sections.
func TestMigrationHistory(t *testing.T) {
 ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second);defer cancel()
 d:=startDB(t,"migration_history")
 var current string;if err:=d.GetDB().QueryRowContext(ctx,"SELECT current_schema()").Scan(&current);err!=nil||current!="migration_history"{t.Fatal("literal schema not selected")}
 m:=db.NewMigrator(d,assetsFS,db.Postgres,log.NewNoopLogger())
 for i:=0;i<2;i++{if err:=m.Start(ctx);err!=nil{t.Fatal(err)}}
 var count int;if err:=d.GetDB().QueryRowContext(ctx,"SELECT count(*) FROM migrations").Scan(&count);err!=nil||count!=2{t.Fatal("migration replay/tracking changed")}
 if _,err:=d.GetDB().ExecContext(ctx,"INSERT INTO notes (id,body,created_at) VALUES ('note-1','kept',CURRENT_TIMESTAMP)");err!=nil{t.Fatal(err)}
 m.SetPath("assets/probe/edited");if err:=m.Start(ctx);err!=nil{t.Fatal("edited tracked migration replayed",err)}
 m.SetPath("assets/probe/order");if err:=m.Start(ctx);err==nil{t.Fatal("failed migration accepted")}
 var first,rolled,later bool
 if err:=d.GetDB().QueryRowContext(ctx,"SELECT to_regclass('first_committed') IS NOT NULL,to_regclass('rollback_marker') IS NOT NULL,to_regclass('later_marker') IS NOT NULL").Scan(&first,&rolled,&later);err!=nil||!first||rolled||later{t.Fatal("per-file transaction/order boundary changed")}
 m.SetPath("assets/probe/empty");if err:=m.Start(ctx);err==nil||!strings.Contains(err.Error(),"no Up section"){t.Fatal("empty Up accepted",err)}
 if err:=d.GetDB().QueryRowContext(ctx,"SELECT count(*) FROM notes WHERE body='kept'").Scan(&count);err!=nil||count!=1{t.Fatal("note lost or Down executed")}
}

// The package's published SQL runs with its UUID default and unique email.
func TestPackageMigration(t *testing.T) {
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
 d:=startDB(t,"package_sql");m:=db.NewMigrator(d,assetsFS,db.Postgres,log.NewNoopLogger());m.SetPath("assets/probe/package")
 if err:=m.Start(ctx);err!=nil{t.Fatal(err)}
 var id string;if err:=d.GetDB().QueryRowContext(ctx,"INSERT INTO users(email) VALUES ('reader@example.com') RETURNING id").Scan(&id);err!=nil{t.Fatal(err)}
 if _,err:=model.ParseID(id);err!=nil{t.Fatal("published SQL UUID default changed",err)}
 if _,err:=d.GetDB().ExecContext(ctx,"INSERT INTO users(email) VALUES ('reader@example.com')");err==nil{t.Fatal("email uniqueness absent")}
}

// A schema error retains a pool, while a failed ping leaves no pool. Cleanup
// belongs to the caller of the failing start and must close its partial state.
func TestConnectionOwnership(t *testing.T) {
 ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel()
 cfg:=startupConfig(t,"forbidden_schema");cfg.Database.User="readonly"
 d:=db.New(assetsFS,db.Postgres,cfg,log.NewNoopLogger())
 if err:=d.Start(ctx);err==nil||!strings.Contains(err.Error(),"cannot ensure schema")||d.GetDB()==nil{t.Fatal("schema failure ownership changed",err)}
 if err:=d.Stop(ctx);err!=nil{t.Fatal(err)};if err:=d.GetDB().PingContext(ctx);err==nil{t.Fatal("partial pool not closed")}
 cfg.Database.Host=filepath.Join(os.Getenv("DB_HOST"),"absent");d=db.New(assetsFS,db.Postgres,cfg,log.NewNoopLogger())
 if err:=d.Start(ctx);err==nil||!strings.Contains(err.Error(),"cannot ping database")||d.GetDB()!=nil{t.Fatal("failed ping retained a pool",err)}
}

// Current database and generated SQLC operations establish the durable
// invoice mapping. The illustrative domain validators remain application-owned.
func TestInvoiceMapping(t *testing.T) {
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
 d:=startDB(t,"invoice_mapping");m:=db.NewMigrator(d,assetsFS,db.Postgres,log.NewNoopLogger());if err:=m.Start(ctx);err!=nil{t.Fatal(err)}
 input:=InvoiceInput{Number:"INV-1",Notes:"note"};invoice,err:=NewInvoice(input);if err!=nil{t.Fatal(err)}
 q:=dal.New(d.GetDB());if err=q.CreateInvoice(ctx,dal.CreateInvoiceParams{ID:invoice.ID,Number:invoice.Number,Notes:invoice.Notes,CreatedAt:invoice.CreatedAt,UpdatedAt:invoice.UpdatedAt});err!=nil{t.Fatal(err)}
 row,err:=q.GetInvoice(ctx,invoice.ID);if err!=nil{t.Fatal(err)};mapped:=invoiceFromRow(row)
 if mapped.ID!=invoice.ID||mapped.Number!="INV-1"||mapped.Notes!="note"{t.Fatal("generated query/domain mapping changed")}
 before:=*invoice;if err=invoice.Update(InvoiceInput{});err==nil||*invoice!=before{t.Fatal("invalid update mutated original")}
 if err=invoice.Update(InvoiceInput{Number:"INV-2",Notes:"updated"});err!=nil||invoice.ID!=before.ID||invoice.CreatedAt!=before.CreatedAt||invoice.UpdatedAt.Before(before.UpdatedAt){t.Fatal("valid update changed immutable state")}
}

type sqlSettings struct{ db *sql.DB }
func (s *sqlSettings) Get(ctx context.Context,key string)(string,error){var raw string;err:=s.db.QueryRowContext(ctx,"SELECT raw FROM doc_settings WHERE key=$1",key).Scan(&raw);if errors.Is(err,sql.ErrNoRows){return "",fmt.Errorf("lookup: %w",settings.ErrNotFound)};return raw,err}
func (s *sqlSettings) Set(ctx context.Context,key,raw string)error{_,err:=s.db.ExecContext(ctx,"INSERT INTO doc_settings(key,raw) VALUES ($1,$2) ON CONFLICT(key) DO UPDATE SET raw=EXCLUDED.raw",key,raw);return err}
func (s *sqlSettings) Delete(ctx context.Context,key string)error{_,err:=s.db.ExecContext(ctx,"DELETE FROM doc_settings WHERE key=$1",key);return err}
func (s *sqlSettings) All(ctx context.Context)([]settings.Value,error){rows,err:=s.db.QueryContext(ctx,"SELECT key,raw FROM doc_settings ORDER BY key");if err!=nil{return nil,err};defer rows.Close();var values []settings.Value;for rows.Next(){var v settings.Value;if err:=rows.Scan(&v.Key,&v.Raw);err!=nil{return nil,err};values=append(values,v)};return values,rows.Err()}

// An application-owned SQL store must preserve defaults, empty strings,
// invalid raw values and cancellation across real persistence operations.
func TestRuntimeSettings(t *testing.T) {
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
 d:=startDB(t,"runtime_settings");if _,err:=d.GetDB().ExecContext(ctx,"CREATE TABLE doc_settings(key text PRIMARY KEY,raw text NOT NULL)");err!=nil{t.Fatal(err)}
 reg:=settings.NewRegistry();minimum,maximum:=1,100
 reg.Register(settings.Schema{Key:"page",Type:settings.Int,Default:"20",Required:true,Min:&minimum,Max:&maximum})
 reg.Register(settings.Schema{Key:"name",Type:settings.String,Default:"default"})
 store:=&sqlSettings{d.GetDB()};svc:=settings.NewService(reg,store)
 if v,err:=svc.GetInt(ctx,"page");err!=nil||v!=20{t.Fatal("absent default changed")}
 if err:=svc.Set(ctx,"page","101");err==nil{t.Fatal("invalid registered write accepted")}
 if err:=svc.Set(ctx,"page","30");err!=nil{t.Fatal(err)};if v,err:=svc.GetInt(ctx,"page");err!=nil||v!=30{t.Fatal("update not persisted")}
 if err:=svc.Set(ctx,"name","");err!=nil{t.Fatal(err)};if v,err:=svc.GetString(ctx,"name");err!=nil||v!=""{t.Fatal("stored empty replaced by default")}
 if err:=store.Set(ctx,"page","");err!=nil{t.Fatal(err)};if _,err:=svc.GetInt(ctx,"page");err==nil{t.Fatal("invalid stored empty accepted")}
 if err:=svc.Delete(ctx,"page");err!=nil{t.Fatal(err)};if v,err:=svc.GetInt(ctx,"page");err!=nil||v!=20{t.Fatal("delete did not restore default")}
 canceled,stop:=context.WithCancel(ctx);stop();if _,err:=svc.GetString(canceled,"missing");!errors.Is(err,context.Canceled){t.Fatal("cancellation converted to absence",err)}
 if err:=store.Set(ctx,"page","101");err!=nil{t.Fatal(err)};if v,err:=svc.GetInt(ctx,"page");err!=nil||v!=101{t.Fatal("getter unexpectedly applies schema bounds")}
 if err:=d.Stop(ctx);err!=nil{t.Fatal(err)};if _,err:=svc.GetString(ctx,"name");err==nil{t.Fatal("closed pool replaced by default")}
}

type fixtureSeed struct{ db *sql.DB; name string; fail bool; calls *int }
func (s fixtureSeed) Name()string{return s.name}
func (s fixtureSeed) Seed(ctx context.Context)error{*s.calls++;_,err:=s.db.ExecContext(ctx,"INSERT INTO seed_effects(id) VALUES ($1) ON CONFLICT DO NOTHING",s.name);if err!=nil{return err};if s.fail{return errors.New("fixture seed failure")};return nil}

// Tracking follows effects. A failed seeder can be retried, later work stops,
// and the exact published UserSeeder remains safe after tracking loss.
func TestSeedTracking(t *testing.T) {
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
 d:=startDB(t,"seed_tracking");if _,err:=d.GetDB().ExecContext(ctx,"CREATE TABLE users(id text PRIMARY KEY,email text NOT NULL UNIQUE);CREATE TABLE seed_effects(id text PRIMARY KEY)");err!=nil{t.Fatal(err)}
 refs:=seed.NewRefMap();user:=&UserSeeder{db:d.GetDB(),refs:refs};runner:=seed.NewRunner(d,[]seed.Seeder{user},log.NewNoopLogger())
 if err:=runner.Start(ctx);err!=nil{t.Fatal(err)};first,err:=refs.Resolve(seed.Ref("admin"));if err!=nil{t.Fatal(err)}
 if err:=runner.Start(ctx);err!=nil{t.Fatal(err)}
 if _,err:=d.GetDB().ExecContext(ctx,"DELETE FROM _seeds WHERE id='users'");err!=nil{t.Fatal(err)}
 if err:=runner.Start(ctx);err!=nil{t.Fatal(err)};second,err:=refs.Resolve(seed.Ref("admin"));if err!=nil||first!=second{t.Fatal("retry created a different durable user")}
 calls,later:=0,0;r:=seed.NewRunner(d,[]seed.Seeder{fixtureSeed{d.GetDB(),"retry",true,&calls},fixtureSeed{d.GetDB(),"later",false,&later}},log.NewNoopLogger())
 if err:=r.Start(ctx);err==nil||calls!=1||later!=0{t.Fatal("seed failure did not stop later work")}
 var tracked int;if err:=d.GetDB().QueryRowContext(ctx,"SELECT count(*) FROM _seeds WHERE id='retry'").Scan(&tracked);err!=nil||tracked!=0{t.Fatal("failed seed tracked")}
 r=seed.NewRunner(d,[]seed.Seeder{fixtureSeed{d.GetDB(),"retry",false,&calls}},log.NewNoopLogger());if err:=r.Start(ctx);err!=nil||calls!=2{t.Fatal("untracked seed did not retry")}
 fresh:=seed.NewRefMap();if fresh.Has(seed.Ref("admin")){t.Fatal("references unexpectedly persisted")}
 if _,err:=d.GetDB().ExecContext(ctx,"ALTER TABLE _seeds ADD CONSTRAINT reject_tracking_loss CHECK(id != 'tracking-loss')");err!=nil{t.Fatal(err)}
 lost:=0;r=seed.NewRunner(d,[]seed.Seeder{fixtureSeed{d.GetDB(),"tracking-loss",false,&lost}},log.NewNoopLogger())
 for attempt:=1;attempt<=2;attempt++{if err:=r.Start(ctx);err==nil||!strings.Contains(err.Error(),"mark seed")||lost!=attempt{t.Fatal("tracking failure did not retry completed effects",err)}}
}
type InvoiceInput struct{Number,Notes string}
type Invoice struct{ID,Number,Notes string;CreatedAt,UpdatedAt time.Time}
func (invoice *Invoice) Validate()error{errs:=validation.ValidateAll(validation.Field("number",invoice.Number).Required().MaxLength(40),validation.Field("notes",invoice.Notes).MaxLength(2000).NoHTML());if errs.HasErrors(){return errs};return nil}
`

func (v *verification) dataWorkbench(discovered inventory) error {
	directory := filepath.Join(v.fixture, "data-workbench")

	err := v.module(directory)
	if err != nil {
		return err
	}

	files := map[string]string{
		"workbench_test.go": dataWorkbenchTests,
		"guide.yaml":        discovered.contents["examples/guide/config.yaml"],
		"ticked.yaml":       discovered.contents["examples/ticked/config.yaml"],
		"sqlc.yaml":         "version: '2'\nsql:\n  - engine: postgresql\n    queries: queries.sql\n    schema: assets/migration/postgres\n    gen:\n      go:\n        package: dal\n        out: dal\n",
		"assets/probe/edited/202609270001-create_notes.sql": "-- +migrate Up\nTHIS SQL MUST NOT REPLAY;\n",
		"assets/probe/order/010-first.sql":                  "-- +migrate Up\nCREATE TABLE first_committed(id integer);\n",
		"assets/probe/order/020-failure.sql":                "-- +migrate Up\nCREATE TABLE rollback_marker(id integer);\nINSERT INTO deliberately_absent_table VALUES (1);\n",
		"assets/probe/order/030-later.sql":                  "-- +migrate Up\nCREATE TABLE later_marker(id integer);\n",
		"assets/probe/empty/040-empty.sql":                  "-- +migrate Up\n-- +migrate Down\n",
	}

	for file, binding := range map[string]struct {
		page   string
		number int
	}{
		"connection.yaml": {"docs/how-to/connect-postgres/README.md", 1},
		"assets/migration/postgres/202609270001-create_notes.sql": {"docs/how-to/apply-migrations/README.md", 1},
		"assets/migration/postgres/004-invoices.sql":              {"docs/tutorials/user-guide/persistence-and-migrations.md", 6},
		"queries.sql":                        {"docs/tutorials/user-guide/persistence-and-migrations.md", 7},
		"assets/probe/package/001-users.sql": {"db/readme.md", 3},
	} {
		body, err := sourceBlock(discovered.contents, binding.page, binding.number)
		if err != nil {
			return err
		}

		files[file] = body
	}

	for _, binding := range []struct {
		page   string
		number int
	}{
		{"docs/tutorials/user-guide/models-and-data-flow.md", 2},
		{"docs/tutorials/user-guide/models-and-data-flow.md", 3},
		{"docs/tutorials/user-guide/models-and-data-flow.md", 5},
		{"seed/readme.md", 1},
	} {
		body, err := sourceBlock(discovered.contents, binding.page, binding.number)
		if err != nil {
			return err
		}

		files["workbench_test.go"] += "\n" + body
	}

	for file, body := range files {
		err = writeFile(filepath.Join(directory, file), body)
		if err != nil {
			return err
		}
	}

	err = v.exec(directory, "sqlc", "generate")
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "mod", "tidy")
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "test", "-json", "-count=1", "-timeout=3m", "./...")
	if err != nil {
		return fmt.Errorf("published data workbench: %w", err)
	}

	return nil
}
