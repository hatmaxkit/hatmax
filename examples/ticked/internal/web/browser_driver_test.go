//go:build browser

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type browserMessage struct {
	ID        int             `json:"id"`
	Method    string          `json:"method"`
	SessionID string          `json:"sessionId"`
	Params    json.RawMessage `json:"params"`
	Result    json.RawMessage `json:"result"`
	Error     json.RawMessage `json:"error"`
}

// The private Chromium pipe owns transport and process control in Go. JavaScript
// runs only in Chromium, where it exercises navigator and the served forms.
type browserPipe struct {
	mu      sync.Mutex
	writer  io.Writer
	serial  int
	pending map[int]chan browserMessage
	closed  chan struct{}
	bridge  func(browserMessage)
}

func browserFrame(data []byte, atEOF bool) (int, []byte, error) {
	end := bytes.IndexByte(data, 0)
	if end >= 0 {
		return end + 1, data[:end], nil
	}

	if atEOF && len(data) != 0 {
		return 0, nil, fmt.Errorf("incomplete browser frame")
	}

	return 0, nil, nil
}

func (p *browserPipe) read(reader io.Reader) {
	defer close(p.closed)

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	scanner.Split(browserFrame)

	for scanner.Scan() {
		var message browserMessage

		err := json.Unmarshal(scanner.Bytes(), &message)
		if err != nil {
			return
		}

		p.mu.Lock()
		reply := p.pending[message.ID]
		bridge := p.bridge
		p.mu.Unlock()

		if reply != nil {
			reply <- message
		} else if message.Method == "Runtime.bindingCalled" && bridge != nil {
			bridge(message)
		}
	}
}

func (p *browserPipe) call(ctx context.Context, method string, params any, session string) (json.RawMessage, error) {
	p.mu.Lock()
	p.serial++
	id := p.serial
	reply := make(chan browserMessage, 1)
	p.pending[id] = reply

	request := map[string]any{"id": id, "method": method, "params": params}
	if session != "" {
		request["sessionId"] = session
	}

	encoded, err := json.Marshal(request)
	if err == nil {
		_, err = p.writer.Write(append(encoded, 0))
	}
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		delete(p.pending, id)
		p.mu.Unlock()
	}()

	if err != nil {
		return nil, fmt.Errorf("owned browser transport failed")
	}

	select {
	case response := <-reply:
		if len(response.Error) != 0 {
			return nil, fmt.Errorf("owned browser rejected %s", method)
		}

		return response.Result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("owned browser %s: %w", method, ctx.Err())
	case <-p.closed:
		return nil, fmt.Errorf("owned browser transport closed")
	}
}

func (p *browserPipe) target(ctx context.Context) (string, error) {
	created, err := p.call(ctx, "Target.createTarget", map[string]any{"url": "about:blank"}, "")
	if err != nil {
		return "", err
	}

	var target struct{ TargetID string }

	err = json.Unmarshal(created, &target)
	if err != nil || target.TargetID == "" {
		return "", fmt.Errorf("missing owned browser target")
	}

	attached, err := p.call(ctx, "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}, "")
	if err != nil {
		return "", err
	}

	var session struct{ SessionID string }

	err = json.Unmarshal(attached, &session)
	if err != nil || session.SessionID == "" {
		return "", fmt.Errorf("missing owned browser session")
	}

	return session.SessionID, nil
}

// CDP callbacks service the separate driver tab while the tested tab navigates.
// Only this invocation's browser pipe is reachable; no public fixture endpoint
// accepts CDP requests or supplies authentication authority.
func (p *browserPipe) installBridge(ctx context.Context, driver string) {
	slots := make(chan struct{}, 8)

	p.mu.Lock()
	p.bridge = func(message browserMessage) {
		if message.SessionID != driver {
			return
		}

		var binding struct{ Name, Payload string }

		err := json.Unmarshal(message.Params, &binding)
		if err != nil || binding.Name != "hatmaxCDP" || len(binding.Payload) > 256<<10 {
			return
		}

		var request struct {
			ID        int
			Method    string
			Params    json.RawMessage
			SessionID string
		}

		err = json.Unmarshal([]byte(binding.Payload), &request)
		if err != nil || request.ID < 1 {
			return
		}

		select {
		case slots <- struct{}{}:
		default:
			return
		}

		go func() {
			defer func() { <-slots }()

			callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			result, callErr := p.call(callCtx, request.Method, request.Params, request.SessionID)

			var failure string
			if callErr != nil {
				failure = "owned CDP request failed"
				result = json.RawMessage("null")
			}

			reply, marshalErr := json.Marshal([]any{request.ID, result, failure})
			if marshalErr == nil {
				_, _ = p.call(callCtx, "Runtime.evaluate", map[string]any{"expression": "nativeReply(..." + string(reply) + ")"}, driver)
			}
		}()
	}
	p.mu.Unlock()
}

func runNativeBrowser(ctx context.Context, binary, origin, profile string, recovery bool) (output string, returned error) {
	scratch, err := os.MkdirTemp("/tmp", "hatmax-browser-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(scratch)

	input, send, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer input.Close()
	defer send.Close()

	receive, outputPipe, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer receive.Close()
	defer outputPipe.Close()

	command := exec.CommandContext(ctx, binary, "--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check", "--disable-background-networking", "--remote-debugging-pipe", "--user-data-dir="+profile, "about:blank")
	command.ExtraFiles = []*os.File{input, outputPipe}

	command.Env = append(os.Environ(), "TMPDIR="+scratch)
	command.WaitDelay = 10 * time.Second

	err = command.Start()
	if err != nil {
		return "", err
	}

	_ = input.Close()
	_ = outputPipe.Close()

	p := &browserPipe{writer: send, pending: make(map[int]chan browserMessage), closed: make(chan struct{})}
	go p.read(receive)

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, _ = p.call(stopCtx, "Browser.close", map[string]any{}, "")

		finished := make(chan error, 1)
		go func() { finished <- command.Wait() }()

		var waitErr error

		select {
		case waitErr = <-finished:
		case <-stopCtx.Done():
			_ = command.Process.Kill()
			waitErr = <-finished
			returned = fmt.Errorf("owned browser exceeded shutdown deadline")
		}

		if returned == nil && waitErr != nil {
			returned = fmt.Errorf("owned browser shutdown failed: %w", waitErr)
		}
	}()

	startupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	version, err := p.call(startupCtx, "Browser.getVersion", map[string]any{}, "")
	if err != nil {
		return "", err
	}

	driver, err := p.target(startupCtx)
	if err != nil {
		return "", err
	}

	_, err = p.call(startupCtx, "Runtime.addBinding", map[string]any{"name": "hatmaxCDP"}, driver)
	if err != nil {
		return "", err
	}

	p.installBridge(ctx, driver)

	var program strings.Builder

	program.WriteString("(async () => {\n" + nativeBrowserPrelude + "\n")

	originJSON, _ := json.Marshal(origin)

	concern := "authenticators"
	if recovery {
		concern = "account-recovery"
	}

	fmt.Fprintf(&program, "const origin=%s, concern=%q, version=%s;\n", originJSON, concern, version)
	// The documentation gate supplies the exact published JSON-conversion
	// fragment. Its actual registration must pass the production finish handler.
	published := ""

	if file := os.Getenv("HATMAX_DOC_REGISTRATION"); file != "" {
		fragment, readErr := os.ReadFile(file)
		if readErr != nil {
			return "", readErr
		}

		published = string(fragment)
	}

	publishedJSON, _ := json.Marshal(published)
	fmt.Fprintf(&program, "const publishedRegistration=%s;\n", publishedJSON)
	program.WriteString("try {\n")

	for _, file := range []string{"controls.js", "recovery.js", "authenticators.js"} {
		contents, readErr := os.ReadFile(filepath.Join("testdata", file))
		if readErr != nil {
			return "", readErr
		}

		program.Write(contents)
		program.WriteByte('\n')
	}

	program.WriteString("return {stages, browser:version.product}; } catch(error) {return {stages, browser:version.product, failure:error.hatmaxAssertion || 'browser/transport failure'};} })()")

	result, err := p.call(ctx, "Runtime.evaluate", map[string]any{"expression": program.String(), "awaitPromise": true, "returnByValue": true}, driver)
	if err != nil {
		return "", err
	}

	var evaluation struct {
		ExceptionDetails json.RawMessage
		Result           struct {
			Value struct {
				Stages  []string
				Browser string
				Failure string
			}
		}
	}

	err = json.Unmarshal(result, &evaluation)
	if err != nil || len(evaluation.ExceptionDetails) != 0 {
		return "", fmt.Errorf("native browser journey failed; protocol values redacted")
	}

	if evaluation.Result.Value.Failure != "" {
		return "", fmt.Errorf("native browser fixture assertion: %s; completed stages: %s", evaluation.Result.Value.Failure, strings.Join(evaluation.Result.Value.Stages, "; "))
	}

	requiredStages := 8
	if recovery {
		requiredStages = 12
	}

	if len(evaluation.Result.Value.Stages) != requiredStages || evaluation.Result.Value.Browser == "" {
		return "", fmt.Errorf("native browser journey incomplete")
	}

	return strings.Join(evaluation.Result.Value.Stages, "\n") + "\nBROWSER " + evaluation.Result.Value.Browser + "; CTAP2/internal+USB/RK/UV; journeys complete", nil
}

const nativeBrowserPrelude = `
const stages=[];
const assert=(condition,label='browser condition')=>{if(!condition){const error=Error('fixture assertion');error.hatmaxAssertion=label;throw error}};
assert.equal=(actual,expected,label)=>assert(Object.is(actual,expected),label);
assert.notEqual=(actual,expected,label)=>assert(!Object.is(actual,expected),label);
assert.deepEqual=(actual,expected,label)=>assert(JSON.stringify(actual)===JSON.stringify(expected),label);
const delay=milliseconds=>new Promise(resolve=>setTimeout(resolve,milliseconds));
let nativeSerial=0;
const nativePending=new Map();
globalThis.nativeReply=(id,result,failure)=>{
  const request=nativePending.get(id); if(!request)return;
  clearTimeout(request.timer); nativePending.delete(id);
  if(failure)request.reject(Error(failure));else request.resolve(result);
};
const call=(method,params={},sessionId='')=>new Promise((resolve,reject)=>{
  const id=++nativeSerial;
  const timer=setTimeout(()=>{nativePending.delete(id);reject(Error('CDP timeout: '+method))},15000);
  nativePending.set(id,{resolve,reject,timer});
  hatmaxCDP(JSON.stringify({id,method,params,sessionId}));
});
async function waitFor(check,label){
  const deadline=Date.now()+10000;
  while(Date.now()<deadline){if(await check())return;await delay(25)}
  throw Error('Browser condition timeout: '+label);
}
`
