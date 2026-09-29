package hatmaxstate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"hatmax.adrianpk.com/generator/conversation"
)

const indexSchemaVersion = 1

var conversationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// Config supplies explicit local-state dependencies. An empty root uses the
// platform default.
type Config struct {
	Root         string
	Now          func() time.Time
	NewID        func() string
	BeforeRename func(string) error
}

// Store is the versioned JSON implementation of conversation.Store.
type Store struct {
	root         string
	now          func() time.Time
	newID        func() string
	beforeRename func(string) error
}

// New constructs a local JSON store without creating directories.
func New(config Config) (*Store, error) {
	root := config.Root
	if strings.TrimSpace(root) == "" {
		var err error

		root, err = DefaultRoot()
		if err != nil {
			return nil, err
		}
	}

	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, stateError("state_root_unavailable", "root", "resolve state root: %v", err)
	}

	now := config.Now
	if now == nil {
		now = time.Now
	}

	newID := config.NewID
	if newID == nil {
		newID = uuid.NewString
	}

	return &Store{
		root:         filepath.Clean(absolute),
		now:          now,
		newID:        newID,
		beforeRename: config.BeforeRename,
	}, nil
}

// Root returns the resolved version-independent state root.
func (s *Store) Root() string {
	return s.root
}

// Open resumes or creates one mutable scope session while holding its lock.
func (s *Store) Open(ctx context.Context, request conversation.OpenRequest) (conversation.Session, error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	if request.Fresh && strings.TrimSpace(request.ConversationID) != "" {
		return nil, stateError("state_request_invalid", "conversation_id", "fresh and explicit conversation selection are mutually exclusive")
	}

	if request.ConversationID != "" && !conversationIDPattern.MatchString(request.ConversationID) {
		return nil, stateError("state_request_invalid", "conversation_id", "conversation identifier is not safe")
	}

	err = validateOpenRequest(request)
	if err != nil {
		return nil, err
	}

	scopePath := s.scopePath(request.Scope)

	err = os.MkdirAll(filepath.Join(scopePath, "conversations"), 0o700)
	if err != nil {
		return nil, stateError("state_persistence_failed", "scope", "create scope directory: %v", err)
	}

	err = secureDirectories(s.root, s.versionPath(), filepath.Dir(scopePath), scopePath, filepath.Join(scopePath, "conversations"))
	if err != nil {
		return nil, err
	}

	lock, err := acquireFileLock(filepath.Join(scopePath, "lock"))
	if err != nil {
		return nil, err
	}

	session := &session{
		store:     s,
		scopePath: scopePath,
		lock:      lock,
	}

	err = session.open(request)
	if err != nil {
		_ = lock.Close()

		return nil, err
	}

	return session, nil
}

// List returns recent conversation metadata for one scope without loading turn
// content.
func (s *Store) List(ctx context.Context, scope conversation.Scope) ([]conversation.Summary, error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	err = validateScope(scope)
	if err != nil {
		return nil, err
	}

	scopePath := s.scopePath(scope)

	index, err := s.loadIndex(scopePath, scope)
	if err != nil {
		if isNotExist(err) {
			return []conversation.Summary{}, nil
		}

		return nil, err
	}

	ids := make([]string, 0, 1+len(index.ArchiveOrder))
	if index.Active != "" {
		ids = append(ids, index.Active)
	}

	ids = append(ids, index.ArchiveOrder...)
	result := make([]conversation.Summary, 0, len(ids))

	for _, id := range ids {
		value, readErr := s.loadConversation(scopePath, id)
		if readErr != nil {
			return nil, readErr
		}

		result = append(result, conversation.Summary{
			ID:        value.ID,
			Status:    value.Status,
			CreatedAt: value.CreatedAt,
			UpdatedAt: value.UpdatedAt,
		})
	}

	sort.SliceStable(result, func(left, right int) bool {
		return result[left].UpdatedAt.After(result[right].UpdatedAt)
	})

	return result, nil
}

func (s *Store) versionPath() string {
	return filepath.Join(s.root, "v1")
}

func (s *Store) scopePath(scope conversation.Scope) string {
	return filepath.Join(s.versionPath(), "scopes", scope.Key)
}

func (s *Store) loadIndex(scopePath string, scope conversation.Scope) (scopeIndex, error) {
	var value scopeIndex

	err := readJSON(filepath.Join(scopePath, "index.json"), &value)
	if err != nil {
		return scopeIndex{}, err
	}

	if value.SchemaVersion != indexSchemaVersion || value.Scope != scope {
		return scopeIndex{}, stateError("state_incompatible", "index", "scope index schema or identity is incompatible")
	}

	if value.ArchiveOrder == nil {
		value.ArchiveOrder = []string{}
	}

	if value.IncompatibleIDs == nil {
		value.IncompatibleIDs = []string{}
	}

	return value, nil
}

func (s *Store) loadConversation(scopePath, id string) (conversation.Conversation, error) {
	var value conversation.Conversation

	err := readJSON(conversationPath(scopePath, id), &value)
	if err != nil {
		return conversation.Conversation{}, err
	}

	err = conversation.Validate(value)
	if err != nil {
		return conversation.Conversation{}, stateError("state_incompatible", "conversation", "%v", err)
	}

	err = validatePrivacy(value)
	if err != nil {
		return conversation.Conversation{}, err
	}

	return value, nil
}

func (s *Store) saveIndex(scopePath string, value scopeIndex) error {
	return writeJSONAtomic(filepath.Join(scopePath, "index.json"), value, s.beforeRename)
}

func (s *Store) saveConversation(scopePath string, value conversation.Conversation) error {
	if !conversationIDPattern.MatchString(value.ID) {
		return stateError("state_identity_invalid", "conversation.id", "conversation identifier is not safe")
	}

	pruned, err := conversation.Prune(value)
	if err != nil {
		return err
	}

	err = validatePrivacy(pruned)
	if err != nil {
		return err
	}

	return writeJSONAtomic(conversationPath(scopePath, value.ID), pruned, s.beforeRename)
}

type session struct {
	mu        sync.Mutex
	store     *Store
	scopePath string
	index     scopeIndex
	current   conversation.Conversation
	lock      *fileLock
	closed    bool
}

func (s *session) open(request conversation.OpenRequest) error {
	index, err := s.store.loadIndex(s.scopePath, request.Scope)
	if err != nil {
		if !isNotExist(err) {
			return err
		}

		index = scopeIndex{
			SchemaVersion:   indexSchemaVersion,
			Scope:           request.Scope,
			ArchiveOrder:    []string{},
			IncompatibleIDs: []string{},
		}
	}

	s.index = index

	if request.Fresh {
		return s.startFresh(request)
	}

	id := request.ConversationID
	if id == "" {
		id = index.Active
	}

	if id == "" {
		return s.createInitial(request)
	}

	value, err := s.store.loadConversation(s.scopePath, id)
	if err != nil {
		markErr := s.markIncompatible(id)
		if markErr != nil {
			return errors.Join(err, markErr)
		}

		return err
	}

	if value.Scope != request.Scope || value.BookContract != request.BookContract {
		markErr := s.markIncompatible(id)
		if markErr != nil {
			return markErr
		}

		return stateError("state_incompatible", "conversation", "stored conversation contract is incompatible")
	}

	if value.Status == conversation.StatusIncompatible {
		markErr := s.markIncompatible(id)
		if markErr != nil {
			return markErr
		}

		return stateError("state_incompatible", "conversation", "stored conversation is marked incompatible")
	}

	if id == index.Active && value.Status != conversation.StatusActive {
		value.Status = conversation.StatusActive
		value.UpdatedAt = s.store.now().UTC()

		err = s.store.saveConversation(s.scopePath, value)
		if err != nil {
			return err
		}
	}

	if value.BackendIdentity != request.BackendIdentity {
		value.BackendIdentity = request.BackendIdentity
		value.BackendThreadID = ""
		value.UpdatedAt = s.store.now().UTC()

		err = s.store.saveConversation(s.scopePath, value)
		if err != nil {
			return err
		}
	}

	if request.ConversationID != "" && request.ConversationID != index.Active {
		err = s.selectConversation(value)
		if err != nil {
			return err
		}

		value, err = s.store.loadConversation(s.scopePath, id)
		if err != nil {
			return err
		}
	}

	s.current = conversation.Clone(value)

	return nil
}

func (s *session) createInitial(request conversation.OpenRequest) error {
	value, err := conversation.New(
		s.store.newID(),
		request.Scope,
		request.BookContract,
		request.BackendIdentity,
		s.store.now().UTC(),
	)
	if err != nil {
		return err
	}

	err = s.store.saveConversation(s.scopePath, value)
	if err != nil {
		return err
	}

	s.index.Active = value.ID

	err = s.store.saveIndex(s.scopePath, s.index)
	if err != nil {
		return err
	}

	s.current = conversation.Clone(value)

	return nil
}

func (s *session) startFresh(request conversation.OpenRequest) error {
	if s.index.Active == "" {
		return s.createInitial(request)
	}

	previous, err := s.store.loadConversation(s.scopePath, s.index.Active)
	if err != nil {
		s.index.IncompatibleIDs = appendUnique(s.index.IncompatibleIDs, s.index.Active)
	} else {
		previous.Status = conversation.StatusReset
		previous.BackendThreadID = ""
		previous.UpdatedAt = s.store.now().UTC()

		err = s.store.saveConversation(s.scopePath, previous)
		if err != nil {
			return err
		}

		s.index.ArchiveOrder = prependUnique(s.index.ArchiveOrder, previous.ID)
	}

	s.index.Active = ""

	var dropped []string

	s.index.ArchiveOrder, dropped = trimArchives(s.index.ArchiveOrder)

	err = s.createInitial(request)
	if err != nil {
		return err
	}

	return removeConversationFiles(s.scopePath, dropped)
}

func (s *session) selectConversation(value conversation.Conversation) error {
	previousID := s.index.Active
	if previousID != "" {
		previous, err := s.store.loadConversation(s.scopePath, previousID)
		if err != nil {
			return err
		}

		previous.Status = conversation.StatusArchived
		previous.BackendThreadID = ""
		previous.UpdatedAt = s.store.now().UTC()

		err = s.store.saveConversation(s.scopePath, previous)
		if err != nil {
			return err
		}

		s.index.ArchiveOrder = prependUnique(s.index.ArchiveOrder, previous.ID)
	}

	value.Status = conversation.StatusActive
	value.BackendThreadID = ""
	value.UpdatedAt = s.store.now().UTC()

	err := s.store.saveConversation(s.scopePath, value)
	if err != nil {
		return err
	}

	s.index.Active = value.ID
	s.index.ArchiveOrder = removeValue(s.index.ArchiveOrder, value.ID)

	return s.store.saveIndex(s.scopePath, s.index)
}

func (s *session) markIncompatible(id string) error {
	s.index.IncompatibleIDs = appendUnique(s.index.IncompatibleIDs, id)
	if s.index.Active == id {
		s.index.Active = ""
	}

	s.index.ArchiveOrder = removeValue(s.index.ArchiveOrder, id)

	return s.store.saveIndex(s.scopePath, s.index)
}

func (s *session) Current() conversation.Conversation {
	s.mu.Lock()
	defer s.mu.Unlock()

	return conversation.Clone(s.current)
}

func (s *session) Replace(ctx context.Context, value conversation.Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := ctx.Err()
	if err != nil {
		return err
	}

	err = s.ensureOpen()
	if err != nil {
		return err
	}

	if value.ID != s.current.ID || value.Scope != s.current.Scope || value.BookContract != s.current.BookContract {
		return stateError("state_identity_mismatch", "conversation", "replacement must preserve conversation, scope, and contract identity")
	}

	pruned, err := conversation.Prune(value)
	if err != nil {
		return err
	}

	err = s.store.saveConversation(s.scopePath, pruned)
	if err != nil {
		return err
	}

	s.current = conversation.Clone(pruned)

	return nil
}

func (s *session) Reset(ctx context.Context, id string, now time.Time) (conversation.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := ctx.Err()
	if err != nil {
		return conversation.Conversation{}, err
	}

	err = s.ensureOpen()
	if err != nil {
		return conversation.Conversation{}, err
	}

	if !conversationIDPattern.MatchString(id) {
		return conversation.Conversation{}, stateError("state_identity_invalid", "conversation.id", "conversation identifier is not safe")
	}

	previous, next, err := s.current.Reset(id, now.UTC())
	if err != nil {
		return conversation.Conversation{}, err
	}

	err = s.store.saveConversation(s.scopePath, previous)
	if err != nil {
		return conversation.Conversation{}, err
	}

	err = s.store.saveConversation(s.scopePath, next)
	if err != nil {
		return conversation.Conversation{}, err
	}

	s.index.Active = next.ID
	s.index.ArchiveOrder = prependUnique(s.index.ArchiveOrder, previous.ID)

	var dropped []string

	s.index.ArchiveOrder, dropped = trimArchives(s.index.ArchiveOrder)

	err = s.store.saveIndex(s.scopePath, s.index)
	if err != nil {
		return conversation.Conversation{}, err
	}

	err = removeConversationFiles(s.scopePath, dropped)
	if err != nil {
		return conversation.Conversation{}, err
	}

	s.current = conversation.Clone(next)

	return conversation.Clone(next), nil
}

func (s *session) Rebind(ctx context.Context, scope conversation.Scope, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := ctx.Err()
	if err != nil {
		return err
	}

	err = s.ensureOpen()
	if err != nil {
		return err
	}

	value := conversation.Clone(s.current)

	err = value.Rebind(scope, now.UTC())
	if err != nil {
		return err
	}

	newScopePath := s.store.scopePath(scope)

	err = os.MkdirAll(filepath.Join(newScopePath, "conversations"), 0o700)
	if err != nil {
		return stateError("state_persistence_failed", "scope", "create rebound scope directory: %v", err)
	}

	err = secureDirectories(s.store.root, s.store.versionPath(), filepath.Dir(newScopePath), newScopePath, filepath.Join(newScopePath, "conversations"))
	if err != nil {
		return err
	}

	newLock, err := acquireFileLock(filepath.Join(newScopePath, "lock"))
	if err != nil {
		return err
	}

	keepLock := false
	defer func() {
		if !keepLock {
			_ = newLock.Close()
		}
	}()

	newIndex, err := s.store.loadIndex(newScopePath, scope)
	if err != nil {
		if !isNotExist(err) {
			return err
		}

		newIndex = scopeIndex{
			SchemaVersion:   indexSchemaVersion,
			Scope:           scope,
			ArchiveOrder:    []string{},
			IncompatibleIDs: []string{},
		}
	}

	if newIndex.Active != "" && newIndex.Active != value.ID {
		return stateError("state_rebind_conflict", "scope", "target scope already has an active conversation")
	}

	err = s.store.saveConversation(newScopePath, value)
	if err != nil {
		return err
	}

	newIndex.Active = value.ID

	err = s.store.saveIndex(newScopePath, newIndex)
	if err != nil {
		return err
	}

	oldScopePath := s.scopePath
	s.index.Active = ""

	err = s.store.saveIndex(oldScopePath, s.index)
	if err != nil {
		return err
	}

	err = os.Remove(conversationPath(oldScopePath, value.ID))
	if err != nil && !isNotExist(err) {
		return stateError("state_persistence_failed", "conversation", "remove rebound source snapshot: %v", err)
	}

	err = s.lock.Close()
	if err != nil {
		return err
	}

	s.lock = newLock
	s.scopePath = newScopePath
	s.index = newIndex
	s.current = conversation.Clone(value)
	keepLock = true

	return nil
}

func (s *session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	return s.lock.Close()
}

func (s *session) ensureOpen() error {
	if s.closed {
		return stateError("state_session_closed", "session", "conversation session is closed")
	}

	return nil
}

func trimArchives(values []string) ([]string, []string) {
	if len(values) <= conversation.MaximumArchivedConversations {
		return values, []string{}
	}

	kept := append([]string{}, values[:conversation.MaximumArchivedConversations]...)
	dropped := append([]string{}, values[conversation.MaximumArchivedConversations:]...)

	return kept, dropped
}

func removeConversationFiles(scopePath string, ids []string) error {
	for _, id := range ids {
		err := os.Remove(conversationPath(scopePath, id))
		if err != nil && !isNotExist(err) {
			return stateError("state_persistence_failed", "conversation", "prune archived conversation: %v", err)
		}
	}

	return nil
}

func validateOpenRequest(request conversation.OpenRequest) error {
	err := validateScope(request.Scope)
	if err != nil {
		return err
	}

	if request.BookContract.BookVersion < 1 || request.BookContract.InterpreterVersion < 1 {
		return stateError("state_request_invalid", "book_contract", "Book and interpreter versions must be positive")
	}

	if strings.TrimSpace(request.BackendIdentity.Adapter) == "" {
		return stateError("state_request_invalid", "backend_identity.adapter", "backend adapter is required")
	}

	return nil
}

func validateScope(scope conversation.Scope) error {
	if scope.Kind != conversation.ScopeProject && scope.Kind != conversation.ScopePreProject {
		return stateError("state_request_invalid", "scope.kind", "unknown scope kind %q", scope.Kind)
	}

	if len(scope.Key) != 64 {
		return stateError("state_request_invalid", "scope.key", "scope key must be a lowercase SHA-256 digest")
	}

	for _, character := range scope.Key {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return stateError("state_request_invalid", "scope.key", "scope key must be a lowercase SHA-256 digest")
		}
	}

	return nil
}

func secureDirectories(paths ...string) error {
	for _, path := range paths {
		err := os.MkdirAll(path, 0o700)
		if err != nil {
			return stateError("state_persistence_failed", "directory", "create state directory: %v", err)
		}

		err = os.Chmod(path, 0o700)
		if err != nil {
			return stateError("state_persistence_failed", "directory", "secure state directory: %v", err)
		}
	}

	return nil
}

func conversationPath(scopePath, id string) string {
	return filepath.Join(scopePath, "conversations", id+".json")
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}

	return append(values, value)
}

func prependUnique(values []string, value string) []string {
	return append([]string{value}, removeValue(values, value)...)
}

func removeValue(values []string, value string) []string {
	result := make([]string, 0, len(values))
	for _, existing := range values {
		if existing != value {
			result = append(result, existing)
		}
	}

	return result
}

var _ conversation.Store = (*Store)(nil)
