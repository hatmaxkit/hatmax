package codex

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/eval"
)

func TestContextIdentityIsOpaqueStableAndContractBound(t *testing.T) {
	// Context identities must not expose their inputs and must rotate with any
	// compatibility boundary.
	first, err := NewContextIdentity("project-secret", "account-secret", 1, 1)
	if err != nil {
		t.Fatalf("NewContextIdentity() error = %v", err)
	}

	same, err := NewContextIdentity("project-secret", "account-secret", 1, 1)
	if err != nil {
		t.Fatalf("NewContextIdentity() error = %v", err)
	}

	changed, err := NewContextIdentity("project-secret", "account-secret", 2, 1)
	if err != nil {
		t.Fatalf("NewContextIdentity() error = %v", err)
	}

	if first != same || first == changed {
		t.Fatalf("identities = %q, %q, %q", first, same, changed)
	}

	if strings.Contains(string(first), "project") || strings.Contains(string(first), "account") {
		t.Fatalf("identity exposes source values: %q", first)
	}
}

func TestThreadManagerStartsIsolatedToolFreeThread(t *testing.T) {
	// A missing context thread starts in the Hatmax-owned directory with no
	// model override, writable sandbox, approval channel, or enabled tool.
	client := &fakeCaller{responses: []any{
		threadListResponse{},
		threadResponse{Thread: threadSummary{ID: "thread-new"}, Model: "gpt-5.4"},
	}}
	root := t.TempDir()

	manager, err := NewThreadManager(client, root, "")
	if err != nil {
		t.Fatalf("NewThreadManager() error = %v", err)
	}

	identity := mustContextIdentity(t)

	session, err := manager.Acquire(context.Background(), identity)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	if session.ID != "thread-new" || session.Reused || session.ModelSelection != eval.ModelBackendDefault || session.EffectiveModel != "gpt-5.4" {
		t.Fatalf("session = %#v", session)
	}

	if filepath.Dir(session.Directory) != filepath.Clean(root) || strings.Contains(session.Directory, "target-project") {
		t.Fatalf("context directory = %q", session.Directory)
	}

	information, err := os.Stat(session.Directory)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if information.Mode().Perm() != contextDirectoryMode {
		t.Fatalf("directory mode = %o", information.Mode().Perm())
	}

	start, ok := client.calls[1].parameters.(threadStartParams)
	if !ok {
		t.Fatalf("start parameters = %T", client.calls[1].parameters)
	}

	if start.Cwd != session.Directory || start.Sandbox != "read-only" || start.ApprovalPolicy != "never" || start.Model != "" {
		t.Fatalf("start parameters = %#v", start)
	}

	features := start.Config["features"].(map[string]bool)
	for name, enabled := range features {
		if enabled {
			t.Fatalf("feature %q is enabled", name)
		}
	}
}

func TestThreadManagerResumesMatchingThread(t *testing.T) {
	// Exact isolated cwd matching selects and resumes the newest context thread.
	root := t.TempDir()
	identity := mustContextIdentity(t)
	directory := filepath.Join(root, string(identity))
	client := &fakeCaller{responses: []any{
		threadListResponse{Data: []threadSummary{{ID: "thread-old", Cwd: directory}}},
		threadResponse{Thread: threadSummary{ID: "thread-old"}, Model: "gpt-5.4"},
	}}

	manager, err := NewThreadManager(client, root, "gpt-explicit")
	if err != nil {
		t.Fatalf("NewThreadManager() error = %v", err)
	}

	session, err := manager.Acquire(context.Background(), identity)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	if !session.Reused || session.ModelSelection != eval.ModelExplicit {
		t.Fatalf("session = %#v", session)
	}

	methods := []string{client.calls[0].method, client.calls[1].method}
	if !reflect.DeepEqual(methods, []string{"thread/list", "thread/resume"}) {
		t.Fatalf("methods = %#v", methods)
	}

	resume := client.calls[1].parameters.(threadResumeParams)
	if resume.ThreadID != "thread-old" || resume.Model != "gpt-explicit" || !resume.ExcludeTurns {
		t.Fatalf("resume parameters = %#v", resume)
	}
}

func TestThreadManagerReplacesInvalidListedThread(t *testing.T) {
	// A thread that disappears after listing is discarded and replaced once.
	root := t.TempDir()
	identity := mustContextIdentity(t)
	directory := filepath.Join(root, string(identity))
	client := &fakeCaller{
		responses: []any{
			threadListResponse{Data: []threadSummary{{ID: "thread-stale", Cwd: directory}}},
			nil,
			threadResponse{Thread: threadSummary{ID: "thread-new"}, Model: "gpt-5.4"},
		},
		errors: map[int]error{1: errors.New("not found")},
	}

	manager, err := NewThreadManager(client, root, "")
	if err != nil {
		t.Fatalf("NewThreadManager() error = %v", err)
	}

	session, err := manager.Acquire(context.Background(), identity)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	methods := []string{client.calls[0].method, client.calls[1].method, client.calls[2].method}
	if session.Reused || !reflect.DeepEqual(methods, []string{"thread/list", "thread/resume", "thread/start"}) {
		t.Fatalf("session = %#v, methods = %#v", session, methods)
	}
}

func TestForbiddenToolNotificationsFailClosed(t *testing.T) {
	// Direct tool events and executable item kinds are protocol violations.
	cases := []Notification{
		{Method: "item/commandExecution/outputDelta", Params: json.RawMessage(`{}`)},
		{Method: "item/started", Params: json.RawMessage(`{"item":{"type":"webSearch"}}`)},
		{Method: "item/completed", Params: json.RawMessage(`{"item":{"type":"dynamicToolCall"}}`)},
	}

	for _, item := range cases {
		err := forbiddenToolNotification(item)
		assertBackendCode(t, err, eval.BackendProtocolViolation)
	}

	allowed := Notification{Method: "item/completed", Params: json.RawMessage(`{"item":{"type":"agentMessage"}}`)}

	err := forbiddenToolNotification(allowed)
	if err != nil {
		t.Fatalf("passive item error = %v", err)
	}
}

func TestClientRejectsToolNotifications(t *testing.T) {
	// The transport guard closes the connection before exposing tool activity.
	clientProcess, serverProcess := newPipeProcesses()
	client := NewClient(clientProcess)

	t.Cleanup(func() { _ = client.Close() })

	encoder := json.NewEncoder(serverProcess)

	err := encoder.Encode(map[string]any{
		"method": "item/started",
		"params": map[string]any{"item": map[string]string{"type": "commandExecution"}},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	<-client.done
	assertBackendCode(t, client.connectionError("test"), eval.BackendProtocolViolation)
}

func mustContextIdentity(t *testing.T) ContextIdentity {
	t.Helper()

	identity, err := NewContextIdentity("target-project", "account", 1, 1)
	if err != nil {
		t.Fatalf("NewContextIdentity() error = %v", err)
	}

	return identity
}

type fakeCall struct {
	method     string
	parameters any
}

type fakeCaller struct {
	calls     []fakeCall
	responses []any
	errors    map[int]error
}

func (client *fakeCaller) Call(_ context.Context, method string, parameters any, result any) error {
	index := len(client.calls)
	client.calls = append(client.calls, fakeCall{method: method, parameters: parameters})

	err := client.errors[index]
	if err != nil {
		return err
	}

	if result == nil || index >= len(client.responses) || client.responses[index] == nil {
		return nil
	}

	encoded, err := json.Marshal(client.responses[index])
	if err != nil {
		return err
	}

	return json.Unmarshal(encoded, result)
}
