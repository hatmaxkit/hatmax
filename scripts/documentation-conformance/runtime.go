// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"hatmax.adrianpk.com/modal"
	"hatmax.adrianpk.com/ui"
	"hatmax.adrianpk.com/web"
)

type commandReceipt struct {
	Directory string   `json:"directory"`
	Arguments []string `json:"arguments"`
	Expected  string   `json:"expected"`
	Exit      int      `json:"exit"`
	Log       string   `json:"log"`
	LogDigest string   `json:"log_sha256"`
}

type processReceipt struct {
	Arguments         []string `json:"arguments"`
	Directory         string   `json:"directory"`
	Environment       []string `json:"fixture_environment"`
	Configuration     string   `json:"configuration_sha256"`
	ConfigurationFile string   `json:"configuration_file,omitempty"`
	Expected          string   `json:"expected_exit"`
	Actual            string   `json:"actual_exit"`
	Log               string   `json:"log"`
	LogDigest         string   `json:"log_sha256,omitempty"`
}

type proof struct {
	ID      string `json:"id"`
	Digest  string `json:"sha256"`
	Method  string `json:"method"`
	Outcome string `json:"outcome"`
}

type runtimeReceipt struct {
	Head         string           `json:"head"`
	Inputs       string           `json:"inputs_sha256"`
	Go           string           `json:"go"`
	Compiler     string           `json:"compiler_configuration"`
	Processes    []processReceipt `json:"processes"`
	Commands     []commandReceipt `json:"commands"`
	Proofs       []proof          `json:"proofs"`
	Observations []string         `json:"observations"`
}

type verification struct {
	root, fixture string
	receipt       runtimeReceipt
}

func writeFile(file, text string) error {
	err := os.MkdirAll(filepath.Dir(file), 0o700)
	if err != nil {
		return err
	}

	return os.WriteFile(file, []byte(text), 0o600)
}

func executionInputs(discovered inventory) (string, error) {
	var bound strings.Builder
	for _, r := range discovered.rows {
		fmt.Fprintf(&bound, "%s\x00%s\x00", r.id, r.digest)
	}

	files, err := filepath.Glob("scripts/documentation-conformance/*.go")
	if err != nil {
		return "", err
	}

	files = append(files, "scripts/check-documentation-conformance.sh", "scripts/documentation-conformance/data-fixture.sh",
		"scripts/documentation-conformance/native-tools.sh", "scripts/docs-check.sh", "scripts/source-license-check.sh",
		"scripts/generator-playground.sh", "Makefile")
	for _, file := range files {
		contents, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}

		fmt.Fprintf(&bound, "%s\x00%s\x00", file, contents)
	}

	return digest(bound.String()), nil
}

func (v *verification) exec(directory, name string, args ...string) error {
	return v.execBound(directory, 5*time.Minute, "", name, args...)
}

func (v *verification) execBound(directory string, bound time.Duration, input, name string, args ...string) error {
	// Build and module commands have finite bounds independent of the external controller.
	ctx, cancel := context.WithTimeout(context.Background(), bound)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)

	cmd.Dir = directory
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if input != "" {
		// Password prompts open /dev/tty before stdin. A private session keeps
		// supplied fixture input authoritative even under a tmux controller.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}

	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second
	output, err := cmd.CombinedOutput()

	exit := 0
	if err != nil {
		exit = 1

		failure, ok := err.(*exec.ExitError)
		if ok {
			exit = failure.ExitCode()
		}
	}

	logName := fmt.Sprintf("runtime-command-%02d.log", len(v.receipt.Commands)+1)

	writeErr := writeFile(filepath.Join(v.fixture, logName), string(output))
	if writeErr != nil {
		return writeErr
	}

	relative, relErr := filepath.Rel(v.root, directory)
	if relErr != nil {
		return relErr
	}

	v.receipt.Commands = append(v.receipt.Commands, commandReceipt{
		Directory: filepath.ToSlash(relative), Arguments: append([]string{name}, args...),
		Expected: "exit 0", Exit: exit, Log: logName, LogDigest: digest(string(output)),
	})
	if err != nil {
		return fmt.Errorf("%s %s failed: %w\n%s", name, strings.Join(args, " "), err, output)
	}

	return nil
}

func (v *verification) module(directory string) error {
	// Module replacement binds every example to this immutable candidate, not a moving release.
	text := fmt.Sprintf("module example.com/docfixture\n\ngo 1.27.1\n\nrequire hatmax.adrianpk.com v0.0.0\n\nreplace hatmax.adrianpk.com => %s\n", strconv.Quote(v.root))

	return writeFile(filepath.Join(directory, "go.mod"), text)
}

func (v *verification) record(r row, method, outcome string) {
	v.receipt.Proofs = append(v.receipt.Proofs, proof{ID: r.id, Digest: r.digest, Method: method, Outcome: outcome})
}

func rowBlock(discovered inventory, r row) (string, error) {
	parts := strings.Split(r.source, "#block-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid block identity: %s", r.id)
	}

	number, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}

	all := blocks(discovered.contents[parts[0]])
	if number < 1 || number > len(all) {
		return "", fmt.Errorf("missing block: %s", r.id)
	}

	return all[number-1], nil
}

func expectedMethod(r row, discovered inventory) (string, error) {
	if r.id == "package:render" {
		return "rendered", nil
	}

	if strings.HasPrefix(r.id, "package:") {
		return "package-tests", nil
	}

	if r.id == "command:examples/guide" || r.id == "example-composition:examples/guide" {
		return "executed", nil
	}

	if strings.HasPrefix(r.id, "walkthrough:") {
		if strings.Contains(r.id, "bootstrap-application") {
			return "executed", nil
		}

		return "source-inspected", nil
	}

	if !strings.HasPrefix(r.id, "example:") {
		return "source-inspected", nil
	}

	block, err := rowBlock(discovered, r)
	if err != nil {
		return "", err
	}

	switch {
	case strings.HasPrefix(block, "```go\n"):
		if strings.Contains(block, "package main") {
			return "executed", nil
		}

		if _, executable := contextOutcomes[r.id]; executable {
			return "executed", nil
		}

		return "compiled", nil
	case strings.HasPrefix(block, "```html\n"):
		return "rendered", nil
	case blockMethod(block) == "published command procedure":
		return "executed", nil
	case strings.HasPrefix(block, "```yaml\n"):
		if strings.Contains(r.id, "bootstrap-application") || r.id == "example:i18n/readme.md#block-3" {
			return "executed", nil
		}

		return "source-inspected", nil
	default:
		return "source-inspected", nil
	}
}

func runRuntime(fixture string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	fixture, err = filepath.Abs(fixture)
	if err != nil {
		return err
	}

	relative, err := filepath.Rel(root, fixture)
	if err != nil || !strings.HasPrefix(filepath.ToSlash(relative), ".tmp/documentation-conformance/slice-2.") {
		return fmt.Errorf("runtime requires owned slice-2 fixture storage")
	}

	_, err = command("git", "check-ignore", filepath.Join(fixture, "runtime-receipts.json"))
	if err != nil {
		return err
	}

	_, statErr := os.Stat(filepath.Join(fixture, "runtime-receipts.json"))
	if !os.IsNotExist(statErr) {
		return fmt.Errorf("fixture already contains runtime evidence")
	}

	discovered, err := discover()
	if err != nil {
		return err
	}

	head, err := command("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}

	version, err := command("go", "version")
	if err != nil {
		return err
	}

	compiler, err := command("go", "env", "-json", "GOOS", "GOARCH", "CGO_ENABLED", "GOTOOLCHAIN", "GOFLAGS", "GOWORK")
	if err != nil {
		return err
	}

	inputs, err := executionInputs(discovered)
	if err != nil {
		return err
	}

	v := verification{root: root, fixture: fixture, receipt: runtimeReceipt{Head: strings.TrimSpace(head), Go: strings.TrimSpace(version), Compiler: compiler, Inputs: inputs}}

	err = checkProcedures(discovered)
	if err != nil {
		return err
	}

	err = v.packageChecks(discovered)
	if err != nil {
		return err
	}

	err = compileContexts(&v, discovered)
	if err != nil {
		return err
	}

	err = v.renderExamples(discovered)
	if err != nil {
		return err
	}

	err = v.processExamples(discovered)
	if err != nil {
		return err
	}

	err = v.httpContracts()
	if err != nil {
		return err
	}
	// Inspection is explicitly separate from compiled/executed evidence.
	for _, r := range discovered.rows {
		if r.slice != 2 {
			continue
		}

		method, err := expectedMethod(r, discovered)
		if err != nil {
			return err
		}

		if method == "source-inspected" {
			v.record(r, method, "T2.1 source/API inspection; contextual notation or conceptual chapter, not a runnable workflow")
		}
	}

	err = verifyProofs(discovered, v.receipt)
	if err != nil {
		return err
	}

	output, err := json.MarshalIndent(v.receipt, "", "  ")
	if err != nil {
		return err
	}

	err = writeFile(filepath.Join(fixture, "runtime-receipts.json"), string(output)+"\n")
	if err != nil {
		return err
	}

	fmt.Printf("Runtime/presentation receipts passed: %d bindings; local HTTP, owned process shutdown, exact fragments and non-empty package suites.\n", len(v.receipt.Proofs))

	return nil
}

func (v *verification) packageChecks(discovered inventory) error {
	return v.packageGroup(discovered, 2)
}

func (v *verification) packageGroup(discovered inventory, group int) error {
	args := []string{"test", "-json", "-count=1", "-timeout=3m"}

	var selected []row

	for _, r := range discovered.rows {
		if r.slice == group && (strings.HasPrefix(r.id, "package:") || r.id == "example-package:examples/ticked/internal/feat/list" || (group == 4 && strings.HasPrefix(r.id, "example-package:")) || (group == 5 && r.id == "implementation:image/internal/ingest") || (group == 6 && strings.HasPrefix(r.id, "implementation:"))) && r.id != "package:render" {
			args = append(args, "./"+r.source)
			selected = append(selected, r)
		}
	}

	err := v.exec(v.root, "go", args...)
	if err != nil {
		return err
	}

	log := v.receipt.Commands[len(v.receipt.Commands)-1].Log

	output, err := os.ReadFile(filepath.Join(v.fixture, log))
	if err != nil {
		return err
	}

	passed := make(map[string]int)

	for _, line := range bytes.Split(output, []byte("\n")) {
		var event struct{ Action, Package, Test string }
		if json.Unmarshal(line, &event) == nil && event.Action == "pass" && event.Test != "" {
			passed[event.Package]++
		}
	}

	for _, r := range selected {
		count := passed["hatmax.adrianpk.com/"+r.source]
		if count == 0 {
			return fmt.Errorf("empty test execution cannot verify %s", r.id)
		}

		v.record(r, "package-tests", fmt.Sprintf("%d actual named tests passed uncached; API contracts, not complete published workflows", count))
	}

	return nil
}

func (v *verification) renderExamples(discovered inventory) error {
	view := map[string]any{
		"Locale": "en", "Amount": 99.99, "Currency": "USD", "Count": 1234,
		"Modal": modal.DefaultConfig("delete-modal", "Confirm Delete"),
		"Form":  ui.NewForm().Action("/submit"), "Items": []string{"item"}, "Invoices": []string{"invoice"},
		"HasMore": true, "NextPage": 2, "StartIndex": 1, "EndIndex": 1, "TotalCount": 1,
	}
	count := 0

	for _, r := range discovered.rows {
		if r.slice != 2 || !strings.HasPrefix(r.id, "example:") {
			continue
		}

		block, err := rowBlock(discovered, r)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(block, "```html\n") {
			continue
		}

		source := blockBody(block)
		// The chapter identifies its row template as application-owned context.
		source += `{{define "assets/templates/invoices/row.html"}}<div>invoice</div>{{end}}`

		tmpl, err := template.New("fragment").Funcs(ui.FuncMap()).Parse(source)
		if err != nil {
			return fmt.Errorf("%s: %w", r.id, err)
		}

		var output bytes.Buffer

		err = tmpl.Execute(&output, view)
		if err != nil {
			return fmt.Errorf("%s: %w", r.id, err)
		}

		if output.Len() == 0 || strings.Contains(output.String(), "ZgotmplZ") || strings.Contains(output.String(), "<no value>") {
			return fmt.Errorf("invalid rendered fragment: %s", r.id)
		}

		rendered := regexp.MustCompile(`(?s)<!--.*?-->`).ReplaceAllString(output.String(), "")
		for marker, expected := range map[string]string{"formatPrice": "$100", "formatNumber": "1,234", "SwapOuter.HTML": `hx-swap="outerHTML"`, "Trigger": "keyup delay:300ms", ".Form.Open": `action="/submit"`, "range seq": "1 2 3 4 5"} {
			if strings.Contains(source, marker) && !strings.Contains(rendered, expected) {
				return fmt.Errorf("%s: missing rendered outcome %q", r.id, expected)
			}
		}

		err = writeFile(filepath.Join(v.fixture, fmt.Sprintf("rendered-%02d.html", count)), output.String())
		if err != nil {
			return err
		}

		v.record(r, "rendered", "actual html/template execution in typed view context; expected attributes and text; application row adapter; no browser DOM-swap claim")

		count++
	}

	if count == 0 {
		return fmt.Errorf("no published templates executed")
	}

	for _, r := range discovered.rows {
		if r.id == "package:render" {
			v.record(r, "rendered", "published render template executed through ui.FuncMap including base string, math, sequence and key fallback; no empty package suite claim")
		}
	}

	return nil
}

type ownedProcess struct {
	cmd          *exec.Cmd
	done         chan error
	log          *os.File
	verification *verification
	index        int
}

func (v *verification) start(binary, directory string, env []string, name string) (*ownedProcess, error) {
	return v.startCommand([]string{binary}, directory, env, name)
}

func (v *verification) startCommand(arguments []string, directory string, env []string, name string) (*ownedProcess, error) {
	log, err := os.OpenFile(filepath.Join(v.fixture, name+"-process.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(arguments[0], arguments[1:]...)

	cmd.Dir = directory
	if strings.HasPrefix(name, "ticked-make") {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = log, log

	err = cmd.Start()
	if err != nil {
		_ = log.Close()

		return nil, err
	}

	configuration, err := os.ReadFile(filepath.Join(directory, "config.yaml"))
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = log.Close()

		return nil, err
	}

	expected := "exit 0 after Ctrl+C"
	if strings.HasPrefix(name, "guide") {
		expected = "signal interrupt; coordinator absent"
	}

	if strings.HasPrefix(name, "ticked-make") {
		expected = "owned make group interrupted; application coordinated shutdown"
	}

	safeEnv := append([]string(nil), env...)
	for index, value := range safeEnv {
		if strings.HasPrefix(value, "TICKED_CREDENTIAL_KEY=") {
			safeEnv[index] = "TICKED_CREDENTIAL_KEY=<private fixture value>"
		}
	}

	index := len(v.receipt.Processes)
	v.receipt.Processes = append(v.receipt.Processes, processReceipt{
		Arguments: cmd.Args, Directory: directory, Environment: safeEnv, Configuration: digest(string(configuration)),
		Expected: expected, Log: name + "-process.log",
	})

	process := &ownedProcess{cmd: cmd, done: make(chan error, 1), log: log, verification: v, index: index}
	go func() { process.done <- cmd.Wait() }()

	return process, nil
}

func (p *ownedProcess) stop(graceful bool) error {
	defer p.log.Close()

	group := p.cmd.SysProcAttr != nil && p.cmd.SysProcAttr.Setpgid
	if group {
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGINT)
	} else {
		_ = p.cmd.Process.Signal(os.Interrupt)
	}

	select {
	case err := <-p.done:
		actual := "exit 0"
		if err != nil {
			actual = err.Error()
		}

		p.verification.receipt.Processes[p.index].Actual = actual

		if group {
			// Make may report the interrupted recipe after its owned application
			// closes ingress. The caller separately inspects coordinator completion.
			return nil
		}

		if graceful {
			return err
		}

		if err == nil {
			return nil
		}

		exit, ok := err.(*exec.ExitError)
		if ok && exit.ProcessState.Sys().(syscall.WaitStatus).Signal() == syscall.SIGINT {
			return nil
		}

		return err
	case <-time.After(10 * time.Second):
		if group {
			_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		} else {
			_ = p.cmd.Process.Kill()
		}

		<-p.done
		p.verification.receipt.Processes[p.index].Actual = "shutdown deadline exceeded; owned process killed"

		return fmt.Errorf("owned process exceeded shutdown deadline")
	}
}

func freeAddress() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	address := listener.Addr().String()
	err = listener.Close()

	return address, err
}

func localClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func request(client *http.Client, method, endpoint, body string, headers map[string]string) (int, http.Header, string, error) {
	req, err := http.NewRequest(method, endpoint, strings.NewReader(body))
	if err != nil {
		return 0, nil, "", err
	}

	for name, value := range headers {
		req.Header.Set(name, value)
	}

	response, err := client.Do(req)
	if err != nil {
		return 0, nil, "", err
	}
	defer response.Body.Close()

	output, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))

	return response.StatusCode, response.Header, string(output), err
}

func awaitHealth(client *http.Client, endpoint string) error {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		status, headers, body, err := request(client, "GET", endpoint+"/ping", "", nil)
		if err == nil && status == http.StatusOK && strings.TrimSpace(body) == `{"status":"ok"}` && strings.Contains(headers.Get("Content-Type"), "application/json") {
			return nil
		}

		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("health endpoint failed exact JSON/content-type assertion")
}

func (v *verification) processExamples(discovered inventory) error {
	page := "docs/how-to/bootstrap-application/README.md"
	clone := filepath.Join(v.fixture, "hatmax")

	err := v.exec(v.root, "git", "clone", "--quiet", "--no-hardlinks", "--no-checkout", v.root, clone)
	if err != nil {
		return err
	}

	err = v.exec(clone, "git", "checkout", "--quiet", "--detach", v.receipt.Head)
	if err != nil {
		return err
	}

	directory := filepath.Join(v.fixture, "myapp")

	err = os.Mkdir(directory, 0o700)
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "mod", "init", "example.com/myapp")
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "mod", "edit", "-replace=hatmax.adrianpk.com=../hatmax")
	if err != nil {
		return err
	}

	err = v.exec(directory, "go", "get", "hatmax.adrianpk.com")
	if err != nil {
		return err
	}

	cloneHead, err := command("git", "-C", clone, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(cloneHead) != v.receipt.Head {
		return fmt.Errorf("bootstrap clone revision mismatch: %v", err)
	}

	for _, source := range []string{"app", "config", "log"} {
		sources, err := filepath.Glob(source + "/*.go")
		if err != nil {
			return err
		}

		for _, file := range sources {
			cloned, err := os.ReadFile(filepath.Join(clone, file))
			if err != nil {
				return err
			}

			if string(cloned) != discovered.contents[file] {
				return fmt.Errorf("bootstrap clone source drift: %s", file)
			}
		}
	}

	for number, file := range map[int]string{2: "config.yaml", 3: "main.go"} {
		body, err := sourceBlock(discovered.contents, page, number)
		if err != nil {
			return err
		}

		err = writeFile(filepath.Join(directory, file), body)
		if err != nil {
			return err
		}
	}

	err = v.exec(directory, "go", "mod", "tidy")
	if err != nil {
		return err
	}

	binary := filepath.Join(directory, "bootstrap")

	err = v.exec(directory, "go", "build", "-o", binary, ".")
	if err != nil {
		return err
	}

	address, err := freeAddress()
	if err != nil {
		return err
	}

	client := localClient()
	defer client.CloseIdleConnections()

	process, err := v.start(binary, directory, []string{"MYAPP_SERVER_PORT=" + address}, "bootstrap")
	if err != nil {
		return err
	}

	healthErr := awaitHealth(client, "http://"+address)
	stopErr := process.stop(true)

	if healthErr != nil {
		return healthErr
	}

	if stopErr != nil {
		return fmt.Errorf("bootstrap Ctrl+C shutdown failed: %w", stopErr)
	}

	_, _, _, err = request(client, "GET", "http://"+address+"/ping", "", nil)
	if err == nil {
		return fmt.Errorf("bootstrap listener remained after successful shutdown")
	}

	v.receipt.Observations = append(v.receipt.Observations, "bootstrap: exact published Go/YAML, owned loopback port via MYAPP_SERVER_PORT; GET /ping 200 JSON; Ctrl+C exit 0; listener unavailable after shutdown; local source replacement, no moving remote checkout claim")

	for _, r := range discovered.rows {
		if r.slice == 2 && (strings.HasPrefix(r.id, "example:"+page) || r.id == "walkthrough:"+page) {
			v.record(r, "executed", "published module init/edit/get/tidy executed with immutable local clone; go build plus owned binary execution replaces go run supervisor; HTTP health and Ctrl+C shutdown observed")
		}
	}

	guideDir := filepath.Join(v.fixture, "guide")

	err = writeFile(filepath.Join(guideDir, "config.yaml"), discovered.contents["examples/guide/config.yaml"])
	if err != nil {
		return err
	}

	binary = filepath.Join(guideDir, "guide")

	err = v.exec(v.root, "go", "build", "-o", binary, "./examples/guide")
	if err != nil {
		return err
	}

	address, err = freeAddress()
	if err != nil {
		return err
	}

	process, err = v.start(binary, guideDir, []string{"GUIDE_SERVER_PORT=" + address, "GUIDE_DATABASE_ENABLED=false"}, "guide")
	if err != nil {
		return err
	}

	err = func() error {
		base := "http://" + address

		err := awaitHealth(client, base)
		if err != nil {
			return err
		}

		cases := []struct {
			method, route, body, origin, expected string
			status                                int
		}{
			{method: "GET", route: "/", status: 200, expected: "Hatmax"},
			{method: "POST", route: "/name", body: "name=Alice", origin: base, status: 200, expected: "Accepted Alice"},
			{method: "POST", route: "/name", body: "name=Alice", status: 403, expected: "Forbidden"},
			{method: "POST", route: "/name", body: "name=Alice", origin: "http://other.invalid", status: 403, expected: "Forbidden"},
			{method: "POST", route: "/name", body: "name=" + url.QueryEscape("<script>alert(1)</script>"), origin: base, status: 200, expected: "&lt;script&gt;"},
			{method: "POST", route: "/name", body: "name=A", origin: base, status: 200, expected: "name"},
			{method: "POST", route: "/greeting", body: "greeting=Good+day", origin: base, status: 303},
		}
		for _, test := range cases {
			status, headers, body, err := request(client, test.method, base+test.route, test.body, map[string]string{"Content-Type": "application/x-www-form-urlencoded", "Origin": test.origin})
			if err != nil || status != test.status || !strings.Contains(body, test.expected) || strings.Contains(body, "<script>alert") {
				return fmt.Errorf("guide %s %s: status %d, wanted %d / %q: %v", test.method, test.route, status, test.status, test.expected, err)
			}

			if test.status == 303 && headers.Get("Location") != "/" {
				return fmt.Errorf("guide redirect target mismatch")
			}

			if test.route == "/" && !strings.Contains(headers.Get("Content-Type"), "text/html") {
				return fmt.Errorf("guide full-page content type mismatch")
			}
		}

		return nil
	}()
	stopErr = process.stop(false)

	if err != nil {
		return err
	}

	if stopErr != nil {
		return stopErr
	}

	v.receipt.Observations = append(v.receipt.Observations, "Guide plain mode: real full page and name form; accepted, invalid and escaped values; missing/cross-site Origin denied; ordinary 303 Location /; owned interrupt termination, not graceful-shutdown proof")

	for _, r := range discovered.rows {
		if r.slice == 2 && (r.id == "command:examples/guide" || r.id == "example-composition:examples/guide" || strings.HasPrefix(r.id, "example:examples/guide/README.md")) {
			v.record(r, "executed", "Guide plain mode built and executed with source-bound config, owned loopback port, full page/name/form/redirect observations; database procedure remains Slice 3")
		}
	}

	return nil
}

func (v *verification) httpContracts() error {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		web.RedirectOrHXRedirect(w, r, "/dashboard")
	}))
	defer server.Close()

	client := localClient()
	defer client.CloseIdleConnections()

	for _, test := range []struct {
		header string
		status int
		name   string
	}{
		{header: "true", status: 200, name: "HX-Redirect"},
		{header: "TRUE", status: 303, name: "Location"},
		{status: 303, name: "Location"},
	} {
		status, headers, _, err := request(client, "GET", server.URL, "", map[string]string{"HX-Request": test.header})
		if err != nil || status != test.status || headers.Get(test.name) != "/dashboard" {
			return fmt.Errorf("web redirect mismatch for HX-Request %q: %d: %v", test.header, status, err)
		}
	}

	v.receipt.Observations = append(v.receipt.Observations, "web.RedirectOrHXRedirect on real owned HTTP listener: exact HX-Request true gives 200 HX-Redirect; absent or TRUE gives 303 Location; automatic redirect following disabled")

	return nil
}

func verifyProofs(discovered inventory, receipt runtimeReceipt) error {
	return verifyGroupProofs(discovered, receipt, 2, expectedMethod)
}

func verifyGroupProofs(discovered inventory, receipt runtimeReceipt, group int, methodFor func(row, inventory) (string, error)) error {
	known := make(map[string]proof)
	for _, p := range receipt.Proofs {
		if _, duplicate := known[p.ID]; duplicate {
			return fmt.Errorf("duplicate execution binding: %s", p.ID)
		}

		known[p.ID] = p
	}

	for _, r := range discovered.rows {
		if r.slice != group {
			continue
		}

		p, found := known[r.id]

		method, err := methodFor(r, discovered)
		if err != nil {
			return err
		}

		if !found || p.Digest != r.digest || p.Method != method || p.Outcome == "" {
			return fmt.Errorf("missing, stale or misclassified runtime receipt: %s", r.id)
		}

		delete(known, r.id)
	}

	if len(known) != 0 || len(receipt.Commands) == 0 || len(receipt.Observations) < 3 {
		return fmt.Errorf("unaccounted or empty runtime evidence")
	}

	for _, cmd := range receipt.Commands {
		if cmd.Exit != 0 || cmd.Expected != "exit 0" || len(cmd.Arguments) == 0 || cmd.Log == "" || cmd.LogDigest == "" {
			return fmt.Errorf("unsuccessful or incomplete command evidence")
		}
	}

	return nil
}

func checkEvidence(file string) error {
	discovered, err := discover()
	if err != nil {
		return err
	}

	output, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	var receipt runtimeReceipt

	err = json.Unmarshal(output, &receipt)
	if err != nil {
		return err
	}

	head, err := command("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}

	version, err := command("go", "version")
	if err != nil {
		return err
	}

	compiler, err := command("go", "env", "-json", "GOOS", "GOARCH", "CGO_ENABLED", "GOTOOLCHAIN", "GOFLAGS", "GOWORK")
	if err != nil {
		return err
	}

	inputs, err := executionInputs(discovered)
	if err != nil {
		return err
	}

	if receipt.Head != strings.TrimSpace(head) || receipt.Inputs != inputs || receipt.Go != strings.TrimSpace(version) || receipt.Compiler != compiler {
		return fmt.Errorf("runtime evidence source/head/tool identity mismatch")
	}

	err = verifyProofs(discovered, receipt)
	if err != nil {
		return err
	}

	coverage, err := os.ReadFile(coveragePath)
	if err != nil {
		return err
	}

	rows, err := parseRows(string(coverage))
	if err != nil {
		return err
	}

	for _, r := range rows {
		if r.slice != 2 {
			continue
		}

		method, err := expectedMethod(r, discovered)
		if err != nil {
			return err
		}

		if r.status != method || r.receipt != "slice-2/runtime-receipts.json" {
			return fmt.Errorf("coverage status disagrees with actual verification: %s", r.id)
		}
	}

	for _, cmd := range receipt.Commands {
		if filepath.Base(cmd.Log) != cmd.Log {
			return fmt.Errorf("invalid command diagnostic path")
		}

		output, err := os.ReadFile(filepath.Join(filepath.Dir(file), cmd.Log))
		if err != nil {
			return err
		}

		if digest(string(output)) != cmd.LogDigest {
			return fmt.Errorf("command diagnostic changed: %s", cmd.Log)
		}
	}

	if len(receipt.Processes) != 2 || receipt.Processes[0].Actual != "exit 0" || receipt.Processes[1].Actual != "signal: interrupt" {
		return fmt.Errorf("missing or unexpected owned process completion")
	}

	for _, process := range receipt.Processes {
		configuration, err := os.ReadFile(filepath.Join(process.Directory, "config.yaml"))
		if err != nil {
			return err
		}

		if process.Configuration != digest(string(configuration)) || process.Expected == "" || len(process.Arguments) != 1 || len(process.Environment) == 0 {
			return fmt.Errorf("owned process configuration or command mismatch")
		}
	}

	fmt.Printf("Exact-head runtime evidence reconciled: %d bindings.\n", len(receipt.Proofs))

	return nil
}
