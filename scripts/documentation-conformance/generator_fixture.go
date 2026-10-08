// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

// The interpreter supplies bounded fixture decisions. Production coordinators,
// persistence, planners, renderers and project commands retain their real owners.
const generatorWorkflowTest = `package fixture
import (
 "bytes"
 "context"
 "crypto/sha256"
 "encoding/json"
 "fmt"
 "io/fs"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "syscall"
 "testing"
 "time"
 "golang.org/x/sys/unix"
 "hatmax.adrianpk.com/generator/conversation"
 "hatmax.adrianpk.com/generator/eval"
 "hatmax.adrianpk.com/generator/intent"
 "hatmax.adrianpk.com/generator/interaction"
 "hatmax.adrianpk.com/generator/project"
 "hatmax.adrianpk.com/internal/hatmaxcli"
 "hatmax.adrianpk.com/internal/hatmaxstate"
)

const detailed = __DETAILED__
const terse = __TERSE__
const addField = __ADD_FIELD__
const createFeature = __CREATE_FEATURE__
const install = __INSTALL__
const open = __OPEN__
const reopen = __REOPEN__
const selectConversation = __SELECT__
const resetConversation = __RESET__
const playground = __PLAYGROUND__

type interpreter struct{}
func(interpreter)Interpret(_ context.Context,request eval.Request)(eval.InterpreterResult,error){
 result:=eval.InterpreterResult{Provenance:eval.Provenance{Adapter:"fixture",ContractVersion:eval.CurrentContractVersion,ModelSelection:eval.ModelNotApplicable,Timing:eval.TimingNotMeasured}}
 output:=eval.Interpretation{SchemaVersion:eval.CurrentInterpretationSchemaVersion,Kind:eval.InterpretationIntent}
 if request.Prompt=="What is this application?" {output.Kind=eval.InterpretationConversation;output.Response=&eval.ConversationResponse{Content:"We can discuss an application before proposing a change."};result.Interpretation=output;return result,nil}
 if request.Target!=nil{
  value:=intent.Intent{SchemaVersion:intent.ApplicationSchemaVersion,Operation:intent.OperationCreateApplication,SourceFingerprint:request.Target.SourceFingerprint,HatmaxVersion:request.Target.HatmaxVersion,BookVersion:request.Book.Version,Archetype:"server_rendered_hatmax_application",Capabilities:[]string{},Documentation:intent.DocumentationNotRequested,Application:&intent.ApplicationIdentity{DisplayName:"Ledger",ModulePath:"example.com/alex/ledger"},Target:&intent.ApplicationTarget{Base:"session_directory"}}
  switch request.Prompt{
  case detailed:value.InitialFeatures=[]intent.InitialFeature{{Feature:"invoice",Domain:intent.Domain{Fields:[]intent.Field{{Name:"number",Type:"string",Required:true},{Name:"notes",Type:"text"}}}}}
  case terse:if !request.Target.Resolved{value.Application.ModulePath=""}else if len(request.Clarifications)>0{value.Application.ModulePath=request.Clarifications[0].Answer}
  default:return result,fmt.Errorf("unknown target fixture prompt %q",request.Prompt)
  }
  output.Intent=&value
 }else{
  value:=intent.Intent{SchemaVersion:intent.CurrentSchemaVersion,ProjectFingerprint:request.Project.Fingerprint,HatmaxVersion:request.Project.HatmaxVersion,BookVersion:request.Book.Version,Archetype:"server_rendered_crud",Feature:"invoice",Capabilities:[]string{"postgres_persistence","runtime_validation","htmx_form"},Documentation:intent.DocumentationNotRequested}
  switch request.Prompt{
  case createFeature:value.Operation=intent.OperationCreateFeature;value.Domain.Fields=[]intent.Field{{Name:"number",Type:"string",Required:true}}
  case addField:value.Operation=intent.OperationAddField;value.Domain.Field=&intent.Field{Name:"issued_at",Type:"timestamp",Required:true}
  case "Validate invoice numbers.","Require four characters.":value.Operation=intent.OperationAddValidation;value.Domain.Validation=&intent.ValidationRule{Field:"number",Kind:"min_length",Value:"3",Scope:intent.ValidationDurable,Message:"Invoice number is too short."};if request.Prompt=="Require four characters."{value.Domain.Validation.Value="4"}
  case "Document every invoice reader need.":
   value.Operation=intent.OperationDocumentFeature;value.Documentation=intent.DocumentationExisting
   value.DocumentationTargets=[]intent.DocumentationTarget{{Quadrant:intent.DocumentationTutorial,Subject:"invoice_basics",ReaderGoal:"Learn invoice creation."},{Quadrant:intent.DocumentationHowTo,Subject:"invoice_workflow",ReaderGoal:"Create an invoice."},{Quadrant:intent.DocumentationReference,Subject:"invoice",ReaderGoal:"Find invoice contracts."},{Quadrant:intent.DocumentationExplanation,Subject:"invoice_ownership",ReaderGoal:"Understand invoice ownership."}}
  case "Use SQLite.":value.Operation=intent.OperationCreateFeature;value.Feature="other";value.Domain.Fields=[]intent.Field{{Name:"number",Type:"string"}};value.Capabilities=[]string{"sqlite"}
  default:return result,fmt.Errorf("unknown project fixture prompt %q",request.Prompt)
  }
  output.Intent=&value
 }
 result.Interpretation=output
 return result,nil
}

type approver struct{}
func(approver)Approve(context.Context,interaction.ApprovalRequest)(interaction.ApprovalDecision,error){return interaction.ApprovalRejected,nil}
func must(t *testing.T,err error){t.Helper();if err!=nil{t.Fatal(err)}}
func outcome(t *testing.T,result conversation.SessionResult,expected interaction.Outcome){t.Helper();if result.Interaction.Outcome!=expected{encoded,_:=json.Marshal(result.Interaction);t.Fatalf("want %s: %s",expected,encoded)}}
func turn(t *testing.T,session *conversation.ActiveSession,prompt string)conversation.SessionResult{t.Helper();result,err:=session.Turn(t.Context(),conversation.TurnRequest{Content:prompt});must(t,err);return result}
func approve(t *testing.T,session *conversation.ActiveSession,proposal conversation.SessionResult)conversation.SessionResult{t.Helper();result,err:=session.Approve(t.Context(),proposal.OperationID,proposal.Interaction.Plan.Digest);must(t,err);return result}
func snapshot(t *testing.T,root string,implementationOnly bool)map[string]string{
 t.Helper();result:=map[string]string{}
 must(t,filepath.WalkDir(root,func(file string,entry fs.DirEntry,err error)error{
  if err!=nil{return err};if entry.IsDir(){return nil};relative,err:=filepath.Rel(root,file);if err!=nil{return err};if implementationOnly&&(strings.HasPrefix(relative,"docs/")||relative=="README.md"){return nil};data,err:=os.ReadFile(file);if err!=nil{return err};result[relative]=fmt.Sprintf("%x",sha256.Sum256(data));return nil
 }));return result
}
func same(t *testing.T,before,after map[string]string){t.Helper();a,_:=json.Marshal(before);b,_:=json.Marshal(after);if !bytes.Equal(a,b){t.Fatal("unauthorized source mutation")}}
func read(t *testing.T,root,file string)string{t.Helper();data,err:=os.ReadFile(filepath.Join(root,file));must(t,err);return string(data)}
func run(t *testing.T,root,name string,args ...string)string{t.Helper();command:=exec.CommandContext(t.Context(),name,args...);command.Dir=root;output,err:=command.CombinedOutput();if err!=nil{t.Fatalf("%s %v: %v\n%s",name,args,err,output)};return string(output)}

// A real conversational kernel publishes, rebinds, resumes and evolves source;
// fixture interpretation establishes no production model acceptance.
func TestBuilder(t *testing.T){
 parent:=t.TempDir();store,err:=hatmaxstate.New(hatmaxstate.Config{Root:filepath.Join(t.TempDir(),"state")});must(t,err)
 engine,err:=interaction.New(interaction.Config{Interpreter:interpreter{},Approver:approver{}});must(t,err)
 coordinator,err:=conversation.NewCoordinator(conversation.CoordinatorConfig{Store:store,Engine:engine,BookContract:conversation.BookContract{BookVersion:1,InterpreterVersion:eval.CurrentContractVersion},ApplicationBookContract:conversation.BookContract{BookVersion:2,InterpreterVersion:eval.CurrentContractVersion},BackendIdentity:conversation.BackendIdentity{Adapter:"fixture"}});must(t,err)
 session,err:=coordinator.Open(t.Context(),parent,conversation.SessionOptions{});must(t,err);defer session.Close()
 before:=snapshot(t,parent,false);dialogue:=turn(t,session,"What is this application?");outcome(t,dialogue,interaction.OutcomeConversationResponse);same(t,before,snapshot(t,parent,false))
 proposal:=turn(t,session,detailed);outcome(t,proposal,interaction.OutcomePlanReady);same(t,before,snapshot(t,parent,false))
 _,err=session.Cancel(t.Context(),proposal.OperationID);must(t,err);same(t,before,snapshot(t,parent,false))
 proposal=turn(t,session,detailed);outcome(t,approve(t,session,proposal),interaction.OutcomeCompleted)
 root:=filepath.Join(parent,"ledger");identity:=session.Current().ID
 t.Setenv("XDG_STATE_HOME",t.TempDir());t.Setenv("XDG_CACHE_HOME",t.TempDir());terminal(t,parent,reopen)
 if session.Current().Scope.Kind!=conversation.ScopeProject{t.Fatal("bootstrap scope did not rebind")};must(t,session.Close())
 session,err=coordinator.Open(t.Context(),root,conversation.SessionOptions{});must(t,err);defer session.Close()
 if session.Current().ID!=identity{t.Fatal("created project lost conversation")}
 before=snapshot(t,root,false);proposal=turn(t,session,addField);outcome(t,proposal,interaction.OutcomePlanReady);same(t,before,snapshot(t,root,false))
 blocked:=approve(t,session,proposal);outcome(t,blocked,interaction.OutcomeExecutionFailed)
 found:=false;for _,diagnostic:=range blocked.Interaction.Diagnostics{if diagnostic.Code=="HMGEN-EXECUTION-LAYOUT-MISSING"{found=true}}
 if !found{t.Fatal("fresh-scaffold failure changed")};same(t,before,snapshot(t,root,false))
 t.Log("fresh-scaffold evolution blocked: HMGEN-EXECUTION-LAYOUT-MISSING; no mutation")
 run(t,root,"go","build","./...");run(t,root,"go","test","-count=1","./...")
 must(t,session.Close());root=existing(t)
 session,err=coordinator.Open(t.Context(),root,conversation.SessionOptions{});must(t,err);defer session.Close()
 proposal=turn(t,session,createFeature);outcome(t,proposal,interaction.OutcomePlanReady);outcome(t,approve(t,session,proposal),interaction.OutcomeCompleted)
 proposal=turn(t,session,addField);outcome(t,proposal,interaction.OutcomePlanReady);timestampBlocked(t,approve(t,session,proposal))
 if !strings.Contains(strings.Join(strings.Fields(read(t,root,"internal/feat/invoice/model.go"))," "),"IssuedAt time.Time"){t.Fatal("required timestamp not retained after failed check")}
 must(t,session.Close());root=existing(t)
 session,err=coordinator.Open(t.Context(),root,conversation.SessionOptions{});must(t,err);defer session.Close()
 proposal=turn(t,session,createFeature);outcome(t,approve(t,session,proposal),interaction.OutcomeCompleted)
 proposal=turn(t,session,"Validate invoice numbers.");validationBlocked(t,approve(t,session,proposal))
 must(t,session.Close());root=existing(t)
 session,err=coordinator.Open(t.Context(),root,conversation.SessionOptions{});must(t,err);defer session.Close()
 proposal=turn(t,session,createFeature);outcome(t,approve(t,session,proposal),interaction.OutcomeCompleted)
 before=snapshot(t,root,true)
 proposal=turn(t,session,"Document every invoice reader need.");outcome(t,approve(t,session,proposal),interaction.OutcomeCompleted);same(t,before,snapshot(t,root,true))
 inventory,err:=project.Inspect(t.Context(),root);must(t,err)
 for _,file:=range inventory.Documentation.Files{for _,link:=range file.LocalLinks{if !link.Exists{t.Fatal("generated navigation target missing",link.Target)}}}
 file:=filepath.Join(root,"docs/reference/invoice/README.md");original,err:=os.ReadFile(file);must(t,err)
 must(t,os.WriteFile(file,append(append([]byte("User prefix.\n"),original...),[]byte("\nUser suffix.\n")...),0600))
 proposal=turn(t,session,"Document every invoice reader need.");outcome(t,approve(t,session,proposal),interaction.OutcomeCompleted)
 page:=read(t,root,"docs/reference/invoice/README.md");if !strings.HasPrefix(page,"User prefix.\n")||!strings.HasSuffix(page,"\nUser suffix.\n")||!strings.Contains(page,"<!-- hatmax:generated:start -->"){t.Fatal("managed regeneration overwrote user text")}
 proposal=turn(t,session,"Require four characters.");outcome(t,proposal,interaction.OutcomePlanReady)
 must(t,os.WriteFile(filepath.Join(root,"main.go"),[]byte(read(t,root,"main.go")+"\n// fixture source drift\n"),0600))
 before=snapshot(t,root,false);outcome(t,approve(t,session,proposal),interaction.OutcomePlanStale);same(t,before,snapshot(t,root,false))
 _,err=session.Reset(t.Context());must(t,err);same(t,before,snapshot(t,root,false))
 denied:=turn(t,session,"Use SQLite.");if denied.Interaction.Outcome==interaction.OutcomePlanReady||denied.Interaction.Outcome==interaction.OutcomeCompleted{t.Fatal("unsupported capability admitted")};same(t,before,snapshot(t,root,false))
 run(t,root,"go","build","./...");run(t,root,"go","test","-count=1","./...")
}

// The real line-oriented command parses published prompts and obtains explicit
// terminal clarification/approval; generated checks use native executables.
func TestHeadless(t *testing.T){
 parent:=t.TempDir();var output,stderr bytes.Buffer
 app,err:=hatmaxcli.New(hatmaxcli.Config{Input:strings.NewReader("example.com/alex/ledger\nyes\n"),Output:&output,ErrorOutput:&stderr,WorkingDirectory:func()(string,error){return parent,nil},CommandName:"hm",CoordinatorFactory:func(_ string,a interaction.Approver,c interaction.Clarifier)(hatmaxcli.Runner,error){return interaction.New(interaction.Config{Interpreter:interpreter{},Approver:a,Clarifier:c})}});must(t,err)
 if status:=app.Run(t.Context(),[]string{"generate",terse});status!=0{t.Fatalf("terse request exit %d: %s\n%s",status,stderr.String(),output.String())}
 root:=filepath.Join(parent,"ledger")
 if strings.Contains(run(t,root,"go","list","-m","-json","hatmax.adrianpk.com"),"Replace"){t.Fatal("published bare scaffold replaced toolkit")}
 run(t,root,"go","build","./...");run(t,root,"go","test","-count=1","./...")
 root=existing(t)
 for _,request:=range []string{createFeature,addField}{
  output.Reset();stderr.Reset()
  app,err=hatmaxcli.New(hatmaxcli.Config{Input:strings.NewReader("yes\n"),Output:&output,ErrorOutput:&stderr,WorkingDirectory:func()(string,error){return root,nil},CommandName:"hm",CoordinatorFactory:func(_ string,a interaction.Approver,c interaction.Clarifier)(hatmaxcli.Runner,error){return interaction.New(interaction.Config{Interpreter:interpreter{},Approver:a,Clarifier:c})}});must(t,err)
  status:=app.Run(t.Context(),[]string{"generate",request})
  if request==addField{if status!=8||!strings.Contains(output.String(),"HMGEN-EXECUTION-COMMAND-FAILED"){t.Fatalf("timestamp blocked result changed: %d %s %s",status,stderr.String(),output.String())};return}
  if status!=0{t.Fatalf("published request exit %d: %s\n%s",status,stderr.String(),output.String())}
 }
 run(t,root,"go","build","./...");run(t,root,"go","test","-count=1","./...")
}

func timestampBlocked(t *testing.T,result conversation.SessionResult){
 t.Helper();outcome(t,result,interaction.OutcomeExecutionFailed)
 found:=false;if result.Interaction.Report!=nil{for _,command:=range result.Interaction.Report.Commands{if command.Name=="validation.check"&&command.ExitCode!=0&&strings.Contains(command.Output,"cannot use row")&&strings.Contains(command.Output,"as dal.Invoice"){found=true}}}
 if !found||len(result.Interaction.RetainedChanges)==0{t.Fatal("timestamp compiler failure or retained-change evidence missing")}
 t.Log("timestamp evolution blocked: SQLC row mapping compilation failed; changes retained")
}

func validationBlocked(t *testing.T,result conversation.SessionResult){
 t.Helper();outcome(t,result,interaction.OutcomeExecutionFailed)
 found:=false;if result.Interaction.Report!=nil{for _,command:=range result.Interaction.Report.Commands{if command.Name=="validation.check"&&command.ExitCode!=0&&strings.Contains(command.Output,"model_test.go")&&strings.Contains(command.Output,"wsl_v5"){found=true}}}
 if !found||len(result.Interaction.RetainedChanges)==0{t.Fatal("generated validation lint failure evidence missing")}
 t.Log("validation evolution blocked: generated model test fails wsl_v5; changes retained")
}
`

// Entry-point checks isolate local state and never request backend inference.
const generatorCommandTest = `
func TestCommands(t *testing.T){
 directory,err:=os.Getwd();must(t,err);root:=t.TempDir()
 t.Setenv("XDG_STATE_HOME",t.TempDir());t.Setenv("XDG_CACHE_HOME",t.TempDir())
 t.Setenv("GOBIN",directory);run(t,__SOURCE_ROOT__,"bash","-eu","-c",install)
 before:=snapshot(t,root,false)
 for _,name:=range []string{"hm","hatmax"}{
  command:=exec.CommandContext(t.Context(),filepath.Join(directory,name),"--help");command.Dir=root
  output,err:=command.CombinedOutput();failure,ok:=err.(*exec.ExitError)
  if !ok||failure.ExitCode()!=2||!strings.Contains(string(output),name+" conversation resume <conversation-id>"){t.Fatalf("usage %s: %v %s",name,err,output)}
 }
 hm:=filepath.Join(directory,"hm")
 terminal(t,root,open)
 created:=run(t,root,hm,"conversation","new")
 id:=strings.TrimSpace(strings.Split(strings.TrimPrefix(created,"Conversation: "),"\n")[0])
 if id==""||!strings.Contains(run(t,root,hm,"conversation","list"),id){t.Fatal("created conversation missing")}
 if !strings.Contains(run(t,root,hm,"conversation","resume",id),id){t.Fatal("conversation did not resume")}
 terminal(t,root,strings.ReplaceAll(selectConversation,"<conversation-id>",id))
 if run(t,root,hm,"conversation","new")==created{t.Fatal("reset reused conversation")}
 if !strings.Contains(run(t,root,hm,"conversation","list"),id){t.Fatal("reset lost retained conversation")}
 terminal(t,root,resetConversation)
 same(t,before,snapshot(t,root,false))
 base:=t.TempDir();t.Setenv("HATMAX_PLAYGROUND_BASE",base);t.Setenv("HATMAX_PLAYGROUND_NO_RUN","1")
 output:=run(t,__SOURCE_ROOT__,"bash","-eu","-c",playground)
 children,err:=os.ReadDir(base);must(t,err);if len(children)!=1||!strings.Contains(output,"Prepared without launching"){t.Fatal("playground preparation missing")}
 for _,name:=range []string{"hm","sqlc"}{info,err:=os.Stat(filepath.Join(base,children[0].Name(),".tmp/bin",name));must(t,err);if info.Mode()&0111==0{t.Fatal("playground executable missing",name)}}
}

// A native owned PTY exercises the real no-argument entrypoint, help and quit.
// No composer message is sent, so it cannot request model inference.
func terminal(t *testing.T,root,script string){
 terminalBound(t,root,script,10*time.Second)
}
func terminalBound(t *testing.T,root,script string,bound time.Duration){
 t.Helper();fd,err:=unix.Open("/dev/ptmx",unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC,0);must(t,err)
 master:=os.NewFile(uintptr(fd),"owned-terminal");defer master.Close()
 must(t,unix.IoctlSetPointerInt(fd,unix.TIOCSPTLCK,0));number,err:=unix.IoctlGetInt(fd,unix.TIOCGPTN);must(t,err)
 slave,err:=os.OpenFile(fmt.Sprintf("/dev/pts/%d",number),os.O_RDWR,0);must(t,err);defer slave.Close()
 must(t,unix.IoctlSetWinsize(fd,unix.TIOCSWINSZ,&unix.Winsize{Row:40,Col:120}))
 ctx,cancel:=context.WithTimeout(t.Context(),bound);defer cancel()
 directory,err:=os.Getwd();must(t,err)
 command:=exec.CommandContext(ctx,"bash","-eu","-c",script);command.Dir=root;command.Env=append(os.Environ(),"TERM=xterm-256color","PATH="+directory+":"+os.Getenv("PATH"))
 command.Stdin=slave;command.Stdout=slave;command.Stderr=slave;command.SysProcAttr=&syscall.SysProcAttr{Setsid:true,Setctty:true,Ctty:0}
 command.Cancel=func()error{return syscall.Kill(-command.Process.Pid,syscall.SIGKILL)};command.WaitDelay=2*time.Second
 must(t,command.Start());defer func(){cancel();_ = command.Wait()}()
 chunks:=make(chan string,128)
 go func(){defer close(chunks);buffer:=make([]byte,8192);for{n,err:=master.Read(buffer);if n>0{select{case chunks<-string(buffer[:n]):case <-ctx.Done():return}};if err!=nil{return}}}()
 var captured strings.Builder
 wait:=func(needle string){t.Helper();for !strings.Contains(captured.String(),needle){select{case chunk,ok:=<-chunks:if !ok{t.Fatalf("terminal ended before %s: %s",needle,captured.String())};captured.WriteString(chunk);if captured.Len()>1<<20{t.Fatal("terminal output exceeded bound")};case <-ctx.Done():t.Fatalf("terminal timed out before %s: %s",needle,captured.String())}}}
 wait("F1 help");_,err=master.Write([]byte{8});must(t,err);wait("Ctrl+N new conversation")
 _,err=master.Write([]byte{3});must(t,err);must(t,command.Wait())
}
`

// The existing-project context matches the repository's real CLI acceptance
// fixture. Its module replacement is explicit; standalone scaffold checks above
// retain the published dependency without a replacement.
const generatorExistingTest = `
func existing(t *testing.T)string{
 t.Helper();root:=t.TempDir();source:=filepath.Join(__SOURCE_ROOT__,"generator/project/testdata/supported")
 t.Setenv("GOLANGCI_LINT_CACHE",t.TempDir())
 must(t,filepath.WalkDir(source,func(file string,entry fs.DirEntry,err error)error{
  if err!=nil{return err};relative,err:=filepath.Rel(source,file);if err!=nil{return err};target:=filepath.Join(root,relative);if entry.IsDir(){return os.MkdirAll(target,0700)};data,err:=os.ReadFile(file);if err!=nil{return err};return os.WriteFile(target,data,0600)
 }))
 must(t,os.WriteFile(filepath.Join(root,"main.go"),[]byte(__COMPOSITION__),0600))
 run(t,root,"go","mod","edit","-replace=hatmax.adrianpk.com="+__SOURCE_ROOT__);run(t,root,"go","mod","tidy")
 return root
}
`
