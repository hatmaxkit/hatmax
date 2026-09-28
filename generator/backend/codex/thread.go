package codex

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hatmax.adrianpk.com/generator/eval"
)

const (
	contextIdentityVersion = "hatmax-codex-context-v1"
	contextDirectoryMode   = 0o700
)

var disabledFeatures = map[string]bool{
	"apps":             false,
	"browser_use":      false,
	"computer_use":     false,
	"hooks":            false,
	"image_generation": false,
	"multi_agent":      false,
	"plugins":          false,
	"shell_tool":       false,
	"skill_search":     false,
	"sleep_tool":       false,
	"tool_suggest":     false,
	"view_image":       false,
}

// ContextIdentity is an opaque project-and-contract identifier. It can be
// used in local paths without exposing the target project path or contents.
type ContextIdentity string

// NewContextIdentity derives a stable opaque identity for one project,
// compatible Book and adapter contract, and authenticated Codex identity.
func NewContextIdentity(projectIdentity, authenticatedIdentity string, bookVersion, contractVersion int) (ContextIdentity, error) {
	if strings.TrimSpace(projectIdentity) == "" {
		return "", fmt.Errorf("project identity is required")
	}

	if strings.TrimSpace(authenticatedIdentity) == "" {
		return "", fmt.Errorf("authenticated identity is required")
	}

	if bookVersion < 1 || contractVersion < 1 {
		return "", fmt.Errorf("positive Book and contract versions are required")
	}

	hash := sha256.New()
	writeIdentityPart(hash, contextIdentityVersion)
	writeIdentityPart(hash, projectIdentity)
	writeIdentityPart(hash, authenticatedIdentity)
	writeIdentityPart(hash, fmt.Sprintf("%d", bookVersion))
	writeIdentityPart(hash, fmt.Sprintf("%d", contractVersion))

	return ContextIdentity(hex.EncodeToString(hash.Sum(nil))), nil
}

// ThreadSession records one started or resumed isolated project thread.
type ThreadSession struct {
	ID             string
	Context        ContextIdentity
	Directory      string
	EffectiveModel string
	ModelSelection eval.ModelSelection
	Reused         bool
}

// Caller is the App Server request boundary needed by thread management.
type Caller interface {
	Call(context.Context, string, any, any) error
}

// ThreadManager starts and resumes isolated interpretation threads.
type ThreadManager struct {
	client Caller
	root   string
	model  string
}

// NewThreadManager constructs a manager rooted in a Hatmax-owned directory.
// An empty model delegates model selection to the backend.
func NewThreadManager(client Caller, root, model string) (*ThreadManager, error) {
	if client == nil {
		return nil, fmt.Errorf("App Server client is required")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil || strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("absolute context root is required")
	}

	return &ThreadManager{client: client, root: filepath.Clean(absoluteRoot), model: strings.TrimSpace(model)}, nil
}

// Acquire resumes the newest matching thread or starts one when none remains
// valid. The working directory contains no target-project files.
func (manager *ThreadManager) Acquire(ctx context.Context, identity ContextIdentity) (ThreadSession, error) {
	if !validContextIdentity(identity) {
		return ThreadSession{}, backendError(eval.BackendThreadFailed, "context_identity", "Hatmax context identity is invalid")
	}

	directory := filepath.Join(manager.root, string(identity))

	err := ensureContextDirectory(directory)
	if err != nil {
		return ThreadSession{}, backendError(eval.BackendThreadFailed, "context_directory", "isolated Hatmax context directory could not be prepared")
	}

	threadID, err := manager.findThread(ctx, directory)
	if err != nil {
		return ThreadSession{}, err
	}

	if threadID != "" {
		session, resumeErr := manager.resume(ctx, identity, directory, threadID)
		if resumeErr == nil {
			return session, nil
		}
	}

	return manager.start(ctx, identity, directory)
}

func (manager *ThreadManager) findThread(ctx context.Context, directory string) (string, error) {
	parameters := threadListParams{
		Cwd:            directory,
		Limit:          10,
		SortDirection:  "desc",
		UseStateDBOnly: true,
	}

	var response threadListResponse

	err := manager.client.Call(ctx, "thread/list", parameters, &response)
	if err != nil {
		return "", backendError(eval.BackendThreadFailed, "thread_list", "isolated Codex threads could not be listed")
	}

	for _, thread := range response.Data {
		if thread.ID != "" && filepath.Clean(thread.Cwd) == directory {
			return thread.ID, nil
		}
	}

	return "", nil
}

func (manager *ThreadManager) resume(ctx context.Context, identity ContextIdentity, directory, threadID string) (ThreadSession, error) {
	parameters := threadResumeParams{
		ThreadID:       threadID,
		Cwd:            directory,
		ApprovalPolicy: "never",
		Sandbox:        "read-only",
		Model:          manager.model,
		Config:         isolatedThreadConfig(),
		ExcludeTurns:   true,
	}

	var response threadResponse

	err := manager.client.Call(ctx, "thread/resume", parameters, &response)
	if err != nil || response.Thread.ID == "" {
		return ThreadSession{}, backendError(eval.BackendThreadFailed, "thread_resume", "isolated Codex thread could not be resumed")
	}

	return manager.session(identity, directory, response, true), nil
}

func (manager *ThreadManager) start(ctx context.Context, identity ContextIdentity, directory string) (ThreadSession, error) {
	parameters := threadStartParams{
		Cwd:                   directory,
		ApprovalPolicy:        "never",
		Sandbox:               "read-only",
		Model:                 manager.model,
		Config:                isolatedThreadConfig(),
		Ephemeral:             false,
		BaseInstructions:      interpretationBaseInstructions,
		DeveloperInstructions: interpretationDeveloperInstructions,
		ThreadSource:          "hatmax",
	}

	var response threadResponse

	err := manager.client.Call(ctx, "thread/start", parameters, &response)
	if err != nil || response.Thread.ID == "" {
		return ThreadSession{}, backendError(eval.BackendThreadFailed, "thread_start", "isolated Codex thread could not be started")
	}

	return manager.session(identity, directory, response, false), nil
}

func (manager *ThreadManager) session(identity ContextIdentity, directory string, response threadResponse, reused bool) ThreadSession {
	selection := eval.ModelBackendDefault
	if manager.model != "" {
		selection = eval.ModelExplicit
	}

	return ThreadSession{
		ID:             response.Thread.ID,
		Context:        identity,
		Directory:      directory,
		EffectiveModel: response.Model,
		ModelSelection: selection,
		Reused:         reused,
	}
}

func ensureContextDirectory(directory string) error {
	err := os.MkdirAll(directory, contextDirectoryMode)
	if err != nil {
		return fmt.Errorf("create context directory: %w", err)
	}

	err = os.Chmod(directory, contextDirectoryMode)
	if err != nil {
		return fmt.Errorf("secure context directory: %w", err)
	}

	return nil
}

func isolatedThreadConfig() map[string]any {
	features := make(map[string]bool, len(disabledFeatures))
	for name, enabled := range disabledFeatures {
		features[name] = enabled
	}

	return map[string]any{
		"features":    features,
		"mcp_servers": map[string]any{},
	}
}

func validContextIdentity(identity ContextIdentity) bool {
	if len(identity) != sha256.Size*2 {
		return false
	}

	_, err := hex.DecodeString(string(identity))

	return err == nil
}

func writeIdentityPart(hash interface{ Write([]byte) (int, error) }, value string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = hash.Write(length[:])
	_, _ = hash.Write([]byte(value))
}

func forbiddenToolNotification(notification Notification) error {
	if directToolNotification(notification.Method) {
		return backendError(eval.BackendProtocolViolation, notification.Method, "Codex attempted forbidden tool activity")
	}

	if notification.Method != "item/started" && notification.Method != "item/completed" {
		return nil
	}

	var parameters struct {
		Item struct {
			Type string `json:"type"`
		} `json:"item"`
	}

	err := json.Unmarshal(notification.Params, &parameters)
	if err != nil || parameters.Item.Type == "" {
		return backendError(eval.BackendProtocolViolation, notification.Method, "Codex emitted an invalid item event")
	}

	if allowedPassiveItem(parameters.Item.Type) {
		return nil
	}

	return backendError(eval.BackendProtocolViolation, notification.Method, "Codex attempted forbidden item activity")
}

func directToolNotification(method string) bool {
	return strings.HasPrefix(method, "item/commandExecution/") ||
		strings.HasPrefix(method, "item/fileChange/") ||
		strings.HasPrefix(method, "item/mcpToolCall/") ||
		strings.HasPrefix(method, "thread/realtime/")
}

func allowedPassiveItem(itemType string) bool {
	switch itemType {
	case "agentMessage", "reasoning", "userMessage", "plan", "contextCompaction":
		return true
	default:
		return false
	}
}

type threadListParams struct {
	Cwd            string `json:"cwd"`
	Limit          int    `json:"limit"`
	SortDirection  string `json:"sortDirection"`
	UseStateDBOnly bool   `json:"useStateDbOnly"`
}

type threadListResponse struct {
	Data []threadSummary `json:"data"`
}

type threadSummary struct {
	ID  string `json:"id"`
	Cwd string `json:"cwd"`
}

type threadStartParams struct {
	Cwd                   string         `json:"cwd"`
	ApprovalPolicy        string         `json:"approvalPolicy"`
	Sandbox               string         `json:"sandbox"`
	Model                 string         `json:"model,omitempty"`
	Config                map[string]any `json:"config"`
	Ephemeral             bool           `json:"ephemeral"`
	BaseInstructions      string         `json:"baseInstructions"`
	DeveloperInstructions string         `json:"developerInstructions"`
	ThreadSource          string         `json:"threadSource"`
}

type threadResumeParams struct {
	ThreadID       string         `json:"threadId"`
	Cwd            string         `json:"cwd"`
	ApprovalPolicy string         `json:"approvalPolicy"`
	Sandbox        string         `json:"sandbox"`
	Model          string         `json:"model,omitempty"`
	Config         map[string]any `json:"config"`
	ExcludeTurns   bool           `json:"excludeTurns"`
}

type threadResponse struct {
	Thread threadSummary `json:"thread"`
	Model  string        `json:"model"`
}

const interpretationBaseInstructions = "Interpret bounded Hatmax generator requests. Return only the requested structured output."

const interpretationDeveloperInstructions = "Do not inspect files, run commands, use tools, access external context, or propose implementation details. Hatmax owns planning and execution."
