package hatmaxstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/conversation"
)

func TestStorePersistsAndResumesBoundedConversation(t *testing.T) {
	root := t.TempDir()
	clock := newTestClock()
	store := newTestStore(t, root, clock, idSequence("conversation-1"), nil)
	request := testOpenRequest(t, conversation.ScopeProject)
	projectRoot := filepath.Join(t.TempDir(), "private-project")

	err := os.MkdirAll(projectRoot, 0o700)
	if err != nil {
		t.Fatalf("MkdirAll(project) error = %v", err)
	}

	request.Scope, err = conversation.ResolveScope(conversation.ScopeProject, projectRoot)
	if err != nil {
		t.Fatalf("ResolveScope(project) error = %v", err)
	}

	session, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	value := session.Current()

	err = value.AddOperation("operation-1", "Create an invoice feature", clock.next())
	if err != nil {
		t.Fatalf("AddOperation() error = %v", err)
	}

	err = value.AppendTurn(conversation.RoleUser, conversation.TurnGoal, "Create an invoice feature", "operation-1", clock.next())
	if err != nil {
		t.Fatalf("AppendTurn() error = %v", err)
	}

	err = session.Replace(context.Background(), value)
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	err = session.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open(restart) error = %v", err)
	}
	defer reopened.Close()

	got := reopened.Current()
	if got.ID != "conversation-1" || len(got.Turns) != 1 || len(got.Operations) != 1 {
		t.Fatalf("reopened conversation = %#v", got)
	}

	summaries, err := store.List(context.Background(), request.Scope)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(summaries) != 1 || summaries[0].ID != "conversation-1" {
		t.Fatalf("List() = %#v", summaries)
	}

	assertStateLayoutAndPermissions(t, store, request.Scope, value.ID)
	assertStateDoesNotContain(t, store.Root(), projectRoot)
}

func TestStoreLocksOneMutableScopeSession(t *testing.T) {
	store := newTestStore(t, t.TempDir(), newTestClock(), idSequence("conversation-1"), nil)
	request := testOpenRequest(t, conversation.ScopeProject)

	first, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open(first) error = %v", err)
	}
	defer first.Close()

	_, err = store.Open(context.Background(), request)
	requireStateCode(t, err, "state_busy")
}

func TestStoreInterruptedReplacementKeepsPreviousSnapshot(t *testing.T) {
	root := t.TempDir()
	clock := newTestClock()
	failReplacement := false
	store := newTestStore(t, root, clock, idSequence("conversation-1"), func(path string) error {
		if failReplacement && strings.HasSuffix(path, "conversation-1.json") {
			return errors.New("simulated interruption")
		}

		return nil
	})
	request := testOpenRequest(t, conversation.ScopeProject)

	session, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	value := session.Current()

	err = value.AppendTurn(conversation.RoleUser, conversation.TurnDialogue, "not published", "", clock.next())
	if err != nil {
		t.Fatalf("AppendTurn() error = %v", err)
	}

	failReplacement = true
	err = session.Replace(context.Background(), value)
	requireStateCode(t, err, "state_persistence_failed")

	if len(session.Current().Turns) != 0 {
		t.Fatal("failed replacement changed in-memory durable snapshot")
	}

	failReplacement = false

	err = session.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	restarted := newTestStore(t, root, clock, idSequence("unused"), nil)

	reopened, err := restarted.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open(restart) error = %v", err)
	}
	defer reopened.Close()

	if len(reopened.Current().Turns) != 0 {
		t.Fatalf("interrupted write replaced snapshot: %#v", reopened.Current().Turns)
	}
}

func TestStorePreservesIncompatibleStateUntilExplicitFreshConversation(t *testing.T) {
	root := t.TempDir()
	clock := newTestClock()
	store := newTestStore(t, root, clock, idSequence("conversation-1", "conversation-2"), nil)
	request := testOpenRequest(t, conversation.ScopeProject)

	session, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	err = session.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	path := conversationPath(store.scopePath(request.Scope), "conversation-1")

	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var document map[string]any

	err = json.Unmarshal(original, &document)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	document["schema_version"] = float64(99)

	incompatible, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	err = os.WriteFile(path, incompatible, 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err = store.Open(context.Background(), request)
	requireStateCode(t, err, "state_incompatible")

	afterFailure, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(after failure) error = %v", err)
	}

	if !reflect.DeepEqual(afterFailure, incompatible) {
		t.Fatal("incompatible snapshot was overwritten")
	}

	request.Fresh = true

	fresh, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open(fresh) error = %v", err)
	}
	defer fresh.Close()

	if fresh.Current().ID != "conversation-2" {
		t.Fatalf("fresh conversation ID = %q", fresh.Current().ID)
	}

	_, err = os.Stat(path)
	if err != nil {
		t.Fatalf("incompatible snapshot was not preserved: %v", err)
	}
}

func TestStoreRejectsCredentialLikeContentWithoutChangingSnapshot(t *testing.T) {
	clock := newTestClock()
	store := newTestStore(t, t.TempDir(), clock, idSequence("conversation-1"), nil)
	request := testOpenRequest(t, conversation.ScopeProject)

	session, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer session.Close()

	value := session.Current()

	err = value.AppendTurn(conversation.RoleUser, conversation.TurnDialogue, "OPENAI_API_KEY=secret", "", clock.next())
	if err != nil {
		t.Fatalf("AppendTurn() error = %v", err)
	}

	err = session.Replace(context.Background(), value)
	requireStateCode(t, err, "state_sensitive_content")

	if len(session.Current().Turns) != 0 {
		t.Fatal("sensitive content entered the durable session snapshot")
	}
}

func TestStoreResetSelectionAndArchivePruning(t *testing.T) {
	ids := make([]string, 12)
	for index := range ids {
		ids[index] = fmt.Sprintf("conversation-%02d", index)
	}

	clock := newTestClock()
	store := newTestStore(t, t.TempDir(), clock, idSequence(ids...), nil)
	request := testOpenRequest(t, conversation.ScopeProject)

	session, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	for index := 1; index < len(ids); index++ {
		_, err = session.Reset(context.Background(), ids[index], clock.next())
		if err != nil {
			t.Fatalf("Reset(%d) error = %v", index, err)
		}
	}

	err = session.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	summaries, err := store.List(context.Background(), request.Scope)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(summaries) != 1+conversation.MaximumArchivedConversations {
		t.Fatalf("List() length = %d, want %d", len(summaries), 1+conversation.MaximumArchivedConversations)
	}

	request.ConversationID = ids[len(ids)-2]

	selected, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open(selected) error = %v", err)
	}
	defer selected.Close()

	if selected.Current().ID != request.ConversationID || selected.Current().Status != conversation.StatusActive {
		t.Fatalf("selected conversation = %#v", selected.Current())
	}
}

func TestStoreRebindsScopeAndReplacesBackendThreadWithoutLosingContext(t *testing.T) {
	root := t.TempDir()
	clock := newTestClock()
	store := newTestStore(t, root, clock, idSequence("conversation-1"), nil)
	request := testOpenRequest(t, conversation.ScopePreProject)

	session, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	value := session.Current()
	value.BackendThreadID = "thread-1"

	err = value.AppendTurn(conversation.RoleUser, conversation.TurnDialogue, "Keep this context", "", clock.next())
	if err != nil {
		t.Fatalf("AppendTurn() error = %v", err)
	}

	err = session.Replace(context.Background(), value)
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	projectScope, err := conversation.ResolveScope(conversation.ScopeProject, t.TempDir())
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}

	err = session.Rebind(context.Background(), projectScope, clock.next())
	if err != nil {
		t.Fatalf("Rebind() error = %v", err)
	}

	if session.Current().Scope != projectScope || session.Current().BackendThreadID != "" {
		t.Fatalf("rebound session = %#v", session.Current())
	}

	value = session.Current()
	value.BackendThreadID = "thread-after-rebind"

	err = session.Replace(context.Background(), value)
	if err != nil {
		t.Fatalf("Replace(rebound thread) error = %v", err)
	}

	err = session.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	oldSummaries, err := store.List(context.Background(), request.Scope)
	if err != nil {
		t.Fatalf("List(old scope) error = %v", err)
	}

	if len(oldSummaries) != 0 {
		t.Fatalf("old scope summaries = %#v", oldSummaries)
	}

	request.Scope = projectScope
	request.BackendIdentity = conversation.BackendIdentity{Adapter: "codex-app-server", Version: "0.159.0", Model: "backend_default"}

	reopened, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open(rebound) error = %v", err)
	}
	defer reopened.Close()

	got := reopened.Current()
	if got.BackendThreadID != "" || len(got.Turns) != 1 || got.Turns[0].Content != "Keep this context" {
		t.Fatalf("backend replacement lost or reused state: %#v", got)
	}
}

func TestStoreRejectsUnsafeConversationSelection(t *testing.T) {
	store := newTestStore(t, t.TempDir(), newTestClock(), idSequence("conversation-1"), nil)
	request := testOpenRequest(t, conversation.ScopeProject)
	request.ConversationID = "../../outside"

	_, err := store.Open(context.Background(), request)
	requireStateCode(t, err, "state_request_invalid")

	request.ConversationID = ""

	session, err := store.Open(context.Background(), request)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer session.Close()

	before := session.Current()
	_, err = session.Reset(context.Background(), "../outside", time.Now().UTC())
	requireStateCode(t, err, "state_identity_invalid")

	if !reflect.DeepEqual(session.Current(), before) {
		t.Fatal("unsafe reset identifier changed the active snapshot")
	}
}

type testClock struct {
	current time.Time
}

func newTestClock() *testClock {
	return &testClock{current: time.Date(2026, time.September, 29, 19, 0, 0, 0, time.UTC)}
}

func (c *testClock) next() time.Time {
	c.current = c.current.Add(time.Second)

	return c.current
}

func newTestStore(
	t *testing.T,
	root string,
	clock *testClock,
	newID func() string,
	beforeRename func(string) error,
) *Store {
	t.Helper()

	store, err := New(Config{
		Root:         root,
		Now:          clock.next,
		NewID:        newID,
		BeforeRename: beforeRename,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return store
}

func testOpenRequest(t *testing.T, kind conversation.ScopeKind) conversation.OpenRequest {
	t.Helper()

	scope, err := conversation.ResolveScope(kind, t.TempDir())
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}

	return conversation.OpenRequest{
		Scope:        scope,
		BookContract: conversation.BookContract{BookVersion: 1, InterpreterVersion: 3},
		BackendIdentity: conversation.BackendIdentity{
			Adapter: "codex-app-server",
			Version: "0.158.0",
			Model:   "backend_default",
		},
	}
}

func idSequence(values ...string) func() string {
	index := 0

	return func() string {
		if index >= len(values) {
			return fmt.Sprintf("unexpected-%d", index)
		}

		value := values[index]
		index++

		return value
	}
}

func assertStateLayoutAndPermissions(t *testing.T, store *Store, scope conversation.Scope, conversationID string) {
	t.Helper()

	scopePath := store.scopePath(scope)
	for _, path := range []string{
		store.root,
		store.versionPath(),
		filepath.Dir(scopePath),
		scopePath,
		filepath.Join(scopePath, "conversations"),
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}

		if info.Mode().Perm() != 0o700 {
			t.Errorf("directory %q permissions = %o, want 700", path, info.Mode().Perm())
		}
	}

	for _, path := range []string{
		filepath.Join(scopePath, "index.json"),
		filepath.Join(scopePath, "lock"),
		conversationPath(scopePath, conversationID),
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}

		if info.Mode().Perm() != 0o600 {
			t.Errorf("file %q permissions = %o, want 600", path, info.Mode().Perm())
		}
	}
}

func assertStateDoesNotContain(t *testing.T, root, forbidden string) {
	t.Helper()

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		if strings.Contains(string(data), forbidden) {
			t.Errorf("state file %q contains forbidden path %q", path, forbidden)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}
}

func requireStateCode(t *testing.T, err error, code string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want %q", code)
	}

	var stateErr Error
	if !errors.As(err, &stateErr) {
		t.Fatalf("error = %T %v, want hatmaxstate.Error", err, err)
	}

	if stateErr.Code != code {
		t.Fatalf("error code = %q, want %q", stateErr.Code, code)
	}
}
