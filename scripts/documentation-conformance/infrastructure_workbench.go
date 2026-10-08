// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var infrastructureProgramOutputs = map[string]string{
	"mailer": "[NOOP MAILER]", "pubsub": "user.created user-1", "telemetry": "500 1 1", "slug": "hello-world-3995fd11\ncafe-resume",
}

var infrastructureBindings = map[string]string{
	"configuration-api:scheduler/config.go":                                   "published YAML loaded through Config; defaults, root mapping and persisted retry budget verified",
	"example:mailer/readme.md#block-1":                                        "exact main executed; fixture sender/recipient/subject logged without transport",
	"example:pubsub/readme.md#block-1":                                        "exact main executed; synchronous memory fan-out printed its documented topic/payload",
	"example:telemetry/readme.md#block-1":                                     "exact main executed; real middleware observed 500, one request and one crash group",
	"example:slug/readme.md#block-1":                                          "exact main executed; both documented slug strings observed",
	"example:fake/readme.md#block-1":                                          "exact TestWelcome executed uncached; validated capture and reset assertions passed",
	"example:testhelper/readme.md#block-1":                                    "exact TestDatabase executed against the owned cluster; pool selected its generated schema",
	"example:image/readme.md#block-1":                                         "exact savePNG executed with generated PNG; resized readback, URL and caller deletion verified",
	"example:scheduler/README.md#block-1":                                     "exact run executed against fresh PostgreSQL; immediate due job, JSON output, cancellation and joined shutdown verified",
	"example:scheduler/README.md#block-4":                                     "exact schedule fragment executed; daily/weekly UTC and interval arithmetic asserted",
	"example:scheduler/README.md#block-5":                                     "exact fake-store tick executed; successful one-shot output and retirement verified",
	"example:scheduler/README.md#block-6":                                     "exact PostgreSQL schema/store fragment executed in an owned schema",
	"example:docs/how-to/configure-mailer/README.md#block-1":                  "exact YAML loaded with fixture-only environment expansion; dry-run/active/unknown selection verified",
	"example:docs/how-to/configure-mailer/README.md#block-2":                  "exact runtime constructor executed for dry-run and owned active SMTP",
	"example:docs/how-to/configure-mailer/README.md#block-3":                  "exact Send fragment executed; local SMTP captured matching message and canceled send was rejected",
	"example:docs/how-to/store-images/README.md#block-1":                      "exact local constructor executed under owned working directory",
	"example:docs/how-to/store-images/README.md#block-3":                      "exact resize/store fragment executed with PNG; extension, 800x400 readback, deletion and canceled input verified",
	"example:docs/how-to/run-background-jobs/README.md#block-1":               "exact store/runner constructors executed against owned PostgreSQL and published config",
	"example:docs/how-to/run-background-jobs/README.md#block-2":               "exact registered handler executed and its JSON output inspected in job_runs",
	"example:docs/how-to/run-background-jobs/README.md#block-3":               "exact scheduler YAML loaded; published polling/worker/retry values asserted",
	"example:docs/how-to/run-background-jobs/README.md#block-4":               "exact INSERT executed; daily recurring slots advanced without repeating the consumed slot",
	"example:docs/how-to/run-background-jobs/README.md#block-5":               "exact fresh-context Stop executed; active handler cancellation and repeat stops joined",
	"example:docs/how-to/use-pubsub/README.md#block-1":                        "exact Setup assembled and started with an application-owned subscriber adapter; broker closed before database",
	"example:docs/how-to/use-pubsub/README.md#block-2":                        "exact named subscription executed; pending JSON delivery retained across owned restart",
	"example:docs/how-to/use-pubsub/README.md#block-3":                        "exact Publish fragment executed; independent success acknowledgement and failed-message retry observed",
	"example:docs/tutorials/user-guide/events-and-background-work.md#block-2": "exact invoice publication fragment executed with a typed application-owned service; real delivery observed",
	"example:docs/tutorials/user-guide/events-and-background-work.md#block-3": "exact AuditService.Start executed with real PostgreSQL; subscribed invoice delivered as decoded JSON",
	"example:docs/how-to/test-with-postgres/README.md#block-1":                "exact TestStore executed uncached in its compiled package context; cleanup inspected externally",
	"example:docs/how-to/test-with-postgres/README.md#block-2":                "exact published go test command executed in an isolated module containing TestStore",
	"example:docs/how-to/test-with-postgres/README.md#block-3":                "exact published environment/server command executed with owned socket/role/database values",
	"walkthrough:docs/how-to/configure-mailer/README.md":                      "published config and send bytes executed through no-op and owned SMTP; validation, reconstruction, cancellation and transport capture observed; external inbox acceptance excluded",
	"walkthrough:docs/how-to/store-images/README.md":                          "published local selection/resize plus complete savePNG readback executed; dimensions, extension, URL, deletion and context limits checked; S3 selection separately compiled and its fixtures tested",
	"walkthrough:docs/how-to/run-background-jobs/README.md":                   "published constructors, config, handler, SQL and Stop executed; one-shot/recurring slots, errors, panic, persistent retry/restart budget and cooperative shutdown observed",
	"walkthrough:docs/how-to/use-pubsub/README.md":                            "published Setup, Subscribe and Publish executed with real owned PostgreSQL; JSON payload, success acknowledgement, pending named restart and cooperative Close verified",
	"walkthrough:docs/how-to/test-with-postgres/README.md":                    "exact server-path test and both commands ran; generated schemas removed; WithConfig consumer pool closed before cleanup; container path source-inspected, not claimed executed",
	"walkthrough:docs/tutorials/user-guide/application-services.md":           "real local mail/image/telemetry compositions exercised; application-owned metadata/export policy remains explicit conceptual context",
	"walkthrough:docs/tutorials/user-guide/events-and-background-work.md":     "exact subscriber/publication methods plus real persisted event/job workflows; retry, recurrence and cancellation observed; no transactional outbox claim",
	"walkthrough:docs/tutorials/user-guide/testing-and-evolution.md":          "published fake/helper tests and owned real persistence workflows executed; conceptual feature-evolution/review guidance source-inspected; complete runtime make check remains outside this documentation gate",
}

func (v *verification) infrastructureWorkbench(discovered inventory) error {
	directory := filepath.Join(v.fixture, "infrastructure-workbench")

	err := v.module(directory)
	if err != nil {
		return err
	}

	for _, owner := range []string{"mailer", "image", "pubsub", "scheduler", "telemetry", "testhelper", "fake", "slug"} {
		page := owner + "/readme.md"
		if owner == "scheduler" {
			page = "scheduler/README.md"
		}

		body, err := sourceBlock(discovered.contents, page, 1)
		if err != nil {
			return err
		}

		file := "published.go"
		if owner == "fake" || owner == "testhelper" {
			file = "published_test.go"
		}

		err = writeFile(filepath.Join(directory, owner, file), body)
		if err != nil {
			return err
		}
	}

	for file, text := range map[string]string{"image/workflow_test.go": infrastructureImageTest, "scheduler/workflow_test.go": infrastructurePollingTest} {
		err = writeFile(filepath.Join(directory, file), text)
		if err != nil {
			return err
		}
	}

	for file, page := range map[string]string{"mailer.yaml": "docs/how-to/configure-mailer/README.md", "scheduler.yaml": "docs/how-to/run-background-jobs/README.md"} {
		number := 1
		if file == "scheduler.yaml" {
			number = 3
		}

		text, err := sourceBlock(discovered.contents, page, number)
		if err != nil {
			return err
		}

		err = writeFile(filepath.Join(directory, file), text)
		if err != nil {
			return err
		}
	}

	replacements := make([]string, 0)

	for marker, input := range map[string]struct {
		page   string
		number int
		quoted bool
	}{
		"__MAIL_CREATE__": {"docs/how-to/configure-mailer/README.md", 2, false}, "__MAIL_SEND__": {"docs/how-to/configure-mailer/README.md", 3, false},
		"__IMAGE_STORE__": {"docs/how-to/store-images/README.md", 1, false}, "__IMAGE_RESIZE__": {"docs/how-to/store-images/README.md", 3, false},
		"__JOB_CREATE__": {"docs/how-to/run-background-jobs/README.md", 1, false}, "__JOB_HANDLER__": {"docs/how-to/run-background-jobs/README.md", 2, false},
		"__JOB_SQL__": {"docs/how-to/run-background-jobs/README.md", 4, true}, "__JOB_STOP__": {"docs/how-to/run-background-jobs/README.md", 5, false},
		"__BROKER_CREATE__": {"docs/how-to/use-pubsub/README.md", 1, false}, "__BROKER_SUBSCRIBE__": {"docs/how-to/use-pubsub/README.md", 2, false},
		"__BROKER_PUBLISH__": {"docs/how-to/use-pubsub/README.md", 3, false}, "__AUDIT_START__": {"docs/tutorials/user-guide/events-and-background-work.md", 3, false},
		"__INVOICE_PUBLISH__": {"docs/tutorials/user-guide/events-and-background-work.md", 2, false}, "__FAKE_TICK__": {"scheduler/README.md", 5, false},
		"__CALENDAR__": {"scheduler/README.md", 4, false}, "__STORE_SCHEMA__": {"scheduler/README.md", 6, false},
	} {
		body, err := sourceBlock(discovered.contents, input.page, input.number)
		if err != nil {
			return err
		}

		if input.quoted {
			body = strconv.Quote(body)
		}

		replacements = append(replacements, marker, body)
	}

	text := strings.NewReplacer(replacements...).Replace(infrastructureWorkflowTest)
	if regexp.MustCompile(`__[A-Z_]+__`).MatchString(text) {
		return fmt.Errorf("unbound published infrastructure fragment")
	}

	err = writeFile(filepath.Join(directory, "workflow", "published_test.go"), text)
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "mod", "tidy")
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "test", "-json", "-count=1", "-timeout=90s", "./...")
	if err != nil {
		return err
	}

	output, err := os.ReadFile(filepath.Join(v.fixture, v.receipt.Commands[len(v.receipt.Commands)-1].Log))
	if err != nil {
		return err
	}

	for _, name := range []string{"TestWelcome", "TestDatabase", "TestReadback", "TestPolling", "TestMail", "TestImages", "TestEvents", "TestJobs", "TestStop", "TestClock", "TestConfiguration"} {
		if !namedTestPassed(output, name) {
			return fmt.Errorf("missing actual infrastructure workflow: %s", name)
		}
	}

	for owner, expected := range infrastructureProgramOutputs {
		err = v.exec(directory, "go", "run", "./"+owner)
		if err != nil {
			return err
		}

		log, err := os.ReadFile(filepath.Join(v.fixture, v.receipt.Commands[len(v.receipt.Commands)-1].Log))
		if err != nil || !bytes.Contains(log, []byte(expected)) {
			return fmt.Errorf("published %s output differs", owner)
		}
	}

	// Both published server-path commands run in a standalone module with the
	// exact test, rather than treating repository helper utility tests as proof.
	helper := filepath.Join(v.fixture, "published-helper")

	err = v.module(helper)
	if err != nil {
		return err
	}

	body, err := sourceBlock(discovered.contents, "docs/how-to/test-with-postgres/README.md", 1)
	if err != nil {
		return err
	}

	err = writeFile(filepath.Join(helper, "published_test.go"), "package fixture\nimport(\"context\";\"testing\";\"hatmax.adrianpk.com/testhelper\")\n"+body)
	if err != nil {
		return err
	}

	err = v.exec(helper, "go", "mod", "tidy")
	if err != nil {
		return err
	}

	for _, number := range []int{2, 3} {
		body, err := sourceBlock(discovered.contents, "docs/how-to/test-with-postgres/README.md", number)
		if err != nil {
			return err
		}

		err = v.execBound(helper, 2*time.Minute, "", "bash", "-eu", "-c", body)
		if err != nil {
			return err
		}
	}

	err = v.exec(v.root, "psql", "-h", os.Getenv("DB_HOST"), "-p", os.Getenv("DB_PORT"), "-U", os.Getenv("DB_USER"), "-d", os.Getenv("DB_NAME"), "-At", "-v", "ON_ERROR_STOP=1", "-c", "SELECT COUNT(*) FROM pg_namespace WHERE nspname LIKE 'test_%'")
	if err != nil {
		return err
	}

	log, err := os.ReadFile(filepath.Join(v.fixture, v.receipt.Commands[len(v.receipt.Commands)-1].Log))
	if err != nil || strings.TrimSpace(string(log)) != "0" {
		return fmt.Errorf("owned helper schemas were not removed")
	}

	for _, r := range discovered.rows {
		if outcome, ok := infrastructureBindings[r.id]; ok {
			v.bindData(r, "executed", outcome)
		}
	}

	return nil
}

func namedTestPassed(output []byte, name string) bool {
	if bytes.Contains(output, []byte("(cached)")) || bytes.Contains(output, []byte("no tests to run")) {
		return false
	}

	for _, line := range bytes.Split(output, []byte("\n")) {
		var event struct{ Action, Test string }
		if json.Unmarshal(line, &event) == nil && event.Action == "pass" && event.Test == name {
			return true
		}
	}

	return false
}

const infrastructureImageTest = `package example
import("bytes";"context";"image";"image/png";"os";"testing";"hatmax.adrianpk.com/image/local")
// Read back the exact published PNG function's variant, then remove it.
func TestReadback(t *testing.T){
 var encoded bytes.Buffer
 if err:=png.Encode(&encoded,image.NewRGBA(image.Rect(0,0,1600,800)));err!=nil{t.Fatal(err)}
 root:=t.TempDir();url,err:=savePNG(context.Background(),root,bytes.NewReader(encoded.Bytes()))
 if err!=nil||url!="/uploads/images/photo-medium.png"{t.Fatalf("URL/readback: %q %v",url,err)}
 store:=local.NewStore(root,"/uploads");reader,err:=store.Get(context.Background(),"images/photo-medium.png");if err!=nil{t.Fatal(err)}
 config,format,err:=image.DecodeConfig(reader);reader.Close()
 if err!=nil||format!="png"||config.Width!=800||config.Height!=400{t.Fatal("stored PNG dimensions differ")}
 if err:=store.Delete(context.Background(),"images/photo-medium.png");err!=nil{t.Fatal(err)}
 if _,err:=store.Get(context.Background(),"images/photo-medium.png");!os.IsNotExist(err){t.Fatal("deleted object remains")}
}
`

const infrastructurePollingTest = `package example
import("context";"testing";"time";"hatmax.adrianpk.com/scheduler/postgres";"hatmax.adrianpk.com/testhelper")
// The exact function must poll immediately, persist output and join on cancellation.
func TestPolling(t *testing.T){
 database,_,cleanup:=testhelper.SetupTestDB(t);t.Cleanup(cleanup)
 if _,err:=database.Exec(postgres.Schema);err!=nil{t.Fatal(err)}
 if _,err:=database.Exec("INSERT INTO scheduled_jobs(id,name,task_type,next_run_at) VALUES('mail','mail','send-email',NOW())");err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithCancel(context.Background());defer cancel()
 done:=make(chan error,1);go func(){done<-run(ctx,database,testhelper.TestLogger())}()
 deadline:=time.Now().Add(3*time.Second)
 complete:=false
 for time.Now().Before(deadline){var status string;var sent bool
  err:=database.QueryRow("SELECT status,(output->>'sent')::boolean FROM job_runs WHERE job_id='mail'").Scan(&status,&sent)
  if err==nil&&status=="success"&&sent{complete=true;break};time.Sleep(10*time.Millisecond)
 }
 cancel()
 select{case err:=<-done:if err!=nil{t.Fatal(err)};case<-time.After(6*time.Second):t.Fatal("published polling function failed to join")}
 if !complete{t.Fatal("published polling function did not complete due mail job")}
}
`
