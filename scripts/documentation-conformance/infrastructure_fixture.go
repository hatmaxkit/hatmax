// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

// This Go test fixture receives exact source blocks at the marked boundaries.
// It owns every server, pool and schema and runs in an ignored isolated module.
const infrastructureWorkflowTest = `package workflow
import (
 "bytes"
 "context"
 "database/sql"
 "errors"
 "fmt"
 goimage "image"
 "image/png"
 "io"
 "net"
 "net/textproto"
 "os"
 "strconv"
 "strings"
 "sync/atomic"
 "testing"
 "time"
 "github.com/go-chi/chi/v5"
 "hatmax.adrianpk.com/app"
 "hatmax.adrianpk.com/config"
 "hatmax.adrianpk.com/image/local"
 "hatmax.adrianpk.com/image/stdprocessor"
 hatlog "hatmax.adrianpk.com/log"
 "hatmax.adrianpk.com/mailer"
 "hatmax.adrianpk.com/pubsub"
 postgres "hatmax.adrianpk.com/pubsub/postgres"
 "hatmax.adrianpk.com/scheduler"
 schedulerpostgres "hatmax.adrianpk.com/scheduler/postgres"
 "hatmax.adrianpk.com/testhelper"
)

func publishedMail(ctx context.Context,cfg *config.Config,logger hatlog.Logger) error {
 __MAIL_CREATE__
 __MAIL_SEND__
 return nil
}

func captureSMTP(t *testing.T)(string,int,<-chan string){
 t.Helper()
 listener,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{t.Fatal(err)}
 captured:=make(chan string,1);done:=make(chan struct{})
 t.Cleanup(func(){listener.Close();select{case <-done:case <-time.After(6*time.Second):t.Error("owned SMTP did not stop")}})
 go func(){defer close(done)
  connection,err:=listener.Accept();if err!=nil{return};defer connection.Close()
  connection.SetDeadline(time.Now().Add(5*time.Second))
  peer:=textproto.NewConn(connection);defer peer.Close();peer.PrintfLine("220 localhost fixture")
  for {line,err:=peer.ReadLine();if err!=nil{return}
   switch {
    case strings.HasPrefix(line,"EHLO"),strings.HasPrefix(line,"HELO"):peer.PrintfLine("250 localhost")
    case strings.HasPrefix(line,"MAIL FROM:"),strings.HasPrefix(line,"RCPT TO:"):peer.PrintfLine("250 accepted")
    case line=="DATA":peer.PrintfLine("354 send data");body,err:=io.ReadAll(peer.DotReader());if err!=nil{return};captured<-string(body);peer.PrintfLine("250 queued")
    case line=="QUIT":peer.PrintfLine("221 closing");return
    default:peer.PrintfLine("500 unsupported fixture command")
   }
  }
 }()
 host,port,err:=net.SplitHostPort(listener.Addr().String());if err!=nil{t.Fatal(err)}
 number,err:=strconv.Atoi(port);if err!=nil{t.Fatal(err)};return host,number,captured
}

// Execute the exact constructor/send with loaded YAML and local transport only.
func TestMail(t *testing.T){
 t.Setenv("SMTP_PASSWORD","fixture-only")
 cfg,err:=config.Load("../mailer.yaml","DOC_MAIL",[]string{"docfixture"});if err!=nil{t.Fatal(err)}
 if cfg.Mailer.SMTP.Password!="fixture-only"||cfg.Mailer.Mode!=mailer.ModeDryRun||!cfg.Mailer.SMTP.StartTLS{t.Fatal("published mail config differs")}
 logger:=hatlog.NewTestLogger("info")
 if _,ok:=mailer.New(cfg,logger).(*mailer.NoopMailer);!ok{t.Fatal("dry run selected transport")}
 if err:=publishedMail(context.Background(),cfg,logger);err!=nil{t.Fatal(err)}
 original:=mailer.New(cfg,logger)
 host,port,captured:=captureSMTP(t)
 cfg.Mailer.Mode=mailer.ModeActive
 cfg.Mailer.SMTP.Host,cfg.Mailer.SMTP.Port=host,port
 // This owned sink advertises neither TLS nor authentication; the how-to
 // explicitly permits selecting its advertised capabilities for local capture.
 cfg.Mailer.SMTP.Username,cfg.Mailer.SMTP.Password="",""
 cfg.Mailer.SMTP.TLS,cfg.Mailer.SMTP.StartTLS=false,false
 if _,ok:=original.(*mailer.NoopMailer);!ok{t.Fatal("existing mailer changed after config mutation")}
 if _,ok:=mailer.New(cfg,logger).(*mailer.SMTPMailer);!ok{t.Fatal("active SMTP was not selected")}
 ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second);defer cancel()
 if err:=publishedMail(ctx,cfg,logger);err!=nil{t.Fatal(err)}
 select{case body:=<-captured:
  for _,part:=range []string{"noreply@example.com","reader@example.com","Subject: Welcome","Welcome to Example."}{if !strings.Contains(body,part){t.Fatalf("capture missing %s",part)}}
 case<-ctx.Done():t.Fatal("missing captured message")}
 canceled,stop:=context.WithCancel(context.Background());stop()
 if err:=publishedMail(canceled,cfg,logger);!errors.Is(err,context.Canceled){t.Fatal("canceled SMTP accepted")}
 cfg.Mailer.Mode="unknown"
 if _,ok:=mailer.New(cfg,logger).(*mailer.NoopMailer);!ok{t.Fatal("unknown mode selected transport")}
 invalid:=&mailer.Message{To:[]mailer.Address{{Email:"reader@example.com"}},Subject:"Welcome",Text:"Hello"}
 if err:=mailer.NewNoopMailer(logger).Send(context.Background(),invalid);err==nil{t.Fatal("no-op accepted missing sender")}
}

func publishedImage(ctx context.Context,source io.Reader,contentType string)error{
 __IMAGE_STORE__
 __IMAGE_RESIZE__
 reader,err:=store.Get(ctx,path);if err!=nil{return err};defer reader.Close()
 width,height,err:=processor.GetDimensions(ctx,reader,result.ContentType);if err!=nil{return err}
 if width!=800||height!=400||path!="images/photo-medium.png"||store.URL(path)!="/uploads/images/photo-medium.png"{return errors.New("published PNG outcome differs")}
 if err:=store.Delete(ctx,path);err!=nil{return err}
 if _,err:=store.Get(ctx,path);!os.IsNotExist(err){return errors.New("variant was not deleted")}
 return nil
}

// Execute local selection and resize; cancellation fails before storage.
func TestImages(t *testing.T){
 previous,err:=os.Getwd();if err!=nil{t.Fatal(err)}
 directory:=t.TempDir();if err:=os.Chdir(directory);err!=nil{t.Fatal(err)};defer os.Chdir(previous)
 var input bytes.Buffer;if err:=png.Encode(&input,goimage.NewRGBA(goimage.Rect(0,0,1600,800)));err!=nil{t.Fatal(err)}
 if err:=publishedImage(context.Background(),bytes.NewReader(input.Bytes()),"image/png");err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithCancel(context.Background());cancel()
 if err:=publishedImage(ctx,bytes.NewReader(input.Bytes()),"image/png");!errors.Is(err,context.Canceled){t.Fatal("canceled processor accepted input")}
 store:=local.NewStore("var/uploads","/uploads")
 if err:=store.Put(ctx,"local.txt",strings.NewReader("fixture"));err!=nil{t.Fatal("local copy unexpectedly promises cancellation",err)}
 if err:=store.Delete(context.Background(),"local.txt");err!=nil{t.Fatal(err)}
 if err:=store.Put(context.Background(),"../outside",strings.NewReader("fixture"));err==nil{t.Fatal("escaping local path accepted")}
}

type databaseProvider struct{connection *sql.DB;started bool}
func (database *databaseProvider)GetDB()*sql.DB{return database.connection}
func (database *databaseProvider)Start(ctx context.Context)error{database.started=true;return database.connection.PingContext(ctx)}
type eventSubscriber struct{broker pubsub.Subscriber;handle pubsub.Handler}
func (subscriber *eventSubscriber)Start(ctx context.Context)error{
 broker,handleOrder:=subscriber.broker,subscriber.handle
 __BROKER_SUBSCRIBE__
 return nil
}
func publishedBroker(ctx context.Context,database *databaseProvider,cfg *config.Config,logger hatlog.Logger,subscriber *eventSubscriber)(*postgres.Broker,error){
 router:=chi.NewRouter()
 __BROKER_CREATE__
 subscriber.broker=broker
 if len(starts)!=3||len(stops)!=1{return nil,errors.New("published lifecycle order differs")}
 if err:=app.Start(ctx,logger,starts,stops,registrars,router);err!=nil{return nil,err}
 return broker,nil
}
func publishedEvent(ctx context.Context,broker pubsub.Publisher,order map[string]string)error{
 __BROKER_PUBLISH__
 return nil
}
type AuditService struct{subscriber pubsub.Subscriber;received chan pubsub.Envelope}
func(service *AuditService)handle(ctx context.Context,event pubsub.Envelope)error{select{case service.received<-event:return nil;case <-ctx.Done():return ctx.Err()}}
__AUDIT_START__
func publishedInvoice(ctx context.Context,publisher pubsub.Publisher)error{
 service:=struct{publisher pubsub.Publisher}{publisher};invoice:=struct{ID string}{"invoice-1"}
 __INVOICE_PUBLISH__
 return nil
}
func waitFor(t *testing.T,description string,ready func()bool){
 t.Helper();deadline:=time.Now().Add(4*time.Second)
 for time.Now().Before(deadline){if ready(){return};time.Sleep(10*time.Millisecond)}
 t.Fatal(description)
}
func acknowledged(t *testing.T,database *sql.DB,id string)bool{
 t.Helper()
 var count int;err:=database.QueryRow("SELECT COUNT(*) FROM pubsub_acknowledgements a JOIN pubsub_messages m ON m.id=a.message_id WHERE a.subscriber_id='billing-orders' AND convert_from(m.payload,'UTF8')::jsonb->>'id'=$1",id).Scan(&count)
 if err!=nil{t.Fatal(err)}
 return count==1
}

// Real named subscriptions retain only failed delivery across an owned restart.
func TestEvents(t *testing.T){
 connection,_,cleanup:=testhelper.SetupTestDB(t);t.Cleanup(cleanup)
 database:=&databaseProvider{connection:connection};cfg:=config.New();cfg.PubSub.PollInterval="20ms";cfg.PubSub.BatchSize=10
 logger:=hatlog.NewNoopLogger();ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
 var failed,healthy atomic.Int64
 subscriber:=&eventSubscriber{handle:func(ctx context.Context,event pubsub.Envelope)error{
  object,ok:=event.Payload.(map[string]any);if !ok{return errors.New("payload is not decoded JSON")}
  if object["id"]=="failed"{failed.Add(1);return errors.New("fixture transient failure")};healthy.Add(1);return nil
 }}
 broker,err:=publishedBroker(ctx,database,cfg,logger,subscriber);if err!=nil{t.Fatal(err)};defer broker.Close()
 if !database.started{t.Fatal("database did not start before broker")}
 for _,id:=range []string{"failed","healthy"}{if err:=publishedEvent(ctx,broker,map[string]string{"id":id});err!=nil{t.Fatal(err)}}
 waitFor(t,"initial fan-out was not persisted",func()bool{return failed.Load()>0&&healthy.Load()>0&&acknowledged(t,connection,"healthy")})
 if err:=broker.Close();err!=nil{t.Fatal(err)}
 if acknowledged(t,connection,"failed"){t.Fatal("failed delivery acknowledged")}
 if healthy.Load()!=1{t.Fatal("healthy delivery repeated")}
 var resumed,badReplay atomic.Int64
 restarted:=&eventSubscriber{handle:func(ctx context.Context,event pubsub.Envelope)error{
  object,ok:=event.Payload.(map[string]any);if !ok{return errors.New("resume payload is not JSON")};if object["id"]!="failed"{badReplay.Add(1)};resumed.Add(1);return nil
 }}
 broker,err=publishedBroker(ctx,database,cfg,logger,restarted);if err!=nil{t.Fatal(err)};defer broker.Close()
 waitFor(t,"failed delivery did not resume",func()bool{return acknowledged(t,connection,"failed")})
 if resumed.Load()!=1||badReplay.Load()!=0{t.Fatal("named restart replayed successful delivery")}
 audit:=&AuditService{subscriber:broker,received:make(chan pubsub.Envelope,1)}
 if err:=audit.Start(ctx);err!=nil{t.Fatal(err)}
 if err:=publishedInvoice(ctx,broker);err!=nil{t.Fatal(err)}
 select{case envelope:=<-audit.received:if envelope.Payload!="invoice-1"||envelope.Topic!="invoices"{t.Fatal("invoice envelope differs")};case<-ctx.Done():t.Fatal("invoice audit delivery missing")}
 if err:=broker.Close();err!=nil{t.Fatal(err)}
 if err:=publishedEvent(ctx,broker,map[string]string{"id":"closed"});err==nil{t.Fatal("closed broker accepted publication")}
}

func publishedRunner(ctx context.Context,database *databaseProvider,cfg *config.Config,logger hatlog.Logger)*scheduler.Runner{
 var settingsService scheduler.SettingsProvider
 __JOB_CREATE__
 __JOB_HANDLER__
 return runner
}
func publishedSchema(ctx context.Context,database *sql.DB)error{
 __STORE_SCHEMA__
 _=store
 return nil
}
func publishedStop(runner *scheduler.Runner,logger hatlog.Logger)error{
 __JOB_STOP__
 return err
}
func tick(t *testing.T,runner *scheduler.Runner){t.Helper();if err:=runner.Tick(context.Background());err!=nil{t.Fatal(err)}}
func insertJob(t *testing.T,database *sql.DB,id,task string,due time.Time){
 t.Helper();if _,err:=database.Exec("INSERT INTO scheduled_jobs(id,name,task_type,next_run_at) VALUES($1,$1,$2,$3)",id,task,due);err!=nil{t.Fatal(err)}
}

// Inspect actual slot, recurrence and durable retries through the published runner.
func TestJobs(t *testing.T){
 connection,_,cleanup:=testhelper.SetupTestDB(t);t.Cleanup(cleanup);ctx:=context.Background();database:=&databaseProvider{connection:connection}
 if err:=publishedSchema(ctx,connection);err!=nil{t.Fatal(err)}
 cfg,err:=config.Load("../scheduler.yaml","DOC_JOBS",[]string{"docfixture"});if err!=nil{t.Fatal(err)}
 if _,err:=connection.Exec(__JOB_SQL__);err!=nil{t.Fatal(err)}
 var due time.Time;if err:=connection.QueryRow("SELECT next_run_at FROM scheduled_jobs WHERE id='daily-report'").Scan(&due);err!=nil{t.Fatal(err)}
 clock:=scheduler.NewFakeClock(due);runner:=publishedRunner(ctx,database,cfg,hatlog.NewNoopLogger());runner.SetClock(clock)
 tick(t,runner);tick(t,runner)
 var status string;var sent bool
 if err:=connection.QueryRow("SELECT status,(output->>'sent')::boolean FROM job_runs WHERE job_id='daily-report'").Scan(&status,&sent);err!=nil||status!="success"||!sent{t.Fatal("published report output missing",err)}
 var next time.Time;var count int
 if err:=connection.QueryRow("SELECT next_run_at FROM scheduled_jobs WHERE id='daily-report'").Scan(&next);err!=nil{t.Fatal(err)}
 if !next.Equal((scheduler.Daily{Hour:9}).Next(due)){t.Fatal("recurrence did not advance from completion")}
 if err:=connection.QueryRow("SELECT COUNT(*) FROM job_runs WHERE job_id='daily-report'").Scan(&count);err!=nil||count!=1{t.Fatal("consumed slot repeated",err)}
 clock.Set(next);tick(t,runner)
 if err:=connection.QueryRow("SELECT COUNT(*) FROM job_runs WHERE job_id='daily-report'").Scan(&count);err!=nil||count!=2{t.Fatal("second distinct slot missing",err)}
 insertJob(t,connection,"retry","retry",clock.Now());insertJob(t,connection,"panic","panic",clock.Now());insertJob(t,connection,"unknown","missing",clock.Now())
 var attempts []scheduler.Job
 runner.Register("retry",func(ctx context.Context,job scheduler.Job)scheduler.Result{attempts=append(attempts,job);return scheduler.Result{Err:errors.New("fixture failure")}})
 runner.Register("panic",func(context.Context,scheduler.Job)scheduler.Result{panic("fixture panic")})
 tick(t,runner)
 var runID string;var retryAt time.Time;var limit int
 if err:=connection.QueryRow("SELECT id,status,retry_at,retry_limit FROM job_runs WHERE job_id='retry'").Scan(&runID,&status,&retryAt,&limit);err!=nil||status!="retry_wait"||limit!=3{t.Fatal("retry wait/budget missing",err)}
 if len(attempts)!=1||attempts[0].Attempt!=1||attempts[0].RunID!=runID{t.Fatal("initial attempt identity differs")}
 clock.Set(retryAt.Add(-time.Millisecond));tick(t,runner);if len(attempts)!=1{t.Fatal("retry ran early")}
 changed:=*cfg;changed.Scheduler.RetryAttempts=9
 runner=publishedRunner(ctx,database,&changed,hatlog.NewNoopLogger());runner.SetClock(clock)
 runner.Register("retry",func(ctx context.Context,job scheduler.Job)scheduler.Result{attempts=append(attempts,job);return scheduler.Result{Err:errors.New("fixture failure")}})
 runner.Register("panic",func(context.Context,scheduler.Job)scheduler.Result{panic("fixture panic")})
 for attempt:=2;attempt<=3;attempt++{
  clock.Set(retryAt);tick(t,runner)
  if attempts[len(attempts)-1].Attempt!=attempt||attempts[len(attempts)-1].MaxAttempts!=3||attempts[len(attempts)-1].RunID!=runID{t.Fatal("restart reset retry identity/budget")}
  if attempt<3{if err:=connection.QueryRow("SELECT retry_at FROM job_runs WHERE id=$1",runID).Scan(&retryAt);err!=nil{t.Fatal(err)}}
 }
 tick(t,runner)
 if len(attempts)!=3{t.Fatal("exhausted one-shot repeated")}
 for _,id:=range []string{"retry","panic","unknown"}{var enabled bool;var detail string
  if err:=connection.QueryRow("SELECT r.status,j.enabled,COALESCE(r.error,'') FROM job_runs r JOIN scheduled_jobs j ON j.id=r.job_id WHERE j.id=$1",id).Scan(&status,&enabled,&detail);err!=nil||status!="failed"||enabled{t.Fatal("terminal failure did not retire slot",id,err)}
  if id=="panic"&&!strings.Contains(detail,"handler panic:"){t.Fatal("panic failure missing")}
 }
 // Success after one failed attempt keeps the same slot and retires once.
 insertJob(t,connection,"recover","recover",clock.Now());recoveries:=0
 runner.Register("recover",func(context.Context,scheduler.Job)scheduler.Result{recoveries++;if recoveries==1{return scheduler.Result{Err:errors.New("once")}};return scheduler.Result{Output:map[string]any{"sent":true}}})
 tick(t,runner);if err:=connection.QueryRow("SELECT retry_at FROM job_runs WHERE job_id='recover'").Scan(&retryAt);err!=nil{t.Fatal(err)}
 clock.Set(retryAt);tick(t,runner);tick(t,runner)
 if recoveries!=2{t.Fatal("recovered job repeated")}
}

// Stop's fresh context joins admitted cooperative work before the database closes.
func TestStop(t *testing.T){
 connection,_,cleanup:=testhelper.SetupTestDB(t);t.Cleanup(cleanup)
 if err:=publishedSchema(context.Background(),connection);err!=nil{t.Fatal(err)}
 cfg,err:=config.Load("../scheduler.yaml","DOC_STOP",[]string{"docfixture"});if err!=nil{t.Fatal(err)}
 runner:=publishedRunner(context.Background(),&databaseProvider{connection:connection},cfg,hatlog.NewNoopLogger())
 entered:=make(chan struct{});exited:=make(chan struct{})
 runner.Register("block",func(ctx context.Context,job scheduler.Job)scheduler.Result{close(entered);<-ctx.Done();close(exited);return scheduler.Result{Err:ctx.Err()}})
 insertJob(t,connection,"block","block",time.Now().Add(-time.Second))
 ctx,cancel:=context.WithCancel(context.Background());defer cancel();defer runner.Stop(context.Background())
 if err:=runner.Start(ctx);err!=nil{t.Fatal(err)}
 select{case<-entered:case<-time.After(3*time.Second):t.Fatal("handler was not admitted")}
 if err:=publishedStop(runner,hatlog.NewNoopLogger());err!=nil{t.Fatal(err)}
 select{case<-exited:default:t.Fatal("stop returned before handler exit")}
 if err:=publishedStop(runner,hatlog.NewNoopLogger());err!=nil{t.Fatal("repeat stop",err)}
 if err:=runner.Start(context.Background());!errors.Is(err,scheduler.ErrStopped){t.Fatal("stopped runner restarted")}
}

func calendar()error{
 __CALENDAR__
 from:=time.Date(2026,1,2,9,0,0,0,time.UTC)
 if !daily.Next(from).Equal(from.Add(24*time.Hour))||weekly.Next(from).Location()!=time.UTC||!weekly.Next(from).After(from)||!interval.Next(from).Equal(from.Add(30*time.Minute))||next.Location()!=time.UTC{return errors.New("schedule arithmetic differs")}
 return nil
}
// The exact fake-store fragment must retire its one-shot; calendar rules use UTC.
func TestClock(t *testing.T){
 ctx:=context.Background();baseTime:=time.Date(2026,1,1,0,0,0,0,time.UTC)
 __FAKE_TICK__
 runs:=store.AllRuns();if len(runs)!=1||runs[0].Status!="success"||!bytes.Contains(runs[0].Output,[]byte("true")){t.Fatal("fake job output missing")}
 if err:=sched.Tick(ctx);err!=nil||len(store.AllRuns())!=1{t.Fatal("fake one-shot repeated")}
 if err:=calendar();err!=nil{t.Fatal(err)}
}

// Real config mapping and helper-created consumer pools must preserve ownership.
func TestConfiguration(t *testing.T){
 root,err:=config.Load("../scheduler.yaml","DOC_CONFIG",[]string{"docfixture"});if err!=nil{t.Fatal(err)}
 got:=scheduler.ConfigFromRoot(root)
 if !got.Enabled||got.Interval!=30*time.Second||got.BatchSize!=20||got.Workers!=2||got.RetryAttempts!=3||got.RetryBackoff!=time.Minute{t.Fatal("published scheduler config differs")}
 defaults:=scheduler.Config{}.WithDefaults()
 if defaults.Enabled||defaults.Interval!=time.Minute||defaults.Workers!=1||defaults.RetryAttempts!=3||defaults.RetryBackoff!=time.Minute{t.Fatal("scheduler defaults differ")}
 cfg,cleanup:=testhelper.SetupTestDBWithConfig(t)
 dsn:=fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable search_path=%s",cfg.Database.Host,cfg.Database.Port,cfg.Database.User,cfg.Database.Password,cfg.Database.Database,cfg.Database.Schema)
 connection,err:=sql.Open("pgx",dsn);if err!=nil{cleanup();t.Fatal(err)}
 t.Cleanup(func(){connection.Close();cleanup()})
 var selected string;if err:=connection.QueryRow("SELECT current_schema()").Scan(&selected);err!=nil||selected!=cfg.Database.Schema{t.Fatal("WithConfig schema differs",err)}
}
`
