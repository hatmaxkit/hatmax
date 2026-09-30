// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/eval"
)

func TestInterpreterReturnsStrictResultAndProvenance(t *testing.T) {
	// One normal request uses one turn and reports runtime, thread, model, and
	// protocol provenance without account data.
	client := newInterpreterFakeClient()

	client.threadReused = true
	client.notifications <- agentMessageNotification("thread-1", "turn-1", validUnsupportedOutput())

	client.notifications <- completedTurnNotification("thread-1", "turn-1", "completed")

	interpreter := newTestInterpreter(t, client, time.Second)

	result, err := interpreter.Interpret(context.Background(), promptTestRequest())
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}

	if result.Interpretation.Kind != eval.InterpretationUnsupported {
		t.Fatalf("interpretation = %#v", result.Interpretation)
	}

	want := eval.Provenance{
		Adapter:         "codex-app-server",
		ThreadID:        "thread-1",
		ContractVersion: eval.CurrentContractVersion,
		BackendVersion:  "0.153.0",
		ProtocolVersion: ProtocolVersion,
		ModelSelection:  eval.ModelBackendDefault,
		EffectiveModel:  "gpt-5.4",
		RuntimeReused:   true,
		ThreadReused:    true,
		Timing:          eval.TimingUnderSecond,
	}
	if !reflect.DeepEqual(result.Provenance, want) {
		t.Fatalf("provenance = %#v, want %#v", result.Provenance, want)
	}

	if client.turnStarts != 1 {
		t.Fatalf("turn starts = %d, want 1", client.turnStarts)
	}
}

func TestInterpreterRetriesInvalidSchemaOnce(t *testing.T) {
	// Invalid structured output receives exactly one schema correction turn.
	client := newInterpreterFakeClient()
	client.notifications <- agentMessageNotification("thread-1", "turn-1", `{"kind":"bad"}`)

	client.notifications <- completedTurnNotification("thread-1", "turn-1", "completed")

	client.notifications <- agentMessageNotification("thread-1", "turn-2", validUnsupportedOutput())

	client.notifications <- completedTurnNotification("thread-1", "turn-2", "completed")

	interpreter := newTestInterpreter(t, client, time.Second)

	result, err := interpreter.Interpret(context.Background(), promptTestRequest())
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}

	if result.Interpretation.Kind != eval.InterpretationUnsupported || client.turnStarts != 2 {
		t.Fatalf("result = %#v, turn starts = %d", result, client.turnStarts)
	}

	if len(client.turnPrompts) != 2 || client.turnPrompts[0] == client.turnPrompts[1] {
		t.Fatalf("turn prompts = %#v", client.turnPrompts)
	}
}

func TestInterpreterRejectsInvalidOutputTwice(t *testing.T) {
	// A second schema failure terminates without further model turns.
	client := newInterpreterFakeClient()
	client.notifications <- agentMessageNotification("thread-1", "turn-1", `{}`)

	client.notifications <- completedTurnNotification("thread-1", "turn-1", "completed")

	client.notifications <- agentMessageNotification("thread-1", "turn-2", `{}`)

	client.notifications <- completedTurnNotification("thread-1", "turn-2", "completed")

	interpreter := newTestInterpreter(t, client, time.Second)
	_, err := interpreter.Interpret(context.Background(), promptTestRequest())
	assertBackendCode(t, err, eval.BackendOutputInvalid)

	if client.turnStarts != 2 {
		t.Fatalf("turn starts = %d, want 2", client.turnStarts)
	}
}

func TestInterpreterInterruptsTimedOutTurn(t *testing.T) {
	// A missing completion is interrupted at the configured turn deadline.
	client := newInterpreterFakeClient()
	interpreter := newTestInterpreter(t, client, time.Millisecond)

	_, err := interpreter.Interpret(context.Background(), promptTestRequest())
	assertBackendCode(t, err, eval.BackendTimeout)

	if client.interrupts != 1 {
		t.Fatalf("interrupts = %d, want 1", client.interrupts)
	}
}

func TestInterpreterRequiresChatGPTAuthentication(t *testing.T) {
	// Missing managed ChatGPT authentication is a distinct stable failure.
	client := newInterpreterFakeClient()
	client.accountMissing = true
	interpreter := newTestInterpreter(t, client, time.Second)

	_, err := interpreter.Interpret(context.Background(), promptTestRequest())
	assertBackendCode(t, err, eval.BackendAuthenticationRequired)

	if client.turnStarts != 0 {
		t.Fatalf("turn starts = %d, want 0", client.turnStarts)
	}
}

func TestInterpreterInterruptsCancelledTurn(t *testing.T) {
	// User cancellation interrupts the active turn and remains distinct from a
	// deadline failure.
	client := newInterpreterFakeClient()
	interpreter := newTestInterpreter(t, client, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := interpreter.Interpret(ctx, promptTestRequest())
	assertBackendCode(t, err, eval.BackendCancelled)
}

func TestInterpreterBoundsConsumedTurnEvents(t *testing.T) {
	// Consuming notifications cannot turn the transport buffer into an
	// unbounded event stream.
	client := newInterpreterFakeClient()

	client.notifications = make(chan Notification, maximumTurnEvents+1)

	for index := 0; index <= maximumTurnEvents; index++ {
		client.notifications <- notification("thread/status/changed", map[string]any{})
	}

	interpreter := newTestInterpreter(t, client, time.Second)

	_, err := interpreter.Interpret(context.Background(), promptTestRequest())
	assertBackendCode(t, err, eval.BackendProtocolViolation)

	if client.interrupts != 1 {
		t.Fatalf("interrupts = %d, want 1", client.interrupts)
	}
}

func TestInterpreterContinuesClarificationOnReusedThread(t *testing.T) {
	// Clarification answers are resent as authoritative state while the project
	// thread provides discardable conversation context.
	client := newInterpreterFakeClient()

	client.threadReused = true
	client.notifications <- agentMessageNotification("thread-1", "turn-1", validUnsupportedOutput())

	client.notifications <- completedTurnNotification("thread-1", "turn-1", "completed")

	request := promptTestRequest()
	request.Clarifications = []eval.ClarificationExchange{{
		Field:    "feature",
		Question: "Which feature?",
		Answer:   "invoice",
	}}

	interpreter := newTestInterpreter(t, client, time.Second)

	_, err := interpreter.Interpret(context.Background(), request)
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}

	if len(client.turnPrompts) != 1 || !strings.Contains(client.turnPrompts[0], `"answer":"invoice"`) {
		t.Fatalf("turn prompts = %#v", client.turnPrompts)
	}
}

func newTestInterpreter(t *testing.T, client *interpreterFakeClient, timeout time.Duration) *Interpreter {
	t.Helper()

	runtime := &interpreterFakeRuntime{process: &fakeProcess{}, info: RuntimeInfo{
		Executable:    "/usr/bin/codex",
		CLIVersion:    "0.153.0",
		ServerVersion: "0.153.0",
		Reused:        true,
	}}

	interpreter, err := NewInterpreter(runtime, func(Process) AppServerClient { return client }, InterpreterConfig{
		ContextRoot:     t.TempDir(),
		ProjectIdentity: "opaque-project",
		TurnTimeout:     timeout,
	}, fakeClock{})
	if err != nil {
		t.Fatalf("NewInterpreter() error = %v", err)
	}

	return interpreter
}

func validUnsupportedOutput() string {
	return `{"schema_version":3,"kind":"unsupported","diagnostics":[{"code":"REQUEST-UNSUPPORTED","field":"prompt","message":"Unsupported request"}]}`
}

func agentMessageNotification(threadID, turnID, text string) Notification {
	return notification("item/completed", map[string]any{
		"threadId": threadID,
		"turnId":   turnID,
		"item": map[string]string{
			"type": "agentMessage",
			"text": text,
		},
	})
}

func completedTurnNotification(threadID, turnID, status string) Notification {
	return notification("turn/completed", map[string]any{
		"threadId": threadID,
		"turn": map[string]any{
			"id":     turnID,
			"status": status,
			"items":  []any{},
		},
	})
}

func notification(method string, parameters any) Notification {
	encoded, _ := json.Marshal(parameters)

	return Notification{Method: method, Params: encoded}
}

type interpreterFakeRuntime struct {
	process Process
	info    RuntimeInfo
	err     error
}

func (runtime *interpreterFakeRuntime) Ensure(context.Context) (RuntimeInfo, error) {
	return runtime.info, runtime.err
}

func (runtime *interpreterFakeRuntime) OpenProxy(context.Context, RuntimeInfo) (Process, error) {
	return runtime.process, runtime.err
}

type interpreterFakeClient struct {
	mutex          sync.Mutex
	notifications  chan Notification
	accountMissing bool
	threadReused   bool
	turnStarts     int
	turnPrompts    []string
	interrupts     int
	closed         bool
}

func newInterpreterFakeClient() *interpreterFakeClient {
	return &interpreterFakeClient{notifications: make(chan Notification, 32)}
}

func (client *interpreterFakeClient) Initialize(context.Context) (InitializeResult, error) {
	return InitializeResult{UserAgent: "codex/0.153.0"}, nil
}

func (client *interpreterFakeClient) Call(_ context.Context, method string, parameters any, result any) error {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	switch method {
	case "account/read":
		response := result.(*accountReadResponse)
		response.RequiresOpenAIAuth = true

		if !client.accountMissing {
			response.Account = &account{Type: "chatgpt", Email: "hidden@example.com", PlanType: "plus"}
		}
	case "thread/list":
		response := result.(*threadListResponse)

		if client.threadReused {
			parameters := parameters.(threadListParams)
			response.Data = []threadSummary{{ID: "thread-1", Cwd: parameters.Cwd}}
		}
	case "thread/resume", "thread/start":
		response := result.(*threadResponse)
		response.Thread.ID = "thread-1"
		response.Model = "gpt-5.4"
	case "turn/start":
		client.turnStarts++
		parameters := parameters.(turnStartParameters)
		client.turnPrompts = append(client.turnPrompts, parameters.Input[0].Text)
		response := result.(*turnStartResponse)
		response.Turn = turn{ID: fmtTurnID(client.turnStarts), Status: "inProgress", Items: []turnItem{}}
	default:
		return errors.New("unexpected method: " + method)
	}

	return nil
}

func (client *interpreterFakeClient) Notifications() <-chan Notification {
	return client.notifications
}

func (client *interpreterFakeClient) Interrupt(context.Context, string, string) error {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	client.interrupts++

	return nil
}

func (client *interpreterFakeClient) Failure(string) error {
	return backendError(eval.BackendUnavailable, "test", "connection closed")
}

func (client *interpreterFakeClient) Close() error {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	client.closed = true

	return nil
}

func fmtTurnID(index int) string {
	if index == 1 {
		return "turn-1"
	}

	return "turn-2"
}

var _ io.Closer = (*interpreterFakeClient)(nil)
